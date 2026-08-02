package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func completeBody(paymentType string, amountPaid *int64) []byte {
	m := map[string]any{"payment_type": paymentType}
	if amountPaid != nil {
		m["amount_paid"] = *amountPaid
	}
	b, _ := json.Marshal(m)
	return b
}

func ptrInt64(v int64) *int64 { return &v }

func postComplete(r http.Handler, id string, body []byte) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, "/v1/admin/orders/"+id+"/complete", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	return rr
}

func decodeComplete(t *testing.T, rr *httptest.ResponseRecorder) completeOrderView {
	t.Helper()
	var got completeOrderView
	if err := json.Unmarshal(rr.Body.Bytes(), &got); err != nil {
		t.Fatalf("json: %v body=%s", err, rr.Body.String())
	}
	return got
}

func TestCompleteOrderPartial(t *testing.T) {
	// AC — Hoàn tất + công nợ: PARTIAL paid=100_000, total=450_000 → debt=350_000.
	billing := &stubBillingRecorder{}
	svc, r := testOrderRouterWithBilling(t, &stubGeo{}, &stubCatalog{}, billing)
	insertTestOrderWithTotal(t, svc, "ord-partial", "u1", "PENDING", "2026-08-02T09:00:00Z", 450000)

	rr := postComplete(r, "ord-partial", completeBody("PARTIAL", ptrInt64(100000)))
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	got := decodeComplete(t, rr)
	if got.Status != "COMPLETED" {
		t.Fatalf("status=%q", got.Status)
	}
	if got.PaymentType != paymentPartial || got.AmountPaid != 100000 || got.AmountDue != 450000 || got.Debt != 350000 {
		t.Fatalf("payment=%+v", got)
	}
	if got.CompletedAt == "" {
		t.Fatal("missing completed_at")
	}
	if got.ID != "ord-partial" || len(got.Items) != 1 {
		t.Fatalf("order snapshot incomplete: %+v", got)
	}

	assertOrderCompleted(t, svc, "ord-partial", paymentPartial, 100000)

	if len(billing.calls) != 1 {
		t.Fatalf("billing calls=%d want 1", len(billing.calls))
	}
	c := billing.calls[0]
	if c.OrderID != "ord-partial" || c.CustomerKey != "uid:u1" || c.PaymentType != paymentPartial ||
		c.AmountDue != 450000 || c.AmountPaid != 100000 {
		t.Fatalf("billing call=%+v", c)
	}
}

