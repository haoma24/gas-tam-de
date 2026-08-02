package main

import (
	"encoding/json"
	"testing"
	"time"

	"gas-tam-de/pkg/events"
	"gas-tam-de/pkg/natsx"

	"github.com/nats-io/nats-server/v2/server"
)

func seedStock(t *testing.T, svc *inventoryService, productID, sku, name string, onHand, cost int64) {
	t.Helper()
	qty := onHand
	_, err := svc.applyMovement(postMovementBody{
		MovementType: movementIN,
		ProductID:    productID,
		Qty:          &qty,
		UnitCost:     &cost,
		SKU:          sku,
		Name:         name,
	}, "seed")
	if err != nil {
		t.Fatalf("seed stock: %v", err)
	}
}

func TestApplyOrderCompletedDeductsStockAndSnapshotsCost(t *testing.T) {
	svc := &inventoryService{db: openInventoryTestDB(t)}
	seedStock(t, svc, "p1", "GAS12", "Gas 12kg", 10, 150000)

	err := svc.applyOrderCompleted("evt-1", orderCompletedPayload{
		OrderID: "ord-1",
		Items: []orderCompletedItem{
			{ProductID: "p1", SKU: "GAS12", Qty: 3},
		},
	})
	if err != nil {
		t.Fatalf("apply: %v", err)
	}

	item, err := loadStockItemTxMust(t, svc, "p1")
	if err != nil {
		t.Fatal(err)
	}
	if item.OnHand != 7 {
		t.Fatalf("on_hand=%d want 7", item.OnHand)
	}
	if item.CostPrice != 150000 {
		t.Fatalf("cost_price=%d want 150000", item.CostPrice)
	}

	var mt string
	var qty, unitCost int64
	var refType, refID string
	err = svc.db.QueryRow(`
		SELECT movement_type, qty, unit_cost, ref_type, ref_id
		FROM stock_movements WHERE product_id = 'p1' AND ref_type = 'ORDER'`).
		Scan(&mt, &qty, &unitCost, &refType, &refID)
	if err != nil {
		t.Fatal(err)
	}
	if mt != movementOUT || qty != 3 || unitCost != 150000 || refType != refTypeOrder || refID != "ord-1" {
		t.Fatalf("movement mt=%s qty=%d cost=%d ref=%s/%s", mt, qty, unitCost, refType, refID)
	}

	var processed int
	if err := svc.db.QueryRow(`SELECT COUNT(*) FROM processed_events WHERE event_id = 'evt-1'`).Scan(&processed); err != nil {
		t.Fatal(err)
	}
	if processed != 1 {
		t.Fatalf("processed=%d want 1", processed)
	}
}

func TestApplyOrderCompletedIdempotent(t *testing.T) {
	svc := &inventoryService{db: openInventoryTestDB(t)}
	seedStock(t, svc, "p1", "GAS12", "Gas 12kg", 5, 100000)

	payload := orderCompletedPayload{
		OrderID: "ord-dup",
		Items:   []orderCompletedItem{{ProductID: "p1", SKU: "GAS12", Qty: 2}},
	}
	if err := svc.applyOrderCompleted("evt-dup", payload); err != nil {
		t.Fatalf("first: %v", err)
	}
	if err := svc.applyOrderCompleted("evt-dup", payload); err != nil {
		t.Fatalf("second: %v", err)
	}

	item, err := loadStockItemTxMust(t, svc, "p1")
	if err != nil {
		t.Fatal(err)
	}
	if item.OnHand != 3 {
		t.Fatalf("on_hand=%d want 3 (deducted once)", item.OnHand)
	}

	var movCount, evtCount int
	_ = svc.db.QueryRow(`SELECT COUNT(*) FROM stock_movements WHERE ref_id = 'ord-dup'`).Scan(&movCount)
	_ = svc.db.QueryRow(`SELECT COUNT(*) FROM processed_events WHERE event_id = 'evt-dup'`).Scan(&evtCount)
	if movCount != 1 || evtCount != 1 {
		t.Fatalf("movements=%d events=%d want 1/1", movCount, evtCount)
	}
}

