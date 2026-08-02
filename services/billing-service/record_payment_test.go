package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"gas-tam-de/pkg/httpx"
	"gas-tam-de/pkg/sqlite"
)

func openTestBillingDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sqlite.Open(filepath.Join(t.TempDir(), "billing.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := migrate(db); err != nil {
		t.Fatal(err)
	}
	return db
}

func testBillingRouter(t *testing.T) (*billingService, http.Handler) {
	t.Helper()
	svc := &billingService{db: openTestBillingDB(t), bus: noopBillingPublisher{}}
	r := httpx.NewRouter("billing-test")
	r.Post("/v1/internal/payments", svc.handleRecordPayment)
	return svc, r
}

func testBillingRouterWithBus(t *testing.T, bus billingEventPublisher) (*billingService, http.Handler) {
	t.Helper()
	svc := &billingService{db: openTestBillingDB(t), bus: bus}
	r := httpx.NewRouter("billing-test")
	r.Post("/v1/internal/payments", svc.handleRecordPayment)
	return svc, r
}

func postRecordPayment(h http.Handler, body any) *httptest.ResponseRecorder {
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/v1/internal/payments", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	return rr
}

func TestRecordPaymentPartial(t *testing.T) {
	// AC — Hoàn tất + công nợ: PARTIAL paid=100_000, due=450_000 → debt delta 350_000.
	svc, h := testBillingRouter(t)
	rr := postRecordPayment(h, recordPaymentInput{
		OrderID:     "ord-partial",
		CustomerKey: "uid:u1",
		PhoneMasked: "090***1111",
		PaymentType: paymentPartial,
		AmountDue:   450000,
		AmountPaid:  100000,
		RecordedBy:  "admin-1",
	})
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var got recordPaymentResult
	if err := json.Unmarshal(rr.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.DebtDelta != 350000 || got.Balance != 350000 || got.PaymentID == "" {
		t.Fatalf("got=%+v", got)
	}

	var payType string
	var paid, due int64
	if err := svc.db.QueryRow(`
		SELECT payment_type, amount_paid, amount_due FROM payments WHERE order_id = ?`,
		"ord-partial").Scan(&payType, &paid, &due); err != nil {
		t.Fatal(err)
	}
	if payType != paymentPartial || paid != 100000 || due != 450000 {
		t.Fatalf("payment row type=%s paid=%d due=%d", payType, paid, due)
	}

	var bal int64
	if err := svc.db.QueryRow(`SELECT balance FROM debts WHERE customer_key = ?`, "uid:u1").Scan(&bal); err != nil {
		t.Fatal(err)
	}
	if bal != 350000 {
		t.Fatalf("balance=%d", bal)
	}

	var ledgerCount int
	if err := svc.db.QueryRow(`SELECT COUNT(*) FROM debt_ledger WHERE order_id = ?`, "ord-partial").Scan(&ledgerCount); err != nil {
		t.Fatal(err)
	}
	if ledgerCount != 1 {
		t.Fatalf("ledger count=%d", ledgerCount)
	}
}

func TestRecordPaymentFullNoDebt(t *testing.T) {
	svc, h := testBillingRouter(t)
	rr := postRecordPayment(h, recordPaymentInput{
		OrderID:     "ord-full",
		CustomerKey: "uid:u1",
		PhoneMasked: "090***1111",
		PaymentType: paymentFull,
		AmountDue:   450000,
		AmountPaid:  450000,
		RecordedBy:  "admin-1",
	})
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var got recordPaymentResult
	_ = json.Unmarshal(rr.Body.Bytes(), &got)
	if got.DebtDelta != 0 || got.Balance != 0 {
		t.Fatalf("got=%+v", got)
	}

	var n int
	if err := svc.db.QueryRow(`SELECT COUNT(*) FROM debts`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Fatalf("debts rows=%d want 0", n)
	}
	if err := svc.db.QueryRow(`SELECT COUNT(*) FROM debt_ledger`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Fatalf("ledger rows=%d want 0", n)
	}
}

func TestRecordPaymentUnpaid(t *testing.T) {
	_, h := testBillingRouter(t)
	rr := postRecordPayment(h, recordPaymentInput{
		OrderID:     "ord-unpaid",
		CustomerKey: "uid:u2",
		PhoneMasked: "091***2222",
		PaymentType: paymentUnpaid,
		AmountDue:   200000,
		AmountPaid:  0,
		RecordedBy:  "admin-1",
	})
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var got recordPaymentResult
	_ = json.Unmarshal(rr.Body.Bytes(), &got)
	if got.DebtDelta != 200000 || got.Balance != 200000 {
		t.Fatalf("got=%+v", got)
	}
}

func TestRecordPaymentAccumulatesDebt(t *testing.T) {
	svc, h := testBillingRouter(t)
	in := recordPaymentInput{
		OrderID:     "ord-a",
		CustomerKey: "uid:u1",
		PhoneMasked: "090***1111",
		PaymentType: paymentUnpaid,
		AmountDue:   100000,
		AmountPaid:  0,
		RecordedBy:  "admin-1",
	}
	if rr := postRecordPayment(h, in); rr.Code != http.StatusOK {
		t.Fatalf("first: %d %s", rr.Code, rr.Body.String())
	}
	in.OrderID = "ord-b"
	in.AmountDue = 50000
	if rr := postRecordPayment(h, in); rr.Code != http.StatusOK {
		t.Fatalf("second: %d %s", rr.Code, rr.Body.String())
	}
	var bal int64
	if err := svc.db.QueryRow(`SELECT balance FROM debts WHERE customer_key = ?`, "uid:u1").Scan(&bal); err != nil {
		t.Fatal(err)
	}
	if bal != 150000 {
		t.Fatalf("balance=%d", bal)
	}
}

func TestRecordPaymentIdempotent(t *testing.T) {
	svc, h := testBillingRouter(t)
	body := recordPaymentInput{
		OrderID:     "ord-idem",
		CustomerKey: "uid:u1",
		PhoneMasked: "090***1111",
		PaymentType: paymentPartial,
		AmountDue:   450000,
		AmountPaid:  100000,
		RecordedBy:  "admin-1",
	}
	rr1 := postRecordPayment(h, body)
	rr2 := postRecordPayment(h, body)
	if rr1.Code != http.StatusOK || rr2.Code != http.StatusOK {
		t.Fatalf("status1=%d status2=%d", rr1.Code, rr2.Code)
	}
	var a, b recordPaymentResult
	_ = json.Unmarshal(rr1.Body.Bytes(), &a)
	_ = json.Unmarshal(rr2.Body.Bytes(), &b)
	if !b.Idempotent || a.PaymentID != b.PaymentID {
		t.Fatalf("a=%+v b=%+v", a, b)
	}
	var n int
	if err := svc.db.QueryRow(`SELECT COUNT(*) FROM payments WHERE order_id = ?`, "ord-idem").Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("payments=%d", n)
	}
	var bal int64
	if err := svc.db.QueryRow(`SELECT balance FROM debts WHERE customer_key = ?`, "uid:u1").Scan(&bal); err != nil {
		t.Fatal(err)
	}
	if bal != 350000 {
		t.Fatalf("double-applied balance=%d", bal)
	}
}

func TestRecordPaymentValidation(t *testing.T) {
	_, h := testBillingRouter(t)
	cases := []recordPaymentInput{
		{OrderID: "", CustomerKey: "k", PhoneMasked: "m", PaymentType: paymentFull, AmountDue: 10, AmountPaid: 10},
		{OrderID: "o", CustomerKey: "k", PhoneMasked: "m", PaymentType: "X", AmountDue: 10, AmountPaid: 10},
		{OrderID: "o", CustomerKey: "k", PhoneMasked: "m", PaymentType: paymentPartial, AmountDue: 10, AmountPaid: 10},
		{OrderID: "o", CustomerKey: "k", PhoneMasked: "m", PaymentType: paymentFull, AmountDue: 10, AmountPaid: 5},
	}
	for i, body := range cases {
		rr := postRecordPayment(h, body)
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("case %d status=%d body=%s", i, rr.Code, rr.Body.String())
		}
	}
}
