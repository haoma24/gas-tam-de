package main

import (
	"testing"
	"time"

	"gas-tam-de/pkg/events"
)

func TestSumSaleRevenue(t *testing.T) {
	got := SumSaleRevenue([]SaleLine{
		{Qty: 2, UnitPrice: 100_000},
		{Qty: 1, UnitPrice: 50_000},
	})
	if got != 250_000 {
		t.Fatalf("SumSaleRevenue=%d want 250000", got)
	}
	if SumSaleRevenue(nil) != 0 {
		t.Fatal("empty sale lines should be 0")
	}
}

func TestSumCOGS(t *testing.T) {
	got := SumCOGS([]COGSLine{
		{Qty: 2, UnitCost: 80_000},
		{Qty: 1, UnitCost: 40_000},
	})
	if got != 200_000 {
		t.Fatalf("SumCOGS=%d want 200000", got)
	}
	if SumCOGS(nil) != 0 {
		t.Fatal("empty cogs lines should be 0")
	}
}

func TestComputeProfit(t *testing.T) {
	if got := ComputeProfit(250_000, 200_000); got != 50_000 {
		t.Fatalf("ComputeProfit=%d want 50000", got)
	}
	// COGS can exceed revenue (e.g. promo / underpriced) — allow negative profit.
	if got := ComputeProfit(100_000, 150_000); got != -50_000 {
		t.Fatalf("negative profit=%d want -50000", got)
	}
}

func TestBuildDailyStatsAmounts_IgnoresDeliveryFeeInProfit(t *testing.T) {
	got := BuildDailyStatsAmounts(
		[]SaleLine{{Qty: 1, UnitPrice: 300_000}},
		[]COGSLine{{Qty: 1, UnitCost: 200_000}},
		25_000, // delivery fee tracked separately
	)
	want := DailyStatsAmounts{
		RevenueVnd:     300_000,
		CogsVnd:        200_000,
		DeliveryFeeVnd: 25_000,
		ProfitVnd:      100_000, // 300k - 200k; fee NOT subtracted
	}
	if got != want {
		t.Fatalf("got %+v want %+v", got, want)
	}
}

func TestDailyStatsFromTotals(t *testing.T) {
	got := DailyStatsFromTotals(500_000, 350_000, 40_000)
	want := DailyStatsAmounts{
		RevenueVnd:     500_000,
		CogsVnd:        350_000,
		DeliveryFeeVnd: 40_000,
		ProfitVnd:      150_000,
	}
	if got != want {
		t.Fatalf("got %+v want %+v", got, want)
	}
}

func TestApplyProfit(t *testing.T) {
	a := &DailyStatsAmounts{
		RevenueVnd:     400_000,
		CogsVnd:        250_000,
		DeliveryFeeVnd: 30_000,
		ProfitVnd:      999, // stale — must be recomputed
	}
	ApplyProfit(a)
	if a.ProfitVnd != 150_000 {
		t.Fatalf("ProfitVnd=%d want 150000", a.ProfitVnd)
	}
	if a.DeliveryFeeVnd != 30_000 {
		t.Fatalf("DeliveryFeeVnd mutated: %d", a.DeliveryFeeVnd)
	}
	ApplyProfit(nil) // no panic
}

func TestBuildDailyStatsAmounts_MultiLineOrder(t *testing.T) {
	// Mirrors architecture example: product subtotal = revenue; COGS from snapshots.
	got := BuildDailyStatsAmounts(
		[]SaleLine{
			{Qty: 2, UnitPrice: 360_000},
			{Qty: 1, UnitPrice: 50_000},
		},
		[]COGSLine{
			{Qty: 2, UnitCost: 280_000},
			{Qty: 1, UnitCost: 30_000},
		},
		10_000,
	)
	if got.RevenueVnd != 770_000 {
		t.Fatalf("revenue=%d want 770000", got.RevenueVnd)
	}
	if got.CogsVnd != 590_000 {
		t.Fatalf("cogs=%d want 590000", got.CogsVnd)
	}
	if got.ProfitVnd != 180_000 {
		t.Fatalf("profit=%d want 180000", got.ProfitVnd)
	}
	if got.DeliveryFeeVnd != 10_000 {
		t.Fatalf("fee=%d want 10000", got.DeliveryFeeVnd)
	}
}

// TestOrderCompletedWithUnitCostShrinksProfit is the shop owner's bug in one
// test: the event used to arrive without unit_cost, COGS summed to 0, and
// profit came back equal to revenue. With the cost on the line the two numbers
// must diverge by exactly the purchase price.
func TestOrderCompletedWithUnitCostShrinksProfit(t *testing.T) {
	env := events.Envelope{
		Subject:    events.OrderCompleted,
		EventID:    "evt-cogs-1",
		OccurredAt: time.Date(2026, 8, 2, 3, 0, 0, 0, time.UTC),
		Payload: map[string]any{
			"order_id":     "ord-1",
			"total":        float64(950000),
			"delivery_fee": float64(50000),
			"items": []any{
				map[string]any{"qty": float64(2), "unit_price": float64(450000), "unit_cost": float64(380000)},
			},
		},
	}

	got, day, err := parseOrderCompleted(env)
	if err != nil {
		t.Fatal(err)
	}
	if day != "2026-08-02" {
		t.Fatalf("day=%q", day)
	}
	if got.Amounts.RevenueVnd != 900000 {
		t.Fatalf("revenue=%d, want 900000", got.Amounts.RevenueVnd)
	}
	if got.Amounts.CogsVnd != 760000 {
		t.Fatalf("cogs=%d, want 760000", got.Amounts.CogsVnd)
	}
	if got.Amounts.ProfitVnd != 140000 {
		t.Fatalf("profit=%d, want 140000 (revenue − COGS)", got.Amounts.ProfitVnd)
	}
	if got.Amounts.ProfitVnd == got.Amounts.RevenueVnd {
		t.Fatal("profit must not equal revenue once COGS is known")
	}
	// Delivery fee is tracked, never folded into profit.
	if got.Amounts.DeliveryFeeVnd != 50000 {
		t.Fatalf("delivery_fee=%d, want 50000", got.Amounts.DeliveryFeeVnd)
	}
}

// TestOrderCompletedWithoutUnitCostStillWorks — orders placed before the field
// existed carry no cost; those days keep reporting profit = revenue rather than
// failing to be counted at all.
func TestOrderCompletedWithoutUnitCostStillWorks(t *testing.T) {
	env := events.Envelope{
		Subject:    events.OrderCompleted,
		EventID:    "evt-cogs-2",
		OccurredAt: time.Date(2026, 8, 2, 3, 0, 0, 0, time.UTC),
		Payload: map[string]any{
			"order_id": "ord-2",
			"total":    float64(450000),
			"items": []any{
				map[string]any{"qty": float64(1), "unit_price": float64(450000)},
			},
		},
	}

	got, _, err := parseOrderCompleted(env)
	if err != nil {
		t.Fatal(err)
	}
	if got.Amounts.CogsVnd != 0 || got.Amounts.ProfitVnd != 450000 {
		t.Fatalf("cogs=%d profit=%d, want 0 / 450000", got.Amounts.CogsVnd, got.Amounts.ProfitVnd)
	}
}
