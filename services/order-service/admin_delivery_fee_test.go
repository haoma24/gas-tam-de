package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"gas-tam-de/pkg/httpx"
)

func testDeliveryFeeAdminRouter(t *testing.T) (*orderService, http.Handler) {
	t.Helper()
	db := openDeliveryFeeTestDB(t)
	if err := seedDeliveryFee(db, deliveryFeeSeedConfig{Enabled: false, Seed: true}); err != nil {
		t.Fatal(err)
	}
	svc := &orderService{db: db}
	r := httpx.NewRouter("order-test")
	r.Get("/v1/admin/delivery-fee", svc.handleGetAdminDeliveryFee)
	r.Put("/v1/admin/delivery-fee", svc.handlePutAdminDeliveryFee)
	return svc, r
}

func TestHandleGetAdminDeliveryFee(t *testing.T) {
	_, h := testDeliveryFeeAdminRouter(t)
	req := httptest.NewRequest(http.MethodGet, "/v1/admin/delivery-fee", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}

	var body deliveryFeeConfig
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Enabled {
		t.Fatal("seed default enabled=false")
	}
	if body.UpdatedAt == "" {
		t.Fatal("expected updated_at")
	}
	if len(body.Rules) != 3 {
		t.Fatalf("rules=%d want 3", len(body.Rules))
	}
	if body.Rules[0].ID != "rule-0-5" || body.Rules[0].FeeVnd != 10000 {
		t.Fatalf("first rule: %+v", body.Rules[0])
	}
	if body.Rules[0].MaxKm == nil || *body.Rules[0].MaxKm != 5 {
		t.Fatalf("first max_km: %+v", body.Rules[0].MaxKm)
	}
	if body.Rules[2].MaxKm != nil {
		t.Fatalf("open-ended rule max_km want null, got %v", *body.Rules[2].MaxKm)
	}
	if !body.Rules[0].Active {
		t.Fatal("seed rules should be active")
	}
}

func TestHandleGetAdminDeliveryFeeNotConfigured(t *testing.T) {
	db := openDeliveryFeeTestDB(t)
	svc := &orderService{db: db}
	r := httpx.NewRouter("order-test")
	r.Get("/v1/admin/delivery-fee", svc.handleGetAdminDeliveryFee)

	req := httptest.NewRequest(http.MethodGet, "/v1/admin/delivery-fee", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status=%d", rec.Code)
	}
}

func TestHandlePutAdminDeliveryFeeToggleAndRules(t *testing.T) {
	_, h := testDeliveryFeeAdminRouter(t)

	payload := `{
		"enabled": true,
		"rules": [
			{"id":"r1","min_km":0,"max_km":4,"fee_vnd":12000,"sort_order":0,"active":true},
			{"id":"r2","min_km":4,"max_km":null,"fee_vnd":25000,"sort_order":1,"active":true}
		]
	}`
	req := httptest.NewRequest(http.MethodPut, "/v1/admin/delivery-fee", bytes.NewBufferString(payload))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}

	var body deliveryFeeConfig
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if !body.Enabled {
		t.Fatal("want enabled=true")
	}
	if len(body.Rules) != 2 {
		t.Fatalf("rules=%d want 2", len(body.Rules))
	}
	if body.Rules[0].ID != "r1" || body.Rules[0].FeeVnd != 12000 {
		t.Fatalf("rule0=%+v", body.Rules[0])
	}
	if body.Rules[1].MaxKm != nil {
		t.Fatalf("rule1 max_km want null")
	}

	// GET reflects persist.
	req2 := httptest.NewRequest(http.MethodGet, "/v1/admin/delivery-fee", nil)
	rec2 := httptest.NewRecorder()
	h.ServeHTTP(rec2, req2)
	if rec2.Code != http.StatusOK {
		t.Fatalf("get status=%d", rec2.Code)
	}
	var got deliveryFeeConfig
	if err := json.Unmarshal(rec2.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if !got.Enabled || len(got.Rules) != 2 {
		t.Fatalf("get after put: %+v", got)
	}
}

func TestHandlePutAdminDeliveryFeeEnabledOnly(t *testing.T) {
	_, h := testDeliveryFeeAdminRouter(t)

	req := httptest.NewRequest(http.MethodPut, "/v1/admin/delivery-fee",
		bytes.NewBufferString(`{"enabled":true}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}

	var body deliveryFeeConfig
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if !body.Enabled {
		t.Fatal("want enabled")
	}
	if len(body.Rules) != 3 {
		t.Fatalf("rules must stay seed set, got %d", len(body.Rules))
	}
}

func TestHandlePutAdminDeliveryFeeRejectsOverlap(t *testing.T) {
	_, h := testDeliveryFeeAdminRouter(t)

	payload := `{
		"rules": [
			{"min_km":0,"max_km":6,"fee_vnd":10000},
			{"min_km":5,"max_km":10,"fee_vnd":20000}
		]
	}`
	req := httptest.NewRequest(http.MethodPut, "/v1/admin/delivery-fee", bytes.NewBufferString(payload))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var errBody map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &errBody); err != nil {
		t.Fatal(err)
	}
	errObj, _ := errBody["error"].(map[string]any)
	if errObj["code"] != "INVALID_RULES" {
		t.Fatalf("code=%v", errObj["code"])
	}
}

func TestHandlePutAdminDeliveryFeeRejectsEmptyBody(t *testing.T) {
	_, h := testDeliveryFeeAdminRouter(t)
	req := httptest.NewRequest(http.MethodPut, "/v1/admin/delivery-fee", bytes.NewBufferString(`{}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status=%d", rec.Code)
	}
}

func TestHandlePutAdminDeliveryFeeGeneratesID(t *testing.T) {
	_, h := testDeliveryFeeAdminRouter(t)
	payload := `{"rules":[{"min_km":0,"max_km":null,"fee_vnd":15000}]}`
	req := httptest.NewRequest(http.MethodPut, "/v1/admin/delivery-fee", bytes.NewBufferString(payload))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var body deliveryFeeConfig
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if len(body.Rules) != 1 || body.Rules[0].ID == "" {
		t.Fatalf("want generated id: %+v", body.Rules)
	}
}

func TestValidateActiveRuleBands(t *testing.T) {
	ok := []deliveryFeeRule{
		{MinKm: 0, MaxKm: floatPtr(5), Active: true},
		{MinKm: 5, MaxKm: nil, Active: true},
		{MinKm: 0, MaxKm: floatPtr(1), Active: false}, // inactive overlap ignored
	}
	if err := validateActiveRuleBands(ok); err != nil {
		t.Fatal(err)
	}

	bad := []deliveryFeeRule{
		{MinKm: 0, MaxKm: nil, Active: true},
		{MinKm: 10, MaxKm: floatPtr(20), Active: true},
	}
	if err := validateActiveRuleBands(bad); err == nil {
		t.Fatal("expected open-ended-not-last error")
	}
}
