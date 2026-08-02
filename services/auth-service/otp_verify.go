package main

import (
	"crypto/subtle"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"gas-tam-de/pkg/httpx"

	"github.com/google/uuid"
)

type otpVerifyBody struct {
	Phone string `json:"phone"`
	Code  string `json:"code"`
}

type otpChallengeRow struct {
	ID         string
	CodeHash   string
	ExpiresAt  time.Time
	Attempts   int
	ConsumedAt sql.NullString
}

func (s *otpService) handleOTPVerify(w http.ResponseWriter, r *http.Request) {
	var body otpVerifyBody
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
	code := strings.TrimSpace(body.Code)
	if len(code) != 6 || !allDigits(code) {
		httpx.Error(w, http.StatusBadRequest, "INVALID_CODE", "OTP code must be 6 digits")
		return
	}

	phoneHash := hashPhone(e164, s.phonePepper)
	now := time.Now().UTC()

	tx, err := s.db.Begin()
	if err != nil {
		slog.Error("begin tx", "err", err)
		httpx.Error(w, http.StatusInternalServerError, "INTERNAL", "could not verify OTP")
		return
	}
	defer func() { _ = tx.Rollback() }()

	ch, err := s.lockLatestChallenge(tx, phoneHash)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			httpx.Error(w, http.StatusUnauthorized, "OTP_NOT_FOUND", "no active OTP challenge for this phone")
			return
		}
		slog.Error("load otp challenge", "err", err)
		httpx.Error(w, http.StatusInternalServerError, "INTERNAL", "could not verify OTP")
		return
	}

	if ch.ConsumedAt.Valid {
		httpx.Error(w, http.StatusUnauthorized, "OTP_CONSUMED", "OTP already used; request a new code")
		return
	}
	if !now.Before(ch.ExpiresAt) {
		httpx.Error(w, http.StatusUnauthorized, "OTP_EXPIRED", "OTP expired; request a new code")
		return
	}
	if ch.Attempts >= s.maxAttempts {
		httpx.Error(w, http.StatusTooManyRequests, "OTP_LOCKED", "too many invalid OTP attempts; request a new code")
		return
	}

	wantHash := hashOTPCode(code, ch.ID, s.otpPepper)
	match := subtle.ConstantTimeCompare([]byte(wantHash), []byte(ch.CodeHash)) == 1

	if !match {
		newAttempts := ch.Attempts + 1
		if err := bumpChallengeAttempts(tx, ch.ID, newAttempts); err != nil {
			slog.Error("bump otp attempts", "err", err)
			httpx.Error(w, http.StatusInternalServerError, "INTERNAL", "could not verify OTP")
			return
		}
		if err := tx.Commit(); err != nil {
			slog.Error("commit otp fail", "err", err)
			httpx.Error(w, http.StatusInternalServerError, "INTERNAL", "could not verify OTP")
			return
		}
		remaining := s.maxAttempts - newAttempts
		if remaining <= 0 {
			httpx.Error(w, http.StatusTooManyRequests, "OTP_LOCKED", "too many invalid OTP attempts; request a new code")
			return
		}
		httpx.JSON(w, http.StatusUnauthorized, map[string]any{
			"error": map[string]any{
				"code":               "OTP_INVALID",
				"message":            "invalid OTP code",
				"attempts_remaining": remaining,
			},
		})
		return
	}

	if err := consumeChallenge(tx, ch.ID, now); err != nil {
		slog.Error("consume otp", "err", err)
		httpx.Error(w, http.StatusInternalServerError, "INTERNAL", "could not verify OTP")
		return
	}
	// Invalidate any other open challenges for this phone.
	if err := invalidateOpenChallenges(tx, phoneHash, ch.ID, now); err != nil {
		slog.Error("invalidate otp challenges", "err", err)
		httpx.Error(w, http.StatusInternalServerError, "INTERNAL", "could not verify OTP")
		return
	}

	masked := maskPhoneE164(e164)
	userID, err := s.upsertCustomer(tx, phoneHash, e164, masked, now)
	if err != nil {
		slog.Error("upsert user", "err", err)
		httpx.Error(w, http.StatusInternalServerError, "INTERNAL", "could not verify OTP")
		return
	}

	refreshRaw, refreshHash, err := generateRefreshToken()
	if err != nil {
		slog.Error("generate refresh", "err", err)
		httpx.Error(w, http.StatusInternalServerError, "INTERNAL", "could not issue tokens")
		return
	}
	sessionID := uuid.NewString()
	sessionExp := now.Add(s.refreshTTL)
	if err := insertSession(tx, sessionID, userID, "customer", refreshHash, sessionExp, now); err != nil {
		slog.Error("insert session", "err", err)
		httpx.Error(w, http.StatusInternalServerError, "INTERNAL", "could not issue tokens")
		return
	}

	access, err := issueAccessToken(s.jwtSecret, userID, "customer", masked, sessionID, s.accessTTL, now)
	if err != nil {
		slog.Error("issue access jwt", "err", err)
		httpx.Error(w, http.StatusInternalServerError, "INTERNAL", "could not issue tokens")
		return
	}

	if err := tx.Commit(); err != nil {
		slog.Error("commit otp verify", "err", err)
		httpx.Error(w, http.StatusInternalServerError, "INTERNAL", "could not verify OTP")
		return
	}

	slog.Info("otp verified",
		"user_id", userID,
		"session_id", sessionID,
		"phone_hash", phoneHash,
	)

	httpx.JSON(w, http.StatusOK, map[string]any{
		"ok":            true,
		"access_token":  access,
		"refresh_token": refreshRaw,
		"token_type":    "Bearer",
		"expires_in":    int(s.accessTTL.Seconds()),
		"user": map[string]any{
			"id":           userID,
			"role":         "customer",
			"phone_masked": masked,
		},
	})
}

