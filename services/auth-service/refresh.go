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

// handleLogout revokes the current device session. The response is idempotent
// so callers can always clear local state, even when the token was rotated or
// already revoked.
func (s *tokenService) handleLogout(w http.ResponseWriter, r *http.Request) {
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
	_, err := s.db.Exec(
		`UPDATE sessions SET revoked_at = ? WHERE refresh_hash = ? AND revoked_at IS NULL`,
		time.Now().UTC().Format(time.RFC3339Nano), hashRefreshToken(raw),
	)
	if err != nil {
		slog.Error("logout session", "err", err)
		httpx.Error(w, http.StatusInternalServerError, "INTERNAL", "could not logout")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

type sessionRow struct {
	ID          string
	UserID      string
	Role        string
	RefreshHash string
	ExpiresAt   time.Time
	Persistent  bool
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
	if !sess.Persistent && !now.Before(sess.ExpiresAt) {
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

	principal, err := resolveRefreshPrincipal(tx, sess)
	if err != nil {
		if errors.Is(err, errPrincipalGone) {
			httpx.Error(w, http.StatusUnauthorized, "INVALID_TOKEN", "invalid or expired refresh token")
			return
		}
		slog.Error("resolve principal for refresh", "err", err)
		httpx.Error(w, http.StatusInternalServerError, "INTERNAL", "could not refresh")
		return
	}

	newSessionID := uuid.NewString()
	sessionExp := now.Add(s.refreshTTL)
	if err := insertSession(tx, newSessionID, sess.UserID, principal.Role, refreshHash, sessionExp, now, sess.Persistent); err != nil {
		slog.Error("insert rotated session", "err", err)
		httpx.Error(w, http.StatusInternalServerError, "INTERNAL", "could not issue tokens")
		return
	}

	access, err := issueAccessToken(
		s.jwtSecret, sess.UserID, principal.Role, principal.PhoneMasked, newSessionID, s.accessTTL, now,
	)
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
		"role", principal.Role,
		"old_session_id", sess.ID,
		"session_id", newSessionID,
	)

	httpx.JSON(w, http.StatusOK, map[string]any{
		"ok":            true,
		"access_token":  access,
		"refresh_token": refreshRaw,
		"token_type":    "Bearer",
		"expires_in":    int(s.accessTTL.Seconds()),
		"user":          principal.userPayload(sess.UserID),
	})
}

// errPrincipalGone marks a session whose account no longer exists, is disabled,
// or carries an unknown role — the caller turns it into a 401.
var errPrincipalGone = errors.New("refresh principal unavailable")

// refreshPrincipal is who a rotated session belongs to and what it may do.
type refreshPrincipal struct {
	Role        string
	PhoneMasked string
	Username    string
	DisplayName string
	Email       string
	PictureURL  string
}

func (p refreshPrincipal) userPayload(userID string) map[string]any {
	out := map[string]any{"id": userID, "role": p.Role}
	if p.PhoneMasked != "" {
		out["phone_masked"] = p.PhoneMasked
	}
	if p.Username != "" {
		out["username"] = p.Username
	}
	if p.DisplayName != "" {
		out["display_name"] = p.DisplayName
	}
	if p.Email != "" {
		out["email"] = p.Email
	}
	if p.PictureURL != "" {
		out["picture_url"] = p.PictureURL
	}
	return out
}

// resolveRefreshPrincipal re-derives the role for a rotating session.
//
// A username/password admin is identified by admin_accounts. A phone admin
// signs in through the customer OTP flow, so its identity lives in users and
// its privilege in admin_phones — which is re-read here, letting the allow-list
// grant or revoke admin on the next rotation instead of on the next login.
func resolveRefreshPrincipal(tx *sql.Tx, sess *sessionRow) (refreshPrincipal, error) {
	if sess.Role == roleAdmin {
		admin, err := loadAdminByID(tx, sess.UserID)
		switch {
		case err == nil:
			if admin.DisabledAt.Valid {
				return refreshPrincipal{}, errPrincipalGone
			}
			p := refreshPrincipal{Role: roleAdmin, Username: admin.Username}
			if admin.DisplayName.Valid {
				p.DisplayName = admin.DisplayName.String
			}
			return p, nil
		case !errors.Is(err, sql.ErrNoRows):
			return refreshPrincipal{}, err
		}
	} else if sess.Role != roleCustomer {
		return refreshPrincipal{}, errPrincipalGone
	}

	user, err := loadPhoneUser(tx, sess.UserID)
	if errors.Is(err, sql.ErrNoRows) {
		return refreshPrincipal{}, errPrincipalGone
	}
	if err != nil {
		return refreshPrincipal{}, err
	}
	if user.GoogleSub.Valid {
		return refreshPrincipal{
			Role:        roleCustomer,
			PhoneMasked: user.PhoneMasked,
			DisplayName: user.FullName.String,
			Email:       user.Email.String,
			PictureURL:  user.PictureURL.String,
		}, nil
	}
	role, err := roleForPhone(tx, user.PhoneHash)
	if err != nil {
		return refreshPrincipal{}, err
	}
	return refreshPrincipal{Role: role, PhoneMasked: user.PhoneMasked}, nil
}

func lockSessionByRefreshHash(tx *sql.Tx, refreshHash string) (*sessionRow, error) {
	row := tx.QueryRow(`
		SELECT id, user_id, role, refresh_hash, expires_at, persistent, revoked_at
		FROM sessions
		WHERE refresh_hash = ?
	`, refreshHash)
	var s sessionRow
	var expiresRaw string
	if err := row.Scan(&s.ID, &s.UserID, &s.Role, &s.RefreshHash, &expiresRaw, &s.Persistent, &s.RevokedAt); err != nil {
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

// phoneUser is the OTP-backed identity behind a customer or phone-admin session.
type phoneUser struct {
	PhoneHash   string
	PhoneMasked string
	GoogleSub   sql.NullString
	Email       sql.NullString
	FullName    sql.NullString
	PictureURL  sql.NullString
}

func loadPhoneUser(tx *sql.Tx, userID string) (phoneUser, error) {
	var u phoneUser
	err := tx.QueryRow(
		`SELECT phone_hash, COALESCE(NULLIF(contact_phone_masked, ''), phone_masked), google_sub, email, full_name, picture_url FROM users WHERE id = ?`, userID,
	).Scan(&u.PhoneHash, &u.PhoneMasked, &u.GoogleSub, &u.Email, &u.FullName, &u.PictureURL)
	return u, err
}
