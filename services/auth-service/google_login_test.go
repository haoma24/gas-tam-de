package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"gas-tam-de/pkg/httpx"
)

type fakeGoogleVerifier struct {
	identity googleIdentity
	err      error
}

func (f fakeGoogleVerifier) Verify(context.Context, string) (googleIdentity, error) {
	return f.identity, f.err
}

func googleLogin(t *testing.T, handler http.Handler) map[string]any {
	t.Helper()
	body, _ := json.Marshal(map[string]string{"id_token": "signed-google-token"})
	req := httptest.NewRequest(http.MethodPost, "/v1/auth/google", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("login status=%d body=%s", rr.Code, rr.Body.String())
	}
	var response map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	return response
}

func TestGoogleLoginCreatesPersistentSessionAndReusesUser(t *testing.T) {
	db := openTestDB(t)
	identity := googleIdentity{
		Subject: "google-123", Email: "customer@example.com",
		DisplayName: "Khach Hang", PictureURL: "https://example.com/avatar.jpg",
	}
	svc := newGoogleAuthService(
		db, "test-jwt-secret", 15*time.Minute, 24*time.Hour,
		fakeGoogleVerifier{identity: identity},
	)
	r := httpx.NewRouter("auth-test")
	r.Post("/v1/auth/google", svc.handleLogin)

	first := googleLogin(t, r)
	second := googleLogin(t, r)
	firstUser := first["user"].(map[string]any)
	secondUser := second["user"].(map[string]any)
	if firstUser["id"] != secondUser["id"] {
		t.Fatalf("Google subject created multiple users: %v != %v", firstUser["id"], secondUser["id"])
	}
	if firstUser["email"] != identity.Email || firstUser["display_name"] != identity.DisplayName {
		t.Fatalf("user=%v", firstUser)
	}
	var users, persistentSessions int
	if err := db.QueryRow(`SELECT COUNT(*) FROM users WHERE google_sub = ?`, identity.Subject).Scan(&users); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow(`SELECT COUNT(*) FROM sessions WHERE persistent = 1`).Scan(&persistentSessions); err != nil {
		t.Fatal(err)
	}
	if users != 1 || persistentSessions != 2 {
		t.Fatalf("users=%d persistent_sessions=%d", users, persistentSessions)
	}
}

func TestGoogleLoginRejectsInvalidToken(t *testing.T) {
	db := openTestDB(t)
	svc := newGoogleAuthService(
		db, "test-jwt-secret", 15*time.Minute, 24*time.Hour,
		fakeGoogleVerifier{err: errors.New("bad signature")},
	)
	r := httpx.NewRouter("auth-test")
	r.Post("/v1/auth/google", svc.handleLogin)
	body, _ := json.Marshal(map[string]string{"id_token": "bad"})
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, httptest.NewRequest(http.MethodPost, "/v1/auth/google", bytes.NewReader(body)))
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestPersistentGoogleSessionRefreshesPastConfiguredTTLAndLogoutRevokesIt(t *testing.T) {
	db := openTestDB(t)
	identity := googleIdentity{Subject: "google-456", Email: "forever@example.com"}
	googleSvc := newGoogleAuthService(
		db, "test-jwt-secret", 15*time.Minute, time.Hour,
		fakeGoogleVerifier{identity: identity},
	)
	loginRouter := httpx.NewRouter("auth-test")
	loginRouter.Post("/v1/auth/google", googleSvc.handleLogin)
	login := googleLogin(t, loginRouter)
	refresh := login["refresh_token"].(string)
	if _, err := db.Exec(`UPDATE sessions SET expires_at = ?`, time.Now().Add(-time.Hour).Format(time.RFC3339Nano)); err != nil {
		t.Fatal(err)
	}

	tokens := newTokenService(db, "test-jwt-secret", 15*time.Minute, time.Hour)
	r := httpx.NewRouter("auth-test")
	r.Post("/v1/auth/refresh", tokens.handleRefresh)
	r.Post("/v1/auth/logout", tokens.handleLogout)
	body, _ := json.Marshal(map[string]string{"refresh_token": refresh})
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, httptest.NewRequest(http.MethodPost, "/v1/auth/refresh", bytes.NewReader(body)))
	if rr.Code != http.StatusOK {
		t.Fatalf("refresh status=%d body=%s", rr.Code, rr.Body.String())
	}
	var response map[string]any
	_ = json.Unmarshal(rr.Body.Bytes(), &response)
	rotated := response["refresh_token"].(string)

	logoutBody, _ := json.Marshal(map[string]string{"refresh_token": rotated})
	logout := httptest.NewRecorder()
	r.ServeHTTP(logout, httptest.NewRequest(http.MethodPost, "/v1/auth/logout", bytes.NewReader(logoutBody)))
	if logout.Code != http.StatusNoContent {
		t.Fatalf("logout status=%d body=%s", logout.Code, logout.Body.String())
	}
	reuse := httptest.NewRecorder()
	r.ServeHTTP(reuse, httptest.NewRequest(http.MethodPost, "/v1/auth/refresh", bytes.NewReader(logoutBody)))
	if reuse.Code != http.StatusUnauthorized {
		t.Fatalf("revoked refresh status=%d body=%s", reuse.Code, reuse.Body.String())
	}
}