func (s *otpService) lockLatestChallenge(tx *sql.Tx, phoneHash string) (*otpChallengeRow, error) {
	row := tx.QueryRow(
		`SELECT id, code_hash, expires_at, attempts, consumed_at
		 FROM otp_challenges
		 WHERE phone_hash = ?
		 ORDER BY created_at DESC
		 LIMIT 1`,
		phoneHash,
	)
	var ch otpChallengeRow
	var expiresRaw string
	if err := row.Scan(&ch.ID, &ch.CodeHash, &expiresRaw, &ch.Attempts, &ch.ConsumedAt); err != nil {
		return nil, err
	}
	exp, err := time.Parse(time.RFC3339Nano, expiresRaw)
	if err != nil {
		exp, err = time.Parse(time.RFC3339, expiresRaw)
		if err != nil {
			return nil, fmt.Errorf("parse expires_at: %w", err)
		}
	}
	ch.ExpiresAt = exp
	return &ch, nil
}

func bumpChallengeAttempts(tx *sql.Tx, id string, attempts int) error {
	_, err := tx.Exec(`UPDATE otp_challenges SET attempts = ? WHERE id = ?`, attempts, id)
	return err
}

func consumeChallenge(tx *sql.Tx, id string, at time.Time) error {
	_, err := tx.Exec(
		`UPDATE otp_challenges SET consumed_at = ?, attempts = attempts + 1 WHERE id = ? AND consumed_at IS NULL`,
		at.Format(time.RFC3339Nano), id,
	)
	return err
}

func invalidateOpenChallenges(tx *sql.Tx, phoneHash, exceptID string, at time.Time) error {
	_, err := tx.Exec(
		`UPDATE otp_challenges SET consumed_at = ? WHERE phone_hash = ? AND id != ? AND consumed_at IS NULL`,
		at.Format(time.RFC3339Nano), phoneHash, exceptID,
	)
	return err
}

func (s *otpService) upsertCustomer(tx *sql.Tx, phoneHash, e164, masked string, now time.Time) (string, error) {
	var id string
	err := tx.QueryRow(`SELECT id FROM users WHERE phone_hash = ?`, phoneHash).Scan(&id)
	if err == nil {
		_, err = tx.Exec(`UPDATE users SET phone_masked = ?, updated_at = ? WHERE id = ?`, masked, now.Format(time.RFC3339Nano), id)
		return id, err
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return "", err
	}

	enc, err := encryptPhoneE164(e164, s.phoneKey)
	if err != nil {
		return "", err
	}
	id = uuid.NewString()
	ts := now.Format(time.RFC3339Nano)
	_, err = tx.Exec(
		`INSERT INTO users (id, phone_e164_enc, phone_hash, phone_masked, full_name, created_at, updated_at)
		 VALUES (?, ?, ?, ?, NULL, ?, ?)`,
		id, enc, phoneHash, masked, ts, ts,
	)
	return id, err
}

func insertSession(tx *sql.Tx, id, userID, role, refreshHash string, expiresAt, createdAt time.Time) error {
	_, err := tx.Exec(
		`INSERT INTO sessions (id, user_id, role, refresh_hash, expires_at, revoked_at, created_at)
		 VALUES (?, ?, ?, ?, ?, NULL, ?)`,
		id, userID, role, refreshHash,
		expiresAt.Format(time.RFC3339Nano),
		createdAt.Format(time.RFC3339Nano),
	)
	return err
}

func allDigits(s string) bool {
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}
