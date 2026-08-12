package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"gas-tam-de/pkg/httpx"
)

func TestUpstreamHealthReadyWhenHealthzOK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/healthz" {
			t.Errorf("probe path=%q, want /healthz", r.URL.Path)
		}
		httpx.JSON(w, http.StatusOK, map[string]any{"status": "ok"})
	}))
	defer srv.Close()

	if err := newUpstreamHealth("inventory", srv.URL).Check(); err != nil {
		t.Fatalf("Check() = %v, want nil", err)
	}
}

func TestUpstreamHealthNamesURLWhenUnreachable(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	unreachable := srv.URL
	srv.Close()

	err := newUpstreamHealth("inventory", unreachable).Check()
	if err == nil {
		t.Fatal("Check() = nil, want error for a closed listener")
	}
	// /readyz shows this text verbatim — the URL is the whole diagnosis.
	if !strings.Contains(err.Error(), unreachable) {
		t.Fatalf("error %q does not name %q", err, unreachable)
	}
	if !strings.Contains(err.Error(), "inventory") {
		t.Fatalf("error %q does not name the dependency", err)
	}
}

func TestUpstreamHealthRejectsNon200(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer srv.Close()

	err := newUpstreamHealth("catalog", srv.URL).Check()
	if err == nil {
		t.Fatal("Check() = nil, want error for status 503")
	}
	if !strings.Contains(err.Error(), "503") {
		t.Fatalf("error %q does not carry the status", err)
	}
}

func TestUpstreamHealthRejectsEmptyURL(t *testing.T) {
	if err := newUpstreamHealth("inventory", "   ").Check(); err == nil {
		t.Fatal("Check() = nil, want error for an empty URL")
	}
}

// TestUpstreamReadyCheckReportsDependencyName pins the /readyz contract:
// 503 plus the failing dependency's name, which is how an operator learns that
// checkout is about to fail before a customer finds out.
func TestUpstreamReadyCheckReportsDependencyName(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	unreachable := srv.URL
	srv.Close()

	r := httpx.NewRouter("order-test")
	httpx.MountReady(r, "order-service", upstreamReadyCheck("inventory", unreachable))

	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/readyz", nil))

	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("status=%d, want 503; body=%s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), "inventory") {
		t.Fatalf("body %s does not name the failing dependency", rr.Body.String())
	}
}
