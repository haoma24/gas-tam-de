package main

import (
	"context"
	"log/slog"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"gas-tam-de/pkg/httpx"
)

// Vietnam (ICT, UTC+7). orders.created_at is RFC3339 UTC, but the shop counts
// days in local time — the same convention report-service uses for
// daily_stats.day, so the two numbers on the Báo cáo screen agree.
var vnLocation = time.FixedZone("Asia/Ho_Chi_Minh", 7*3600)

const (
	// customerStatsDefaultDays is the window when no from/to is given.
	customerStatsDefaultDays  = 30
	customerStatsDefaultLimit = 200
	customerStatsMaxLimit     = 1000
)

// customerStat is one customer's activity over the requested period.
type customerStat struct {
	UserID       string `json:"user_id"`
	CustomerName string `json:"customer_name"`
	// CustomerPhone is the callable number; PhoneMasked is the fallback for
	// customers auth-service has no number on file for.
	CustomerPhone string `json:"customer_phone,omitempty"`
	PhoneMasked   string `json:"phone_masked"`
	AddressText   string `json:"address_text,omitempty"`

	OrdersTotal     int64 `json:"orders_total"`
	OrdersCompleted int64 `json:"orders_completed"`
	OrdersCancelled int64 `json:"orders_cancelled"`
	OrdersPending   int64 `json:"orders_pending"`

	// Money counts completed orders only — the same rule the rest of the system
	// follows: nothing is earned until the order is handed over.
	SpentVnd int64 `json:"spent_vnd"`
	PaidVnd  int64 `json:"paid_vnd"`
	DebtVnd  int64 `json:"debt_vnd"`

	FirstOrderAt string `json:"first_order_at"`
	LastOrderAt  string `json:"last_order_at"`
}

// handleListCustomerStats serves GET /v1/admin/orders/customers — «khách nào đã
// đặt bao nhiêu đơn» for the Báo cáo tab.
//
// Query: from=&to= (YYYY-MM-DD, VN days, inclusive) or nothing for the last 30
// days; limit= caps the rows returned. Aggregated straight from `orders`: this
// is a GROUP BY over one table, not a read-model worth its own event stream.
//
// Chi routes this before /v1/admin/orders/{id} — a static segment wins over a
// param — and the gateway already proxies /v1/admin/orders/* to this service.
func (s *orderService) handleListCustomerStats(w http.ResponseWriter, r *http.Request) {
	from, to, err := parseCustomerStatsRange(r, time.Now().UTC())
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}
	limit, err := parseCustomerStatsLimit(r.URL.Query().Get("limit"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}

	stats, err := s.customerStats(from, to)
	if err != nil {
		slog.Error("customer stats", "err", err, "from", from, "to", to)
		httpx.Error(w, http.StatusInternalServerError, "INTERNAL", "could not load customer stats")
		return
	}

	s.fillStatPhones(r.Context(), stats)

	total := len(stats)
	if len(stats) > limit {
		stats = stats[:limit]
	}

	httpx.JSON(w, http.StatusOK, map[string]any{
		"from":      from,
		"to":        to,
		"timezone":  "Asia/Ho_Chi_Minh",
		"count":     len(stats),
		"total":     total,
		"customers": stats,
	})
}

