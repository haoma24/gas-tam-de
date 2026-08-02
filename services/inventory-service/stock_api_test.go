package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"gas-tam-de/pkg/httpx"
)

func testInventoryRouter(t *testing.T) (*inventoryService, http.Handler) {
	t.Helper()
	svc := &inventoryService{db: openInventoryTestDB(t)}
	r := httpx.NewRouter("inventory-test")
	r.Get("/v1/admin/inventory", svc.handleListStock)
	r.Post("/v1/admin/inventory", svc.handlePostMovement)
	return svc, r
}

func postMovement(h http.Handler, body map[string]any, userID string) *httptest.ResponseRecorder {
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/v1/admin/inventory", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	if userID != "" {
		req.Header.Set("X-User-Id", userID)
	}
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	return rr
}

func getListStock(h http.Handler) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, "/v1/admin/inventory", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	return rr
}

func TestListStockEmpty(t *testing.T) {
	_, h := testInventoryRouter(t)
	rr := getListStock(h)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var got listStockResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.Count != 0 || got.Items == nil || len(got.Items) != 0 {
		t.Fatalf("got=%+v", got)
	}
}

func TestStockINCreateAndList(t *testing.T) {
	_, h := testInventoryRouter(t)

	rr := postMovement(h, map[string]any{
		"movement_type": "IN",
		"product_id":    "p-gas12",
		"sku":           "GAS12",
		"name":          "Gas 12kg",
		"qty":           10,
		"unit_cost":     150000,
		"reorder_level": 3,
		"note":          "nhập đầu kỳ",
	}, "admin-1")
	if rr.Code != http.StatusCreated {
		t.Fatalf("IN status=%d body=%s", rr.Code, rr.Body.String())
	}
	var created postMovementResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}
	if created.Item.OnHand != 10 || created.Item.CostPrice != 150000 || created.Item.SKU != "GAS12" {
		t.Fatalf("item=%+v", created.Item)
	}
	if created.Movement.MovementType != movementIN || created.Movement.Qty != 10 || created.Movement.Delta != 10 {
		t.Fatalf("movement=%+v", created.Movement)
	}
	if created.Movement.UnitCost == nil || *created.Movement.UnitCost != 150000 {
		t.Fatalf("unit_cost=%v", created.Movement.UnitCost)
	}
	if created.Movement.CreatedBy == nil || *created.Movement.CreatedBy != "admin-1" {
		t.Fatalf("created_by=%v", created.Movement.CreatedBy)
	}

	rr = getListStock(h)
	if rr.Code != http.StatusOK {
		t.Fatalf("list status=%d", rr.Code)
	}
	var list listStockResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &list); err != nil {
		t.Fatal(err)
	}
	if list.Count != 1 || list.Items[0].ProductID != "p-gas12" || list.Items[0].OnHand != 10 {
		t.Fatalf("list=%+v", list)
	}
}

func TestStockINUpdatesCostAndOnHand(t *testing.T) {
	_, h := testInventoryRouter(t)

	if rr := postMovement(h, map[string]any{
		"movement_type": "IN",
		"product_id":    "p1",
		"sku":           "SKU1",
		"name":          "Gas",
		"qty":           5,
		"unit_cost":     100000,
	}, "admin"); rr.Code != http.StatusCreated {
		t.Fatalf("first IN: %s", rr.Body.String())
	}

	rr := postMovement(h, map[string]any{
		"movement_type": "IN",
		"product_id":    "p1",
		"qty":           5,
		"unit_cost":     120000,
	}, "admin")
	if rr.Code != http.StatusCreated {
		t.Fatalf("second IN status=%d body=%s", rr.Code, rr.Body.String())
	}
	var got postMovementResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.Item.OnHand != 10 || got.Item.CostPrice != 120000 {
		t.Fatalf("want on_hand=10 cost=120000 got=%+v", got.Item)
	}
}

func TestStockOUTSnapshotsCostAndAllowsNegative(t *testing.T) {
	_, h := testInventoryRouter(t)

	if rr := postMovement(h, map[string]any{
		"movement_type": "IN",
		"product_id":    "p1",
		"sku":           "SKU1",
		"name":          "Gas",
		"qty":           2,
		"unit_cost":     150000,
	}, "admin"); rr.Code != http.StatusCreated {
		t.Fatalf("IN: %s", rr.Body.String())
	}

	rr := postMovement(h, map[string]any{
		"movement_type": "OUT",
		"product_id":    "p1",
		"qty":           5,
		"note":          "xuất tay",
	}, "admin")
	if rr.Code != http.StatusCreated {
		t.Fatalf("OUT status=%d body=%s", rr.Code, rr.Body.String())
	}
	var got postMovementResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.Item.OnHand != -3 {
		t.Fatalf("on_hand=%d want -3", got.Item.OnHand)
	}
	if got.Movement.Delta != -5 || got.Movement.Qty != 5 {
		t.Fatalf("movement=%+v", got.Movement)
	}
	if got.Movement.UnitCost == nil || *got.Movement.UnitCost != 150000 {
		t.Fatalf("OUT should snapshot cost_price: %v", got.Movement.UnitCost)
	}
}

