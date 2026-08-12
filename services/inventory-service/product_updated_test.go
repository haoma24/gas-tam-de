package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"gas-tam-de/pkg/events"
	"gas-tam-de/pkg/natsx"

	"github.com/nats-io/nats-server/v2/server"
)

// TestApplyProductUpdatedCreatesEmptyStockRow is the point of the consumer: a
// product in catalog gets a stock row with the *same* product_id, so checkout
// reserve can find it instead of failing with «Không đủ tồn kho».
func TestApplyProductUpdatedCreatesEmptyStockRow(t *testing.T) {
	svc := &inventoryService{db: openInventoryTestDB(t)}

	err := svc.applyProductUpdated("evt-1", productUpdatedPayload{
		ProductID: "prod-uuid-1",
		SKU:       "GAS12",
		Name:      "Gas 12kg",
	})
	if err != nil {
		t.Fatalf("apply: %v", err)
	}

	item, err := loadStockItemTxMust(t, svc, "prod-uuid-1")
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if item.SKU != "GAS12" || item.Name != "Gas 12kg" {
		t.Fatalf("identity=%+v", item)
	}
	if item.OnHand != 0 || item.CostPrice != 0 {
		t.Fatalf("new row must start empty, got on_hand=%d cost=%d", item.OnHand, item.CostPrice)
	}
	// Creating a row is not a stock movement — the ledger must stay untouched.
	var movements int
	if err := svc.db.QueryRow(`SELECT COUNT(*) FROM stock_movements`).Scan(&movements); err != nil {
		t.Fatal(err)
	}
	if movements != 0 {
		t.Fatalf("stock_movements=%d want 0", movements)
	}
}

// TestApplyProductUpdatedKeepsQuantityOnRename guards the split of ownership:
// catalog owns sku/name, the stock ledger owns quantity and cost.
func TestApplyProductUpdatedKeepsQuantityOnRename(t *testing.T) {
	svc := &inventoryService{db: openInventoryTestDB(t)}
	seedStock(t, svc, "prod-uuid-1", "GAS12", "Gas 12kg", 7, 150000)

	err := svc.applyProductUpdated("evt-rename", productUpdatedPayload{
		ProductID: "prod-uuid-1",
		SKU:       "GAS12-NEW",
		Name:      "Gas 12kg (bình xanh)",
	})
	if err != nil {
		t.Fatalf("apply: %v", err)
	}

	item, err := loadStockItemTxMust(t, svc, "prod-uuid-1")
	if err != nil {
		t.Fatal(err)
	}
	if item.SKU != "GAS12-NEW" || item.Name != "Gas 12kg (bình xanh)" {
		t.Fatalf("identity not synced: %+v", item)
	}
	if item.OnHand != 7 || item.CostPrice != 150000 {
		t.Fatalf("quantity/cost changed: on_hand=%d cost=%d", item.OnHand, item.CostPrice)
	}
}

// TestApplyProductUpdatedFixesPlaceholderRow covers rows created by
// applyOrderOUTTx, which names them after the SKU "until catalog sync".
func TestApplyProductUpdatedFixesPlaceholderRow(t *testing.T) {
	svc := &inventoryService{db: openInventoryTestDB(t)}
	if err := svc.applyOrderCompleted("evt-order", orderCompletedPayload{
		OrderID: "ord-1",
		Items:   []orderCompletedItem{{ProductID: "prod-uuid-1", SKU: "GAS12", Qty: 2}},
	}); err != nil {
		t.Fatalf("seed placeholder: %v", err)
	}

	before, _ := loadStockItemTxMust(t, svc, "prod-uuid-1")
	if before.Name != "GAS12" {
		t.Fatalf("expected placeholder name, got %q", before.Name)
	}

	if err := svc.applyProductUpdated("evt-sync", productUpdatedPayload{
		ProductID: "prod-uuid-1",
		SKU:       "GAS12",
		Name:      "Gas 12kg",
	}); err != nil {
		t.Fatalf("apply: %v", err)
	}

	after, _ := loadStockItemTxMust(t, svc, "prod-uuid-1")
	if after.Name != "Gas 12kg" {
		t.Fatalf("name=%q want the catalog name", after.Name)
	}
	if after.OnHand != -2 {
		t.Fatalf("on_hand=%d want -2 (sync must not touch quantity)", after.OnHand)
	}
}

// TestApplyProductUpdatedIdempotent pins the replay guarantee: DeliverAll
// backfills the whole CATALOG stream on first attach, and a redelivered event
// must not undo a later state.
func TestApplyProductUpdatedIdempotent(t *testing.T) {
	svc := &inventoryService{db: openInventoryTestDB(t)}
	payload := productUpdatedPayload{
		ProductID: "prod-uuid-1",
		SKU:       "GAS12",
		Name:      "Gas 12kg",
	}
	if err := svc.applyProductUpdated("evt-1", payload); err != nil {
		t.Fatalf("first apply: %v", err)
	}
	seedStock(t, svc, "prod-uuid-1", "GAS12", "Gas 12kg", 5, 100000)

	if err := svc.applyProductUpdated("evt-1", payload); err != nil {
		t.Fatalf("replay: %v", err)
	}

	item, _ := loadStockItemTxMust(t, svc, "prod-uuid-1")
	if item.OnHand != 5 {
		t.Fatalf("on_hand=%d want 5 — replay must be a no-op", item.OnHand)
	}
	var rows int
	if err := svc.db.QueryRow(
		`SELECT COUNT(*) FROM stock_items WHERE product_id = 'prod-uuid-1'`,
	).Scan(&rows); err != nil {
		t.Fatal(err)
	}
	if rows != 1 {
		t.Fatalf("stock_items rows=%d want 1", rows)
	}
}

