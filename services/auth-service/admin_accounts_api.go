package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"gas-tam-de/pkg/httpx"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// adminAccountService manages username/password accounts that can sign in at
// /v1/auth/admin/login. The gateway also protects these routes, but handlers
// verify the forwarded identity so auth-service is safe when reached directly.
type adminAccountService struct {
	db *sql.DB
}

type adminAccountView struct {
	ID          string `json:"id"`
	Username    string `json:"username"`
	DisplayName string `json:"display_name,omitempty"`
	CreatedAt   string `json:"created_at"`
	IsSelf      bool   `json:"is_self"`
}

type createAdminAccountBody struct {
	Username    string `json:"username"`
	Password    string `json:"password"`
	DisplayName string `json:"display_name"`
}

type updateAdminAccountBody struct {
	Username        *string `json:"username"`
	DisplayName     *string `json:"display_name"`
	NewPassword     *string `json:"new_password"`
	CurrentPassword string  `json:"current_password"`
}

func (s *adminAccountService) handleList(w http.ResponseWriter, r *http.Request) {
	actor, ok := requireAdminIdentity(w, r)
	if !ok {
		return
	}
	rows, err := s.db.Query(`
		SELECT id, username, display_name, created_at
		FROM admin_accounts
		WHERE disabled_at IS NULL
		ORDER BY lower(username), created_at
	`)
	if err != nil {
		slog.Error("list admin accounts", "err", err)
		httpx.Error(w, http.StatusInternalServerError, "INTERNAL", "could not load admin accounts")
		return
	}
	defer rows.Close()

	items := make([]adminAccountView, 0)
	for rows.Next() {
		var item adminAccountView
		var display sql.NullString
		if err := rows.Scan(&item.ID, &item.Username, &display, &item.CreatedAt); err != nil {
			slog.Error("scan admin account", "err", err)
			httpx.Error(w, http.StatusInternalServerError, "INTERNAL", "could not load admin accounts")
			return
		}
		if display.Valid {
			item.DisplayName = display.String
		}
		item.IsSelf = item.ID == actor
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		slog.Error("iterate admin accounts", "err", err)
		httpx.Error(w, http.StatusInternalServerError, "INTERNAL", "could not load admin accounts")
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"admin_accounts": items})
}

func (s *adminAccountService) handleCreate(w http.ResponseWriter, r *http.Request) {
	actor, ok := requireAdminIdentity(w, r)
	if !ok {
		return
	}
	var body createAdminAccountBody
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&body); err != nil {
		httpx.Error(w, http.StatusBadRequest, "INVALID_JSON", "invalid request body")
		return
	}
	username, display, ok := validateAdminAccountInput(w, body.Username, body.DisplayName)
	if !ok || !validateAdminPassword(w, body.Password) {
		return
	}
	if adminUsernameExists(s.db, username, "") {
		httpx.Error(w, http.StatusConflict, "USERNAME_EXISTS", "username already exists")
		return
	}
	hash, err := hashAdminPassword(body.Password)
	if err != nil {
		slog.Error("hash admin password", "err", err)
		httpx.Error(w, http.StatusInternalServerError, "INTERNAL", "could not create admin account")
		return
	}
	item := adminAccountView{
		ID: uuid.NewString(), Username: username, DisplayName: display,
		CreatedAt: time.Now().UTC().Format(time.RFC3339Nano),
	}
	var displayArg any
	if display != "" {
		displayArg = display
	}
	_, err = s.db.Exec(`
		INSERT INTO admin_accounts (id, username, password_hash, display_name, created_at, disabled_at)
		VALUES (?, ?, ?, ?, ?, NULL)
	`, item.ID, item.Username, hash, displayArg, item.CreatedAt)
	if err != nil {
		if isUniqueConstraint(err) {
			httpx.Error(w, http.StatusConflict, "USERNAME_EXISTS", "username already exists")
			return
		}
		slog.Error("create admin account", "err", err)
		httpx.Error(w, http.StatusInternalServerError, "INTERNAL", "could not create admin account")
		return
	}
	slog.Info("admin account created", "id", item.ID, "username", item.Username, "actor_id", actor)
	httpx.JSON(w, http.StatusCreated, item)
}

