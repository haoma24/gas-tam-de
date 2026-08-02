package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"gas-tam-de/pkg/httpx"
	"gas-tam-de/pkg/sqlite"
)

func TestNormalizePhoneVN(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"0901234567", "+84901234567"},
		{"090 123 4567", "+84901234567"},
		{"+84901234567", "+84901234567"},
		{"84901234567", "+84901234567"},
	}
	for _, tc := range cases {
		got, err := normalizePhoneVN(tc.in)
		if err != nil {
			t.Fatalf("normalizePhoneVN(%q): %v", tc.in, err)
		}
		if got != tc.want {
			t.Fatalf("normalizePhoneVN(%q)=%q want %q", tc.in, got, tc.want)
		}
	}
	if _, err := normalizePhoneVN("123"); err == nil {
		t.Fatal("expected error for invalid phone")
	}
}

func TestMaskPhoneE164(t *testing.T) {
	got := maskPhoneE164("+84901234567")
	if got != "090***4567" {
		t.Fatalf("mask=%q", got)
	}
}

func TestOTPRateLimiterCooldown(t *testing.T) {
	l := newOTPRateLimiter(60, 5, 20)
	now := time.Date(2026, 8, 2, 7, 0, 0, 0, time.UTC)
	if r := l.Allow("p1", "1.1.1.1", now); !r.Allowed {
		t.Fatalf("first allow: %+v", r)
	}
	r := l.Allow("p1", "1.1.1.1", now.Add(10*time.Second))
	if r.Allowed || r.Reason != "cooldown" || r.RetryAfterSec < 1 {
		t.Fatalf("cooldown: %+v", r)
	}
	if r := l.Allow("p1", "1.1.1.1", now.Add(61*time.Second)); !r.Allowed {
		t.Fatalf("after cooldown: %+v", r)
	}
}

func TestOTPRateLimiterPhoneQuota(t *testing.T) {
	l := newOTPRateLimiter(1, 2, 100)
	now := time.Date(2026, 8, 2, 8, 0, 0, 0, time.UTC)
	if r := l.Allow("p2", "2.2.2.2", now); !r.Allowed {
		t.Fatal(r)
	}
	if r := l.Allow("p2", "2.2.2.2", now.Add(2*time.Second)); !r.Allowed {
		t.Fatal(r)
	}
	r := l.Allow("p2", "2.2.2.2", now.Add(4*time.Second))
	if r.Allowed || r.Reason != "phone_quota" {
		t.Fatalf("quota: %+v", r)
	}
}

func TestOTPRequestHandler(t *testing.T) {
	db := openTestDB(t)
	mockSMS := NewMockSMSSender()
	svc := &otpService{
		db:           db,
		limiter:      newOTPRateLimiter(60, 5, 20),
		sms:          mockSMS,
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

	r := httpx.NewRouter("auth-test")
	r.Post("/v1/auth/otp/request", svc.handleOTPRequest)

	body, _ := json.Marshal(map[string]string{"phone": "0901234567"})
	req := httptest.NewRequest(http.MethodPost, "/v1/auth/otp/request", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.RemoteAddr = "127.0.0.1:12345"
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var resp map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp["ok"] != true {
		t.Fatalf("resp=%v", resp)
	}
	if resp["phone_masked"] != "090***4567" {
		t.Fatalf("masked=%v", resp["phone_masked"])
	}
	code, _ := resp["dev_code"].(string)
	if len(code) != 6 {
		t.Fatalf("dev_code=%v", resp["dev_code"])
	}

	sent, ok := mockSMS.Last()
	if !ok || sent.PhoneE164 != "+84901234567" || sent.Code != code {
		t.Fatalf("sms send=%+v ok=%v", sent, ok)
	}

	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM otp_challenges`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("challenges=%d", n)
	}

	// Second request within cooldown → 429
	req2 := httptest.NewRequest(http.MethodPost, "/v1/auth/otp/request", bytes.NewReader(body))
	req2.Header.Set("Content-Type", "application/json")
	req2.RemoteAddr = "127.0.0.1:12345"
	rr2 := httptest.NewRecorder()
	r.ServeHTTP(rr2, req2)
	if rr2.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429 got %d body=%s", rr2.Code, rr2.Body.String())
	}
	if rr2.Header().Get("Retry-After") == "" {
		t.Fatal("missing Retry-After")
	}
}

func TestOTPRequestSMSFailure(t *testing.T) {
	db := openTestDB(t)
	svc := &otpService{
		db:           db,
		limiter:      newOTPRateLimiter(60, 5, 20),
		sms:          failingSMSSender{},
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
	r := httpx.NewRouter("auth-test")
	r.Post("/v1/auth/otp/request", svc.handleOTPRequest)

	body, _ := json.Marshal(map[string]string{"phone": "0901234567"})
	req := httptest.NewRequest(http.MethodPost, "/v1/auth/otp/request", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.RemoteAddr = "127.0.0.1:12345"
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadGateway {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestOTPRequestInvalidPhone(t *testing.T) {
	db := openTestDB(t)
	svc := &otpService{
		db:          db,
		limiter:     newOTPRateLimiter(60, 5, 20),
		sms:         NewMockSMSSender(),
		phonePepper: "pepper",
		otpPepper:   "otp-pepper",
		phoneKey:    derivePhoneKey("test-phone-enc-key"),
		jwtSecret:   "test-jwt-secret",
		ttl:         5 * time.Minute,
		accessTTL:   15 * time.Minute,
		refreshTTL:  24 * time.Hour,
		maxAttempts: 5,
		cooldownSec: 60,
	}
	r := httpx.NewRouter("auth-test")
	r.Post("/v1/auth/otp/request", svc.handleOTPRequest)

	body, _ := json.Marshal(map[string]string{"phone": "abc"})
	req := httptest.NewRequest(http.MethodPost, "/v1/auth/otp/request", bytes.NewReader(body))
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status=%d", rr.Code)
	}
}

func openTestDB(t *testing.T) *sql.DB {
	t.Helper()
	dir := t.TempDir()
	db, err := sqlite.Open(filepath.Join(dir, "auth.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := migrate(db); err != nil {
		t.Fatal(err)
	}
	return db
}

type failingSMSSender struct{}

func (failingSMSSender) SendOTP(context.Context, string, string) error {
	return ErrSMSNotConfigured
}
