package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSecurityHeaders_PresentOnOK(t *testing.T) {
	r := testRouter(t, "secret", upstreams{})
	req := httptest.NewRequest(http.MethodGet, "/v1/hello", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	assertSecurityHeaders(t, rec.Header())
}

func TestSecurityHeaders_PresentOnError(t *testing.T) {
	r := testRouter(t, "secret", upstreams{auth: "http://127.0.0.1:1"})
	req := httptest.NewRequest(http.MethodPost, "/v1/auth/otp/request", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadGateway {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	assertSecurityHeaders(t, rec.Header())
}

func TestSecurityHeaders_PresentOnPreflight(t *testing.T) {
	auth, _, _ := startMockUpstream(t)
	r := testRouter(t, "secret", upstreams{auth: auth.URL})
	req := httptest.NewRequest(http.MethodOptions, "/v1/auth/otp/request", nil)
	req.Header.Set("Origin", "http://localhost:54321")
	req.Header.Set("Access-Control-Request-Method", "POST")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("status=%d", rec.Code)
	}
	assertSecurityHeaders(t, rec.Header())
}

func assertSecurityHeaders(t *testing.T, h http.Header) {
	t.Helper()
	cases := map[string]string{
		"X-Content-Type-Options":  "nosniff",
		"X-Frame-Options":         "DENY",
		"Referrer-Policy":         "no-referrer",
		"Permissions-Policy":      "geolocation=(), microphone=(), camera=()",
		"Content-Security-Policy": "frame-ancestors 'none'",
	}
	for k, want := range cases {
		if got := h.Get(k); got != want {
			t.Fatalf("%s=%q want %q", k, got, want)
		}
	}
}

func TestProxy_UpstreamDown_NoInternalLeak(t *testing.T) {
	r := testRouter(t, "secret", upstreams{auth: "http://127.0.0.1:1"})
	req := httptest.NewRequest(http.MethodPost, "/v1/auth/otp/request", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadGateway {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	var parsed map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &parsed); err != nil {
		t.Fatalf("json: %v body=%s", err, body)
	}
	errObj, _ := parsed["error"].(map[string]any)
	if errObj["code"] != "BAD_GATEWAY" {
		t.Fatalf("code=%v", errObj["code"])
	}
	if errObj["message"] != "upstream unavailable" {
		t.Fatalf("message=%v", errObj["message"])
	}
	lower := strings.ToLower(body)
	for _, leak := range []string{"127.0.0.1", "connection refused", "dial tcp", "stack", "goroutine"} {
		if strings.Contains(lower, leak) {
			t.Fatalf("response leaked %q: %s", leak, body)
		}
	}
}

func TestProxy_InvalidUpstream_NoURLLeak(t *testing.T) {
	r := testRouter(t, "secret", upstreams{auth: "not-a-url"})
	req := httptest.NewRequest(http.MethodPost, "/v1/auth/otp/request", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadGateway {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	if strings.Contains(body, "not-a-url") || strings.Contains(strings.ToLower(body), "invalid upstream") {
		t.Fatalf("leaked upstream config: %s", body)
	}
	var parsed map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &parsed); err != nil {
		t.Fatal(err)
	}
	errObj, _ := parsed["error"].(map[string]any)
	if errObj["message"] != "upstream unavailable" {
		t.Fatalf("message=%v", errObj["message"])
	}
}

func TestProxy_StripsUpstreamServerHeader(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Server", "internal-auth/1.0")
		w.Header().Set("X-Powered-By", "secret-stack")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	t.Cleanup(srv.Close)

	r := testRouter(t, "secret", upstreams{auth: srv.URL})
	req := httptest.NewRequest(http.MethodPost, "/v1/auth/otp/request", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if got := rec.Header().Get("Server"); got != "" {
		t.Fatalf("Server header leaked: %q", got)
	}
	if got := rec.Header().Get("X-Powered-By"); got != "" {
		t.Fatalf("X-Powered-By leaked: %q", got)
	}
}