// customerStats groups the period's orders by user. Aggregating in Go rather
// than in SQL keeps the "latest name/address wins" rule explicit instead of
// leaning on SQLite's bare-column-with-max behaviour, which is undefined when a
// query mixes min() and max().
func (s *orderService) customerStats(from, to string) ([]customerStat, error) {
	rows, err := s.db.Query(`
		SELECT user_id, customer_name, phone_masked, customer_phone, address_text,
		       status, total, COALESCE(amount_paid, 0), created_at
		FROM orders
		WHERE date(created_at, '+7 hours') >= ? AND date(created_at, '+7 hours') <= ?
		ORDER BY created_at ASC`, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	byUser := make(map[string]*customerStat)
	order := make([]string, 0)

	for rows.Next() {
		var (
			userID, name, masked, phone, address string
			status, createdAt                    string
			total, amountPaid                    int64
		)
		if err := rows.Scan(&userID, &name, &masked, &phone, &address,
			&status, &total, &amountPaid, &createdAt); err != nil {
			return nil, err
		}

		c, seen := byUser[userID]
		if !seen {
			c = &customerStat{UserID: userID, FirstOrderAt: createdAt}
			byUser[userID] = c
			order = append(order, userID)
		}

		// Rows arrive oldest first, so the last write wins: the customer is
		// shown under the name, address and number of their most recent order.
		c.CustomerName = name
		c.PhoneMasked = masked
		c.AddressText = address
		if p := displayPhone(phone); p != "" {
			c.CustomerPhone = p
		}
		c.LastOrderAt = createdAt

		c.OrdersTotal++
		switch status {
		case orderStatusCompleted:
			c.OrdersCompleted++
			c.SpentVnd += total
			c.PaidVnd += amountPaid
		case orderStatusCancelled:
			c.OrdersCancelled++
		case orderStatusPending:
			c.OrdersPending++
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	out := make([]customerStat, 0, len(order))
	for _, id := range order {
		c := byUser[id]
		c.DebtVnd = c.SpentVnd - c.PaidVnd
		out = append(out, *c)
	}

	// Biggest customers first; ties broken by order count so a frequent buyer
	// of cheap items does not sink below a one-off large order.
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].SpentVnd != out[j].SpentVnd {
			return out[i].SpentVnd > out[j].SpentVnd
		}
		return out[i].OrdersTotal > out[j].OrdersTotal
	})
	return out, nil
}

// fillStatPhones resolves numbers for customers whose orders all predate the
// customer_phone column. One batched call, best-effort — same contract as
// fillCustomerPhones, without persisting (the order rows get theirs when the
// desk lists them).
func (s *orderService) fillStatPhones(ctx context.Context, stats []customerStat) {
	if s.authDir == nil {
		return
	}
	missing := make([]string, 0)
	for _, c := range stats {
		if c.CustomerPhone == "" {
			missing = append(missing, c.UserID)
		}
	}
	if len(missing) == 0 {
		return
	}
	phones, err := s.authDir.PhonesByUserID(ctx, missing)
	if err != nil {
		slog.Error("customer stats phones", "customers", len(missing), "err", err)
		return
	}
	for i := range stats {
		if stats[i].CustomerPhone != "" {
			continue
		}
		if p := strings.TrimSpace(phones[stats[i].UserID]); p != "" {
			stats[i].CustomerPhone = displayPhone(p)
		}
	}
}

// parseCustomerStatsRange returns inclusive VN day keys. Omitted → the last
// [customerStatsDefaultDays] days ending today.
func parseCustomerStatsRange(r *http.Request, now time.Time) (from, to string, err error) {
	q := r.URL.Query()
	fromRaw := strings.TrimSpace(q.Get("from"))
	toRaw := strings.TrimSpace(q.Get("to"))

	if fromRaw == "" && toRaw == "" {
		today := now.In(vnLocation)
		return dayKeyVN(today.AddDate(0, 0, -(customerStatsDefaultDays - 1))), dayKeyVN(today), nil
	}
	if fromRaw == "" || toRaw == "" {
		return "", "", errString("from and to are both required")
	}
	if err := validateDayKey(fromRaw); err != nil {
		return "", "", err
	}
	if err := validateDayKey(toRaw); err != nil {
		return "", "", err
	}
	if fromRaw > toRaw {
		return "", "", errString("from must be <= to")
	}
	return fromRaw, toRaw, nil
}

func parseCustomerStatsLimit(raw string) (int, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return customerStatsDefaultLimit, nil
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n < 1 {
		return 0, errString("limit must be a positive integer")
	}
	if n > customerStatsMaxLimit {
		n = customerStatsMaxLimit
	}
	return n, nil
}

func dayKeyVN(t time.Time) string {
	return t.In(vnLocation).Format("2006-01-02")
}

func validateDayKey(day string) error {
	if _, err := time.ParseInLocation("2006-01-02", day, vnLocation); err != nil {
		return errString("day must be YYYY-MM-DD")
	}
	return nil
}
