package main

import (
	"database/sql"
	"embed"
	"log/slog"
	"os"
	"strings"
	"time"

	"gas-tam-de/pkg/config"
	"gas-tam-de/pkg/httpx"
	"gas-tam-de/pkg/sqlite"
)

const serviceName = "auth-service"

//go:embed schema.sql
var schemaFS embed.FS

func main() {
	addr := config.ListenAddr("AUTH_ADDR", ":8081")
	dbPath := config.Get("AUTH_DB", "data/auth.db")

	db, err := sqlite.Open(dbPath)
	if err != nil {
		slog.Error("open db", "err", err)
		os.Exit(1)
	}
	defer db.Close()

	if err := migrate(db); err != nil {
		slog.Error("migrate", "err", err)
		os.Exit(1)
	}

	if err := seedAdminAccount(db, loadAdminSeedConfig()); err != nil {
		slog.Error("seed admin", "err", err)
		os.Exit(1)
	}

	phonePepper := loadPhonePepper()
	if err := seedAdminPhones(db, loadAdminPhonesSeed(), phonePepper); err != nil {
		slog.Error("seed admin phones", "err", err)
		os.Exit(1)
	}

	accessTTLSec := config.GetInt("JWT_ACCESS_TTL_SEC", 900)
	refreshTTLSec := config.GetInt("JWT_REFRESH_TTL_SEC", 2592000) // 30 days
	jwtSecret := config.Get("JWT_SECRET", "dev-jwt-secret-change-me")
	accessTTL := time.Duration(accessTTLSec) * time.Second
	refreshTTL := time.Duration(refreshTTLSec) * time.Second

	// api-gateway must print the same fingerprint, otherwise it rejects every
	// token this service signs ("invalid or expired access token").
	slog.Info("access token signing key",
		"jwt_secret_fp", config.SecretFingerprint(jwtSecret),
		"access_ttl_sec", accessTTLSec,
		"refresh_ttl_sec", refreshTTLSec,
	)

	otp := newOTPService(db, jwtSecret, accessTTL, refreshTTL)
	tokens := newTokenService(db, jwtSecret, accessTTL, refreshTTL)
	me := &meService{db: db}
	adminPhones := &adminPhoneService{db: db, phonePepper: phonePepper}

	r := httpx.NewRouter(serviceName)
	httpx.MountHealth(r, serviceName)

	r.Post("/v1/auth/otp/request", otp.handleOTPRequest)
	r.Post("/v1/auth/otp/verify", otp.handleOTPVerify)
	r.Post("/v1/auth/admin/login", tokens.handleAdminLogin)
	r.Post("/v1/auth/refresh", tokens.handleRefresh)
	r.Get("/v1/me", me.handleGetMe)
	r.Patch("/v1/me", me.handlePatchMe)
	r.Get("/v1/admin/admin-phones", adminPhones.handleList)
	r.Post("/v1/admin/admin-phones", adminPhones.handleCreate)
	r.Delete("/v1/admin/admin-phones/{id}", adminPhones.handleDelete)

	if err := httpx.ListenAndServe(addr, serviceName, r); err != nil {
		slog.Error("server stopped", "err", err)
		os.Exit(1)
	}
}

func migrate(db *sql.DB) error {
	sqlBytes, err := schemaFS.ReadFile("schema.sql")
	if err != nil {
		return err
	}
	_, err = db.Exec(string(sqlBytes))
	return err
}

func newOTPService(db *sql.DB, jwtSecret string, accessTTL, refreshTTL time.Duration) *otpService {
	cooldown := config.GetInt("OTP_COOLDOWN_SEC", 60)
	ttlSec := config.GetInt("OTP_TTL_SEC", 300)
	maxPhone := config.GetInt("OTP_MAX_PER_PHONE_HOUR", 5)
	maxIP := config.GetInt("OTP_MAX_PER_IP_HOUR", 20)
	maxAttempts := config.GetInt("OTP_MAX_ATTEMPTS", 5)

	otpPepper := config.Get("OTP_HASH_PEPPER", config.Get("JWT_SECRET", "dev-otp-pepper-change-me"))
	phoneEncKey := config.Get("PHONE_ENC_KEY", "dev-phone-enc-key-32bytes-min!!")

	// Default local/dev reveals OTP in JSON so clients can test without real SMS.
	// Set OTP_DEV_REVEAL=0 in non-dev environments.
	devReveal := true
	if v := strings.TrimSpace(os.Getenv("OTP_DEV_REVEAL")); v != "" {
		devReveal = strings.EqualFold(v, "1") || strings.EqualFold(v, "true")
	}

	sms := newSMSSenderFromEnv()
	slog.Info("sms provider selected", "provider", smsProviderName(sms))

	return &otpService{
		db:           db,
		limiter:      newOTPRateLimiter(cooldown, maxPhone, maxIP),
		sms:          sms,
		phonePepper:  loadPhonePepper(),
		otpPepper:    otpPepper,
		phoneKey:     derivePhoneKey(phoneEncKey),
		jwtSecret:    jwtSecret,
		ttl:          time.Duration(ttlSec) * time.Second,
		accessTTL:    accessTTL,
		refreshTTL:   refreshTTL,
		maxAttempts:  maxAttempts,
		cooldownSec:  cooldown,
		devRevealOTP: devReveal,
	}
}

// loadPhonePepper resolves the pepper behind users.phone_hash. It is the
// customer identity key, so every caller must derive it the same way — changing
// it against a populated auth.db orphans existing accounts.
func loadPhonePepper() string {
	return config.Get("PHONE_HASH_PEPPER", config.Get("PHONE_ENC_KEY", "dev-phone-pepper-change-me"))
}

// loadAdminPhonesSeed returns the numbers bootstrapped into the admin
// allow-list. Setting ADMIN_PHONES to an empty value seeds nothing, which is
// why LookupEnv is used instead of the usual fallback helper.
func loadAdminPhonesSeed() string {
	if v, ok := os.LookupEnv("ADMIN_PHONES"); ok {
		return v
	}
	return defaultAdminPhones
}

const defaultAdminPhones = "0909777020"

func smsProviderName(s SMSSender) string {
	switch s.(type) {
	case *MockSMSSender:
		return "mock"
	case *StringeeSMSSender:
		return "stringee"
	case *ProductionSMSSender:
		return "production"
	default:
		return "unknown"
	}
}
