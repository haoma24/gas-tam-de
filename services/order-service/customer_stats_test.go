package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// insertTestOrderMoney inserts an order with an explicit settlement so the
// customer aggregate has something to sum.
func insertTestOrderMoney(
	t *testing.T, svc *orderService,
	id, userID, customerName, status, createdAt string,
	total, amountPaid int64,
) {
	t.Helper()
	_, err := svc.db.Exec(`
		INSERT INTO orders (
			id, user_id, customer_name, phone_hash, phone_masked, customer_phone,
			address_text, lat, lng, distance_km, delivery_fee, subtotal, total,
			status, created_at, amount_paid
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		id, userID, customerName, "uid:"+userID, "090***1111", "",
		"1 Le Loi", 10.78, 106.70, 2.5, 0, total, total,
		status, createdAt, amountPaid,
	)
	if err != nil {
		t.Fatal(err)
	}
}

func getCustomerStats(t *testing.T, r http.Handler, query string) []customerStat {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/v1/admin/orders/customers"+query, nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var payload struct {
		Customers []customerStat `json:"customers"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	return payload.Customers
}

// TestCustomerStatsGroupsByUser is the shop owner's question — "khách nào đã
// đặt bao nhiêu đơn" — answered from the orders table.
func TestCustomerStatsGroupsByUser(t *testing.T) {
	svc, r := testOrderRouter(t, &stubGeo{}, &stubCatalog{})

	insertTestOrderMoney(t, svc, "o1", "user-a", "Chi Lan", orderStatusCompleted, "2026-08-01T03:00:00Z", 300000, 300000)
	insertTestOrderMoney(t, svc, "o2", "user-a", "Chi Lan", orderStatusCompleted, "2026-08-02T03:00:00Z", 200000, 50000)
	insertTestOrderMoney(t, svc, "o3", "user-a", "Chi Lan", orderStatusCancelled, "2026-08-03T03:00:00Z", 100000, 0)
	insertTestOrderMoney(t, svc, "o4", "user-a", "Chi Lan", orderStatusPending, "2026-08-04T03:00:00Z", 100000, 0)
	insertTestOrderMoney(t, svc, "o5", "user-b", "Anh Tam", orderStatusCompleted, "2026-08-02T03:00:00Z", 120000, 120000)

	got := getCustomerStats(t, r, "?from=2026-08-01&to=2026-08-31")
	if len(got) != 2 {
		t.Fatalf("want 2 customers, got %d", len(got))
	}

	// Sorted by spend: user-a (500k completed) before user-b (120k).
	a := got[0]
	if a.UserID != "user-a" {
		t.Fatalf("first customer=%q, want user-a", a.UserID)
	}
	if a.OrdersTotal != 4 {
		t.Fatalf("orders_total=%d, want 4", a.OrdersTotal)
	}
	if a.OrdersCompleted != 2 || a.OrdersCancelled != 1 || a.OrdersPending != 1 {
		t.Fatalf("counts completed=%d cancelled=%d pending=%d",
			a.OrdersCompleted, a.OrdersCancelled, a.OrdersPending)
	}
	// Cancelled and pending orders must not add revenue.
	if a.SpentVnd != 500000 {
		t.Fatalf("spent=%d, want 500000", a.SpentVnd)
	}
	if a.PaidVnd != 350000 {
		t.Fatalf("paid=%d, want 350000", a.PaidVnd)
	}
	if a.DebtVnd != 150000 {
		t.Fatalf("debt=%d, want 150000", a.DebtVnd)
	}
	if a.FirstOrderAt != "2026-08-01T03:00:00Z" {
		t.Fatalf("first_order_at=%q", a.FirstOrderAt)
	}
	if a.LastOrderAt != "2026-08-04T03:00:00Z" {
		t.Fatalf("last_order_at=%q", a.LastOrderAt)
	}
	if got[1].UserID != "user-b" {
		t.Fatalf("second customer=%q, want user-b", got[1].UserID)
	}
}

// TestCustomerStatsRangeIsVietnamDays guards the timezone the shop actually
// counts in: 2026-08-01T18:00Z is already 2026-08-02 in Vietnam, so a range
// ending 2026-08-01 must not include it — otherwise the customer list and
// report-service's daily_stats would disagree about which day an order landed on.
func TestCustomerStatsRangeIsVietnamDays(t *testing.T) {
	svc, r := testOrderRouter(t, &stubGeo{}, &stubCatalog{})

	insertTestOrderMoney(t, svc, "early", "user-a", "A", orderStatusCompleted, "2026-08-01T02:00:00Z", 100000, 100000)
	insertTestOrderMoney(t, svc, "late", "user-b", "B", orderStatusCompleted, "2026-08-01T18:00:00Z", 100000, 100000)

	got := getCustomerStats(t, r, "?from=2026-08-01&to=2026-08-01")
	if len(got) != 1 || got[0].UserID != "user-a" {
		t.Fatalf("want only user-a on 2026-08-01, got %+v", got)
	}

	got = getCustomerStats(t, r, "?from=2026-08-02&to=2026-08-02")
	if len(got) != 1 || got[0].UserID != "user-b" {
		t.Fatalf("want only user-b on 2026-08-02, got %+v", got)
	}
}

// TestCustomerStatsUsesLatestOrderIdentity — a customer who moved should be
// listed under the address and name of their most recent order.
func TestCustomerStatsUsesLatestOrderIdentity(t *testing.T) {
	svc, r := testOrderRouter(t, &stubGeo{}, &stubCatalog{})

	insertTestOrderMoney(t, svc, "old", "user-a", "Ten Cu", orderStatusCompleted, "2026-08-01T03:00:00Z", 100000, 100000)
	insertTestOrderMoney(t, svc, "new", "user-a", "Ten Moi", orderStatusCompleted, "2026-08-05T03:00:00Z", 100000, 100000)

	got := getCustomerStats(t, r, "?from=2026-08-01&to=2026-08-31")
	if len(got) != 1 {
		t.Fatalf("want 1 customer, got %d", len(got))
	}
	if got[0].CustomerName != "Ten Moi" {
		t.Fatalf("customer_name=%q, want Ten Moi", got[0].CustomerName)
	}
}

func TestCustomerStatsRejectsBadRange(t *testing.T) {
	_, r := testOrderRouter(t, &stubGeo{}, &stubCatalog{})

	for _, query := range []string{
		"?from=2026-08-05&to=2026-08-01", // reversed
		"?from=nonsense&to=2026-08-01",
		"?from=2026-08-01", // to missing
		"?limit=0",
	} {
		req := httptest.NewRequest(http.MethodGet, "/v1/admin/orders/customers"+query, nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("query %q: status=%d, want 400", query, rr.Code)
		}
	}
}
