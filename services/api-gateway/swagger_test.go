package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSwaggerDisabledByDefault(t *testing.T) {
	t.Setenv("SWAGGER_ENABLED", "0")
	r := testRouter(t, "secret", upstreams{})
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/swagger/doc.json", nil))
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status=%d want=%d", rec.Code, http.StatusNotFound)
	}
}

func TestSwaggerEnabledServesGeneratedSpec(t *testing.T) {
	t.Setenv("SWAGGER_ENABLED", "1")
	r := testRouter(t, "secret", upstreams{})
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/swagger/doc.json", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var spec struct {
		Info struct {
			Title string `json:"title"`
		} `json:"info"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &spec); err != nil {
		t.Fatalf("decode generated spec: %v", err)
	}
	if spec.Info.Title != "Gas Tam De API" {
		t.Fatalf("generated spec title=%q", spec.Info.Title)
	}
}
