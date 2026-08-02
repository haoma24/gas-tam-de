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

func adminLoginTokens(t *testing.T, svc *tokenService, username, password string) (access, refresh string) {
	t.Helper()
	r := httpx.NewRouter("auth-test")
	r.Post("/v1/auth/admin/login", svc.handleAdminLogin)
	body, _ := json.Marshal(map[string]string{"username": username, "password": password})
	req := httptest.NewRequest(http.MethodPost, "/v1/auth/admin/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("login status=%d body=%s", rr.Code, rr.Body.String())
	}
	var resp map[string]any
	_ = json.Unmarshal(rr.Body.Bytes(), &resp)
	access, _ = resp["access_token"].(string)
	refresh, _ = resp["refresh_token"].(string)
	if access == "" || refresh == "" {
		t.Fatalf("tokens missing: %v", resp)
	}
	return access, refresh
}

func TestRefreshAdminRotate(t *testing.T) {
	svc := testTokenService(t)
	cfg := adminSeedConfig{Username: "admin", Password: "pw", DisplayName: "Admin", Enabled: true}
	if err := seedAdminAccount(svc.db, cfg); err != nil {
		t.Fatal(err)
	}
	_, oldRefresh := adminLoginTokens(t, svc, cfg.Username, cfg.Password)

	r := httpx.NewRouter("auth-test")
	r.Post("/v1/auth/refresh", svc.handleRefresh)

	body, _ := json.Marshal(map[string]string{"refresh_token": oldRefresh})
	req := httptest.NewRequest(http.MethodPost, "/v1/auth/refresh", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}

	var resp map[string]any
	_ = json.Unmarshal(rr.Body.Bytes(), &resp)
	newAccess, _ := resp["access_token"].(string)
	newRefresh, _ := resp["refresh_token"].(string)
	if newAccess == "" || newRefresh == "" || newRefresh == oldRefresh {
		t.Fatalf("rotation failed: %v", resp)
	}

	claims := &AccessClaims{}
	tok, err := jwt.ParseWithClaims(newAccess, claims, func(t *jwt.Token) (any, error) {
		return []byte("test-jwt-secret"), nil
	})
	if err != nil || !tok.Valid || claims.Role != "admin" {
		t.Fatalf("jwt err=%v claims=%+v", err, claims)
	}

	user, _ := resp["user"].(map[string]any)
	if user["role"] != "admin" || user["username"] != cfg.Username {
		t.Fatalf("user=%v", user)
	}

	// Old refresh must be rejected (rotated).
	body2, _ := json.Marshal(map[string]string{"refresh_token": oldRefresh})
	req2 := httptest.NewRequest(http.MethodPost, "/v1/auth/refresh", bytes.NewReader(body2))
	req2.Header.Set("Content-Type", "application/json")
	rr2 := httptest.NewRecorder()
	r.ServeHTTP(rr2, req2)
	if rr2.Code != http.StatusUnauthorized {
		t.Fatalf("reuse old refresh status=%d", rr2.Code)
	}

	var revoked, active int
	_ = svc.db.QueryRow(`SELECT COUNT(*) FROM sessions WHERE revoked_at IS NOT NULL`).Scan(&revoked)
	_ = svc.db.QueryRow(`SELECT COUNT(*) FROM sessions WHERE revoked_at IS NULL`).Scan(&active)
	if revoked != 1 || active != 1 {
		t.Fatalf("revoked=%d active=%d", revoked, active)
	}
}