func (s *adminAccountService) handleUpdate(w http.ResponseWriter, r *http.Request) {
	actor, ok := requireAdminIdentity(w, r)
	if !ok {
		return
	}
	id := strings.TrimSpace(chi.URLParam(r, "id"))
	var body updateAdminAccountBody
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if id == "" || dec.Decode(&body) != nil {
		httpx.Error(w, http.StatusBadRequest, "INVALID_JSON", "invalid request body")
		return
	}
	if body.Username == nil && body.DisplayName == nil && body.NewPassword == nil {
		httpx.Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "no account changes supplied")
		return
	}

	var current adminAccountRow
	var createdAt string
	err := s.db.QueryRow(`
		SELECT id, username, password_hash, display_name, disabled_at, created_at
		FROM admin_accounts WHERE id = ? AND disabled_at IS NULL
	`, id).Scan(&current.ID, &current.Username, &current.PasswordHash, &current.DisplayName, &current.DisabledAt, &createdAt)
	if errors.Is(err, sql.ErrNoRows) {
		httpx.Error(w, http.StatusNotFound, "NOT_FOUND", "admin account not found")
		return
	}
	if err != nil {
		slog.Error("load admin account", "err", err)
		httpx.Error(w, http.StatusInternalServerError, "INTERNAL", "could not update admin account")
		return
	}

	username := current.Username
	display := ""
	if current.DisplayName.Valid {
		display = current.DisplayName.String
	}
	if body.Username != nil {
		username = *body.Username
	}
	if body.DisplayName != nil {
		display = *body.DisplayName
	}
	username, display, ok = validateAdminAccountInput(w, username, display)
	if !ok {
		return
	}
	if body.NewPassword != nil && !validateAdminPassword(w, *body.NewPassword) {
		return
	}
	if adminUsernameExists(s.db, username, id) {
		httpx.Error(w, http.StatusConflict, "USERNAME_EXISTS", "username already exists")
		return
	}
	// Re-authenticate when an account changes its own login credentials. A
	// different admin may reset the account, which is the recovery path when its
	// owner forgot the old password.
	credentialsChanged := username != current.Username || body.NewPassword != nil
	if id == actor && credentialsChanged && !verifyAdminPassword(current.PasswordHash, body.CurrentPassword) {
		httpx.Error(w, http.StatusUnauthorized, "CURRENT_PASSWORD_INVALID", "current password is incorrect")
		return
	}

	passwordHash := current.PasswordHash
	if body.NewPassword != nil {
		passwordHash, err = hashAdminPassword(*body.NewPassword)
		if err != nil {
			slog.Error("hash new admin password", "err", err)
			httpx.Error(w, http.StatusInternalServerError, "INTERNAL", "could not update admin account")
			return
		}
	}
	var displayArg any
	if display != "" {
		displayArg = display
	}
	_, err = s.db.Exec(`
		UPDATE admin_accounts SET username = ?, display_name = ?, password_hash = ?
		WHERE id = ? AND disabled_at IS NULL
	`, username, displayArg, passwordHash, id)
	if err != nil {
		if isUniqueConstraint(err) {
			httpx.Error(w, http.StatusConflict, "USERNAME_EXISTS", "username already exists")
			return
		}
		slog.Error("update admin account", "err", err)
		httpx.Error(w, http.StatusInternalServerError, "INTERNAL", "could not update admin account")
		return
	}
	item := adminAccountView{ID: id, Username: username, DisplayName: display, CreatedAt: createdAt, IsSelf: id == actor}
	slog.Info("admin account updated", "id", id, "username", username, "password_changed", body.NewPassword != nil, "actor_id", actor)
	httpx.JSON(w, http.StatusOK, item)
}

func validateAdminAccountInput(w http.ResponseWriter, rawUsername, rawDisplay string) (string, string, bool) {
	username := strings.TrimSpace(rawUsername)
	display := strings.TrimSpace(rawDisplay)
	if utf8.RuneCountInString(username) < 3 || utf8.RuneCountInString(username) > 80 {
		httpx.Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "username must be 3 to 80 characters")
		return "", "", false
	}
	if strings.IndexFunc(username, unicode.IsSpace) >= 0 {
		httpx.Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "username cannot contain whitespace")
		return "", "", false
	}
	if utf8.RuneCountInString(display) > 80 {
		httpx.Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "display name too long")
		return "", "", false
	}
	return username, display, true
}

func validateAdminPassword(w http.ResponseWriter, password string) bool {
	// bcrypt accepts at most 72 bytes; keeping the lower bound explicit avoids
	// accidentally creating weak admin credentials from this privileged API.
	if len(password) < 8 || len([]byte(password)) > 72 {
		httpx.Error(w, http.StatusBadRequest, "WEAK_PASSWORD", "password must be 8 to 72 bytes")
		return false
	}
	return true
}

func adminUsernameExists(db *sql.DB, username, exceptID string) bool {
	var n int
	_ = db.QueryRow(`SELECT COUNT(*) FROM admin_accounts WHERE username = ? AND id <> ?`, username, exceptID).Scan(&n)
	return n > 0
}

func isUniqueConstraint(err error) bool {
	return err != nil && strings.Contains(strings.ToLower(err.Error()), "unique constraint")
}
