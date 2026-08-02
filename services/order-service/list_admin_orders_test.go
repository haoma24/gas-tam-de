package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func insertTestOrder(t *testing.T, svc *orderService, id, userID, status, createdAt string) {
	t.Helper()
	insertTestOrderDesk(t, svc, id, userID, status, createdAt,
		"Khach "+id, "090***1111", "1 Le Loi", 2.5, 10.78, 106.70)
}

func insertTestOrderDesk(
	t *testing.T, svc *orderService,
	id, userID, status, createdAt,
	customerName, phoneMasked, addressText string,
	distanceKm, lat, lng float64,
) {
	t.Helper()
	_, err := svc.db.Exec(`
		INSERT INTO orders (
			id, user_id, customer_name, phone_hash, phone_masked, address_text,
			lat, lng, distance_km, delivery_fee, subtotal, total, status, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		id, userID, customerName, "uid:"+userID, phoneMasked, addressText,
		lat, lng, distanceKm, 0, 100000, 100000, status, createdAt,
	)
	if err != nil {
		t.Fatal(err)
	}
	_, err = svc.db.Exec(`
		INSERT INTO order_items (
			id, order_id, product_id, product_sku, product_name, unit_price, qty, line_total
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		"item-"+id, id, "p1", "GAS12", "Gas 12kg", 100000, 1, 100000,
	)
	if err != nil {
		t.Fatal(err)
	}
}

func TestListAdminOrdersFIFO(t *testing.T) {
	svc, r := testOrderRouter(t, &stubGeo{}, &stubCatalog{})

	// B created after A; both PENDING — A must appear first (created_at ASC).
	insertTestOrder(t, svc, "ord-b", "user-b", "PENDING", "2026-08-02T10:00:00Z")
	insertTestOrder(t, svc, "ord-a", "user-a", "PENDING", "2026-08-02T09:00:00Z")
	insertTestOrder(t, svc, "ord-done", "user-c", "COMPLETED", "2026-08-02T08:00:00Z")

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
	if len(payload.Orders) != 2 {
		t.Fatalf("want 2 PENDING orders, got %d", len(payload.Orders))
	}
	if payload.Orders[0].ID != "ord-a" || payload.Orders[1].ID != "ord-b" {
		t.Fatalf("FIFO order got %q then %q", payload.Orders[0].ID, payload.Orders[1].ID)
	}
	if payload.Orders[0].Stt != 1 || payload.Orders[1].Stt != 2 {
		t.Fatalf("STT want 1,2 got %d,%d", payload.Orders[0].Stt, payload.Orders[1].Stt)
	}
	if payload.Orders[0].CreatedAt != "2026-08-02T09:00:00Z" {
		t.Fatalf("created_at=%q", payload.Orders[0].CreatedAt)
	}
	if payload.Orders[0].CustomerName == "" || payload.Orders[0].AddressText == "" {
		t.Fatal("expected basic order fields")
	}
	if len(payload.Orders[0].Items) != 1 {
		t.Fatalf("items=%d", len(payload.Orders[0].Items))
	}
}

// TestListAdminOrdersDeskColumns covers T5.1.2: STT, tên, SĐT, địa chỉ, km, thời gian.
func TestListAdminOrdersDeskColumns(t *testing.T) {
	svc, r := testOrderRouter(t, &stubGeo{}, &stubCatalog{})

	insertTestOrderDesk(t, svc, "ord-newer", "u2", "PENDING", "2026-08-02T11:00:00Z",
		"Tran B", "091***2222", "2 Nguyen Hue", 4.75, 10.79, 106.71)
	insertTestOrderDesk(t, svc, "ord-older", "u1", "PENDING", "2026-08-02T09:30:00Z",
		"Nguyen A", "090***4567", "12 Tran Hung Dao", 3.25, 10.75, 106.65)

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
	if len(payload.Orders) != 2 {
		t.Fatalf("want 2, got %d", len(payload.Orders))
	}

	first := payload.Orders[0]
	if first.Stt != 1 {
		t.Fatalf("stt=%d", first.Stt)
	}
	if first.ID != "ord-older" {
		t.Fatalf("id=%q", first.ID)
	}
	if first.CustomerName != "Nguyen A" {
		t.Fatalf("customer_name=%q", first.CustomerName)
	}
	if first.PhoneMasked != "090***4567" {
		t.Fatalf("phone_masked=%q", first.PhoneMasked)
	}
	if first.AddressText != "12 Tran Hung Dao" {
		t.Fatalf("address_text=%q", first.AddressText)
	}
	if first.DistanceKm != 3.25 {
		t.Fatalf("distance_km=%v", first.DistanceKm)
	}
	if first.Lat != 10.75 || first.Lng != 106.65 {
		t.Fatalf("lat/lng=%v,%v", first.Lat, first.Lng)
	}
	if first.CreatedAt != "2026-08-02T09:30:00Z" {
		t.Fatalf("created_at=%q", first.CreatedAt)
	}

	second := payload.Orders[1]
	if second.Stt != 2 || second.CustomerName != "Tran B" || second.DistanceKm != 4.75 {
		t.Fatalf("second desk row unexpected: %+v", second)
	}

	// Customer list must not include stt.
	reqMe := httptest.NewRequest(http.MethodGet, "/v1/orders/me", nil)
	reqMe.Header.Set("X-User-Id", "u1")
	reqMe.Header.Set("X-User-Role", "customer")
	reqMe.Header.Set("X-Phone-Masked", "090***4567")
	rrMe := httptest.NewRecorder()
	r.ServeHTTP(rrMe, reqMe)
	if rrMe.Code != http.StatusOK {
		t.Fatalf("me status=%d body=%s", rrMe.Code, rrMe.Body.String())
	}
	var mePayload struct {
		Orders []orderView `json:"orders"`
	}
	if err := json.Unmarshal(rrMe.Body.Bytes(), &mePayload); err != nil {
		t.Fatal(err)
	}
	if len(mePayload.Orders) != 1 || mePayload.Orders[0].Stt != 0 {
		t.Fatalf("customer list should omit stt, got %+v", mePayload.Orders)
	}
}

func TestListAdminOrdersStatusFilter(t *testing.T) {
	svc, r := testOrderRouter(t, &stubGeo{}, &stubCatalog{})
	insertTestOrder(t, svc, "ord-p", "user-1", "PENDING", "2026-08-02T09:00:00Z")
	insertTestOrder(t, svc, "ord-c", "user-2", "COMPLETED", "2026-08-02T08:00:00Z")

	req := httptest.NewRequest(http.MethodGet, "/v1/admin/orders?status=COMPLETED", nil)
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
	if len(payload.Orders) != 1 || payload.Orders[0].ID != "ord-c" {
		t.Fatalf("want COMPLETED ord-c, got %+v", payload.Orders)
	}
}

func TestListAdminOrdersInvalidStatus(t *testing.T) {
	_, r := testOrderRouter(t, &stubGeo{}, &stubCatalog{})
	req := httptest.NewRequest(http.MethodGet, "/v1/admin/orders?status=SHIPPED", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status=%d", rr.Code)
	}
}

func TestParseAdminOrderStatusFilter(t *testing.T) {
	cases := []struct {
		in   string
		want string
		ok   bool
	}{
		{"", "PENDING", true},
		{"  pending ", "PENDING", true},
		{"CANCELLED", "CANCELLED", true},
		{"bogus", "", false},
	}
	for _, tc := range cases {
		got, ok := parseAdminOrderStatusFilter(tc.in)
		if ok != tc.ok || got != tc.want {
			t.Fatalf("in=%q got=(%q,%v) want=(%q,%v)", tc.in, got, ok, tc.want, tc.ok)
		}
	}
}