func TestApplyOrderCompletedAllowsNegativeAndCreatesMissingStock(t *testing.T) {
	svc := &inventoryService{db: openInventoryTestDB(t)}

	err := svc.applyOrderCompleted("evt-neg", orderCompletedPayload{
		OrderID: "ord-neg",
		Items:   []orderCompletedItem{{ProductID: "p-missing", SKU: "NEW1", Qty: 2}},
	})
	if err != nil {
		t.Fatalf("apply: %v", err)
	}

	item, err := loadStockItemTxMust(t, svc, "p-missing")
	if err != nil {
		t.Fatal(err)
	}
	if item.OnHand != -2 {
		t.Fatalf("on_hand=%d want -2", item.OnHand)
	}
	if item.SKU != "NEW1" || item.CostPrice != 0 {
		t.Fatalf("item=%+v", item)
	}
}

func TestApplyOrderCompletedMultiLine(t *testing.T) {
	svc := &inventoryService{db: openInventoryTestDB(t)}
	seedStock(t, svc, "p1", "GAS12", "Gas 12kg", 10, 150000)
	seedStock(t, svc, "p2", "GAS45", "Gas 45kg", 4, 400000)

	err := svc.applyOrderCompleted("evt-multi", orderCompletedPayload{
		OrderID: "ord-multi",
		Items: []orderCompletedItem{
			{ProductID: "p1", SKU: "GAS12", Qty: 1},
			{ProductID: "p2", SKU: "GAS45", Qty: 2},
		},
	})
	if err != nil {
		t.Fatalf("apply: %v", err)
	}

	p1, _ := loadStockItemTxMust(t, svc, "p1")
	p2, _ := loadStockItemTxMust(t, svc, "p2")
	if p1.OnHand != 9 || p2.OnHand != 2 {
		t.Fatalf("p1=%d p2=%d", p1.OnHand, p2.OnHand)
	}
}

// T7.2.1: order.completed OUT unit_cost stays frozen after a later IN raises cost_price.
func TestOrderCompletedOUTCostSnapshotFrozenAfterLaterIN(t *testing.T) {
	svc := &inventoryService{db: openInventoryTestDB(t)}
	seedStock(t, svc, "p1", "GAS12", "Gas 12kg", 10, 100000)

	if err := svc.applyOrderCompleted("evt-cogs-1", orderCompletedPayload{
		OrderID: "ord-cogs",
		Items:   []orderCompletedItem{{ProductID: "p1", SKU: "GAS12", Qty: 2}},
	}); err != nil {
		t.Fatalf("apply: %v", err)
	}

	var saleCost int64
	if err := svc.db.QueryRow(`
		SELECT unit_cost FROM stock_movements
		WHERE ref_type = 'ORDER' AND ref_id = 'ord-cogs'`).Scan(&saleCost); err != nil {
		t.Fatal(err)
	}
	if saleCost != 100000 {
		t.Fatalf("sale unit_cost=%d want 100000", saleCost)
	}

	qty := int64(1)
	newCost := int64(250000)
	if _, err := svc.applyMovement(postMovementBody{
		MovementType: movementIN,
		ProductID:    "p1",
		Qty:          &qty,
		UnitCost:     &newCost,
	}, "admin"); err != nil {
		t.Fatalf("later IN: %v", err)
	}

	var frozen int64
	if err := svc.db.QueryRow(`
		SELECT unit_cost FROM stock_movements
		WHERE ref_type = 'ORDER' AND ref_id = 'ord-cogs'`).Scan(&frozen); err != nil {
		t.Fatal(err)
	}
	if frozen != 100000 {
		t.Fatalf("ORDER OUT rewritten to %d; want frozen 100000", frozen)
	}

	item, err := loadStockItemTxMust(t, svc, "p1")
	if err != nil {
		t.Fatal(err)
	}
	if item.CostPrice != 250000 {
		t.Fatalf("current cost_price=%d want 250000", item.CostPrice)
	}
}

func TestParseOrderCompletedPayload(t *testing.T) {
	payload := map[string]any{
		"order_id": "ord-1",
		"items": []any{
			map[string]any{"product_id": "p1", "sku": "GAS12", "qty": float64(2), "unit_price": float64(450000)},
		},
		"total":        float64(900000),
		"payment_type": "FULL",
		"amount_paid":  float64(900000),
	}
	got, err := parseOrderCompletedPayload(payload)
	if err != nil {
		t.Fatal(err)
	}
	if got.OrderID != "ord-1" || len(got.Items) != 1 || got.Items[0].Qty != 2 || got.Items[0].ProductID != "p1" {
		t.Fatalf("got=%+v", got)
	}
}

