package main

import (
	"database/sql"
	"encoding/json"
	"path/filepath"
	"testing"
	"time"

	"gas-tam-de/pkg/events"
	"gas-tam-de/pkg/natsx"
	"gas-tam-de/pkg/sqlite"

	"github.com/nats-io/nats-server/v2/server"
)

func openReportTestDB(t *testing.T) *sql.DB {
	t.Helper()
	path := filepath.Join(t.TempDir(), "report.db")
	db, err := sqlite.Open(path)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := migrate(db); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}

func TestApplyOrderPlaced_UpsertsOrdersPlaced(t *testing.T) {
	svc := &reportService{db: openReportTestDB(t)}
	day := "2026-08-02"
	if err := svc.applyOrderPlaced("evt-place-1", "ord-1", day); err != nil {
		t.Fatalf("apply: %v", err)
	}
	row, err := loadDailyStats(svc.db, day)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if row.OrdersPlaced != 1 {
		t.Fatalf("orders_placed=%d want 1", row.OrdersPlaced)
	}
	if row.OrdersCompleted != 0 || row.RevenueVnd != 0 {
		t.Fatalf("unexpected money/completed on place: %+v", row)
	}

	// Idempotent
	if err := svc.applyOrderPlaced("evt-place-1", "ord-1", day); err != nil {
		t.Fatalf("reapply: %v", err)
	}
	row, _ = loadDailyStats(svc.db, day)
	if row.OrdersPlaced != 1 {
		t.Fatalf("orders_placed after dup=%d want 1", row.OrdersPlaced)
	}

	if err := svc.applyOrderPlaced("evt-place-2", "ord-2", day); err != nil {
		t.Fatalf("second place: %v", err)
	}
	row, _ = loadDailyStats(svc.db, day)
	if row.OrdersPlaced != 2 {
		t.Fatalf("orders_placed=%d want 2", row.OrdersPlaced)
	}
}

func TestApplyOrderCompleted_UpsertsRevenueCOGSProfit(t *testing.T) {
	svc := &reportService{db: openReportTestDB(t)}
	day := "2026-08-02"
	payload := orderCompletedStats{
		OrderID: "ord-done",
		Amounts: BuildDailyStatsAmounts(
			[]SaleLine{{Qty: 2, UnitPrice: 100_000}},
			[]COGSLine{{Qty: 2, UnitCost: 70_000}},
			15_000,
		),
	}
	if err := svc.applyOrderCompleted("evt-done-1", day, payload); err != nil {
		t.Fatalf("apply: %v", err)
	}
	row, err := loadDailyStats(svc.db, day)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if row.OrdersCompleted != 1 {
		t.Fatalf("orders_completed=%d want 1", row.OrdersCompleted)
	}
	if row.RevenueVnd != 200_000 {
		t.Fatalf("revenue=%d want 200000", row.RevenueVnd)
	}
	if row.CogsVnd != 140_000 {
		t.Fatalf("cogs=%d want 140000", row.CogsVnd)
	}
	if row.DeliveryFeeVnd != 15_000 {
		t.Fatalf("fee=%d want 15000", row.DeliveryFeeVnd)
	}
	if row.ProfitVnd != 60_000 {
		t.Fatalf("profit=%d want 60000 (fee not subtracted)", row.ProfitVnd)
	}

	// Aggregate second order same day
	payload2 := orderCompletedStats{
		OrderID: "ord-done-2",
		Amounts: BuildDailyStatsAmounts(
			[]SaleLine{{Qty: 1, UnitPrice: 50_000}},
			[]COGSLine{{Qty: 1, UnitCost: 20_000}},
			5_000,
		),
	}
	if err := svc.applyOrderCompleted("evt-done-2", day, payload2); err != nil {
		t.Fatalf("second: %v", err)
	}
	row, _ = loadDailyStats(svc.db, day)
	if row.OrdersCompleted != 2 {
		t.Fatalf("orders_completed=%d want 2", row.OrdersCompleted)
	}
	if row.RevenueVnd != 250_000 || row.CogsVnd != 160_000 || row.ProfitVnd != 90_000 {
		t.Fatalf("aggregated row=%+v", row)
	}
	if row.DeliveryFeeVnd != 20_000 {
		t.Fatalf("fee=%d want 20000", row.DeliveryFeeVnd)
	}

	// Idempotent
	if err := svc.applyOrderCompleted("evt-done-1", day, payload); err != nil {
		t.Fatalf("idempotent: %v", err)
	}
	row, _ = loadDailyStats(svc.db, day)
	if row.OrdersCompleted != 2 || row.RevenueVnd != 250_000 {
		t.Fatalf("dup mutated row=%+v", row)
	}
}

