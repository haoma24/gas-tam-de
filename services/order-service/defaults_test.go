package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestGetMyOrderDefaults(t *testing.T) {
	db := openTestOrderDB(t)
	svc := &orderService{db: db}
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/orders/me/defaults", svc.handleGetMyOrderDefaults)

	reqEmpty := httptest.NewRequest(http.MethodGet, "/v1/orders/me/defaults", nil)
	reqEmpty.Header.Set("X-User-Id", "user-new")
	reqEmpty.Header.Set("X-User-Role", "customer")
	reqEmpty.Header.Set("X-Phone-Masked", "090***0000")
	recEmpty := httptest.NewRecorder()
	mux.ServeHTTP(recEmpty, reqEmpty)
	if recEmpty.Code != http.StatusOK {
		t.Fatalf("empty status=%d %s", recEmpty.Code, recEmpty.Body.String())
	}
	var empty map[string]any
	_ = json.Unmarshal(recEmpty.Body.Bytes(), &empty)
	if empty["has_defaults"] != false {
		t.Fatalf("expected has_defaults=false got %v", empty)
	}

	now := time.Now().UTC().Format(time.RFC3339Nano)
	_, err := db.Exec(`
		INSERT INTO orders (
			id, user_id, customer_name, phone_hash, phone_masked, address_text,
			lat, lng, distance_km, delivery_fee, subtotal, total, status, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 'PENDING', ?)
	`, "o1", "user-1", "Nguyen A", "ph", "090***4567", "12 Le Loi",
		10.77, 106.70, 2.5, 10000, 100000, 110000, now)
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/v1/orders/me/defaults", nil)
	req.Header.Set("X-User-Id", "user-1")
	req.Header.Set("X-User-Role", "customer")
	req.Header.Set("X-Phone-Masked", "090***4567")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d %s", rec.Code, rec.Body.String())
	}
	var got map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &got)
	if got["has_defaults"] != true {
		t.Fatalf("has_defaults=%v", got["has_defaults"])
	}
	if got["customer_name"] != "Nguyen A" {
		t.Fatalf("customer_name=%v", got["customer_name"])
	}
	if got["address_text"] != "12 Le Loi" {
		t.Fatalf("address_text=%v", got["address_text"])
	}
}