func TestStockADJUSTDelta(t *testing.T) {
	_, h := testInventoryRouter(t)

	if rr := postMovement(h, map[string]any{
		"movement_type": "IN",
		"product_id":    "p1",
		"sku":           "SKU1",
		"name":          "Gas",
		"qty":           10,
		"unit_cost":     100000,
	}, "admin"); rr.Code != http.StatusCreated {
		t.Fatalf("IN: %s", rr.Body.String())
	}

	rr := postMovement(h, map[string]any{
		"movement_type": "ADJUST",
		"product_id":    "p1",
		"delta":         -3,
		"note":          "kiểm kê thiếu",
	}, "admin")
	if rr.Code != http.StatusCreated {
		t.Fatalf("ADJUST status=%d body=%s", rr.Code, rr.Body.String())
	}
	var got postMovementResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.Item.OnHand != 7 || got.Movement.Qty != 3 || got.Movement.Delta != -3 {
		t.Fatalf("got item=%+v movement=%+v", got.Item, got.Movement)
	}

	rr = postMovement(h, map[string]any{
		"movement_type": "ADJUST",
		"product_id":    "p1",
		"delta":         2,
		"unit_cost":     110000,
	}, "admin")
	if rr.Code != http.StatusCreated {
		t.Fatalf("ADJUST+ status=%d body=%s", rr.Code, rr.Body.String())
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.Item.OnHand != 9 || got.Item.CostPrice != 110000 {
		t.Fatalf("after +adjust: %+v", got.Item)
	}
}

func TestStockValidationErrors(t *testing.T) {
	_, h := testInventoryRouter(t)

	cases := []struct {
		name string
		body map[string]any
		code string
		want int
	}{
		{
			name: "missing type",
			body: map[string]any{"product_id": "p1", "qty": 1},
			code: "INVALID_TYPE",
			want: http.StatusBadRequest,
		},
		{
			name: "IN without unit_cost",
			body: map[string]any{
				"movement_type": "IN", "product_id": "p1", "sku": "S", "name": "N", "qty": 1,
			},
			code: "INVALID_UNIT_COST",
			want: http.StatusBadRequest,
		},
		{
			name: "IN create without sku",
			body: map[string]any{
				"movement_type": "IN", "product_id": "p1", "name": "N", "qty": 1, "unit_cost": 1,
			},
			code: "INVALID_PRODUCT",
			want: http.StatusBadRequest,
		},
		{
			name: "OUT missing stock",
			body: map[string]any{"movement_type": "OUT", "product_id": "missing", "qty": 1},
			code: "NOT_FOUND",
			want: http.StatusNotFound,
		},
		{
			name: "ADJUST zero delta",
			body: map[string]any{"movement_type": "ADJUST", "product_id": "p1", "delta": 0},
			code: "INVALID_DELTA",
			want: http.StatusBadRequest,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rr := postMovement(h, tc.body, "admin")
			if rr.Code != tc.want {
				t.Fatalf("status=%d want=%d body=%s", rr.Code, tc.want, rr.Body.String())
			}
			var body map[string]any
			if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
				t.Fatal(err)
			}
			errObj, _ := body["error"].(map[string]any)
			if errObj["code"] != tc.code {
				t.Fatalf("code=%v want %s", errObj["code"], tc.code)
			}
		})
	}
}

func TestStockOUTRequiresExisting(t *testing.T) {
	_, h := testInventoryRouter(t)
	rr := postMovement(h, map[string]any{
		"movement_type": "OUT",
		"product_id":    "nope",
		"qty":           1,
	}, "")
	if rr.Code != http.StatusNotFound {
		t.Fatalf("status=%d", rr.Code)
	}
}

