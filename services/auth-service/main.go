package main

import (
	"context"
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
	googleAuth := newGoogleAuthService(
		db, jwtSecret, accessTTL, refreshTTL,
		newGoogleIDTokenVerifier(config.Get("GOOGLE_CLIENT_IDS", "")),
	)
	me := &meService{
		db:       db,
		phoneKey: derivePhoneKey(config.Get("PHONE_ENC_KEY", "dev-phone-enc-key-32bytes-min!!")),
	}
	adminPhones := &adminPhoneService{db: db, phonePepper: phonePepper}
	adminAccounts := &adminAccountService{db: db}

	r := httpx.NewRouter(serviceName)
	httpx.MountHealth(r, serviceName)

	r.Post("/v1/auth/otp/request", otp.handleOTPRequest)
	r.Post("/v1/auth/otp/verify", otp.handleOTPVerify)
	r.Post("/v1/auth/admin/login", tokens.handleAdminLogin)
	r.Post("/v1/auth/google", googleAuth.handleLogin)
	r.Post("/v1/auth/refresh", tokens.handleRefresh)
	r.Post("/v1/auth/logout", tokens.handleLogout)
	r.Get("/v1/me", me.handleGetMe)
	r.Patch("/v1/me", me.handlePatchMe)
	r.Get("/v1/admin/admin-phones", adminPhones.handleList)
	r.Post("/v1/admin/admin-phones", adminPhones.handleCreate)
	r.Delete("/v1/admin/admin-phones/{id}", adminPhones.handleDelete)
	r.Get("/v1/admin/admin-accounts", adminAccounts.handleList)
	r.Post("/v1/admin/admin-accounts", adminAccounts.handleCreate)
	r.Patch("/v1/admin/admin-accounts/{id}", adminAccounts.handleUpdate)

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
	if _, err = db.Exec(string(sqlBytes)); err != nil {
		return err
	}
	// CREATE TABLE IF NOT EXISTS does not evolve databases already deployed.
	// Keep these additive migrations idempotent for existing OTP installations.
	for _, column := range []struct {
		table, name, definition string
	}{
		{"users", "google_sub", "TEXT"},
		{"users", "email", "TEXT"},
		{"users", "picture_url", "TEXT"},
		{"users", "contact_phone_e164_enc", "BLOB"},
		{"users", "contact_phone_masked", "TEXT"},
		{"sessions", "persistent", "INTEGER NOT NULL DEFAULT 0"},
	} {
		if err := ensureColumn(context.Background(), db, column.table, column.name, column.definition); err != nil {
			return err
		}
	}
	_, err = db.Exec(`CREATE UNIQUE INDEX IF NOT EXISTS idx_users_google_sub ON users(google_sub) WHERE google_sub IS NOT NULL`)
	return err
}

func ensureColumn(ctx context.Context, db *sql.DB, table, name, definition string) error {
	rows, err := db.QueryContext(ctx, "PRAGMA table_info("+table+")")
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var cid int
		var columnName, columnType string
		var notNull, pk int
		var defaultValue sql.NullString
		if err := rows.Scan(&cid, &columnName, &columnType, &notNull, &defaultValue, &pk); err != nil {
			return err
		}
		if columnName == name {
			return nil
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}
	if err := rows.Close(); err != nil {
		return err
	}
	_, err = db.ExecContext(ctx, "ALTER TABLE "+table+" ADD COLUMN "+name+" "+definition)
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
