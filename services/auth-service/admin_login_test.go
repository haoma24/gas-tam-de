package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"gas-tam-de/pkg/httpx"

	"github.com/golang-jwt/jwt/v5"
)

func testTokenService(t *testing.T) *tokenService {
	t.Helper()
	return newTokenService(openTestDB(t), "test-jwt-secret", 15*time.Minute, 24*time.Hour)
}

func TestAdminLoginSuccess(t *testing.T) {
	svc := testTokenService(t)
	cfg := adminSeedConfig{
		Username:    "shopadmin",
		Password:    "s3cret-local",
		DisplayName: "Chủ CH",
		Enabled:     true,
	}
	if err := seedAdminAccount(svc.db, cfg); err != nil {
		t.Fatal(err)
	}

	r := httpx.NewRouter("auth-test")
	r.Post("/v1/auth/admin/login", svc.handleAdminLogin)

	body, _ := json.Marshal(map[string]string{
		"username": cfg.Username,
		"password": cfg.Password,
	})
	req := httptest.NewRequest(http.MethodPost, "/v1/auth/admin/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}

	var resp map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp["ok"] != true || resp["token_type"] != "Bearer" {
		t.Fatalf("resp=%v", resp)
	}
	access, _ := resp["access_token"].(string)
	refresh, _ := resp["refresh_token"].(string)
	if access == "" || refresh == "" {
		t.Fatalf("tokens missing: %v", resp)
	}

	claims := &AccessClaims{}
	tok, err := jwt.ParseWithClaims(access, claims, func(t *jwt.Token) (any, error) {
		return []byte("test-jwt-secret"), nil
	})
	if err != nil || !tok.Valid {
		t.Fatalf("parse jwt: %v", err)
	}
	if claims.Role != "admin" || claims.Subject == "" || claims.SessionID == "" {
		t.Fatalf("claims=%+v", claims)
	}
	if claims.PhoneMasked != "" {
		t.Fatalf("admin should not have phone_masked, got %q", claims.PhoneMasked)
	}

	user, _ := resp["user"].(map[string]any)
	if user["role"] != "admin" || user["username"] != cfg.Username {
		t.Fatalf("user=%v", user)
	}
	if user["display_name"] != cfg.DisplayName {
		t.Fatalf("display_name=%v", user["display_name"])
	}
	if user["id"] != claims.Subject {
		t.Fatalf("user id mismatch")
	}

	var nSessions int
	if err := svc.db.QueryRow(`SELECT COUNT(*) FROM sessions WHERE role = 'admin'`).Scan(&nSessions); err != nil {
		t.Fatal(err)
	}
	if nSessions != 1 {
		t.Fatalf("sessions=%d", nSessions)
	}
}

func TestAdminLoginWrongPassword(t *testing.T) {
	svc := testTokenService(t)
	if err := seedAdminAccount(svc.db, adminSeedConfig{
		Username: "admin",
		Password: "correct-password",
		Enabled:  true,
	}); err != nil {
		t.Fatal(err)
	}

	r := httpx.NewRouter("auth-test")
	r.Post("/v1/auth/admin/login", svc.handleAdminLogin)

	body, _ := json.Marshal(map[string]string{"username": "admin", "password": "wrong"})
	req := httptest.NewRequest(http.MethodPost, "/v1/auth/admin/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d", rr.Code)
	}
	var resp map[string]any
	_ = json.Unmarshal(rr.Body.Bytes(), &resp)
	errObj, _ := resp["error"].(map[string]any)
	if errObj["code"] != "INVALID_CREDENTIALS" {
		t.Fatalf("error=%v", resp)
	}
}

func TestAdminLoginUnknownUser(t *testing.T) {
	svc := testTokenService(t)
	r := httpx.NewRouter("auth-test")
	r.Post("/v1/auth/admin/login", svc.handleAdminLogin)

	body, _ := json.Marshal(map[string]string{"username": "nobody", "password": "x"})
	req := httptest.NewRequest(http.MethodPost, "/v1/auth/admin/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestAdminLoginDisabled(t *testing.T) {
	svc := testTokenService(t)
	cfg := adminSeedConfig{Username: "disabled", Password: "pw", Enabled: true}
	if err := seedAdminAccount(svc.db, cfg); err != nil {
		t.Fatal(err)
	}
	_, err := svc.db.Exec(`UPDATE admin_accounts SET disabled_at = ? WHERE username = ?`,
		time.Now().UTC().Format(time.RFC3339Nano), cfg.Username)
	if err != nil {
		t.Fatal(err)
	}

	r := httpx.NewRouter("auth-test")
	r.Post("/v1/auth/admin/login", svc.handleAdminLogin)
	body, _ := json.Marshal(map[string]string{"username": cfg.Username, "password": cfg.Password})
	req := httptest.NewRequest(http.MethodPost, "/v1/auth/admin/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d", rr.Code)
	}
}

func TestAdminLoginMissingFields(t *testing.T) {
	svc := testTokenService(t)
	r := httpx.NewRouter("auth-test")
	r.Post("/v1/auth/admin/login", svc.handleAdminLogin)

	body, _ := json.Marshal(map[string]string{"username": "admin"})
	req := httptest.NewRequest(http.MethodPost, "/v1/auth/admin/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status=%d", rr.Code)
	}
}