func TestRefreshCustomerFromOTP(t *testing.T) {
	db := openTestDB(t)
	otp := &otpService{
		db:           db,
		limiter:      newOTPRateLimiter(60, 5, 20),
		sms:          NewMockSMSSender(),
		phonePepper:  "pepper",
		otpPepper:    "otp-pepper",
		phoneKey:     derivePhoneKey("test-phone-enc-key"),
		jwtSecret:    "test-jwt-secret",
		ttl:          5 * time.Minute,
		accessTTL:    15 * time.Minute,
		refreshTTL:   24 * time.Hour,
		maxAttempts:  5,
		cooldownSec:  60,
		devRevealOTP: true,
	}
	tokens := newTokenService(db, "test-jwt-secret", 15*time.Minute, 24*time.Hour)

	r := httpx.NewRouter("auth-test")
	r.Post("/v1/auth/otp/request", otp.handleOTPRequest)
	r.Post("/v1/auth/otp/verify", otp.handleOTPVerify)
	r.Post("/v1/auth/refresh", tokens.handleRefresh)

	reqBody, _ := json.Marshal(map[string]string{"phone": "0901234567"})
	req := httptest.NewRequest(http.MethodPost, "/v1/auth/otp/request", bytes.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	req.RemoteAddr = "127.0.0.1:1"
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	var reqResp map[string]any
	_ = json.Unmarshal(rr.Body.Bytes(), &reqResp)
	code, _ := reqResp["dev_code"].(string)

	verifyBody, _ := json.Marshal(map[string]string{"phone": "0901234567", "code": code})
	vreq := httptest.NewRequest(http.MethodPost, "/v1/auth/otp/verify", bytes.NewReader(verifyBody))
	vreq.Header.Set("Content-Type", "application/json")
	vrr := httptest.NewRecorder()
	r.ServeHTTP(vrr, vreq)
	if vrr.Code != http.StatusOK {
		t.Fatalf("verify=%d %s", vrr.Code, vrr.Body.String())
	}
	var vresp map[string]any
	_ = json.Unmarshal(vrr.Body.Bytes(), &vresp)
	oldRefresh, _ := vresp["refresh_token"].(string)

	body, _ := json.Marshal(map[string]string{"refresh_token": oldRefresh})
	rreq := httptest.NewRequest(http.MethodPost, "/v1/auth/refresh", bytes.NewReader(body))
	rreq.Header.Set("Content-Type", "application/json")
	rrr := httptest.NewRecorder()
	r.ServeHTTP(rrr, rreq)
	if rrr.Code != http.StatusOK {
		t.Fatalf("refresh=%d %s", rrr.Code, rrr.Body.String())
	}
	var rresp map[string]any
	_ = json.Unmarshal(rrr.Body.Bytes(), &rresp)
	user, _ := rresp["user"].(map[string]any)
	if user["role"] != "customer" || user["phone_masked"] != "090***4567" {
		t.Fatalf("user=%v", user)
	}

	claims := &AccessClaims{}
	access, _ := rresp["access_token"].(string)
	tok, err := jwt.ParseWithClaims(access, claims, func(t *jwt.Token) (any, error) {
		return []byte("test-jwt-secret"), nil
	})
	if err != nil || !tok.Valid || claims.Role != "customer" || claims.PhoneMasked != "090***4567" {
		t.Fatalf("claims=%+v err=%v", claims, err)
	}
}

func TestRefreshInvalidToken(t *testing.T) {
	svc := testTokenService(t)
	r := httpx.NewRouter("auth-test")
	r.Post("/v1/auth/refresh", svc.handleRefresh)

	body, _ := json.Marshal(map[string]string{"refresh_token": "not-a-real-token"})
	req := httptest.NewRequest(http.MethodPost, "/v1/auth/refresh", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d", rr.Code)
	}
}

func TestRefreshExpiredSession(t *testing.T) {
	svc := testTokenService(t)
	cfg := adminSeedConfig{Username: "admin", Password: "pw", Enabled: true}
	if err := seedAdminAccount(svc.db, cfg); err != nil {
		t.Fatal(err)
	}
	_, refresh := adminLoginTokens(t, svc, cfg.Username, cfg.Password)

	past := time.Now().UTC().Add(-time.Hour).Format(time.RFC3339Nano)
	if _, err := svc.db.Exec(`UPDATE sessions SET expires_at = ?`, past); err != nil {
		t.Fatal(err)
	}

	r := httpx.NewRouter("auth-test")
	r.Post("/v1/auth/refresh", svc.handleRefresh)
	body, _ := json.Marshal(map[string]string{"refresh_token": refresh})
	req := httptest.NewRequest(http.MethodPost, "/v1/auth/refresh", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
}
