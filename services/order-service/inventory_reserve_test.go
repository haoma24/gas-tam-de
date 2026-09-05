package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"gas-tam-de/pkg/httpx"
)

// testOrderRouterWithInventory wires a real HTTP inventory client at srvURL so
// the reserve leg of checkout is exercised end to end (status mapping included).
func testOrderRouterWithInventory(t *testing.T, srvURL string) (*orderService, http.Handler) {
	t.Helper()
	svc := &orderService{
		db:  openTestOrderDB(t),
		geo: &stubGeo{result: geoCheckResult{DistanceKm: 3.2, InRange: true, MaxRadiusKm: 10}},
		catalog: &stubCatalog{products: []catalogProduct{
			{ID: "p1", SKU: "GAS12", Name: "Gas 12kg", SalePrice: 450000, Active: true},
		}},
		billing:   noopBillingRecorder{},
		inventory: newHTTPInventoryClient(srvURL, nil),
		bus:       noopOrderPublisher{},
	}
	return svc, mountOrderTestRoutes(svc)
}

func postValidOrder(t *testing.T, r http.Handler) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/v1/orders", bytes.NewReader(validBody("p1")))
	customerHeaders(req)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	return rr
}

func errorCode(t *testing.T, body []byte) string {
	t.Helper()
	var out struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		t.Fatalf("decode error body %q: %v", body, err)
	}
	return out.Error.Code
}

func orderStatus(t *testing.T, svc *orderService) (id, status string) {
	t.Helper()
	if err := svc.db.QueryRow(`SELECT id, status FROM orders`).Scan(&id, &status); err != nil {
		t.Fatalf("read order: %v", err)
	}
	return id, status
}

// TestCreateOrderReservesStock pins the happy path: checkout posts every line to
// inventory /v1/internal/stock/reserve and the order stays PENDING.
func TestCreateOrderReservesStock(t *testing.T) {
	var gotPath string
	var gotBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		httpx.JSON(w, http.StatusOK, map[string]any{"ok": true})
	}))
	defer srv.Close()

	svc, r := testOrderRouterWithInventory(t, srv.URL)
	rr := postValidOrder(t, r)

	if rr.Code != http.StatusCreated {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	if gotPath != "/v1/internal/stock/reserve" {
		t.Fatalf("inventory path=%q", gotPath)
	}
	items, _ := gotBody["items"].([]any)
	if len(items) != 1 {
		t.Fatalf("reserve items=%v", gotBody["items"])
	}
	line, _ := items[0].(map[string]any)
	if line["product_id"] != "p1" || line["sku"] != "GAS12" || line["qty"] != float64(2) {
		t.Fatalf("reserve line=%v", line)
	}
	if _, status := orderStatus(t, svc); status != "PENDING" {
		t.Fatalf("status=%q, want PENDING", status)
	}
}

// TestCreateOrderInsufficientStock keeps the 409 from inventory mapped to
// INSUFFICIENT_STOCK — the customer must be told stock is short, not that the
// system is broken. The classification reads the upstream body, so it breaks
// silently if inventory ever stops echoing "insufficient".
func TestCreateOrderInsufficientStock(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		httpx.Error(w, http.StatusConflict, "INSUFFICIENT_STOCK",
			"insufficient stock for product p1 (on_hand=0, need=2)")
	}))
	defer srv.Close()

	svc, r := testOrderRouterWithInventory(t, srv.URL)
	rr := postValidOrder(t, r)

	if rr.Code != http.StatusConflict {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	if code := errorCode(t, rr.Body.Bytes()); code != "INSUFFICIENT_STOCK" {
		t.Fatalf("code=%q", code)
	}
	if _, status := orderStatus(t, svc); status != "CANCELLED" {
		t.Fatalf("status=%q, want CANCELLED so the PENDING row is not delivered", status)
	}
}

// TestCreateOrderInventoryUnreachable covers the staging incident: order-service
// without INVENTORY_SERVICE_URL dialled 127.0.0.1:8085 inside its own container
// and every checkout failed with "Không trừ được tồn kho". The order must not be
// left PENDING, since no stock was deducted for it.
func TestCreateOrderInventoryUnreachable(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	unreachable := srv.URL
	srv.Close() // nothing is listening on that port any more

	svc, r := testOrderRouterWithInventory(t, unreachable)
	rr := postValidOrder(t, r)

	if rr.Code != http.StatusBadGateway {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	if code := errorCode(t, rr.Body.Bytes()); code != "INVENTORY_UNAVAILABLE" {
		t.Fatalf("code=%q", code)
	}
	if _, status := orderStatus(t, svc); status != "CANCELLED" {
		t.Fatalf("status=%q, want CANCELLED", status)
	}
}

// TestInventoryClientErrorNamesBaseURL keeps the configured host in the error
// text. Logs showing "127.0.0.1:8085" are what identifies a missing
// INVENTORY_SERVICE_URL instead of a genuinely dead inventory-service.
func TestInventoryClientErrorNamesBaseURL(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		httpx.Error(w, http.StatusNotFound, "NOT_FOUND", "no route")
	}))
	defer srv.Close()

	_, err := newHTTPInventoryClient(srv.URL, nil).
		Reserve(t.Context(), "order-1", []stockLine{{ProductID: "p1", SKU: "GAS12", Qty: 1}})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), srv.URL) {
		t.Fatalf("error %q does not name base URL %q", err, srv.URL)
	}
}