func TestCompleteOrderFull(t *testing.T) {
	svc, r := testOrderRouter(t, &stubGeo{}, &stubCatalog{})
	insertTestOrderWithTotal(t, svc, "ord-full", "u1", "PENDING", "2026-08-02T09:00:00Z", 450000)

	rr := postComplete(r, "ord-full", completeBody("FULL", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	got := decodeComplete(t, rr)
	if got.PaymentType != paymentFull || got.AmountPaid != 450000 || got.Debt != 0 {
		t.Fatalf("payment=%+v", got)
	}
	assertOrderCompleted(t, svc, "ord-full", paymentFull, 450000)
}

func TestCompleteOrderUnpaid(t *testing.T) {
	svc, r := testOrderRouter(t, &stubGeo{}, &stubCatalog{})
	insertTestOrderWithTotal(t, svc, "ord-debt", "u1", "PENDING", "2026-08-02T09:00:00Z", 200000)

	rr := postComplete(r, "ord-debt", completeBody("UNPAID", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	got := decodeComplete(t, rr)
	if got.PaymentType != paymentUnpaid || got.AmountPaid != 0 || got.Debt != 200000 {
		t.Fatalf("payment=%+v", got)
	}
	assertOrderCompleted(t, svc, "ord-debt", paymentUnpaid, 0)
}

func TestCompleteOrderValidation(t *testing.T) {
	svc, r := testOrderRouter(t, &stubGeo{}, &stubCatalog{})
	insertTestOrderWithTotal(t, svc, "ord-v", "u1", "PENDING", "2026-08-02T09:00:00Z", 100000)

	cases := []struct {
		name string
		body []byte
	}{
		{"bad type", completeBody("DEBT", nil)},
		{"partial missing paid", completeBody("PARTIAL", nil)},
		{"partial zero", completeBody("PARTIAL", ptrInt64(0))},
		{"partial equal total", completeBody("PARTIAL", ptrInt64(100000))},
		{"full wrong paid", completeBody("FULL", ptrInt64(50))},
		{"unpaid nonzero", completeBody("UNPAID", ptrInt64(1))},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rr := postComplete(r, "ord-v", tc.body)
			if rr.Code != http.StatusBadRequest {
				t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
			}
		})
	}

	// Still PENDING after failed attempts.
	var status string
	if err := svc.db.QueryRow(`SELECT status FROM orders WHERE id = ?`, "ord-v").Scan(&status); err != nil {
		t.Fatal(err)
	}
	if status != "PENDING" {
		t.Fatalf("status=%q want PENDING", status)
	}
}

func TestCompleteOrderNotFound(t *testing.T) {
	_, r := testOrderRouter(t, &stubGeo{}, &stubCatalog{})
	rr := postComplete(r, "missing", completeBody("FULL", nil))
	if rr.Code != http.StatusNotFound {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestCompleteOrderAlreadyCompleted(t *testing.T) {
	svc, r := testOrderRouter(t, &stubGeo{}, &stubCatalog{})
	insertTestOrderWithTotal(t, svc, "ord-done", "u1", "PENDING", "2026-08-02T09:00:00Z", 100000)

	rr1 := postComplete(r, "ord-done", completeBody("FULL", nil))
	if rr1.Code != http.StatusOK {
		t.Fatalf("first: status=%d body=%s", rr1.Code, rr1.Body.String())
	}
	rr2 := postComplete(r, "ord-done", completeBody("UNPAID", nil))
	if rr2.Code != http.StatusConflict {
		t.Fatalf("second: status=%d body=%s", rr2.Code, rr2.Body.String())
	}
}

func TestCompleteOrderCancelled(t *testing.T) {
	svc, r := testOrderRouter(t, &stubGeo{}, &stubCatalog{})
	insertTestOrderWithTotal(t, svc, "ord-cx", "u1", "CANCELLED", "2026-08-02T09:00:00Z", 100000)

	rr := postComplete(r, "ord-cx", completeBody("FULL", nil))
	if rr.Code != http.StatusConflict {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestSettlePayment(t *testing.T) {
	type want struct {
		ptype string
		paid  int64
		debt  int64
		err   bool
	}
	cases := []struct {
		name string
		due  int64
		body completeOrderBody
		want want
	}{
		{"full omit", 100, completeOrderBody{PaymentType: "full"}, want{paymentFull, 100, 0, false}},
		{"partial", 450000, completeOrderBody{PaymentType: "PARTIAL", AmountPaid: ptrInt64(100000)}, want{paymentPartial, 100000, 350000, false}},
		{"unpaid", 50, completeOrderBody{PaymentType: "UNPAID"}, want{paymentUnpaid, 0, 50, false}},
		{"bad", 10, completeOrderBody{PaymentType: "X"}, want{err: true}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			pt, paid, debt, msg := settlePayment(tc.due, tc.body)
			if tc.want.err {
				if msg == "" {
					t.Fatal("expected error")
				}
				return
			}
			if msg != "" || pt != tc.want.ptype || paid != tc.want.paid || debt != tc.want.debt {
				t.Fatalf("got type=%q paid=%d debt=%d msg=%q", pt, paid, debt, msg)
			}
		})
	}
}

func insertTestOrderWithTotal(t *testing.T, svc *orderService, id, userID, status, createdAt string, total int64) {
	t.Helper()
	_, err := svc.db.Exec(`
		INSERT INTO orders (
			id, user_id, customer_name, phone_hash, phone_masked, address_text,
			lat, lng, distance_km, delivery_fee, subtotal, total, status, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		id, userID, "Khach "+id, "uid:"+userID, "090***1111", "1 Le Loi",
		10.78, 106.70, 2.5, 0, total, total, status, createdAt,
	)
	if err != nil {
		t.Fatal(err)
	}
	_, err = svc.db.Exec(`
		INSERT INTO order_items (
			id, order_id, product_id, product_sku, product_name, unit_price, qty, line_total
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		"item-"+id, id, "p1", "GAS12", "Gas 12kg", total, 1, total,
	)
	if err != nil {
		t.Fatal(err)
	}
}

func assertOrderCompleted(t *testing.T, svc *orderService, id, paymentType string, amountPaid int64) {
	t.Helper()
	var status, completedAt string
	var pt sql.NullString
	var paid sql.NullInt64
	err := svc.db.QueryRow(`
		SELECT status, completed_at, payment_type, amount_paid FROM orders WHERE id = ?`, id,
	).Scan(&status, &completedAt, &pt, &paid)
	if err != nil {
		t.Fatal(err)
	}
	if status != "COMPLETED" || completedAt == "" {
		t.Fatalf("status=%q completed_at=%q", status, completedAt)
	}
	if !pt.Valid || pt.String != paymentType {
		t.Fatalf("payment_type=%v want %s", pt, paymentType)
	}
	if !paid.Valid || paid.Int64 != amountPaid {
		t.Fatalf("amount_paid=%v want %d", paid, amountPaid)
	}
}

type stubBillingRecorder struct {
	calls []billingPaymentInput
	err   error
}

func (s *stubBillingRecorder) RecordPayment(_ context.Context, in billingPaymentInput) error {
	s.calls = append(s.calls, in)
	return s.err
}

func testOrderRouterWithBilling(t *testing.T, geo geoChecker, catalog productCatalog, billing billingRecorder) (*orderService, http.Handler) {
	t.Helper()
	svc := &orderService{
		db:      openTestOrderDB(t),
		geo:     geo,
		catalog: catalog,
		billing: billing,
		bus:     noopOrderPublisher{},
	}
	return svc, mountOrderTestRoutes(svc)
}