func TestStockPersistsMovements(t *testing.T) {
	svc, h := testInventoryRouter(t)

	if rr := postMovement(h, map[string]any{
		"movement_type": "IN",
		"product_id":    "p1",
		"sku":           "SKU1",
		"name":          "Gas",
		"qty":           4,
		"unit_cost":     90000,
	}, "admin"); rr.Code != http.StatusCreated {
		t.Fatalf("IN: %s", rr.Body.String())
	}
	if rr := postMovement(h, map[string]any{
		"movement_type": "OUT",
		"product_id":    "p1",
		"qty":           1,
	}, "admin"); rr.Code != http.StatusCreated {
		t.Fatalf("OUT: %s", rr.Body.String())
	}

	var n int
	if err := svc.db.QueryRow(`SELECT COUNT(*) FROM stock_movements WHERE product_id = 'p1'`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 2 {
		t.Fatalf("movements=%d want 2", n)
	}

	var onHand int64
	if err := svc.db.QueryRow(`SELECT on_hand FROM stock_items WHERE product_id = 'p1'`).Scan(&onHand); err != nil {
		t.Fatal(err)
	}
	if onHand != 3 {
		t.Fatalf("on_hand=%d want 3", onHand)
	}
}

// T7.2.1: OUT must snapshot stock cost_price and ignore any client-supplied unit_cost.
func TestStockOUTIgnoresClientUnitCost(t *testing.T) {
	_, h := testInventoryRouter(t)

	if rr := postMovement(h, map[string]any{
		"movement_type": "IN",
		"product_id":    "p1",
		"sku":           "SKU1",
		"name":          "Gas",
		"qty":           5,
		"unit_cost":     150000,
	}, "admin"); rr.Code != http.StatusCreated {
		t.Fatalf("IN: %s", rr.Body.String())
	}

	rr := postMovement(h, map[string]any{
		"movement_type": "OUT",
		"product_id":    "p1",
		"qty":           1,
		"unit_cost":     1, // malicious / stale client value — must be ignored
	}, "admin")
	if rr.Code != http.StatusCreated {
		t.Fatalf("OUT status=%d body=%s", rr.Code, rr.Body.String())
	}
	var got postMovementResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.Movement.UnitCost == nil || *got.Movement.UnitCost != 150000 {
		t.Fatalf("want snapshotted 150000, got %v", got.Movement.UnitCost)
	}
}

// T7.2.1: historical OUT.unit_cost stays frozen after later IN updates cost_price.
func TestStockOUTCostSnapshotFrozenAfterLaterIN(t *testing.T) {
	svc, h := testInventoryRouter(t)

	if rr := postMovement(h, map[string]any{
		"movement_type": "IN",
		"product_id":    "p1",
		"sku":           "SKU1",
		"name":          "Gas",
		"qty":           10,
		"unit_cost":     100000,
	}, "admin"); rr.Code != http.StatusCreated {
		t.Fatalf("IN1: %s", rr.Body.String())
	}
	if rr := postMovement(h, map[string]any{
		"movement_type": "OUT",
		"product_id":    "p1",
		"qty":           2,
	}, "admin"); rr.Code != http.StatusCreated {
		t.Fatalf("OUT1: %s", rr.Body.String())
	}

	var firstOUTCost int64
	if err := svc.db.QueryRow(`
		SELECT unit_cost FROM stock_movements
		WHERE product_id = 'p1' AND movement_type = 'OUT'
		ORDER BY created_at ASC LIMIT 1`).Scan(&firstOUTCost); err != nil {
		t.Fatal(err)
	}
	if firstOUTCost != 100000 {
		t.Fatalf("first OUT cost=%d want 100000", firstOUTCost)
	}

	if rr := postMovement(h, map[string]any{
		"movement_type": "IN",
		"product_id":    "p1",
		"qty":           1,
		"unit_cost":     200000,
	}, "admin"); rr.Code != http.StatusCreated {
		t.Fatalf("IN2: %s", rr.Body.String())
	}

	var frozen int64
	if err := svc.db.QueryRow(`
		SELECT unit_cost FROM stock_movements
		WHERE product_id = 'p1' AND movement_type = 'OUT'
		ORDER BY created_at ASC LIMIT 1`).Scan(&frozen); err != nil {
		t.Fatal(err)
	}
	if frozen != 100000 {
		t.Fatalf("historical OUT rewritten to %d; want frozen 100000", frozen)
	}

	rr := postMovement(h, map[string]any{
		"movement_type": "OUT",
		"product_id":    "p1",
		"qty":           1,
	}, "admin")
	if rr.Code != http.StatusCreated {
		t.Fatalf("OUT2: %s", rr.Body.String())
	}
	var got postMovementResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.Movement.UnitCost == nil || *got.Movement.UnitCost != 200000 {
		t.Fatalf("second OUT should snapshot new cost: %v", got.Movement.UnitCost)
	}
	if got.Item.CostPrice != 200000 {
		t.Fatalf("cost_price=%d want 200000", got.Item.CostPrice)
	}
}

func TestSnapshotOUTCost(t *testing.T) {
	if got := snapshotOUTCost(150000); got != 150000 {
		t.Fatalf("got %d", got)
	}
	if got := snapshotOUTCost(0); got != 0 {
		t.Fatalf("zero: %d", got)
	}
	if got := snapshotOUTCost(-1); got != 0 {
		t.Fatalf("clamp negative: %d", got)
	}
}
