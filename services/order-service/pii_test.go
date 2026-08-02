package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestMaskPhoneDisplay(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"+84901234567", "090***4567"},
		{"0901234567", "090***4567"},
		{"84901234567", "090***4567"},
		{"090***4567", "090***4567"},
		{"123", "***"},
		{"", "***"},
	}
	for _, tc := range cases {
		got := maskPhoneDisplay(tc.in)
		if got != tc.want {
			t.Fatalf("maskPhoneDisplay(%q)=%q want %q", tc.in, got, tc.want)
		}
	}
}

func TestEnsurePhoneMasked(t *testing.T) {
	if got := ensurePhoneMasked("  0901234567  "); got != "090***4567" {
		t.Fatalf("ensure=%q", got)
	}
	if got := ensurePhoneMasked("090***4567"); got != "090***4567" {
		t.Fatalf("already masked=%q", got)
	}
	if got := ensurePhoneMasked(""); got != "" {
		t.Fatalf("empty=%q", got)
	}
}

func TestCreateOrderRemasksFullPhoneHeader(t *testing.T) {
	geo := &stubGeo{result: geoCheckResult{DistanceKm: 2, InRange: true, MaxRadiusKm: 10}}
	catalog := &stubCatalog{products: []catalogProduct{
		{ID: "p1", SKU: "GAS12", Name: "Gas 12kg", SalePrice: 100000, Active: true},
	}}
	svc, r := testOrderRouter(t, geo, catalog)

	req := httptest.NewRequest(http.MethodPost, "/v1/orders", bytes.NewReader(validBody("p1")))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-Id", "user-1")
	req.Header.Set("X-User-Role", "customer")
	req.Header.Set("X-Phone-Masked", "0901234567") // accidental full number
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}

	raw := rr.Body.String()
	if strings.Contains(raw, "0901234567") {
		t.Fatalf("response leaked full phone: %s", raw)
	}
	if strings.Contains(raw, "phone_hash") || strings.Contains(raw, "phone_e164") {
		t.Fatalf("response leaked secret phone fields: %s", raw)
	}

	var out orderView
	if err := json.Unmarshal(rr.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	if out.PhoneMasked != "090***4567" {
		t.Fatalf("phone_masked=%q", out.PhoneMasked)
	}

	var stored string
	if err := svc.db.QueryRow(`SELECT phone_masked FROM orders WHERE id = ?`, out.ID).Scan(&stored); err != nil {
		t.Fatal(err)
	}
	if stored != "090***4567" {
		t.Fatalf("persisted phone_masked=%q", stored)
	}
}

func TestListMyOrdersMasksPII(t *testing.T) {
	geo := &stubGeo{result: geoCheckResult{DistanceKm: 1.5, InRange: true, MaxRadiusKm: 10}}
	catalog := &stubCatalog{products: []catalogProduct{
		{ID: "p1", SKU: "GAS12", Name: "Gas 12kg", SalePrice: 100000, Active: true},
	}}
	_, r := testOrderRouter(t, geo, catalog)

	createReq := httptest.NewRequest(http.MethodPost, "/v1/orders", bytes.NewReader(validBody("p1")))
	customerHeaders(createReq)
	createRR := httptest.NewRecorder()
	r.ServeHTTP(createRR, createReq)
	if createRR.Code != http.StatusCreated {
		t.Fatalf("create status=%d body=%s", createRR.Code, createRR.Body.String())
	}

	listReq := httptest.NewRequest(http.MethodGet, "/v1/orders/me", nil)
	customerHeaders(listReq)
	listRR := httptest.NewRecorder()
	r.ServeHTTP(listRR, listReq)
	if listRR.Code != http.StatusOK {
		t.Fatalf("list status=%d body=%s", listRR.Code, listRR.Body.String())
	}

	raw := listRR.Body.String()
	if strings.Contains(raw, "phone_hash") || strings.Contains(raw, "phone_e164") {
		t.Fatalf("list leaked secret fields: %s", raw)
	}

	var payload struct {
		Orders []orderView `json:"orders"`
	}
	if err := json.Unmarshal(listRR.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if len(payload.Orders) != 1 {
		t.Fatalf("orders=%d", len(payload.Orders))
	}
	if payload.Orders[0].PhoneMasked != "090***4567" {
		t.Fatalf("phone_masked=%q", payload.Orders[0].PhoneMasked)
	}
	if payload.Orders[0].UserID != "user-1" {
		t.Fatalf("user_id=%q", payload.Orders[0].UserID)
	}
}

func TestListMyOrdersUnauthorized(t *testing.T) {
	_, r := testOrderRouter(t, &stubGeo{}, &stubCatalog{})
	req := httptest.NewRequest(http.MethodGet, "/v1/orders/me", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d", rr.Code)
	}
}