func TestHandleOrderCompletedMsg_DerivesFeeAndOptionalCOGS(t *testing.T) {
	svc := &reportService{db: openReportTestDB(t)}
	occurred := time.Date(2026, 8, 2, 10, 0, 0, 0, time.UTC)
	env := events.NewEnvelope(events.OrderCompleted, "evt-msg-1", map[string]any{
		"order_id": "ord-x",
		"items": []any{
			map[string]any{"product_id": "p1", "qty": 1, "unit_price": 300_000, "unit_cost": 200_000},
		},
		"total":        325_000, // implies delivery_fee 25_000
		"payment_type": "FULL",
		"amount_paid":  325_000,
	})
	env.OccurredAt = occurred
	data, err := json.Marshal(env)
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.handleOrderCompletedMsg(data); err != nil {
		t.Fatalf("handle: %v", err)
	}
	day := DayKeyVN(occurred)
	row, err := loadDailyStats(svc.db, day)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if row.RevenueVnd != 300_000 || row.CogsVnd != 200_000 || row.DeliveryFeeVnd != 25_000 || row.ProfitVnd != 100_000 {
		t.Fatalf("row=%+v", row)
	}
}

func TestHandleOrderPlacedMsg(t *testing.T) {
	svc := &reportService{db: openReportTestDB(t)}
	created := time.Date(2026, 8, 2, 3, 0, 0, 0, time.UTC) // 10:00 VN
	env := events.NewEnvelope(events.OrderPlaced, "evt-place-msg", map[string]any{
		"order_id":    "ord-p",
		"total":       100_000,
		"distance_km": 2.5,
		"created_at":  created.Format(time.RFC3339),
	})
	data, err := json.Marshal(env)
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.handleOrderPlacedMsg(data); err != nil {
		t.Fatalf("handle: %v", err)
	}
	day := DayKeyVN(created)
	row, err := loadDailyStats(svc.db, day)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if row.OrdersPlaced != 1 {
		t.Fatalf("orders_placed=%d want 1", row.OrdersPlaced)
	}
}

func TestDayKeyVN(t *testing.T) {
	// 2026-08-01 20:00 UTC → 2026-08-02 03:00 VN
	utc := time.Date(2026, 8, 1, 20, 0, 0, 0, time.UTC)
	if got := DayKeyVN(utc); got != "2026-08-02" {
		t.Fatalf("DayKeyVN=%q want 2026-08-02", got)
	}
}

func TestDeliveryFeeFromPayload(t *testing.T) {
	if got := deliveryFeeFromPayload(map[string]any{"delivery_fee": 12_000}, 100, 200); got != 12_000 {
		t.Fatalf("explicit fee=%d", got)
	}
	if got := deliveryFeeFromPayload(map[string]any{"total": 325_000}, 300_000, 325_000); got != 25_000 {
		t.Fatalf("derived fee=%d", got)
	}
	if got := deliveryFeeFromPayload(map[string]any{}, 400_000, 350_000); got != 0 {
		t.Fatalf("negative derive clamped=%d", got)
	}
}

func TestReportConsumersJetStream(t *testing.T) {
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

	svc := &reportService{db: openReportTestDB(t)}
	subs, err := startReportConsumers(js, svc)
	if err != nil {
		t.Fatalf("start consumers: %v", err)
	}
	defer func() {
		for _, s := range subs {
			_ = s.Unsubscribe()
		}
	}()

	occurred := time.Date(2026, 8, 2, 8, 0, 0, 0, time.UTC)
	placed := events.NewEnvelope(events.OrderPlaced, "evt-js-place", map[string]any{
		"order_id":    "ord-js",
		"total":       450_000,
		"distance_km": 3.0,
		"created_at":  occurred.Format(time.RFC3339),
	})
	placed.OccurredAt = occurred
	if _, err := natsx.PublishEnvelope(js, placed); err != nil {
		t.Fatalf("publish placed: %v", err)
	}

	completed := events.NewEnvelope(events.OrderCompleted, "evt-js-done", map[string]any{
		"order_id": "ord-js",
		"items": []map[string]any{
			{"product_id": "p1", "qty": 1, "unit_price": 400_000, "unit_cost": 250_000},
		},
		"total":        450_000,
		"payment_type": "FULL",
		"amount_paid":  450_000,
	})
	completed.OccurredAt = occurred
	if _, err := natsx.PublishEnvelope(js, completed); err != nil {
		t.Fatalf("publish completed: %v", err)
	}

	day := DayKeyVN(occurred)
	deadline := time.Now().Add(3 * time.Second)
	for {
		row, err := loadDailyStats(svc.db, day)
		if err == nil && row.OrdersPlaced == 1 && row.OrdersCompleted == 1 && row.RevenueVnd == 400_000 {
			if row.CogsVnd != 250_000 || row.ProfitVnd != 150_000 || row.DeliveryFeeVnd != 50_000 {
				t.Fatalf("money mismatch: %+v", row)
			}
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("timeout waiting for daily_stats; err=%v row=%+v", err, row)
		}
		time.Sleep(50 * time.Millisecond)
	}

	info, err := js.ConsumerInfo("ORDERS", durableOrderCompleted)
	if err != nil {
		t.Fatalf("ConsumerInfo completed: %v", err)
	}
	if info.Name != durableOrderCompleted {
		t.Fatalf("durable=%q", info.Name)
	}
	info, err = js.ConsumerInfo("ORDERS", durableOrderPlaced)
	if err != nil {
		t.Fatalf("ConsumerInfo placed: %v", err)
	}
	if info.Name != durableOrderPlaced {
		t.Fatalf("durable placed=%q", info.Name)
	}
}
