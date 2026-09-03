package main

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"gas-tam-de/pkg/httpx"

	"github.com/google/uuid"
	"google.golang.org/api/idtoken"
)

var errGoogleNotConfigured = errors.New("google sign-in is not configured")

type googleLoginBody struct {
	IDToken string `json:"id_token"`
}

type googleIdentity struct {
	Subject     string
	Email       string
	DisplayName string
	PictureURL  string
}

type googleTokenVerifier interface {
	Verify(context.Context, string) (googleIdentity, error)
}

type googleIDTokenVerifier struct {
	clientIDs map[string]struct{}
}

func newGoogleIDTokenVerifier(rawClientIDs string) *googleIDTokenVerifier {
	clientIDs := make(map[string]struct{})
	for _, value := range strings.Split(rawClientIDs, ",") {
		if id := strings.TrimSpace(value); id != "" {
			clientIDs[id] = struct{}{}
		}
	}
	return &googleIDTokenVerifier{clientIDs: clientIDs}
}

func (v *googleIDTokenVerifier) Verify(ctx context.Context, raw string) (googleIdentity, error) {
	if len(v.clientIDs) == 0 {
		return googleIdentity{}, errGoogleNotConfigured
	}
	parsed, err := idtoken.ParsePayload(raw)
	if err != nil {
		return googleIdentity{}, fmt.Errorf("parse Google ID token: %w", err)
	}
	if _, ok := v.clientIDs[parsed.Audience]; !ok {
		return googleIdentity{}, errors.New("Google ID token audience is not allowed")
	}
	payload, err := idtoken.Validate(ctx, raw, parsed.Audience)
	if err != nil {
		return googleIdentity{}, fmt.Errorf("validate Google ID token: %w", err)
	}

	email := claimString(payload.Claims, "email")
	if payload.Subject == "" || email == "" || !claimBool(payload.Claims, "email_verified") {
		return googleIdentity{}, errors.New("Google account must have a verified email")
	}
	return googleIdentity{
		Subject:     payload.Subject,
		Email:       strings.ToLower(email),
		DisplayName: claimString(payload.Claims, "name"),
		PictureURL:  claimString(payload.Claims, "picture"),
	}, nil
}

func claimString(claims map[string]any, key string) string {
	value, _ := claims[key].(string)
	return strings.TrimSpace(value)
}

func claimBool(claims map[string]any, key string) bool {
	switch value := claims[key].(type) {
	case bool:
		return value
	case string:
		return strings.EqualFold(value, "true")
	default:
		return false
	}
}

type googleAuthService struct {
	db         *sql.DB
	jwtSecret  string
	accessTTL  time.Duration
	refreshTTL time.Duration
	verifier   googleTokenVerifier
}

func newGoogleAuthService(db *sql.DB, jwtSecret string, accessTTL, refreshTTL time.Duration, verifier googleTokenVerifier) *googleAuthService {
	return &googleAuthService{
		db: db, jwtSecret: jwtSecret, accessTTL: accessTTL,
		refreshTTL: refreshTTL, verifier: verifier,
	}
}

func (s *googleAuthService) handleLogin(w http.ResponseWriter, r *http.Request) {
	var body googleLoginBody
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&body); err != nil {
		httpx.Error(w, http.StatusBadRequest, "INVALID_JSON", "invalid request body")
		return
	}
	raw := strings.TrimSpace(body.IDToken)
	if raw == "" {
		httpx.Error(w, http.StatusBadRequest, "INVALID_GOOGLE_TOKEN", "id_token is required")
		return
	}

	identity, err := s.verifier.Verify(r.Context(), raw)
	if err != nil {
		if errors.Is(err, errGoogleNotConfigured) {
			slog.Error("Google sign-in unavailable", "err", err)
			httpx.Error(w, http.StatusServiceUnavailable, "GOOGLE_AUTH_NOT_CONFIGURED", "Google sign-in is not configured")
			return
		}
		slog.Warn("Google ID token rejected", "err", err)
		httpx.Error(w, http.StatusUnauthorized, "INVALID_GOOGLE_TOKEN", "invalid Google identity token")
		return
	}

	now := time.Now().UTC()
	tx, err := s.db.Begin()
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "INTERNAL", "could not login")
		return
	}
	defer func() { _ = tx.Rollback() }()

	userID, phoneMasked, err := upsertGoogleUser(tx, identity, now)
	if err != nil {
		slog.Error("upsert Google user", "err", err)
		httpx.Error(w, http.StatusInternalServerError, "INTERNAL", "could not login")
		return
	}
	refreshRaw, refreshHash, err := generateRefreshToken()
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "INTERNAL", "could not issue tokens")
		return
	}
	sessionID := uuid.NewString()
	if err := insertSession(tx, sessionID, userID, roleCustomer, refreshHash, now.Add(s.refreshTTL), now, true); err != nil {
		slog.Error("insert persistent Google session", "err", err)
		httpx.Error(w, http.StatusInternalServerError, "INTERNAL", "could not issue tokens")
		return
	}
	access, err := issueAccessToken(s.jwtSecret, userID, roleCustomer, phoneMasked, sessionID, s.accessTTL, now)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "INTERNAL", "could not issue tokens")
		return
	}
	if err := tx.Commit(); err != nil {
		httpx.Error(w, http.StatusInternalServerError, "INTERNAL", "could not login")
		return
	}

	slog.Info("Google login", "user_id", userID, "session_id", sessionID)
	httpx.JSON(w, http.StatusOK, map[string]any{
		"ok": true, "access_token": access, "refresh_token": refreshRaw,
		"token_type": "Bearer", "expires_in": int(s.accessTTL.Seconds()),
		"user": map[string]any{
			"id": userID, "role": roleCustomer, "phone_masked": phoneMasked,
			"email": identity.Email, "display_name": identity.DisplayName,
			"picture_url": identity.PictureURL,
		},
	})
}

func upsertGoogleUser(tx *sql.Tx, identity googleIdentity, now time.Time) (string, string, error) {
	var id, phoneMasked string
	err := tx.QueryRow(`
		SELECT id, COALESCE(NULLIF(contact_phone_masked, ''), phone_masked)
		FROM users WHERE google_sub = ?
	`, identity.Subject).Scan(&id, &phoneMasked)
	stamp := now.Format(time.RFC3339Nano)
	if err == nil {
		_, err = tx.Exec(`
			UPDATE users
			SET email = ?, picture_url = COALESCE(NULLIF(?, ''), picture_url),
			    updated_at = ?
			WHERE id = ?
		`, identity.Email, identity.PictureURL, stamp, id)
		return id, phoneMasked, err
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return "", "", err
	}

	id = uuid.NewString()
	sum := sha256.Sum256([]byte(identity.Subject))
	placeholderHash := "google:" + hex.EncodeToString(sum[:])
	_, err = tx.Exec(`
		INSERT INTO users (
			id, phone_e164_enc, phone_hash, phone_masked, google_sub,
			email, picture_url, full_name, created_at, updated_at
		) VALUES (?, ?, ?, '', ?, ?, NULLIF(?, ''), NULLIF(?, ''), ?, ?)
	`, id, []byte{}, placeholderHash, identity.Subject, identity.Email,
		identity.PictureURL, identity.DisplayName, stamp, stamp)
	return id, "", err
}
