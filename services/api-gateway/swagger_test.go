package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
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

func TestSwaggerDocumentsEveryGatewayOperation(t *testing.T) {
	t.Setenv("SWAGGER_ENABLED", "1")
	r := testRouter(t, "secret", upstreams{})
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/swagger/doc.json", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}

	var spec struct {
		Paths map[string]map[string]json.RawMessage `json:"paths"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &spec); err != nil {
		t.Fatalf("decode generated spec: %v", err)
	}

	want := []string{
		"GET /hello",
		"POST /auth/otp/request", "POST /auth/otp/verify",
		"POST /auth/admin/login", "POST /auth/google",
		"POST /auth/refresh", "POST /auth/logout",
		"GET /me", "PATCH /me",
		"GET /admin/admin-phones", "POST /admin/admin-phones", "DELETE /admin/admin-phones/{id}",
		"GET /admin/admin-accounts", "POST /admin/admin-accounts", "PATCH /admin/admin-accounts/{id}",
		"GET /products", "GET /admin/products", "POST /admin/products",
		"GET /admin/products/{id}", "PATCH /admin/products/{id}",
		"GET /geo/store", "GET /geo/search", "POST /geo/check", "PUT /admin/geo/store",
		"POST /orders/quote", "POST /orders", "POST /orders/{id}/cancel",
		"GET /orders/me/defaults", "GET /orders/me",
		"GET /admin/orders", "GET /admin/orders/customers", "GET /admin/orders/{id}",
		"POST /admin/orders/{id}/complete",
		"GET /admin/delivery-fee", "PUT /admin/delivery-fee",
		"GET /admin/desk-settings", "PUT /admin/desk-settings",
		"GET /stock/levels", "GET /admin/inventory", "POST /admin/inventory",
		"GET /admin/debts", "GET /admin/dashboard/summary",
	}

	operationCount := 0
	for _, methods := range spec.Paths {
		operationCount += len(methods)
	}
	if operationCount != len(want) {
		t.Errorf("documented operations=%d want=%d", operationCount, len(want))
	}
	for _, operation := range want {
		method, path, _ := strings.Cut(operation, " ")
		methods, ok := spec.Paths[path]
		if !ok {
			t.Errorf("missing documented path %s", path)
			continue
		}
		if _, ok := methods[strings.ToLower(method)]; !ok {
			t.Errorf("missing documented operation %s", operation)
		}
	}
}
