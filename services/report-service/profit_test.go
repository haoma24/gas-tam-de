package main

import "testing"

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
