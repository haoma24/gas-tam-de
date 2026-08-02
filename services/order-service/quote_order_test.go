package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func quoteBody(productID string, lat, lng float64) []byte {
	b, _ := json.Marshal(map[string]any{
		"lat": lat,
		"lng": lng,
		"items": []map[string]any{
			{"product_id": productID, "qty": 2},
		},
	})
	return b
}

func TestQuoteOrderHappyPathFeeEnabled(t *testing.T) {
	geo := &stubGeo{result: geoCheckResult{DistanceKm: 3.2, InRange: true, MaxRadiusKm: 10}}
	catalog := &stubCatalog{products: []catalogProduct{
		{ID: "p1", SKU: "GAS12", Name: "Gas 12kg", SalePrice: 450000, Active: true},
	}}
	svc, r := testOrderRouter(t, geo, catalog)
	if err := seedDeliveryFee(svc.db, deliveryFeeSeedConfig{Enabled: true, Seed: true}); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodPost, "/v1/orders/quote", bytes.NewReader(quoteBody("p1", 10.78, 106.70)))
	customerHeaders(req)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	if geo.calls != 1 || geo.lastLat != 10.78 || geo.lastLng != 106.70 {
		t.Fatalf("geo calls=%d lat=%v lng=%v", geo.calls, geo.lastLat, geo.lastLng)
	}

	var out quoteOrderView
	if err := json.Unmarshal(rr.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	const wantFee int64 = 10000
	if !out.InRange || out.DistanceKm != 3.2 || out.MaxRadiusKm != 10 {
		t.Fatalf("geo fields=%+v", out)
	}
	if out.DeliveryFee != wantFee || out.Subtotal != 900000 || out.Total != 900000+wantFee {
		t.Fatalf("totals fee=%d sub=%d total=%d", out.DeliveryFee, out.Subtotal, out.Total)
	}

	var orderCount int
	if err := svc.db.QueryRow(`SELECT COUNT(*) FROM orders`).Scan(&orderCount); err != nil {
		t.Fatal(err)
	}
	if orderCount != 0 {
		t.Fatalf("quote must not persist orders, count=%d", orderCount)
	}
}

func TestQuoteOrderOutOfRangeStillReturnsPreview(t *testing.T) {
	geo := &stubGeo{result: geoCheckResult{DistanceKm: 12.5, InRange: false, MaxRadiusKm: 10}}
	catalog := &stubCatalog{products: []catalogProduct{
		{ID: "p1", SKU: "GAS12", Name: "Gas 12kg", SalePrice: 450000, Active: true},
	}}
	svc, r := testOrderRouter(t, geo, catalog)
	if err := seedDeliveryFee(svc.db, deliveryFeeSeedConfig{Enabled: true, Seed: true}); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodPost, "/v1/orders/quote", bytes.NewReader(quoteBody("p1", 10.9, 106.8)))
	customerHeaders(req)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var out quoteOrderView
	if err := json.Unmarshal(rr.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	if out.InRange || out.DistanceKm != 12.5 {
		t.Fatalf("want out-of-range preview, got %+v", out)
	}
	// 12.5km → band [10, +inf) = 30000 when fee enabled
	if out.DeliveryFee != 30000 || out.Subtotal != 900000 || out.Total != 930000 {
		t.Fatalf("totals=%+v", out)
	}
}

func TestQuoteOrderFeeZeroWhenDisabled(t *testing.T) {
	geo := &stubGeo{result: geoCheckResult{DistanceKm: 7.5, InRange: true, MaxRadiusKm: 10}}
	catalog := &stubCatalog{products: []catalogProduct{
		{ID: "p1", SKU: "GAS12", Name: "Gas 12kg", SalePrice: 450000, Active: true},
	}}
	svc, r := testOrderRouter(t, geo, catalog)
	if err := seedDeliveryFee(svc.db, deliveryFeeSeedConfig{Enabled: false, Seed: true}); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodPost, "/v1/orders/quote", bytes.NewReader(quoteBody("p1", 10.78, 106.70)))
	customerHeaders(req)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var out quoteOrderView
	_ = json.Unmarshal(rr.Body.Bytes(), &out)
	if out.DeliveryFee != 0 || out.Total != 900000 || !out.InRange {
		t.Fatalf("want fee 0 in-range, got %+v", out)
	}
}

