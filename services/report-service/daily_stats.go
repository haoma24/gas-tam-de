package main

import (
	"database/sql"
	"fmt"
	"strings"
	"time"
)

// Vietnam (ICT, UTC+7) — daily_stats.day is YYYY-MM-DD in this zone (architecture §6.7).
var vnLocation = time.FixedZone("Asia/Ho_Chi_Minh", 7*3600)

// DailyStatsDelta is an additive contribution to one day's daily_stats row.
type DailyStatsDelta struct {
	RevenueVnd      int64
	CogsVnd         int64
	DeliveryFeeVnd  int64
	OrdersCompleted int64
	OrdersPlaced    int64
}

// DayKeyVN formats t as YYYY-MM-DD in Vietnam time.
func DayKeyVN(t time.Time) string {
	if t.IsZero() {
		t = time.Now().UTC()
	}
	return t.In(vnLocation).Format("2006-01-02")
}

func isEventProcessedTx(tx *sql.Tx, eventID string) (bool, error) {
	var n int
	err := tx.QueryRow(`SELECT COUNT(1) FROM processed_events WHERE event_id = ?`, eventID).Scan(&n)
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

func insertProcessedEventTx(tx *sql.Tx, eventID, processedAt string) error {
	_, err := tx.Exec(`
		INSERT INTO processed_events (event_id, processed_at) VALUES (?, ?)`,
		eventID, processedAt)
	return err
}

// applyDailyStatsDelta upserts daily_stats for day (VN) by adding delta, then recomputes profit.
// Caller must already gate on processed_events for idempotency.
func applyDailyStatsDeltaTx(tx *sql.Tx, day string, delta DailyStatsDelta) error {
	day = strings.TrimSpace(day)
	if day == "" {
		return fmt.Errorf("day is required")
	}

	var row DailyStatsAmounts
	var ordersCompleted, ordersPlaced int64
	err := tx.QueryRow(`
		SELECT revenue_vnd, cogs_vnd, delivery_fee_vnd, orders_completed, orders_placed, profit_vnd
		FROM daily_stats WHERE day = ?`, day,
	).Scan(
		&row.RevenueVnd, &row.CogsVnd, &row.DeliveryFeeVnd,
		&ordersCompleted, &ordersPlaced, &row.ProfitVnd,
	)
	if err != nil && err != sql.ErrNoRows {
		return err
	}

	row.RevenueVnd += delta.RevenueVnd
	row.CogsVnd += delta.CogsVnd
	row.DeliveryFeeVnd += delta.DeliveryFeeVnd
	ordersCompleted += delta.OrdersCompleted
	ordersPlaced += delta.OrdersPlaced
	ApplyProfit(&row)

	_, err = tx.Exec(`
		INSERT INTO daily_stats (
			day, revenue_vnd, cogs_vnd, delivery_fee_vnd,
			orders_completed, orders_placed, profit_vnd
		) VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(day) DO UPDATE SET
			revenue_vnd = excluded.revenue_vnd,
			cogs_vnd = excluded.cogs_vnd,
			delivery_fee_vnd = excluded.delivery_fee_vnd,
			orders_completed = excluded.orders_completed,
			orders_placed = excluded.orders_placed,
			profit_vnd = excluded.profit_vnd`,
		day, row.RevenueVnd, row.CogsVnd, row.DeliveryFeeVnd,
		ordersCompleted, ordersPlaced, row.ProfitVnd,
	)
	return err
}

type dailyStatsRow struct {
	Day             string
	RevenueVnd      int64
	CogsVnd         int64
	DeliveryFeeVnd  int64
	OrdersCompleted int64
	OrdersPlaced    int64
	ProfitVnd       int64
}

func loadDailyStats(db *sql.DB, day string) (dailyStatsRow, error) {
	var r dailyStatsRow
	err := db.QueryRow(`
		SELECT day, revenue_vnd, cogs_vnd, delivery_fee_vnd,
		       orders_completed, orders_placed, profit_vnd
		FROM daily_stats WHERE day = ?`, day,
	).Scan(
		&r.Day, &r.RevenueVnd, &r.CogsVnd, &r.DeliveryFeeVnd,
		&r.OrdersCompleted, &r.OrdersPlaced, &r.ProfitVnd,
	)
	return r, err
}
