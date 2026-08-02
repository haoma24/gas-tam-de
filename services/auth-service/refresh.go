package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"gas-tam-de/pkg/httpx"

	"github.com/google/uuid"
)

type refreshBody struct {
	RefreshToken string `json:"refresh_token"`
}

type sessionRow struct {
	ID          string
	UserID      string
	Role        string
	RefreshHash string
	ExpiresAt   time.Time
	RevokedAt   sql.NullString
}

func (s *tokenService) handleRefresh(w http.ResponseWriter, r *http.Request) {
	var body refreshBody
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&body); err != nil {
		httpx.Error(w, http.StatusBadRequest, "INVALID_JSON", "invalid request body")
		return
	}

	raw := strings.TrimSpace(body.RefreshToken)
	if raw == "" {
		httpx.Error(w, http.StatusBadRequest, "INVALID_TOKEN", "refresh_token is required")
		return
	}

	now := time.Now().UTC()
	wantHash := hashRefreshToken(raw)

	tx, err := s.db.Begin()
	if err != nil {
		slog.Error("begin tx", "err", err)
		httpx.Error(w, http.StatusInternalServerError, "INTERNAL", "could not refresh")
		return
	}
	defer func() { _ = tx.Rollback() }()

	sess, err := lockSessionByRefreshHash(tx, wantHash)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			httpx.Error(w, http.StatusUnauthorized, "INVALID_TOKEN", "invalid or expired refresh token")
			return
		}
		slog.Error("load session", "err", err)
		httpx.Error(w, http.StatusInternalServerError, "INTERNAL", "could not refresh")
		return
	}
	if sess.RevokedAt.Valid {
		httpx.Error(w, http.StatusUnauthorized, "INVALID_TOKEN", "invalid or expired refresh token")
		return
	}
	if !now.Before(sess.ExpiresAt) {
		httpx.Error(w, http.StatusUnauthorized, "INVALID_TOKEN", "invalid or expired refresh token")
		return
	}

	if err := revokeSession(tx, sess.ID, now); err != nil {
		slog.Error("revoke session", "err", err)
		httpx.Error(w, http.StatusInternalServerError, "INTERNAL", "could not refresh")
		return
	}

	refreshRaw, refreshHash, err := generateRefreshToken()
	if err != nil {
		slog.Error("generate refresh", "err", err)
		httpx.Error(w, http.StatusInternalServerError, "INTERNAL", "could not issue tokens")
		return
	}

	newSessionID := uuid.NewString()
	sessionExp := now.Add(s.refreshTTL)
	if err := insertSession(tx, newSessionID, sess.UserID, sess.Role, refreshHash, sessionExp, now); err != nil {
		slog.Error("insert rotated session", "err", err)
		httpx.Error(w, http.StatusInternalServerError, "INTERNAL", "could not issue tokens")
		return
	}

	phoneMasked := ""
	userPayload := map[string]any{
		"id":   sess.UserID,
		"role": sess.Role,
	}
	switch sess.Role {
	case "admin":
		admin, err := loadAdminByID(tx, sess.UserID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				httpx.Error(w, http.StatusUnauthorized, "INVALID_TOKEN", "invalid or expired refresh token")
				return
			}
			slog.Error("load admin for refresh", "err", err)
			httpx.Error(w, http.StatusInternalServerError, "INTERNAL", "could not refresh")
			return
		}
		if admin.DisabledAt.Valid {
			httpx.Error(w, http.StatusUnauthorized, "INVALID_TOKEN", "invalid or expired refresh token")
			return
		}
		userPayload["username"] = admin.Username
		if admin.DisplayName.Valid && admin.DisplayName.String != "" {
			userPayload["display_name"] = admin.DisplayName.String
		}
	case "customer":
		masked, err := loadCustomerPhoneMasked(tx, sess.UserID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				httpx.Error(w, http.StatusUnauthorized, "INVALID_TOKEN", "invalid or expired refresh token")
				return
			}
			slog.Error("load customer for refresh", "err", err)
			httpx.Error(w, http.StatusInternalServerError, "INTERNAL", "could not refresh")
			return
		}
		phoneMasked = masked
		userPayload["phone_masked"] = masked
	default:
		httpx.Error(w, http.StatusUnauthorized, "INVALID_TOKEN", "invalid or expired refresh token")
		return
	}

	access, err := issueAccessToken(s.jwtSecret, sess.UserID, sess.Role, phoneMasked, newSessionID, s.accessTTL, now)
	if err != nil {
		slog.Error("issue access jwt", "err", err)
		httpx.Error(w, http.StatusInternalServerError, "INTERNAL", "could not issue tokens")
		return
	}

	if err := tx.Commit(); err != nil {
		slog.Error("commit refresh", "err", err)
		httpx.Error(w, http.StatusInternalServerError, "INTERNAL", "could not refresh")
		return
	}

	slog.Info("token refreshed",
		"user_id", sess.UserID,
		"role", sess.Role,
		"old_session_id", sess.ID,
		"session_id", newSessionID,
	)

	httpx.JSON(w, http.StatusOK, map[string]any{
		"ok":            true,
		"access_token":  access,
		"refresh_token": refreshRaw,
		"token_type":    "Bearer",
		"expires_in":    int(s.accessTTL.Seconds()),
		"user":          userPayload,
	})
}

func lockSessionByRefreshHash(tx *sql.Tx, refreshHash string) (*sessionRow, error) {
	row := tx.QueryRow(`
		SELECT id, user_id, role, refresh_hash, expires_at, revoked_at
		FROM sessions
		WHERE refresh_hash = ?
	`, refreshHash)
	var s sessionRow
	var expiresRaw string
	if err := row.Scan(&s.ID, &s.UserID, &s.Role, &s.RefreshHash, &expiresRaw, &s.RevokedAt); err != nil {
		return nil, err
	}
	exp, err := parseRFC3339Flexible(expiresRaw)
	if err != nil {
		return nil, err
	}
	s.ExpiresAt = exp
	return &s, nil
}

func revokeSession(tx *sql.Tx, id string, at time.Time) error {
	_, err := tx.Exec(
		`UPDATE sessions SET revoked_at = ? WHERE id = ? AND revoked_at IS NULL`,
		at.Format(time.RFC3339Nano), id,
	)
	return err
}

func loadAdminByID(tx *sql.Tx, id string) (*adminAccountRow, error) {
	row := tx.QueryRow(`
		SELECT id, username, password_hash, display_name, disabled_at
		FROM admin_accounts WHERE id = ?
	`, id)
	var a adminAccountRow
	if err := row.Scan(&a.ID, &a.Username, &a.PasswordHash, &a.DisplayName, &a.DisabledAt); err != nil {
		return nil, err
	}
	return &a, nil
}

func loadCustomerPhoneMasked(tx *sql.Tx, userID string) (string, error) {
	var masked string
	err := tx.QueryRow(`SELECT phone_masked FROM users WHERE id = ?`, userID).Scan(&masked)
	return masked, err
}