// TestApplyProductUpdatedSkuConflictDoesNotBlock keeps a SKU collision from
// becoming a poison message that stalls the durable consumer forever.
func TestApplyProductUpdatedSkuConflictDoesNotBlock(t *testing.T) {
	svc := &inventoryService{db: openInventoryTestDB(t)}
	seedStock(t, svc, "prod-uuid-1", "GAS12", "Gas 12kg", 3, 100000)

	err := svc.applyProductUpdated("evt-conflict", productUpdatedPayload{
		ProductID: "prod-uuid-2",
		SKU:       "GAS12", // already taken by prod-uuid-1
		Name:      "Gas 12kg bản mới",
	})
	if err != nil {
		t.Fatalf("apply must not fail (would Nak forever): %v", err)
	}

	// The event is recorded so it is not redelivered...
	var processed int
	if err := svc.db.QueryRow(
		`SELECT COUNT(*) FROM processed_events WHERE event_id = 'evt-conflict'`,
	).Scan(&processed); err != nil {
		t.Fatal(err)
	}
	if processed != 1 {
		t.Fatalf("processed_events=%d want 1", processed)
	}
	// ...and the existing row keeps its SKU.
	if _, err := loadStockItemTxMust(t, svc, "prod-uuid-2"); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("conflicting row should not exist, err=%v", err)
	}
	existing, _ := loadStockItemTxMust(t, svc, "prod-uuid-1")
	if existing.OnHand != 3 {
		t.Fatalf("existing on_hand=%d want 3", existing.OnHand)
	}
}

func TestParseProductUpdatedPayload(t *testing.T) {
	t.Run("full payload", func(t *testing.T) {
		out, err := parseProductUpdatedPayload(map[string]any{
			"product_id": " prod-1 ",
			"sku":        " GAS12 ",
			"name":       " Gas 12kg ",
			"sale_price": float64(450000),
			"active":     true,
		})
		if err != nil {
			t.Fatal(err)
		}
		if out.ProductID != "prod-1" || out.SKU != "GAS12" || out.Name != "Gas 12kg" {
			t.Fatalf("payload=%+v", out)
		}
	})

	t.Run("name missing falls back to sku", func(t *testing.T) {
		// Events published before `name` was added are still in the stream.
		out, err := parseProductUpdatedPayload(map[string]any{
			"product_id": "prod-1",
			"sku":        "GAS12",
		})
		if err != nil {
			t.Fatal(err)
		}
		if out.Name != "GAS12" {
			t.Fatalf("name=%q want the SKU as fallback", out.Name)
		}
	})

	t.Run("product_id required", func(t *testing.T) {
		if _, err := parseProductUpdatedPayload(map[string]any{"sku": "GAS12"}); err == nil {
			t.Fatal("want error for missing product_id")
		}
	})

	t.Run("nil payload", func(t *testing.T) {
		if _, err := parseProductUpdatedPayload(nil); err == nil {
			t.Fatal("want error for nil payload")
		}
	})
}

func TestHandleProductUpdatedMsg(t *testing.T) {
	svc := &inventoryService{db: openInventoryTestDB(t)}
	env := events.NewEnvelope(events.CatalogProductUpdated, "evt-msg", map[string]any{
		"product_id": "prod-uuid-9",
		"sku":        "GAS45",
		"name":       "Gas 45kg",
		"sale_price": float64(1500000),
		"active":     true,
	})
	data, err := json.Marshal(env)
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.handleProductUpdatedMsg(data); err != nil {
		t.Fatalf("handle: %v", err)
	}
	item, err := loadStockItemTxMust(t, svc, "prod-uuid-9")
	if err != nil {
		t.Fatal(err)
	}
	if item.Name != "Gas 45kg" || item.OnHand != 0 {
		t.Fatalf("item=%+v", item)
	}
}

func TestHandleProductUpdatedMsgRejectsWrongSubject(t *testing.T) {
	svc := &inventoryService{db: openInventoryTestDB(t)}
	env := events.NewEnvelope(events.OrderPlaced, "evt-wrong", map[string]any{
		"product_id": "prod-uuid-9",
	})
	data, _ := json.Marshal(env)
	if err := svc.handleProductUpdatedMsg(data); err == nil {
		t.Fatal("want error for a foreign subject")
	}
}

// TestProductUpdatedConsumerJetStream runs the real durable against an embedded
// broker, including the DeliverAll backfill of an event published before the
// consumer attached.
func TestProductUpdatedConsumerJetStream(t *testing.T) {
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

	// Published first: the consumer must still see it (DeliverAll backfill).
	env := events.NewEnvelope(events.CatalogProductUpdated, "evt-js-1", map[string]any{
		"product_id": "prod-uuid-js",
		"sku":        "GAS12",
		"name":       "Gas 12kg",
		"sale_price": 450000,
		"active":     true,
	})
	if _, err := natsx.PublishEnvelope(js, env); err != nil {
		t.Fatalf("publish: %v", err)
	}

	svc := &inventoryService{db: openInventoryTestDB(t)}
	sub, err := startProductUpdatedConsumer(js, svc)
	if err != nil {
		t.Fatalf("start consumer: %v", err)
	}
	defer func() { _ = sub.Unsubscribe() }()

	deadline := time.Now().Add(5 * time.Second)
	for {
		item, err := loadStockItemTxMust(t, svc, "prod-uuid-js")
		if err == nil && item.Name == "Gas 12kg" {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("stock row not created from catalog.product.updated (last err=%v)", err)
		}
		time.Sleep(50 * time.Millisecond)
	}
}
