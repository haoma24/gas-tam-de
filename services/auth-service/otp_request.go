package main

import (
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"math/big"
	"net"
	"net/http"
	"strings"
	"time"

	"gas-tam-de/pkg/httpx"

	"github.com/google/uuid"
)

type otpRequestBody struct {
	Phone string `json:"phone"`
}

type otpService struct {
	db           *sql.DB
	limiter      *otpRateLimiter
	sms          SMSSender
	phonePepper  string
	otpPepper    string
	phoneKey     []byte
	jwtSecret    string
	ttl          time.Duration
	accessTTL    time.Duration
	refreshTTL   time.Duration
	maxAttempts  int
	cooldownSec  int
	devRevealOTP bool
}

func (s *otpService) handleOTPRequest(w http.ResponseWriter, r *http.Request) {
	var body otpRequestBody
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&body); err != nil {
		httpx.Error(w, http.StatusBadRequest, "INVALID_JSON", "invalid request body")
		return
	}

	e164, err := normalizePhoneVN(body.Phone)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "INVALID_PHONE", "phone must be a valid Vietnam mobile number")
		return
	}

	phoneHash := hashPhone(e164, s.phonePepper)
	ip := clientIP(r)
	now := time.Now().UTC()

	rl := s.limiter.Allow(phoneHash, ip, now)
	if !rl.Allowed {
		w.Header().Set("Retry-After", fmt.Sprintf("%d", rl.RetryAfterSec))
		httpx.JSON(w, http.StatusTooManyRequests, map[string]any{
			"error": map[string]any{
				"code":             "RATE_LIMITED",
				"message":          "too many OTP requests; try again later",
				"retry_after_sec":  rl.RetryAfterSec,
				"limit_reason":     rl.Reason,
			},
		})
		return
	}

	code, err := generateOTPCode(6)
	if err != nil {
		slog.Error("generate otp", "err", err)
		httpx.Error(w, http.StatusInternalServerError, "INTERNAL", "could not issue OTP")
		return
	}

	challengeID := uuid.NewString()
	codeHash := hashOTPCode(code, challengeID, s.otpPepper)
	expiresAt := now.Add(s.ttl)

	if err := s.insertChallenge(challengeID, phoneHash, codeHash, expiresAt, now); err != nil {
		slog.Error("insert otp challenge", "err", err)
		httpx.Error(w, http.StatusInternalServerError, "INTERNAL", "could not issue OTP")
		return
	}

	if err := s.sms.SendOTP(r.Context(), e164, code); err != nil {
		slog.Error("sms send failed",
			"err", err,
			"challenge_id", challengeID,
			"phone_hash", phoneHash,
		)
		httpx.Error(w, http.StatusBadGateway, "SMS_FAILED", "could not send OTP SMS")
		return
	}

	// Never log raw OTP.
	slog.Info("otp issued",
		"challenge_id", challengeID,
		"phone_hash", phoneHash,
		"expires_at", expiresAt.Format(time.RFC3339),
		"ip", ip,
	)

	resp := map[string]any{
		"ok":               true,
		"phone_masked":     maskPhoneE164(e164),
		"expires_in_sec":   int(s.ttl.Seconds()),
		"resend_after_sec": s.cooldownSec,
	}
	if s.devRevealOTP {
		resp["dev_code"] = code
	}
	httpx.JSON(w, http.StatusOK, resp)
}

func (s *otpService) insertChallenge(id, phoneHash, codeHash string, expiresAt, createdAt time.Time) error {
	_, err := s.db.Exec(
		`INSERT INTO otp_challenges (id, phone_hash, code_hash, expires_at, attempts, consumed_at, created_at)
		 VALUES (?, ?, ?, ?, 0, NULL, ?)`,
		id, phoneHash, codeHash,
		expiresAt.Format(time.RFC3339Nano),
		createdAt.Format(time.RFC3339Nano),
	)
	return err
}

func generateOTPCode(digits int) (string, error) {
	max := big.NewInt(1)
	for i := 0; i < digits; i++ {
		max.Mul(max, big.NewInt(10))
	}
	n, err := rand.Int(rand.Reader, max)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%0*d", digits, n.Int64()), nil
}

func hashOTPCode(code, challengeID, pepper string) string {
	sum := sha256.Sum256([]byte(pepper + ":" + challengeID + ":" + code))
	return hex.EncodeToString(sum[:])
}

func clientIP(r *http.Request) string {
	// RealIP middleware sets RemoteAddr; still accept X-Forwarded-For first hop if present.
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		return strings.TrimSpace(parts[0])
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
