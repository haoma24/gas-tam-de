package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"gas-tam-de/pkg/httpx"
	"gas-tam-de/pkg/sqlite"
)

func openTestOrderDB(t *testing.T) *sql.DB {
	t.Helper()
	dir := t.TempDir()
	db, err := sqlite.Open(filepath.Join(dir, "order.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := migrate(db); err != nil {
		t.Fatal(err)
	}
	return db
}

type stubGeo struct {
	result geoCheckResult
	err    error
	calls  int
	lastLat, lastLng float64
}

func (s *stubGeo) Check(_ context.Context, lat, lng float64) (geoCheckResult, error) {
	s.calls++
	s.lastLat, s.lastLng = lat, lng
	if s.err != nil {
		return geoCheckResult{}, s.err
	}
	return s.result, nil
}

type stubCatalog struct {
	products []catalogProduct
	err      error
}

func (s *stubCatalog) ListActive(_ context.Context) ([]catalogProduct, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.products, nil
}

func testOrderRouter(t *testing.T, geo geoChecker, catalog productCatalog) (*orderService, http.Handler) {
	t.Helper()
	svc := &orderService{
		db:      openTestOrderDB(t),
		geo:     geo,
		catalog: catalog,
		billing: noopBillingRecorder{},
		bus:     noopOrderPublisher{},
	}
	return svc, mountOrderTestRoutes(svc)
}

func testOrderRouterWithBus(t *testing.T, geo geoChecker, catalog productCatalog, bus orderPublisher) (*orderService, http.Handler) {
	t.Helper()
	svc := &orderService{
		db:      openTestOrderDB(t),
		geo:     geo,
		catalog: catalog,
		billing: noopBillingRecorder{},
		bus:     bus,
	}
	return svc, mountOrderTestRoutes(svc)
}

func mountOrderTestRoutes(svc *orderService) http.Handler {
	r := httpx.NewRouter("order-test")
	r.Post("/v1/orders/quote", svc.handleQuoteOrder)
	r.Post("/v1/orders", svc.handleCreateOrder)
	r.Get("/v1/orders/me", svc.handleListMyOrders)
	r.Get("/v1/admin/orders", svc.handleListAdminOrders)
	r.Get("/v1/admin/orders/{id}", svc.handleGetAdminOrder)
	r.Post("/v1/admin/orders/{id}/complete", svc.handleCompleteOrder)
	return r
}

func customerHeaders(req *http.Request) {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-Id", "user-1")
	req.Header.Set("X-User-Role", "customer")
	req.Header.Set("X-Phone-Masked", "090***4567")
}

func validBody(productID string) []byte {
	b, _ := json.Marshal(map[string]any{
		"customer_name": "Nguyen Van A",
		"address_text":  "123 Le Loi, Q1",
		"lat":           10.78,
		"lng":           106.70,
		"items": []map[string]any{
			{"product_id": productID, "qty": 2},
		},
	})
	return b
}

func TestCreateOrderHappyPath(t *testing.T) {
	geo := &stubGeo{result: geoCheckResult{DistanceKm: 3.2, InRange: true, MaxRadiusKm: 10}}
	catalog := &stubCatalog{products: []catalogProduct{
		{ID: "p1", SKU: "GAS12", Name: "Gas 12kg", SalePrice: 450000, Active: true},
	}}
	svc, r := testOrderRouter(t, geo, catalog)

	req := httptest.NewRequest(http.MethodPost, "/v1/orders", bytes.NewReader(validBody("p1")))
	customerHeaders(req)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	if geo.calls != 1 || geo.lastLat != 10.78 || geo.lastLng != 106.70 {
		t.Fatalf("geo calls=%d lat=%v lng=%v", geo.calls, geo.lastLat, geo.lastLng)
	}

	var out orderView
	if err := json.Unmarshal(rr.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	if out.ID == "" || out.UserID != "user-1" || out.Status != "PENDING" {
		t.Fatalf("order=%+v", out)
	}
	if out.PhoneMasked != "090***4567" || out.CustomerName != "Nguyen Van A" {
		t.Fatalf("pii/name=%+v", out)
	}
	if out.DistanceKm != 3.2 || out.DeliveryFee != 0 {
		t.Fatalf("distance/fee=%+v", out)
	}
	if out.Subtotal != 900000 || out.Total != 900000 {
		t.Fatalf("totals sub=%d total=%d", out.Subtotal, out.Total)
	}
	if len(out.Items) != 1 || out.Items[0].Qty != 2 || out.Items[0].UnitPrice != 450000 {
		t.Fatalf("items=%+v", out.Items)
	}

	var count int
	if err := svc.db.QueryRow(`SELECT COUNT(*) FROM orders WHERE id = ?`, out.ID).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("orders count=%d", count)
	}
	if err := svc.db.QueryRow(`SELECT COUNT(*) FROM order_items WHERE order_id = ?`, out.ID).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("items count=%d", count)
	}

	var status string
	var total, subtotal, deliveryFee int64
	var distanceKm float64
	if err := svc.db.QueryRow(
		`SELECT status, total, subtotal, delivery_fee, distance_km FROM orders WHERE id = ?`, out.ID,
	).Scan(&status, &total, &subtotal, &deliveryFee, &distanceKm); err != nil {
		t.Fatal(err)
	}
	if status != "PENDING" || total != 900000 || subtotal != 900000 || deliveryFee != 0 || distanceKm != 3.2 {
		t.Fatalf("persisted order status=%s total=%d sub=%d fee=%d dist=%v", status, total, subtotal, deliveryFee, distanceKm)
	}
}