func TestHandleOrderCompletedMsg(t *testing.T) {
	svc := &inventoryService{db: openInventoryTestDB(t)}
	seedStock(t, svc, "p1", "GAS12", "Gas 12kg", 5, 100000)

	env := events.NewEnvelope(events.OrderCompleted, "evt-msg", map[string]any{
		"order_id": "ord-msg",
		"items": []any{
			map[string]any{"product_id": "p1", "sku": "GAS12", "qty": float64(1), "unit_price": float64(450000)},
		},
		"total":        float64(450000),
		"payment_type": "FULL",
		"amount_paid":  float64(450000),
	})
	data, err := json.Marshal(env)
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.handleOrderCompletedMsg(data); err != nil {
		t.Fatalf("handle: %v", err)
	}
	item, _ := loadStockItemTxMust(t, svc, "p1")
	if item.OnHand != 4 {
		t.Fatalf("on_hand=%d want 4", item.OnHand)
	}
}

func TestOrderCompletedConsumerJetStream(t *testing.T) {
	opts := &server.Options{
		Port:      -1,
		JetStream: true,
		StoreDir:  t.TempDir(),
	}
	ns, err := server.NewServer(opts)
	if err != nil {
		t.Fatalf("new server: %v", err)
	}
	go ns.Start()
	if !ns.ReadyForConnections(5 * time.Second) {
		t.Fatal("nats server not ready")
	}
	defer ns.Shutdown()

	nc, js, err := natsx.ConnectJS(ns.ClientURL())
	if err != nil {
		t.Fatalf("ConnectJS: %v", err)
	}
	defer nc.Close()
	if err := natsx.EnsureStreams(js); err != nil {
		t.Fatalf("EnsureStreams: %v", err)
	}

	svc := &inventoryService{db: openInventoryTestDB(t)}
	seedStock(t, svc, "p1", "GAS12", "Gas 12kg", 8, 120000)

	sub, err := startOrderCompletedConsumer(js, svc)
	if err != nil {
		t.Fatalf("start consumer: %v", err)
	}
	defer func() { _ = sub.Unsubscribe() }()

	env := events.NewEnvelope(events.OrderCompleted, "evt-js-1", map[string]any{
		"order_id": "ord-js",
		"items": []map[string]any{
			{"product_id": "p1", "sku": "GAS12", "qty": 3, "unit_price": 450000},
		},
		"total":        1350000,
		"payment_type": "FULL",
		"amount_paid":  1350000,
	})
	if _, err := natsx.PublishEnvelope(js, env); err != nil {
		t.Fatalf("publish: %v", err)
	}

	deadline := time.Now().Add(3 * time.Second)
	for {
		item, err := loadStockItemTxMust(t, svc, "p1")
		if err == nil && item.OnHand == 5 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("timeout waiting for stock deduction; on_hand err=%v", err)
		}
		time.Sleep(50 * time.Millisecond)
	}

	// Republish same event_id (JetStream MsgId dedupe may drop; also re-apply via handler).
	if err := svc.applyOrderCompleted("evt-js-1", orderCompletedPayload{
		OrderID: "ord-js",
		Items:   []orderCompletedItem{{ProductID: "p1", SKU: "GAS12", Qty: 3}},
	}); err != nil {
		t.Fatalf("idempotent reapply: %v", err)
	}
	item, _ := loadStockItemTxMust(t, svc, "p1")
	if item.OnHand != 5 {
		t.Fatalf("on_hand=%d want 5 after idempotent", item.OnHand)
	}

	info, err := js.ConsumerInfo("ORDERS", durableOrderCompleted)
	if err != nil {
		t.Fatalf("ConsumerInfo: %v", err)
	}
	if info.Name != durableOrderCompleted {
		t.Fatalf("durable=%q", info.Name)
	}
}

func loadStockItemTxMust(t *testing.T, svc *inventoryService, productID string) (stockItem, error) {
	t.Helper()
	tx, err := svc.db.Begin()
	if err != nil {
		return stockItem{}, err
	}
	defer func() { _ = tx.Rollback() }()
	return loadStockItemTx(tx, productID)
}
