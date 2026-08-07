package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

type capturedReq struct {
	Method string
	Path   string
	Query  string
	Header http.Header
	Body   string
}

func startMockUpstream(t *testing.T) (*httptest.Server, *capturedReq, *sync.Mutex) {
	t.Helper()
	cap := &capturedReq{}
	var mu sync.Mutex
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		mu.Lock()
		cap.Method = r.Method
		cap.Path = r.URL.Path
		cap.Query = r.URL.RawQuery
		cap.Header = r.Header.Clone()
		cap.Body = string(body)
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"upstream":"ok"}`))
	}))
	t.Cleanup(srv.Close)
	return srv, cap, &mu
}

func testRouter(t *testing.T, secret string, u upstreams) http.Handler {
	t.Helper()
	return testRouterWithLimits(t, secret, u, defaultRateLimitConfig())
}

func testRouterWithLimits(t *testing.T, secret string, u upstreams, cfg rateLimitConfig) http.Handler {
	t.Helper()
	origins := parseCORSOrigins(defaultCORSOrigins)
	return newGatewayRouter(secret, origins, u, newRateLimiters(cfg), NewMemoryAuditRecorder())
}

func testRouterWithAudit(t *testing.T, secret string, u upstreams, audit AuditRecorder) http.Handler {
	t.Helper()
	origins := parseCORSOrigins(defaultCORSOrigins)
	return newGatewayRouter(secret, origins, u, newRateLimiters(defaultRateLimitConfig()), audit)
}

func TestProxy_PublicAuthForwardsPath(t *testing.T) {
	auth, cap, mu := startMockUpstream(t)
	r := testRouter(t, "secret", upstreams{auth: auth.URL})

	req := httptest.NewRequest(http.MethodPost, "/v1/auth/otp/request", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	mu.Lock()
	defer mu.Unlock()
	if cap.Path != "/v1/auth/otp/request" {
		t.Fatalf("path=%q", cap.Path)
	}
}

func TestProxy_AdminSplitsUpstreams(t *testing.T) {
	secret := "secret"
	catalog, catCap, catMu := startMockUpstream(t)
	order, ordCap, ordMu := startMockUpstream(t)
	inventory, invCap, invMu := startMockUpstream(t)
	billing, bilCap, bilMu := startMockUpstream(t)
	report, repCap, repMu := startMockUpstream(t)
	geo, geoCap, geoMu := startMockUpstream(t)

	r := testRouter(t, secret, upstreams{
		catalog:   catalog.URL,
		order:     order.URL,
		inventory: inventory.URL,
		billing:   billing.URL,
		report:    report.URL,
		geo:       geo.URL,
	})
	adminTok := issueTestToken(t, secret, "a1", roleAdmin, "s1", time.Hour)

	cases := []struct {
		path string
		cap  *capturedReq
		mu   *sync.Mutex
		want string
	}{
		{"/v1/admin/products", catCap, catMu, "/v1/admin/products"},
		{"/v1/admin/orders", ordCap, ordMu, "/v1/admin/orders"},
		{"/v1/admin/delivery-fee", ordCap, ordMu, "/v1/admin/delivery-fee"},
		{"/v1/admin/inventory", invCap, invMu, "/v1/admin/inventory"},
		{"/v1/admin/debts", bilCap, bilMu, "/v1/admin/debts"},
		{"/v1/admin/dashboard/summary", repCap, repMu, "/v1/admin/dashboard/summary"},
		{"/v1/admin/geo/store", geoCap, geoMu, "/v1/admin/geo/store"},
	}
	for _, tc := range cases {
		req := httptest.NewRequest(http.MethodGet, tc.path, nil)
		req.Header.Set("Authorization", "Bearer "+adminTok)
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("%s: status=%d body=%s", tc.path, rec.Code, rec.Body.String())
		}
		tc.mu.Lock()
		got := tc.cap.Path
		tc.mu.Unlock()
		if got != tc.want {
			t.Fatalf("%s: upstream path=%q want=%q", tc.path, got, tc.want)
		}
	}
}

// The admin phone allow-list lives in auth-service, unlike every other
// /v1/admin route, so it needs its own upstream mapping.
func TestProxy_AdminPhonesGoToAuthService(t *testing.T) {
	secret := "secret"
	auth, cap, mu := startMockUpstream(t)
	r := testRouter(t, secret, upstreams{auth: auth.URL})
	adminTok := issueTestToken(t, secret, "a1", roleAdmin, "s1", time.Hour)

	for _, tc := range []struct{ method, path string }{
		{http.MethodGet, "/v1/admin/admin-phones"},
		{http.MethodPost, "/v1/admin/admin-phones"},
		{http.MethodDelete, "/v1/admin/admin-phones/abc"},
	} {
		req := httptest.NewRequest(tc.method, tc.path, nil)
		req.Header.Set("Authorization", "Bearer "+adminTok)
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("%s %s: status=%d body=%s", tc.method, tc.path, rec.Code, rec.Body.String())
		}
		mu.Lock()
		gotPath, gotRole := cap.Path, cap.Header.Get("X-User-Role")
		mu.Unlock()
		if gotPath != tc.path {
			t.Fatalf("upstream path=%q want=%q", gotPath, tc.path)
		}
		if gotRole != roleAdmin {
			t.Fatalf("upstream X-User-Role=%q", gotRole)
		}
	}
}

func TestProxy_AdminPhonesRejectCustomer(t *testing.T) {
	secret := "secret"
	auth, _, _ := startMockUpstream(t)
	r := testRouter(t, secret, upstreams{auth: auth.URL})
	tok := issueTestToken(t, secret, "u1", roleCustomer, "s1", time.Hour)

	req := httptest.NewRequest(http.MethodGet, "/v1/admin/admin-phones", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestProxy_CustomerOrdersForwardsIdentityHeaders(t *testing.T) {
	secret := "secret"
	order, cap, mu := startMockUpstream(t)
	r := testRouter(t, secret, upstreams{order: order.URL})
	tok := issueTestToken(t, secret, "user-42", roleCustomer, "sess-9", time.Hour)

	req := httptest.NewRequest(http.MethodGet, "/v1/orders/me", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	// Spoof attempt — must be stripped then replaced by JWT claims.
	req.Header.Set("X-User-Id", "spoofed")
	req.Header.Set("X-Internal-Token", "evil")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	mu.Lock()
	defer mu.Unlock()
	if cap.Header.Get("X-User-Id") != "user-42" {
		t.Fatalf("X-User-Id=%q", cap.Header.Get("X-User-Id"))
	}
	if cap.Header.Get("X-User-Role") != roleCustomer {
		t.Fatalf("X-User-Role=%q", cap.Header.Get("X-User-Role"))
	}
	if cap.Header.Get("X-Session-Id") != "sess-9" {
		t.Fatalf("X-Session-Id=%q", cap.Header.Get("X-Session-Id"))
	}
	if cap.Header.Get("X-Internal-Token") != "" {
		t.Fatal("client X-Internal-Token must not reach upstream")
	}
}

func TestProxy_UpstreamDown_BadGateway(t *testing.T) {
	r := testRouter(t, "secret", upstreams{auth: "http://127.0.0.1:1"})
	req := httptest.NewRequest(http.MethodPost, "/v1/auth/otp/request", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadGateway {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	errObj, _ := body["error"].(map[string]any)
	if errObj["code"] != "BAD_GATEWAY" {
		t.Fatalf("code=%v", errObj["code"])
	}
}
