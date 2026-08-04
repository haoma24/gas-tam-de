package main

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"gas-tam-de/pkg/events"
	"gas-tam-de/pkg/natsx"

	"github.com/nats-io/nats-server/v2/server"
)

type recordingBillingPublisher struct {
	payments []struct {
		OrderID     string
		PaymentType string
		AmountPaid  int64
	}
	debts []struct {
		CustomerKey string
		Balance     int64
	}
}

func (r *recordingBillingPublisher) PublishPaymentRecorded(orderID, paymentType string, amountPaid int64) error {
	r.payments = append(r.payments, struct {
		OrderID     string
		PaymentType string
		AmountPaid  int64
	}{orderID, paymentType, amountPaid})
	return nil
}

func (r *recordingBillingPublisher) PublishDebtUpdated(customerKey string, balance int64) error {
	r.debts = append(r.debts, struct {
		CustomerKey string
		Balance     int64
	}{customerKey, balance})
	return nil
}

func TestRecordPaymentPublishesEvents(t *testing.T) {
	bus := &recordingBillingPublisher{}
	_, h := testBillingRouterWithBus(t, bus)
	rr := postRecordPayment(h, recordPaymentInput{
		OrderID:     "ord-evt",
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
	if len(bus.payments) != 1 {
		t.Fatalf("payments=%d want 1", len(bus.payments))
	}
	p := bus.payments[0]
	if p.OrderID != "ord-evt" || p.PaymentType != paymentPartial || p.AmountPaid != 100000 {
		t.Fatalf("payment event=%+v", p)
	}
	if len(bus.debts) != 1 {
		t.Fatalf("debts=%d want 1", len(bus.debts))
	}
	d := bus.debts[0]
	if d.CustomerKey != "uid:u1" || d.Balance != 350000 {
		t.Fatalf("debt event=%+v", d)
	}
}

func TestRecordPaymentIdempotentDoesNotRepublish(t *testing.T) {
	bus := &recordingBillingPublisher{}
	_, h := testBillingRouterWithBus(t, bus)
	body := recordPaymentInput{
		OrderID:     "ord-idem-evt",
		CustomerKey: "uid:u1",
		PhoneMasked: "090***1111",
		PaymentType: paymentPartial,
		AmountDue:   450000,
		AmountPaid:  100000,
		RecordedBy:  "admin-1",
	}
	if rr := postRecordPayment(h, body); rr.Code != http.StatusOK {
		t.Fatalf("first status=%d", rr.Code)
	}
	if rr := postRecordPayment(h, body); rr.Code != http.StatusOK {
		t.Fatalf("second status=%d", rr.Code)
	}
	if len(bus.payments) != 1 || len(bus.debts) != 1 {
		t.Fatalf("payments=%d debts=%d want 1 each", len(bus.payments), len(bus.debts))
	}
}

func TestRecordPaymentFullPublishesDebtUpdated(t *testing.T) {
	bus := &recordingBillingPublisher{}
	_, h := testBillingRouterWithBus(t, bus)
	rr := postRecordPayment(h, recordPaymentInput{
		OrderID:     "ord-full-evt",
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
	if len(bus.payments) != 1 || len(bus.debts) != 1 {
		t.Fatalf("payments=%d debts=%d", len(bus.payments), len(bus.debts))
	}
	if bus.debts[0].Balance != 0 {
		t.Fatalf("balance=%d want 0", bus.debts[0].Balance)
	}
}

func TestJSBillingPublisherPublishesEnvelopes(t *testing.T) {
	opts := &server.Options{
		Port:      -1,
		JetStream: true,
		StoreDir:  t.TempDir(),
	}
	ns, err := server.NewServer(opts)
	if err != nil {
		t.Fatalf("new server: %v", err)
	}
	go ns.Start()
	if !ns.ReadyForConnections(5 * time.Second) {
		t.Fatal("nats server not ready")
	}
	defer ns.Shutdown()

	nc, js, err := natsx.ConnectJS(ns.ClientURL())
	if err != nil {
		t.Fatalf("ConnectJS: %v", err)
	}
	defer nc.Close()
	if err := natsx.EnsureStreams(js); err != nil {
		t.Fatalf("EnsureStreams: %v", err)
	}

	paySub, err := js.SubscribeSync(events.BillingPaymentRecorded)
	if err != nil {
		t.Fatalf("SubscribeSync payment: %v", err)
	}
	defer func() { _ = paySub.Unsubscribe() }()
	debtSub, err := js.SubscribeSync(events.BillingDebtUpdated)
	if err != nil {
		t.Fatalf("SubscribeSync debt: %v", err)
	}
	defer func() { _ = debtSub.Unsubscribe() }()

	pub := newJSBillingPublisher(natsx.Static(js))
	if err := pub.PublishPaymentRecorded("ord-1", paymentPartial, 100000); err != nil {
		t.Fatalf("PublishPaymentRecorded: %v", err)
	}
	if err := pub.PublishDebtUpdated("uid:u1", 350000); err != nil {
		t.Fatalf("PublishDebtUpdated: %v", err)
	}

	payMsg, err := paySub.NextMsg(2 * time.Second)
	if err != nil {
		t.Fatalf("payment NextMsg: %v", err)
	}
	var payEnv events.Envelope
	if err := json.Unmarshal(payMsg.Data, &payEnv); err != nil {
		t.Fatalf("unmarshal payment: %v", err)
	}
	if payEnv.Subject != events.BillingPaymentRecorded {
		t.Fatalf("subject=%q", payEnv.Subject)
	}
	if payEnv.Payload["order_id"] != "ord-1" || payEnv.Payload["payment_type"] != paymentPartial {
		t.Fatalf("payload=%v", payEnv.Payload)
	}

	debtMsg, err := debtSub.NextMsg(2 * time.Second)
	if err != nil {
		t.Fatalf("debt NextMsg: %v", err)
	}
	var debtEnv events.Envelope
	if err := json.Unmarshal(debtMsg.Data, &debtEnv); err != nil {
		t.Fatalf("unmarshal debt: %v", err)
	}
	if debtEnv.Subject != events.BillingDebtUpdated {
		t.Fatalf("subject=%q", debtEnv.Subject)
	}
	if debtEnv.Payload["customer_key"] != "uid:u1" {
		t.Fatalf("payload=%v", debtEnv.Payload)
	}
	if bal, ok := debtEnv.Payload["balance"].(float64); !ok || int64(bal) != 350000 {
		t.Fatalf("balance=%v", debtEnv.Payload["balance"])
	}

	info, err := js.StreamInfo("BILLING")
	if err != nil {
		t.Fatalf("StreamInfo: %v", err)
	}
	if info.State.Msgs < 2 {
		t.Fatalf("BILLING msgs=%d want >=2", info.State.Msgs)
	}
}