func TestQuoteOrderUnauthorized(t *testing.T) {
	_, r := testOrderRouter(t, &stubGeo{result: geoCheckResult{InRange: true}}, &stubCatalog{})
	req := httptest.NewRequest(http.MethodPost, "/v1/orders/quote", bytes.NewReader(quoteBody("p1", 10, 106)))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d", rr.Code)
	}
}

func TestQuoteOrderForbiddenAdmin(t *testing.T) {
	_, r := testOrderRouter(t, &stubGeo{result: geoCheckResult{InRange: true}}, &stubCatalog{})
	req := httptest.NewRequest(http.MethodPost, "/v1/orders/quote", bytes.NewReader(quoteBody("p1", 10, 106)))
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

func TestQuoteOrderProductNotFound(t *testing.T) {
	geo := &stubGeo{result: geoCheckResult{DistanceKm: 1, InRange: true, MaxRadiusKm: 10}}
	catalog := &stubCatalog{products: []catalogProduct{
		{ID: "other", SKU: "X", Name: "X", SalePrice: 1, Active: true},
	}}
	_, r := testOrderRouter(t, geo, catalog)

	req := httptest.NewRequest(http.MethodPost, "/v1/orders/quote", bytes.NewReader(quoteBody("missing", 10, 106)))
	customerHeaders(req)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestQuoteOrderEmptyItems(t *testing.T) {
	_, r := testOrderRouter(t, &stubGeo{}, &stubCatalog{})
	body, _ := json.Marshal(map[string]any{
		"lat":   10,
		"lng":   106,
		"items": []any{},
	})
	req := httptest.NewRequest(http.MethodPost, "/v1/orders/quote", bytes.NewReader(body))
	customerHeaders(req)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status=%d", rr.Code)
	}
}

func TestQuoteOrderGeoUnavailable(t *testing.T) {
	geo := &stubGeo{err: errors.New("dial timeout")}
	catalog := &stubCatalog{products: []catalogProduct{
		{ID: "p1", SKU: "GAS12", Name: "Gas 12kg", SalePrice: 450000, Active: true},
	}}
	_, r := testOrderRouter(t, geo, catalog)

	req := httptest.NewRequest(http.MethodPost, "/v1/orders/quote", bytes.NewReader(quoteBody("p1", 10, 106)))
	customerHeaders(req)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadGateway {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestQuoteOrderMergesDuplicateProductLines(t *testing.T) {
	geo := &stubGeo{result: geoCheckResult{DistanceKm: 2, InRange: true, MaxRadiusKm: 10}}
	catalog := &stubCatalog{products: []catalogProduct{
		{ID: "p1", SKU: "GAS12", Name: "Gas 12kg", SalePrice: 100000, Active: true},
	}}
	_, r := testOrderRouter(t, geo, catalog)

	body, _ := json.Marshal(map[string]any{
		"lat": 10.7,
		"lng": 106.7,
		"items": []map[string]any{
			{"product_id": "p1", "qty": 1},
			{"product_id": "p1", "qty": 3},
		},
	})
	req := httptest.NewRequest(http.MethodPost, "/v1/orders/quote", bytes.NewReader(body))
	customerHeaders(req)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var out quoteOrderView
	_ = json.Unmarshal(rr.Body.Bytes(), &out)
	if out.Subtotal != 400000 || out.Total != 400000 {
		t.Fatalf("subtotal=%d total=%d", out.Subtotal, out.Total)
	}
}
