package httpx

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestMountHealthIgnoresDependencies(t *testing.T) {
	r := NewRouter("test-service")
	MountHealth(r, "test-service")
	MountReady(r, "test-service", ReadyCheck{
		Name:  "nats",
		Check: func() error { return errors.New("connection refused") },
	})

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/healthz", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("/healthz = %d, want 200 even with a failing dependency", rec.Code)
	}

	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/readyz", nil))
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("/readyz = %d, want 503", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "connection refused") {
		t.Fatalf("/readyz body should name the failing dependency: %s", rec.Body.String())
	}
}

func TestMountReadyOKWhenDependenciesPass(t *testing.T) {
	r := NewRouter("test-service")
	MountReady(r, "test-service", ReadyCheck{Name: "nats", Check: func() error { return nil }})

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/readyz", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("/readyz = %d, want 200", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), `"ready"`) {
		t.Fatalf("/readyz body = %s", rec.Body.String())
	}
}