func TestCreateOrderUnauthorized(t *testing.T) {
	_, r := testOrderRouter(t, &stubGeo{result: geoCheckResult{InRange: true}}, &stubCatalog{})
	req := httptest.NewRequest(http.MethodPost, "/v1/orders", bytes.NewReader(validBody("p1")))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d", rr.Code)
	}
}

func TestCreateOrderForbiddenAdmin(t *testing.T) {
	_, r := testOrderRouter(t, &stubGeo{result: geoCheckResult{InRange: true}}, &stubCatalog{})
	req := httptest.NewRequest(http.MethodPost, "/v1/orders", bytes.NewReader(validBody("p1")))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-Id", "admin-1")
	req.Header.Set("X-User-Role", "admin")
	req.Header.Set("X-Phone-Masked", "090***4567")
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestCreateOrderOutOfRange(t *testing.T) {
	geo := &stubGeo{result: geoCheckResult{DistanceKm: 12.5, InRange: false, MaxRadiusKm: 10}}
	catalog := &stubCatalog{products: []catalogProduct{
		{ID: "p1", SKU: "GAS12", Name: "Gas 12kg", SalePrice: 450000, Active: true},
	}}
	_, r := testOrderRouter(t, geo, catalog)

	req := httptest.NewRequest(http.MethodPost, "/v1/orders", bytes.NewReader(validBody("p1")))
	customerHeaders(req)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var payload map[string]any
	_ = json.Unmarshal(rr.Body.Bytes(), &payload)
	errObj, _ := payload["error"].(map[string]any)
	if errObj["code"] != "OUT_OF_RANGE" {
		t.Fatalf("error=%v", errObj)
	}
}

func TestCreateOrderProductNotFound(t *testing.T) {
	geo := &stubGeo{result: geoCheckResult{DistanceKm: 1, InRange: true, MaxRadiusKm: 10}}
	catalog := &stubCatalog{products: []catalogProduct{
		{ID: "other", SKU: "X", Name: "X", SalePrice: 1, Active: true},
	}}
	_, r := testOrderRouter(t, geo, catalog)

	req := httptest.NewRequest(http.MethodPost, "/v1/orders", bytes.NewReader(validBody("missing")))
	customerHeaders(req)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestCreateOrderValidationEmptyItems(t *testing.T) {
	_, r := testOrderRouter(t, &stubGeo{}, &stubCatalog{})
	body, _ := json.Marshal(map[string]any{
		"customer_name": "A",
		"address_text":  "B",
		"lat":           10,
		"lng":           106,
		"items":         []any{},
	})
	req := httptest.NewRequest(http.MethodPost, "/v1/orders", bytes.NewReader(body))
	customerHeaders(req)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status=%d", rr.Code)
	}
}

func TestCreateOrderGeoUnavailable(t *testing.T) {
	geo := &stubGeo{err: errors.New("dial timeout")}
	catalog := &stubCatalog{products: []catalogProduct{
		{ID: "p1", SKU: "GAS12", Name: "Gas 12kg", SalePrice: 450000, Active: true},
	}}
	_, r := testOrderRouter(t, geo, catalog)

	req := httptest.NewRequest(http.MethodPost, "/v1/orders", bytes.NewReader(validBody("p1")))
	customerHeaders(req)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadGateway {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestCreateOrderMergesDuplicateProductLines(t *testing.T) {
	geo := &stubGeo{result: geoCheckResult{DistanceKm: 2, InRange: true, MaxRadiusKm: 10}}
	catalog := &stubCatalog{products: []catalogProduct{
		{ID: "p1", SKU: "GAS12", Name: "Gas 12kg", SalePrice: 100000, Active: true},
	}}
	_, r := testOrderRouter(t, geo, catalog)

	body, _ := json.Marshal(map[string]any{
		"customer_name": "A",
		"address_text":  "B",
		"lat":           10.7,
		"lng":           106.7,
		"items": []map[string]any{
			{"product_id": "p1", "qty": 1},
			{"product_id": "p1", "qty": 3},
		},
	})
	req := httptest.NewRequest(http.MethodPost, "/v1/orders", bytes.NewReader(body))
	customerHeaders(req)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var out orderView
	_ = json.Unmarshal(rr.Body.Bytes(), &out)
	if len(out.Items) != 1 || out.Items[0].Qty != 4 || out.Subtotal != 400000 {
		t.Fatalf("items=%+v subtotal=%d", out.Items, out.Subtotal)
	}
}

func TestCreateOrderAppliesDeliveryFeeWhenEnabled(t *testing.T) {
	geo := &stubGeo{result: geoCheckResult{DistanceKm: 3.2, InRange: true, MaxRadiusKm: 10}}
	catalog := &stubCatalog{products: []catalogProduct{
		{ID: "p1", SKU: "GAS12", Name: "Gas 12kg", SalePrice: 450000, Active: true},
	}}
	svc, r := testOrderRouter(t, geo, catalog)
	if err := seedDeliveryFee(svc.db, deliveryFeeSeedConfig{Enabled: true, Seed: true}); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodPost, "/v1/orders", bytes.NewReader(validBody("p1")))
	customerHeaders(req)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}

	var out orderView
	if err := json.Unmarshal(rr.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	const wantFee int64 = 10000
	if out.DeliveryFee != wantFee || out.Subtotal != 900000 || out.Total != 900000+wantFee {
		t.Fatalf("fee=%d sub=%d total=%d want fee=%d", out.DeliveryFee, out.Subtotal, out.Total, wantFee)
	}

	var deliveryFee, total int64
	if err := svc.db.QueryRow(
		`SELECT delivery_fee, total FROM orders WHERE id = ?`, out.ID,
	).Scan(&deliveryFee, &total); err != nil {
		t.Fatal(err)
	}
	if deliveryFee != wantFee || total != 900000+wantFee {
		t.Fatalf("persisted fee=%d total=%d", deliveryFee, total)
	}
}

func TestCreateOrderDeliveryFeeZeroWhenDisabled(t *testing.T) {
	geo := &stubGeo{result: geoCheckResult{DistanceKm: 7.5, InRange: true, MaxRadiusKm: 10}}
	catalog := &stubCatalog{products: []catalogProduct{
		{ID: "p1", SKU: "GAS12", Name: "Gas 12kg", SalePrice: 450000, Active: true},
	}}
	svc, r := testOrderRouter(t, geo, catalog)
	if err := seedDeliveryFee(svc.db, deliveryFeeSeedConfig{Enabled: false, Seed: true}); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodPost, "/v1/orders", bytes.NewReader(validBody("p1")))
	customerHeaders(req)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}

	var out orderView
	if err := json.Unmarshal(rr.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	if out.DeliveryFee != 0 || out.Total != 900000 {
		t.Fatalf("fee=%d total=%d want 0 / 900000", out.DeliveryFee, out.Total)
	}
}
