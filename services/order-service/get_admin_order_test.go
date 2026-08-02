package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestGetAdminOrderLatLng covers T5.2.1: admin GET by id returns delivery lat/lng.
func TestGetAdminOrderLatLng(t *testing.T) {
	svc, r := testOrderRouter(t, &stubGeo{}, &stubCatalog{})

	insertTestOrderDesk(t, svc, "ord-nav", "u1", "PENDING", "2026-08-02T09:00:00Z",
		"Nguyen A", "090***4567", "12 Tran Hung Dao", 3.25, 10.762622, 106.660172)

	req := httptest.NewRequest(http.MethodGet, "/v1/admin/orders/ord-nav", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}

	var got orderView
	if err := json.Unmarshal(rr.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.ID != "ord-nav" {
		t.Fatalf("id=%q", got.ID)
	}
	if got.Lat != 10.762622 || got.Lng != 106.660172 {
		t.Fatalf("lat/lng=%v,%v (want destination coords for navigation)", got.Lat, got.Lng)
	}
	if got.CustomerName != "Nguyen A" || got.AddressText != "12 Tran Hung Dao" {
		t.Fatalf("unexpected order fields: %+v", got)
	}
	if got.Stt != 0 {
		t.Fatalf("detail must omit list STT, got %d", got.Stt)
	}
	if len(got.Items) != 1 {
		t.Fatalf("items=%d", len(got.Items))
	}

	// Raw JSON keys must be clearly named lat / lng for clients (T5.2.2/T5.2.3).
	var raw map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &raw); err != nil {
		t.Fatal(err)
	}
	if _, ok := raw["lat"]; !ok {
		t.Fatal("missing json key lat")
	}
	if _, ok := raw["lng"]; !ok {
		t.Fatal("missing json key lng")
	}
}

func TestGetAdminOrderNotFound(t *testing.T) {
	_, r := testOrderRouter(t, &stubGeo{}, &stubCatalog{})

	req := httptest.NewRequest(http.MethodGet, "/v1/admin/orders/missing", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestListAdminOrdersExposesLatLng(t *testing.T) {
	svc, r := testOrderRouter(t, &stubGeo{}, &stubCatalog{})
	insertTestOrderDesk(t, svc, "ord-list", "u1", "PENDING", "2026-08-02T09:00:00Z",
		"Le C", "092***3333", "3 Pasteur", 1.5, 10.771, 106.698)

	req := httptest.NewRequest(http.MethodGet, "/v1/admin/orders", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}

	var payload struct {
		Orders []orderView `json:"orders"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if len(payload.Orders) != 1 {
		t.Fatalf("want 1, got %d", len(payload.Orders))
	}
	o := payload.Orders[0]
	if o.Lat != 10.771 || o.Lng != 106.698 {
		t.Fatalf("list lat/lng=%v,%v", o.Lat, o.Lng)
	}
}
