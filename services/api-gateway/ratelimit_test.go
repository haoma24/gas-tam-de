package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestRateLimit_OTPRequestByIP(t *testing.T) {
	auth, _, _ := startMockUpstream(t)
	cfg := defaultRateLimitConfig()
	cfg.OTPPerIPPerMinute = 2
	r := testRouterWithLimits(t, "secret", upstreams{auth: auth.URL}, cfg)

	for i := 0; i < 2; i++ {
		req := httptest.NewRequest(http.MethodPost, "/v1/auth/otp/request", nil)
		req.RemoteAddr = "203.0.113.10:1234"
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("req %d: status=%d body=%s", i+1, rec.Code, rec.Body.String())
		}
	}

	req := httptest.NewRequest(http.MethodPost, "/v1/auth/otp/request", nil)
	req.RemoteAddr = "203.0.113.10:1234"
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429 got %d body=%s", rec.Code, rec.Body.String())
	}
	if rec.Header().Get("Retry-After") == "" {
		t.Fatal("missing Retry-After")
	}
	code, _ := decodeErr(t, rec)
	if code != "RATE_LIMITED" {
		t.Fatalf("code=%q", code)
	}

	// Different IP still allowed.
	req2 := httptest.NewRequest(http.MethodPost, "/v1/auth/otp/request", nil)
	req2.RemoteAddr = "203.0.113.20:1234"
	rec2 := httptest.NewRecorder()
	r.ServeHTTP(rec2, req2)
	if rec2.Code != http.StatusOK {
		t.Fatalf("other ip status=%d body=%s", rec2.Code, rec2.Body.String())
	}
}

func TestRateLimit_AdminLoginByIP(t *testing.T) {
	auth, _, _ := startMockUpstream(t)
	cfg := defaultRateLimitConfig()
	cfg.LoginPerIPPerMinute = 1
	r := testRouterWithLimits(t, "secret", upstreams{auth: auth.URL}, cfg)

	req1 := httptest.NewRequest(http.MethodPost, "/v1/auth/admin/login", nil)
	req1.RemoteAddr = "198.51.100.5:99"
	rec1 := httptest.NewRecorder()
	r.ServeHTTP(rec1, req1)
	if rec1.Code != http.StatusOK {
		t.Fatalf("first login status=%d", rec1.Code)
	}

	// Google and admin login share the same edge budget.
	req2 := httptest.NewRequest(http.MethodPost, "/v1/auth/google", nil)
	req2.RemoteAddr = "198.51.100.5:99"
	rec2 := httptest.NewRecorder()
	r.ServeHTTP(rec2, req2)
	if rec2.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429 got %d body=%s", rec2.Code, rec2.Body.String())
	}
	if rec2.Header().Get("Retry-After") == "" {
		t.Fatal("missing Retry-After")
	}
}

func TestRateLimit_PlaceOrderByUser(t *testing.T) {
	secret := "secret"
	order, _, _ := startMockUpstream(t)
	cfg := defaultRateLimitConfig()
	cfg.OrderPerUserPerMinute = 2
	cfg.OrderPerIPPerMinute = 100
	r := testRouterWithLimits(t, secret, upstreams{order: order.URL}, cfg)

	tok := issueTestToken(t, secret, "user-1", roleCustomer, "sess-1", time.Hour)

	for i := 0; i < 2; i++ {
		req := httptest.NewRequest(http.MethodPost, "/v1/orders", nil)
		req.Header.Set("Authorization", "Bearer "+tok)
		req.RemoteAddr = "192.0.2.1:1"
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("req %d: status=%d body=%s", i+1, rec.Code, rec.Body.String())
		}
	}

	req := httptest.NewRequest(http.MethodPost, "/v1/orders", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	req.RemoteAddr = "192.0.2.1:1"
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429 got %d body=%s", rec.Code, rec.Body.String())
	}
	if rec.Header().Get("Retry-After") == "" {
		t.Fatal("missing Retry-After")
	}

	// Quote path is not place-order — not rate limited by this bucket.
	tok2 := issueTestToken(t, secret, "user-1", roleCustomer, "sess-2", time.Hour)
	reqQ := httptest.NewRequest(http.MethodPost, "/v1/orders/quote", nil)
	reqQ.Header.Set("Authorization", "Bearer "+tok2)
	reqQ.RemoteAddr = "192.0.2.1:1"
	recQ := httptest.NewRecorder()
	r.ServeHTTP(recQ, reqQ)
	if recQ.Code != http.StatusOK {
		t.Fatalf("quote status=%d body=%s", recQ.Code, recQ.Body.String())
	}

	// Other user still allowed.
	tokOther := issueTestToken(t, secret, "user-2", roleCustomer, "sess-3", time.Hour)
	reqO := httptest.NewRequest(http.MethodPost, "/v1/orders", nil)
	reqO.Header.Set("Authorization", "Bearer "+tokOther)
	reqO.RemoteAddr = "192.0.2.1:1"
	recO := httptest.NewRecorder()
	r.ServeHTTP(recO, reqO)
	if recO.Code != http.StatusOK {
		t.Fatalf("other user status=%d body=%s", recO.Code, recO.Body.String())
	}
}

func TestRateLimit_AuthRefreshNotLimited(t *testing.T) {
	auth, _, _ := startMockUpstream(t)
	cfg := defaultRateLimitConfig()
	cfg.OTPPerIPPerMinute = 1
	cfg.LoginPerIPPerMinute = 1
	r := testRouterWithLimits(t, "secret", upstreams{auth: auth.URL}, cfg)

	for i := 0; i < 3; i++ {
		req := httptest.NewRequest(http.MethodPost, "/v1/auth/refresh", nil)
		req.RemoteAddr = "203.0.113.10:1"
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("refresh %d status=%d body=%s", i+1, rec.Code, rec.Body.String())
		}
	}
}

func TestSlidingWindowLimiter(t *testing.T) {
	l := newSlidingWindowLimiter(2)
	now := time.Now()
	if !l.Allow("a", now).Allowed {
		t.Fatal("1st should allow")
	}
	if !l.Allow("a", now.Add(time.Second)).Allowed {
		t.Fatal("2nd should allow")
	}
	res := l.Allow("a", now.Add(2*time.Second))
	if res.Allowed {
		t.Fatal("3rd should deny")
	}
	if res.RetryAfterSec < 1 {
		t.Fatalf("retry=%d", res.RetryAfterSec)
	}
	if !l.Allow("b", now.Add(2*time.Second)).Allowed {
		t.Fatal("other key should allow")
	}
}
