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

type adminLoginBody struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type adminAccountRow struct {
	ID           string
	Username     string
	PasswordHash string
	DisplayName  sql.NullString
	DisabledAt   sql.NullString
}

func (s *tokenService) handleAdminLogin(w http.ResponseWriter, r *http.Request) {
	var body adminLoginBody
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&body); err != nil {
		httpx.Error(w, http.StatusBadRequest, "INVALID_JSON", "invalid request body")
		return
	}

	username := strings.TrimSpace(body.Username)
	password := body.Password
	if username == "" || password == "" {
		httpx.Error(w, http.StatusBadRequest, "INVALID_CREDENTIALS", "username and password are required")
		return
	}

	admin, err := loadAdminByUsername(s.db, username)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// Same status/code as wrong password — avoid username enumeration.
			_ = verifyAdminPassword(dummyBcryptHash, password)
			httpx.Error(w, http.StatusUnauthorized, "INVALID_CREDENTIALS", "invalid username or password")
			return
		}
		slog.Error("load admin", "err", err)
		httpx.Error(w, http.StatusInternalServerError, "INTERNAL", "could not login")
		return
	}
	if admin.DisabledAt.Valid {
		httpx.Error(w, http.StatusUnauthorized, "INVALID_CREDENTIALS", "invalid username or password")
		return
	}
	if !verifyAdminPassword(admin.PasswordHash, password) {
		httpx.Error(w, http.StatusUnauthorized, "INVALID_CREDENTIALS", "invalid username or password")
		return
	}

	now := time.Now().UTC()
	refreshRaw, refreshHash, err := generateRefreshToken()
	if err != nil {
		slog.Error("generate refresh", "err", err)
		httpx.Error(w, http.StatusInternalServerError, "INTERNAL", "could not issue tokens")
		return
	}

	sessionID := uuid.NewString()
	sessionExp := now.Add(s.refreshTTL)

	tx, err := s.db.Begin()
	if err != nil {
		slog.Error("begin tx", "err", err)
		httpx.Error(w, http.StatusInternalServerError, "INTERNAL", "could not login")
		return
	}
	defer func() { _ = tx.Rollback() }()

	if err := insertSession(tx, sessionID, admin.ID, "admin", refreshHash, sessionExp, now); err != nil {
		slog.Error("insert session", "err", err)
		httpx.Error(w, http.StatusInternalServerError, "INTERNAL", "could not issue tokens")
		return
	}

	access, err := issueAccessToken(s.jwtSecret, admin.ID, "admin", "", sessionID, s.accessTTL, now)
	if err != nil {
		slog.Error("issue access jwt", "err", err)
		httpx.Error(w, http.StatusInternalServerError, "INTERNAL", "could not issue tokens")
		return
	}

	if err := tx.Commit(); err != nil {
		slog.Error("commit admin login", "err", err)
		httpx.Error(w, http.StatusInternalServerError, "INTERNAL", "could not login")
		return
	}

	slog.Info("admin login",
		"admin_id", admin.ID,
		"session_id", sessionID,
		"username", admin.Username,
	)

	user := map[string]any{
		"id":       admin.ID,
		"role":     "admin",
		"username": admin.Username,
	}
	if admin.DisplayName.Valid && admin.DisplayName.String != "" {
		user["display_name"] = admin.DisplayName.String
	}

	httpx.JSON(w, http.StatusOK, map[string]any{
		"ok":            true,
		"access_token":  access,
		"refresh_token": refreshRaw,
		"token_type":    "Bearer",
		"expires_in":    int(s.accessTTL.Seconds()),
		"user":          user,
	})
}

func loadAdminByUsername(db *sql.DB, username string) (*adminAccountRow, error) {
	row := db.QueryRow(`
		SELECT id, username, password_hash, display_name, disabled_at
		FROM admin_accounts WHERE username = ?
	`, username)
	var a adminAccountRow
	if err := row.Scan(&a.ID, &a.Username, &a.PasswordHash, &a.DisplayName, &a.DisabledAt); err != nil {
		return nil, err
	}
	return &a, nil
}

// Precomputed bcrypt of a fixed string — used only to burn similar CPU when username is missing.
const dummyBcryptHash = "$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy"
