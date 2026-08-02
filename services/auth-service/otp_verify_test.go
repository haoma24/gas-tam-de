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

func testOTPService(t *testing.T) *otpService {
	t.Helper()
	return &otpService{
		db:           openTestDB(t),
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
}

func TestPhoneEncryptRoundTrip(t *testing.T) {
	key := derivePhoneKey("secret")
	enc, err := encryptPhoneE164("+84901234567", key)
	if err != nil {
		t.Fatal(err)
	}
	got, err := decryptPhoneE164(enc, key)
	if err != nil {
		t.Fatal(err)
	}
	if got != "+84901234567" {
		t.Fatalf("got %q", got)
	}
}

func TestOTPVerifySuccess(t *testing.T) {
	svc := testOTPService(t)
	r := httpx.NewRouter("auth-test")
	r.Post("/v1/auth/otp/request", svc.handleOTPRequest)
	r.Post("/v1/auth/otp/verify", svc.handleOTPVerify)

	reqBody, _ := json.Marshal(map[string]string{"phone": "0901234567"})
	req := httptest.NewRequest(http.MethodPost, "/v1/auth/otp/request", bytes.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	req.RemoteAddr = "127.0.0.1:12345"
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("request status=%d body=%s", rr.Code, rr.Body.String())
	}
	var reqResp map[string]any
	_ = json.Unmarshal(rr.Body.Bytes(), &reqResp)
	code, _ := reqResp["dev_code"].(string)
	if code == "" {
		t.Fatal("missing dev_code")
	}

	verifyBody, _ := json.Marshal(map[string]string{"phone": "0901234567", "code": code})
	vreq := httptest.NewRequest(http.MethodPost, "/v1/auth/otp/verify", bytes.NewReader(verifyBody))
	vreq.Header.Set("Content-Type", "application/json")
	vrr := httptest.NewRecorder()
	r.ServeHTTP(vrr, vreq)
	if vrr.Code != http.StatusOK {
		t.Fatalf("verify status=%d body=%s", vrr.Code, vrr.Body.String())
	}

	var resp map[string]any
	if err := json.Unmarshal(vrr.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp["ok"] != true {
		t.Fatalf("resp=%v", resp)
	}
	access, _ := resp["access_token"].(string)
	refresh, _ := resp["refresh_token"].(string)
	if access == "" || refresh == "" {
		t.Fatalf("tokens missing: %v", resp)
	}
	if resp["token_type"] != "Bearer" {
		t.Fatalf("token_type=%v", resp["token_type"])
	}

	claims := &AccessClaims{}
	tok, err := jwt.ParseWithClaims(access, claims, func(t *jwt.Token) (any, error) {
		return []byte("test-jwt-secret"), nil
	})
	if err != nil || !tok.Valid {
		t.Fatalf("parse jwt: %v", err)
	}
	if claims.Role != "customer" || claims.Subject == "" || claims.SessionID == "" {
		t.Fatalf("claims=%+v", claims)
	}
	if claims.PhoneMasked != "090***4567" {
		t.Fatalf("phone_masked=%q", claims.PhoneMasked)
	}

	user, _ := resp["user"].(map[string]any)
	if user["id"] != claims.Subject {
		t.Fatalf("user id mismatch %v vs %s", user["id"], claims.Subject)
	}

	var nUsers, nSessions, consumed int
	if err := svc.db.QueryRow(`SELECT COUNT(*) FROM users`).Scan(&nUsers); err != nil {
		t.Fatal(err)
	}
	if err := svc.db.QueryRow(`SELECT COUNT(*) FROM sessions`).Scan(&nSessions); err != nil {
		t.Fatal(err)
	}
	if err := svc.db.QueryRow(`SELECT COUNT(*) FROM otp_challenges WHERE consumed_at IS NOT NULL`).Scan(&consumed); err != nil {
		t.Fatal(err)
	}
	if nUsers != 1 || nSessions != 1 || consumed != 1 {
		t.Fatalf("users=%d sessions=%d consumed=%d", nUsers, nSessions, consumed)
	}

	// Replay same OTP → fail (consumed)
	vreq2 := httptest.NewRequest(http.MethodPost, "/v1/auth/otp/verify", bytes.NewReader(verifyBody))
	vreq2.Header.Set("Content-Type", "application/json")
	vrr2 := httptest.NewRecorder()
	r.ServeHTTP(vrr2, vreq2)
	if vrr2.Code != http.StatusUnauthorized {
		t.Fatalf("replay expected 401 got %d body=%s", vrr2.Code, vrr2.Body.String())
	}
}

func TestOTPVerifyInvalidCode(t *testing.T) {
	svc := testOTPService(t)
	r := httpx.NewRouter("auth-test")
	r.Post("/v1/auth/otp/request", svc.handleOTPRequest)
	r.Post("/v1/auth/otp/verify", svc.handleOTPVerify)

	reqBody, _ := json.Marshal(map[string]string{"phone": "0912345678"})
	req := httptest.NewRequest(http.MethodPost, "/v1/auth/otp/request", bytes.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	req.RemoteAddr = "10.0.0.1:1"
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("request status=%d", rr.Code)
	}

	verifyBody, _ := json.Marshal(map[string]string{"phone": "0912345678", "code": "000000"})
	vreq := httptest.NewRequest(http.MethodPost, "/v1/auth/otp/verify", bytes.NewReader(verifyBody))
	vreq.Header.Set("Content-Type", "application/json")
	vrr := httptest.NewRecorder()
	r.ServeHTTP(vrr, vreq)
	if vrr.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d body=%s", vrr.Code, vrr.Body.String())
	}
	var resp map[string]any
	_ = json.Unmarshal(vrr.Body.Bytes(), &resp)
	errObj, _ := resp["error"].(map[string]any)
	if errObj["code"] != "OTP_INVALID" {
		t.Fatalf("error=%v", resp)
	}
	if int(errObj["attempts_remaining"].(float64)) != 4 {
		t.Fatalf("remaining=%v", errObj["attempts_remaining"])
	}

	var attempts int
	if err := svc.db.QueryRow(`SELECT attempts FROM otp_challenges`).Scan(&attempts); err != nil {
		t.Fatal(err)
	}
	if attempts != 1 {
		t.Fatalf("attempts=%d", attempts)
	}
}

func TestOTPVerifyLockout(t *testing.T) {
	svc := testOTPService(t)
	svc.maxAttempts = 2
	r := httpx.NewRouter("auth-test")
	r.Post("/v1/auth/otp/request", svc.handleOTPRequest)
	r.Post("/v1/auth/otp/verify", svc.handleOTPVerify)

	reqBody, _ := json.Marshal(map[string]string{"phone": "0987654321"})
	req := httptest.NewRequest(http.MethodPost, "/v1/auth/otp/request", bytes.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	req.RemoteAddr = "10.0.0.2:1"
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("request status=%d", rr.Code)
	}

	for i := 0; i < 2; i++ {
		verifyBody, _ := json.Marshal(map[string]string{"phone": "0987654321", "code": "111111"})
		vreq := httptest.NewRequest(http.MethodPost, "/v1/auth/otp/verify", bytes.NewReader(verifyBody))
		vreq.Header.Set("Content-Type", "application/json")
		vrr := httptest.NewRecorder()
		r.ServeHTTP(vrr, vreq)
		if i == 0 && vrr.Code != http.StatusUnauthorized {
			t.Fatalf("attempt1 status=%d", vrr.Code)
		}
		if i == 1 && vrr.Code != http.StatusTooManyRequests {
			t.Fatalf("attempt2 expected 429 got %d body=%s", vrr.Code, vrr.Body.String())
		}
	}
}

func TestOTPVerifyNoChallenge(t *testing.T) {
	svc := testOTPService(t)
	r := httpx.NewRouter("auth-test")
	r.Post("/v1/auth/otp/verify", svc.handleOTPVerify)

	verifyBody, _ := json.Marshal(map[string]string{"phone": "0901111222", "code": "123456"})
	vreq := httptest.NewRequest(http.MethodPost, "/v1/auth/otp/verify", bytes.NewReader(verifyBody))
	vreq.Header.Set("Content-Type", "application/json")
	vrr := httptest.NewRecorder()
	r.ServeHTTP(vrr, vreq)
	if vrr.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d", vrr.Code)
	}
}
