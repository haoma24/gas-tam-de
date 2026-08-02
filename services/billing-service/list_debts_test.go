package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"gas-tam-de/pkg/httpx"
)

func testBillingRouterWithDebts(t *testing.T) (*billingService, http.Handler) {
	t.Helper()
	svc := &billingService{db: openTestBillingDB(t), bus: noopBillingPublisher{}}
	r := httpx.NewRouter("billing-test")
	r.Post("/v1/internal/payments", svc.handleRecordPayment)
	r.Get("/v1/admin/debts", svc.handleListDebts)
	return svc, r
}

func getAdminDebts(h http.Handler) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, "/v1/admin/debts", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	return rr
}

func TestListDebtsEmpty(t *testing.T) {
	_, h := testBillingRouterWithDebts(t)
	rr := getAdminDebts(h)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var got listDebtsResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.Count != 0 || got.TotalBalance != 0 || got.Items == nil || len(got.Items) != 0 {
		t.Fatalf("got=%+v", got)
	}
}

func TestListDebtsAggregateAndOrder(t *testing.T) {
	_, h := testBillingRouterWithDebts(t)

	// Customer A: PARTIAL → 350k debt
	if rr := postRecordPayment(h, recordPaymentInput{
		OrderID:     "ord-a1",
		CustomerKey: "uid:a",
		PhoneMasked: "090***1111",
		PaymentType: paymentPartial,
		AmountDue:   450000,
		AmountPaid:  100000,
		RecordedBy:  "admin",
	}); rr.Code != http.StatusOK {
		t.Fatalf("a1 status=%d body=%s", rr.Code, rr.Body.String())
	}

	// Customer B: UNPAID → 200k debt (lower than A)
	if rr := postRecordPayment(h, recordPaymentInput{
		OrderID:     "ord-b1",
		CustomerKey: "uid:b",
		PhoneMasked: "091***2222",
		PaymentType: paymentUnpaid,
		AmountDue:   200000,
		AmountPaid:  0,
		RecordedBy:  "admin",
	}); rr.Code != http.StatusOK {
		t.Fatalf("b1 status=%d body=%s", rr.Code, rr.Body.String())
	}

	// Customer A accumulates: another UNPAID 50k → balance 400k
	if rr := postRecordPayment(h, recordPaymentInput{
		OrderID:     "ord-a2",
		CustomerKey: "uid:a",
		PhoneMasked: "090***1111",
		PaymentType: paymentUnpaid,
		AmountDue:   50000,
		AmountPaid:  0,
		RecordedBy:  "admin",
	}); rr.Code != http.StatusOK {
		t.Fatalf("a2 status=%d body=%s", rr.Code, rr.Body.String())
	}

	// FULL creates no debt row / no balance bump
	if rr := postRecordPayment(h, recordPaymentInput{
		OrderID:     "ord-c1",
		CustomerKey: "uid:c",
		PhoneMasked: "092***3333",
		PaymentType: paymentFull,
		AmountDue:   100000,
		AmountPaid:  100000,
		RecordedBy:  "admin",
	}); rr.Code != http.StatusOK {
		t.Fatalf("c1 status=%d body=%s", rr.Code, rr.Body.String())
	}

	rr := getAdminDebts(h)
	if rr.Code != http.StatusOK {
		t.Fatalf("list status=%d body=%s", rr.Code, rr.Body.String())
	}
	var got listDebtsResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.Count != 2 || got.TotalBalance != 600000 {
		t.Fatalf("count=%d total=%d want 2 / 600000", got.Count, got.TotalBalance)
	}
	if len(got.Items) != 2 {
		t.Fatalf("items=%+v", got.Items)
	}
	// Highest balance first
	if got.Items[0].CustomerKey != "uid:a" || got.Items[0].Balance != 400000 {
		t.Fatalf("first=%+v", got.Items[0])
	}
	if got.Items[0].PhoneMasked != "090***1111" {
		t.Fatalf("phone_masked=%s", got.Items[0].PhoneMasked)
	}
	if got.Items[1].CustomerKey != "uid:b" || got.Items[1].Balance != 200000 {
		t.Fatalf("second=%+v", got.Items[1])
	}
	if _, err := time.Parse(time.RFC3339, got.Items[0].UpdatedAt); err != nil {
		t.Fatalf("updated_at=%q err=%v", got.Items[0].UpdatedAt, err)
	}
}

func TestListDebtsOmitsZeroBalance(t *testing.T) {
	svc, h := testBillingRouterWithDebts(t)

	now := time.Now().UTC().Format(time.RFC3339)
	_, err := svc.db.Exec(`
		INSERT INTO debts (customer_key, phone_masked, balance, updated_at)
		VALUES (?, ?, ?, ?)`, "uid:zero", "093***0000", 0, now)
	if err != nil {
		t.Fatal(err)
	}
	_, err = svc.db.Exec(`
		INSERT INTO debts (customer_key, phone_masked, balance, updated_at)
		VALUES (?, ?, ?, ?)`, "uid:pos", "094***4444", 10000, now)
	if err != nil {
		t.Fatal(err)
	}

	rr := getAdminDebts(h)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var got listDebtsResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.Count != 1 || got.TotalBalance != 10000 || got.Items[0].CustomerKey != "uid:pos" {
		t.Fatalf("got=%+v", got)
	}
}
