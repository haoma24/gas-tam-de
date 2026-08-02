package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"gas-tam-de/pkg/events"
	"gas-tam-de/pkg/httpx"
)

func testReportRouter(t *testing.T) (*reportService, http.Handler) {
	t.Helper()
	svc := &reportService{db: openReportTestDB(t)}
	r := httpx.NewRouter("report-test")
	r.Get("/v1/admin/dashboard/summary", svc.handleDashboardSummary)
	return svc, r
}

func getDashboardSummary(h http.Handler, rawQuery string) *httptest.ResponseRecorder {
	u := "/v1/admin/dashboard/summary"
	if rawQuery != "" {
		u += "?" + rawQuery
	}
	req := httptest.NewRequest(http.MethodGet, u, nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	return rr
}

func seedDailyStats(t *testing.T, svc *reportService, day string, delta DailyStatsDelta) {
	t.Helper()
	tx, err := svc.db.Begin()
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	defer func() { _ = tx.Rollback() }()
	if err := applyDailyStatsDeltaTx(tx, day, delta); err != nil {
		t.Fatalf("seed daily_stats: %v", err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatalf("commit: %v", err)
	}
}

func TestDashboardSummary_DefaultTodayEmpty(t *testing.T) {
	_, h := testReportRouter(t)
	rr := getDashboardSummary(h, "")
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var got dashboardSummaryResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	today := DayKeyVN(time.Now().UTC())
	if got.From != today || got.To != today {
		t.Fatalf("range=%s..%s want today=%s", got.From, got.To, today)
	}
	if got.Timezone != dashboardTimezone {
		t.Fatalf("timezone=%q", got.Timezone)
	}
	if got.RevenueVnd != 0 || got.ProfitVnd != 0 || got.OrdersCompleted != 0 || got.DebtTotal != 0 {
		t.Fatalf("expected zeros, got=%+v", got)
	}
}

func TestDashboardSummary_DayAndRange(t *testing.T) {
	svc, h := testReportRouter(t)
	seedDailyStats(t, svc, "2026-08-01", DailyStatsDelta{
		RevenueVnd: 100_000, CogsVnd: 40_000, DeliveryFeeVnd: 10_000, OrdersCompleted: 1, OrdersPlaced: 2,
	})
	seedDailyStats(t, svc, "2026-08-02", DailyStatsDelta{
		RevenueVnd: 200_000, CogsVnd: 50_000, DeliveryFeeVnd: 20_000, OrdersCompleted: 2, OrdersPlaced: 3,
	})
	seedDailyStats(t, svc, "2026-08-03", DailyStatsDelta{
		RevenueVnd: 50_000, CogsVnd: 10_000, DeliveryFeeVnd: 5_000, OrdersCompleted: 1, OrdersPlaced: 1,
	})

	rr := getDashboardSummary(h, "day=2026-08-02")
	if rr.Code != http.StatusOK {
		t.Fatalf("day status=%d body=%s", rr.Code, rr.Body.String())
	}
	var day dashboardSummaryResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &day); err != nil {
		t.Fatal(err)
	}
	if day.From != "2026-08-02" || day.To != "2026-08-02" {
		t.Fatalf("day range=%+v", day)
	}
	if day.RevenueVnd != 200_000 || day.CogsVnd != 50_000 || day.DeliveryFeeVnd != 20_000 {
		t.Fatalf("day money=%+v", day)
	}
	if day.ProfitVnd != 150_000 || day.OrdersCompleted != 2 || day.OrdersPlaced != 3 {
		t.Fatalf("day profit/orders=%+v", day)
	}

	rr = getDashboardSummary(h, "from=2026-08-01&to=2026-08-02")
	if rr.Code != http.StatusOK {
		t.Fatalf("range status=%d body=%s", rr.Code, rr.Body.String())
	}
	var rangeResp dashboardSummaryResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &rangeResp); err != nil {
		t.Fatal(err)
	}
	if rangeResp.From != "2026-08-01" || rangeResp.To != "2026-08-02" {
		t.Fatalf("range bounds=%+v", rangeResp)
	}
	if rangeResp.RevenueVnd != 300_000 || rangeResp.CogsVnd != 90_000 || rangeResp.DeliveryFeeVnd != 30_000 {
		t.Fatalf("range money=%+v", rangeResp)
	}
	if rangeResp.ProfitVnd != 210_000 || rangeResp.OrdersCompleted != 3 || rangeResp.OrdersPlaced != 5 {
		t.Fatalf("range profit/orders=%+v", rangeResp)
	}
}

func TestDashboardSummary_BadQuery(t *testing.T) {
	_, h := testReportRouter(t)

	cases := []struct {
		q string
	}{
		{"day=not-a-date"},
		{"from=2026-08-02&to=2026-08-01"},
		{"from=2026-08-01"},
		{"to=2026-08-01"},
		{"day=2026-08-01&from=2026-08-01&to=2026-08-01"},
	}
	for _, tc := range cases {
		rr := getDashboardSummary(h, tc.q)
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("q=%q status=%d body=%s", tc.q, rr.Code, rr.Body.String())
		}
	}
}

func TestDashboardSummary_IncludesDebtTotal(t *testing.T) {
	svc, h := testReportRouter(t)
	if err := svc.applyBillingDebtUpdated("evt-debt-1", "uid:a", 350_000); err != nil {
		t.Fatalf("debt a: %v", err)
	}
	if err := svc.applyBillingDebtUpdated("evt-debt-2", "uid:b", 100_000); err != nil {
		t.Fatalf("debt b: %v", err)
	}
	// FULL settlement → balance 0 should drop from outstanding sum
	if err := svc.applyBillingDebtUpdated("evt-debt-3", "uid:b", 0); err != nil {
		t.Fatalf("debt b clear: %v", err)
	}

	rr := getDashboardSummary(h, "day=2026-08-02")
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var got dashboardSummaryResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.DebtTotal != 350_000 {
		t.Fatalf("debt_total=%d want 350000", got.DebtTotal)
	}
}

func TestApplyBillingDebtUpdated_Idempotent(t *testing.T) {
	svc := &reportService{db: openReportTestDB(t)}
	if err := svc.applyBillingDebtUpdated("evt-same", "uid:x", 200_000); err != nil {
		t.Fatalf("first: %v", err)
	}
	if err := svc.applyBillingDebtUpdated("evt-same", "uid:x", 200_000); err != nil {
		t.Fatalf("dup: %v", err)
	}
	total, err := loadDebtTotal(svc.db)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if total != 200_000 {
		t.Fatalf("debt_total=%d want 200000", total)
	}
}

func TestHandleBillingDebtUpdatedMsg(t *testing.T) {
	svc := &reportService{db: openReportTestDB(t)}
	env := events.NewEnvelope(events.BillingDebtUpdated, "evt-msg-1", map[string]any{
		"customer_key": "uid:z",
		"balance":      99_000,
	})
	raw, err := json.Marshal(env)
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.handleBillingDebtUpdatedMsg(raw); err != nil {
		t.Fatalf("handle: %v", err)
	}
	total, err := loadDebtTotal(svc.db)
	if err != nil {
		t.Fatal(err)
	}
	if total != 99_000 {
		t.Fatalf("debt_total=%d", total)
	}
}
