package main

import (
	"database/sql"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"gas-tam-de/pkg/httpx"
)

const dashboardTimezone = "Asia/Ho_Chi_Minh"

// dashboardSummaryResponse is GET /v1/admin/dashboard/summary (T8.1.2).
// Period totals come from daily_stats; debt_total from the debt read-model.
type dashboardSummaryResponse struct {
	From            string `json:"from"`
	To              string `json:"to"`
	Timezone        string `json:"timezone"`
	RevenueVnd      int64  `json:"revenue_vnd"`
	CogsVnd         int64  `json:"cogs_vnd"`
	DeliveryFeeVnd  int64  `json:"delivery_fee_vnd"`
	ProfitVnd       int64  `json:"profit_vnd"`
	OrdersCompleted int64  `json:"orders_completed"`
	OrdersPlaced    int64  `json:"orders_placed"`
	DebtTotal       int64  `json:"debt_total"`
}

// handleDashboardSummary serves GET /v1/admin/dashboard/summary.
// Query: omitted → today (VN); day=YYYY-MM-DD; or from=&to= inclusive range.
// Gateway mounts under /v1/admin/* with role=admin RBAC.
func (s *reportService) handleDashboardSummary(w http.ResponseWriter, r *http.Request) {
	from, to, err := parseDashboardSummaryRange(r, time.Now().UTC())
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}

	out, err := s.dashboardSummary(from, to)
	if err != nil {
		slog.Error("dashboard summary", "err", err, "from", from, "to", to)
		httpx.Error(w, http.StatusInternalServerError, "INTERNAL", "could not load dashboard summary")
		return
	}
	httpx.JSON(w, http.StatusOK, out)
}

func (s *reportService) dashboardSummary(from, to string) (dashboardSummaryResponse, error) {
	agg, err := sumDailyStats(s.db, from, to)
	if err != nil {
		return dashboardSummaryResponse{}, err
	}
	debt, err := loadDebtTotal(s.db)
	if err != nil {
		return dashboardSummaryResponse{}, err
	}
	return dashboardSummaryResponse{
		From:            from,
		To:              to,
		Timezone:        dashboardTimezone,
		RevenueVnd:      agg.RevenueVnd,
		CogsVnd:         agg.CogsVnd,
		DeliveryFeeVnd:  agg.DeliveryFeeVnd,
		ProfitVnd:       agg.ProfitVnd,
		OrdersCompleted: agg.OrdersCompleted,
		OrdersPlaced:    agg.OrdersPlaced,
		DebtTotal:       debt,
	}, nil
}

type dailyStatsAggregate struct {
	RevenueVnd      int64
	CogsVnd         int64
	DeliveryFeeVnd  int64
	ProfitVnd       int64
	OrdersCompleted int64
	OrdersPlaced    int64
}

func sumDailyStats(db *sql.DB, from, to string) (dailyStatsAggregate, error) {
	var a dailyStatsAggregate
	err := db.QueryRow(`
		SELECT
			COALESCE(SUM(revenue_vnd), 0),
			COALESCE(SUM(cogs_vnd), 0),
			COALESCE(SUM(delivery_fee_vnd), 0),
			COALESCE(SUM(orders_completed), 0),
			COALESCE(SUM(orders_placed), 0)
		FROM daily_stats
		WHERE day >= ? AND day <= ?`, from, to,
	).Scan(
		&a.RevenueVnd, &a.CogsVnd, &a.DeliveryFeeVnd,
		&a.OrdersCompleted, &a.OrdersPlaced,
	)
	if err != nil {
		return dailyStatsAggregate{}, err
	}
	a.ProfitVnd = ComputeProfit(a.RevenueVnd, a.CogsVnd)
	return a, nil
}

func parseDashboardSummaryRange(r *http.Request, now time.Time) (from, to string, err error) {
	q := r.URL.Query()
	day := strings.TrimSpace(q.Get("day"))
	fromRaw := strings.TrimSpace(q.Get("from"))
	toRaw := strings.TrimSpace(q.Get("to"))

	if day != "" && (fromRaw != "" || toRaw != "") {
		return "", "", fmt.Errorf("use either day or from/to, not both")
	}

	if day != "" {
		if err := validateDayKey(day); err != nil {
			return "", "", err
		}
		return day, day, nil
	}

	if fromRaw != "" || toRaw != "" {
		if fromRaw == "" || toRaw == "" {
			return "", "", fmt.Errorf("from and to are both required")
		}
		if err := validateDayKey(fromRaw); err != nil {
			return "", "", fmt.Errorf("from: %w", err)
		}
		if err := validateDayKey(toRaw); err != nil {
			return "", "", fmt.Errorf("to: %w", err)
		}
		if fromRaw > toRaw {
			return "", "", fmt.Errorf("from must be <= to")
		}
		return fromRaw, toRaw, nil
	}

	today := DayKeyVN(now)
	return today, today, nil
}

func validateDayKey(day string) error {
	if _, err := time.ParseInLocation("2006-01-02", day, vnLocation); err != nil {
		return fmt.Errorf("day must be YYYY-MM-DD")
	}
	return nil
}
