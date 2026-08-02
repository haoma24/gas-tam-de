package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCORS_PreflightAllowedOrigin(t *testing.T) {
	auth, _, _ := startMockUpstream(t)
	r := testRouter(t, "secret", upstreams{auth: auth.URL})

	req := httptest.NewRequest(http.MethodOptions, "/v1/auth/otp/request", nil)
	req.Header.Set("Origin", "http://localhost:54321")
	req.Header.Set("Access-Control-Request-Method", "POST")
	req.Header.Set("Access-Control-Request-Headers", "authorization, content-type, x-phone-masked, x-user-id")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status=%d", rec.Code)
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "http://localhost:54321" {
		t.Fatalf("Allow-Origin=%q", got)
	}
	allowHeaders := strings.ToLower(rec.Header().Get("Access-Control-Allow-Headers"))
	if allowHeaders == "" {
		t.Fatal("missing Allow-Headers")
	}
	for _, need := range []string{"authorization", "x-phone-masked", "x-user-id", "x-user-role"} {
		if !strings.Contains(allowHeaders, need) {
			t.Fatalf("Allow-Headers missing %q: %q", need, allowHeaders)
		}
	}
}

func TestCORS_ActualRequestReflectsOrigin(t *testing.T) {
	r := testRouter(t, "secret", upstreams{})
	req := httptest.NewRequest(http.MethodGet, "/v1/hello", nil)
	req.Header.Set("Origin", "http://127.0.0.1:7357")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d", rec.Code)
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "http://127.0.0.1:7357" {
		t.Fatalf("Allow-Origin=%q", got)
	}
}

func TestCORS_DisallowedOrigin(t *testing.T) {
	r := testRouter(t, "secret", upstreams{})
	req := httptest.NewRequest(http.MethodGet, "/v1/hello", nil)
	req.Header.Set("Origin", "https://evil.example")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Fatalf("unexpected Allow-Origin=%q", rec.Header().Get("Access-Control-Allow-Origin"))
	}
}

func TestOriginAllowed(t *testing.T) {
	patterns := parseCORSOrigins(defaultCORSOrigins)
	if !originAllowed("http://localhost:8080", patterns) {
		t.Fatal("localhost port")
	}
	if !originAllowed("http://127.0.0.1:1", patterns) {
		t.Fatal("loopback port")
	}
	if originAllowed("https://localhost:8080", patterns) {
		t.Fatal("https localhost not in default patterns")
	}
	if originAllowed("http://evil.com", patterns) {
		t.Fatal("evil")
	}
	if !originAllowed("https://app.example.com", []string{"https://app.example.com"}) {
		t.Fatal("exact")
	}
	if !originAllowed("https://cdn.example.com", []string{"*"}) {
		t.Fatal("star")
	}
}
