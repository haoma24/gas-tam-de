package main

// Profit formula for report-service (T7.2.2 / architecture §6.7).
//
// MVP:
//
//	profit_vnd = revenue_vnd - cogs_vnd
//
// - revenue_vnd = Σ(qty × unit_price) on completed order sale lines (product subtotal).
// - cogs_vnd    = Σ(qty × unit_cost) from inventory OUT/`ORDER` COGS snapshots (T7.2.1);
//   never use current stock_items.cost_price.
// - delivery_fee_vnd is tracked separately and is NOT subtracted from profit.

// SaleLine is one completed-order product line (sale price snapshot).
type SaleLine struct {
	Qty       int64
	UnitPrice int64
}

// COGSLine is one inventory OUT movement cost snapshot (unit_cost at export time).
type COGSLine struct {
	Qty      int64
	UnitCost int64
}

// DailyStatsAmounts holds the monetary columns of daily_stats (architecture §6.7).
type DailyStatsAmounts struct {
	RevenueVnd     int64
	CogsVnd        int64
	DeliveryFeeVnd int64
	ProfitVnd      int64
}

// SumSaleRevenue returns Σ(qty × unit_price) — doanh thu hàng (product subtotal).
func SumSaleRevenue(lines []SaleLine) int64 {
	var sum int64
	for _, l := range lines {
		sum += l.Qty * l.UnitPrice
	}
	return sum
}

// SumCOGS returns Σ(qty × unit_cost) from COGS snapshots (not live cost_price).
func SumCOGS(lines []COGSLine) int64 {
	var sum int64
	for _, l := range lines {
		sum += l.Qty * l.UnitCost
	}
	return sum
}

// ComputeProfit is the MVP profit formula: revenue − COGS.
// Delivery fee is intentionally excluded (tracked on daily_stats.delivery_fee_vnd).
func ComputeProfit(revenueVnd, cogsVnd int64) int64 {
	return revenueVnd - cogsVnd
}

// BuildDailyStatsAmounts computes daily_stats money fields for one completed-order contribution.
func BuildDailyStatsAmounts(saleLines []SaleLine, cogsLines []COGSLine, deliveryFeeVnd int64) DailyStatsAmounts {
	revenue := SumSaleRevenue(saleLines)
	cogs := SumCOGS(cogsLines)
	return DailyStatsAmounts{
		RevenueVnd:     revenue,
		CogsVnd:        cogs,
		DeliveryFeeVnd: deliveryFeeVnd,
		ProfitVnd:      ComputeProfit(revenue, cogs),
	}
}

// DailyStatsFromTotals builds amounts when revenue/COGS/fee are already aggregated.
func DailyStatsFromTotals(revenueVnd, cogsVnd, deliveryFeeVnd int64) DailyStatsAmounts {
	return DailyStatsAmounts{
		RevenueVnd:     revenueVnd,
		CogsVnd:        cogsVnd,
		DeliveryFeeVnd: deliveryFeeVnd,
		ProfitVnd:      ComputeProfit(revenueVnd, cogsVnd),
	}
}

// ApplyProfit sets ProfitVnd = RevenueVnd − CogsVnd on an aggregate row
// (e.g. after summing daily_stats deltas). DeliveryFeeVnd is left unchanged.
func ApplyProfit(a *DailyStatsAmounts) {
	if a == nil {
		return
	}
	a.ProfitVnd = ComputeProfit(a.RevenueVnd, a.CogsVnd)
}
