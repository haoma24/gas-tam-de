package main

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
)

// recordingPublisher captures the order.completed payload the report service
// would receive.
type recordingPublisher struct {
	completed []orderCompletedEvent
}

func (recordingPublisher) PublishOrderPlaced(orderPlacedEvent) error { return nil }
func (p *recordingPublisher) PublishOrderCompleted(e orderCompletedEvent) error {
	p.completed = append(p.completed, e)
	return nil
}
func (recordingPublisher) PublishOrderCancelled(orderCancelledEvent) error { return nil }

// costReserver stands in for inventory-service, answering with the COGS
// snapshot it wrote on the OUT movement.
type costReserver struct {
	costs map[string]int64
}

func (c costReserver) Reserve(_ context.Context, _ string, _ []stockLine) (map[string]int64, error) {
	return c.costs, nil
}
func (costReserver) Release(_ context.Context, _ string, _ []stockLine) error { return nil }

// TestPlacedOrderStoresUnitCost is the root of the wrong-profit report: without
// a cost on the line, report-service sums COGS to 0 and profit comes out equal
// to revenue.
func TestPlacedOrderStoresUnitCost(t *testing.T) {
	svc := &orderService{
		db:        openTestOrderDB(t),
		geo:       &stubGeo{result: geoCheckResult{DistanceKm: 3.2, InRange: true, MaxRadiusKm: 10}},
		catalog:   &stubCatalog{products: []catalogProduct{{ID: "p1", SKU: "GAS12", Name: "Gas 12kg", SalePrice: 450000, Active: true}}},
		billing:   noopBillingRecorder{},
		inventory: costReserver{costs: map[string]int64{"p1": 380000}},
		bus:       noopOrderPublisher{},
	}
	r := mountOrderTestRoutes(svc)

	rr := postValidOrder(t, r)
	if rr.Code != http.StatusCreated {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}

	var placed orderView
	if err := json.Unmarshal(rr.Body.Bytes(), &placed); err != nil {
		t.Fatal(err)
	}
	if len(placed.Items) == 0 {
		t.Fatal("no items on the placed order")
	}
	if placed.Items[0].UnitCost != 380000 {
		t.Fatalf("unit_cost=%d, want the reserve snapshot 380000", placed.Items[0].UnitCost)
	}

	var stored int64
	if err := svc.db.QueryRow(
		`SELECT unit_cost FROM order_items WHERE order_id = ?`, placed.ID).Scan(&stored); err != nil {
		t.Fatal(err)
	}
	if stored != 380000 {
		t.Fatalf("persisted unit_cost=%d, want 380000", stored)
	}
}

// TestOrderCompletedCarriesUnitCost — report-service reads unit_cost off the
// event, so the cost has to survive all the way into the payload.
func TestOrderCompletedCarriesUnitCost(t *testing.T) {
	bus := &recordingPublisher{}
	svc := &orderService{
		db:        openTestOrderDB(t),
		geo:       &stubGeo{result: geoCheckResult{DistanceKm: 3.2, InRange: true, MaxRadiusKm: 10}},
		catalog:   &stubCatalog{products: []catalogProduct{{ID: "p1", SKU: "GAS12", Name: "Gas 12kg", SalePrice: 450000, Active: true}}},
		billing:   noopBillingRecorder{},
		inventory: costReserver{costs: map[string]int64{"p1": 380000}},
		bus:       bus,
	}
	r := mountOrderTestRoutes(svc)

	rr := postValidOrder(t, r)
	if rr.Code != http.StatusCreated {
		t.Fatalf("place status=%d body=%s", rr.Code, rr.Body.String())
	}
	var placed orderView
	if err := json.Unmarshal(rr.Body.Bytes(), &placed); err != nil {
		t.Fatal(err)
	}

	done := postComplete(r, placed.ID, []byte(`{"payment_type":"FULL"}`))
	if done.Code != http.StatusOK {
		t.Fatalf("complete status=%d body=%s", done.Code, done.Body.String())
	}

	if len(bus.completed) != 1 {
		t.Fatalf("published %d order.completed events, want 1", len(bus.completed))
	}
	items := bus.completed[0].Items
	if len(items) == 0 || items[0].UnitCost != 380000 {
		t.Fatalf("order.completed items=%+v, want unit_cost 380000", items)
	}
}

// TestReserveWithoutCostsLeavesZero — an inventory build that answers without
// `items` (rollout skew) must not break placing an order; those lines simply
// contribute no COGS.
func TestReserveWithoutCostsLeavesZero(t *testing.T) {
	svc := &orderService{
		db:        openTestOrderDB(t),
		geo:       &stubGeo{result: geoCheckResult{DistanceKm: 3.2, InRange: true, MaxRadiusKm: 10}},
		catalog:   &stubCatalog{products: []catalogProduct{{ID: "p1", SKU: "GAS12", Name: "Gas 12kg", SalePrice: 450000, Active: true}}},
		billing:   noopBillingRecorder{},
		inventory: costReserver{costs: nil},
		bus:       noopOrderPublisher{},
	}
	r := mountOrderTestRoutes(svc)

	rr := postValidOrder(t, r)
	if rr.Code != http.StatusCreated {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var placed orderView
	if err := json.Unmarshal(rr.Body.Bytes(), &placed); err != nil {
		t.Fatal(err)
	}
	if placed.Items[0].UnitCost != 0 {
		t.Fatalf("unit_cost=%d, want 0", placed.Items[0].UnitCost)
	}
}
