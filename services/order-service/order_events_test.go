package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"gas-tam-de/pkg/events"
	"gas-tam-de/pkg/natsx"

	"github.com/nats-io/nats-server/v2/server"
)

type recordingOrderPublisher struct {
	placed    []orderPlacedEvent
	completed []orderCompletedEvent
}

func (r *recordingOrderPublisher) PublishOrderPlaced(e orderPlacedEvent) error {
	r.placed = append(r.placed, e)
	return nil
}

func (r *recordingOrderPublisher) PublishOrderCompleted(e orderCompletedEvent) error {
	r.completed = append(r.completed, e)
	return nil
}

func (r *recordingOrderPublisher) PublishOrderCancelled(e orderCancelledEvent) error {
	return nil
}

func TestCreateOrderPublishesOrderPlaced(t *testing.T) {
	geo := &stubGeo{result: geoCheckResult{DistanceKm: 3.2, InRange: true, MaxRadiusKm: 10}}
	catalog := &stubCatalog{products: []catalogProduct{
		{ID: "p1", SKU: "GAS12", Name: "Gas 12kg", SalePrice: 450000, Active: true},
	}}
	bus := &recordingOrderPublisher{}
	_, r := testOrderRouterWithBus(t, geo, catalog, bus)

	req := httptest.NewRequest(http.MethodPost, "/v1/orders", bytes.NewReader(validBody("p1")))
	customerHeaders(req)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var out orderView
	if err := json.Unmarshal(rr.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	if len(bus.placed) != 1 {
		t.Fatalf("placed=%d want 1", len(bus.placed))
	}
	got := bus.placed[0]
	if got.OrderID != out.ID {
		t.Fatalf("order_id=%q want %q", got.OrderID, out.ID)
	}
	if got.Total != 900000 || got.DistanceKm != 3.2 || got.CreatedAt != out.CreatedAt {
		t.Fatalf("event=%+v out.created_at=%s", got, out.CreatedAt)
	}
	if len(bus.completed) != 0 {
		t.Fatalf("completed=%d want 0", len(bus.completed))
	}
}

func TestCreateOrderOutOfRangeDoesNotPublish(t *testing.T) {
	geo := &stubGeo{result: geoCheckResult{DistanceKm: 12.5, InRange: false, MaxRadiusKm: 10}}
	catalog := &stubCatalog{products: []catalogProduct{
		{ID: "p1", SKU: "GAS12", Name: "Gas 12kg", SalePrice: 450000, Active: true},
	}}
	bus := &recordingOrderPublisher{}
	_, r := testOrderRouterWithBus(t, geo, catalog, bus)

	req := httptest.NewRequest(http.MethodPost, "/v1/orders", bytes.NewReader(validBody("p1")))
	customerHeaders(req)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status=%d", rr.Code)
	}
	if len(bus.placed) != 0 {
		t.Fatalf("placed=%d want 0", len(bus.placed))
	}
}

func TestCompleteOrderPublishesOrderCompleted(t *testing.T) {
	bus := &recordingOrderPublisher{}
	svc := &orderService{
		db:      openTestOrderDB(t),
		geo:     &stubGeo{},
		catalog: &stubCatalog{},
		billing: &stubBillingRecorder{},
		bus:     bus,
	}
	r := mountOrderTestRoutes(svc)
	insertTestOrderWithTotal(t, svc, "ord-evt", "u1", "PENDING", "2026-08-02T09:00:00Z", 450000)

	rr := postComplete(r, "ord-evt", completeBody("PARTIAL", ptrInt64(100000)))
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	if len(bus.completed) != 1 {
		t.Fatalf("completed=%d want 1", len(bus.completed))
	}
	got := bus.completed[0]
	if got.OrderID != "ord-evt" || got.Total != 450000 || got.PaymentType != paymentPartial || got.AmountPaid != 100000 {
		t.Fatalf("event=%+v", got)
	}
	if len(got.Items) != 1 || got.Items[0].ProductID == "" || got.Items[0].Qty < 1 {
		t.Fatalf("items=%+v", got.Items)
	}
	if len(bus.placed) != 0 {
		t.Fatalf("placed=%d want 0", len(bus.placed))
	}
}

func TestJSOrderPublisherPublishesEnvelope(t *testing.T) {
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

	sub, err := js.SubscribeSync(events.OrderPlaced)
	if err != nil {
		t.Fatalf("SubscribeSync: %v", err)
	}
	defer func() { _ = sub.Unsubscribe() }()

	pub := newJSOrderPublisher(js)
	e := orderPlacedEvent{
		OrderID:    "ord-1",
		Total:      900000,
		DistanceKm: 3.25,
		CreatedAt:  "2026-08-02T02:00:00Z",
	}
	if err := pub.PublishOrderPlaced(e); err != nil {
		t.Fatalf("PublishOrderPlaced: %v", err)
	}

	msg, err := sub.NextMsg(2 * time.Second)
	if err != nil {
		t.Fatalf("NextMsg: %v", err)
	}
	var env events.Envelope
	if err := json.Unmarshal(msg.Data, &env); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if env.Subject != events.OrderPlaced {
		t.Fatalf("subject=%q", env.Subject)
	}
	if env.EventID == "" || env.SchemaVersion != 1 {
		t.Fatalf("envelope=%+v", env)
	}
	if env.Payload["order_id"] != "ord-1" {
		t.Fatalf("payload=%v", env.Payload)
	}
	if total, ok := env.Payload["total"].(float64); !ok || int64(total) != 900000 {
		t.Fatalf("total=%v", env.Payload["total"])
	}
	if dist, ok := env.Payload["distance_km"].(float64); !ok || dist != 3.25 {
		t.Fatalf("distance_km=%v", env.Payload["distance_km"])
	}
	if env.Payload["created_at"] != "2026-08-02T02:00:00Z" {
		t.Fatalf("created_at=%v", env.Payload["created_at"])
	}
	info, err := js.StreamInfo("ORDERS")
	if err != nil {
		t.Fatalf("StreamInfo: %v", err)
	}
	if info.State.Msgs < 1 {
		t.Fatalf("ORDERS msgs=%d want >=1", info.State.Msgs)
	}
}

func TestJSOrderPublisherPublishesOrderCompleted(t *testing.T) {
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

	sub, err := js.SubscribeSync(events.OrderCompleted)
	if err != nil {
		t.Fatalf("SubscribeSync: %v", err)
	}
	defer func() { _ = sub.Unsubscribe() }()

	pub := newJSOrderPublisher(js)
	e := orderCompletedEvent{
		OrderID: "ord-done",
		Items: []orderItemView{
			{ProductID: "p1", ProductSKU: "GAS12", Qty: 2, UnitPrice: 450000},
		},
		Total:       900000,
		PaymentType: paymentPartial,
		AmountPaid:  100000,
	}
	if err := pub.PublishOrderCompleted(e); err != nil {
		t.Fatalf("PublishOrderCompleted: %v", err)
	}

	msg, err := sub.NextMsg(2 * time.Second)
	if err != nil {
		t.Fatalf("NextMsg: %v", err)
	}
	var env events.Envelope
	if err := json.Unmarshal(msg.Data, &env); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if env.Subject != events.OrderCompleted {
		t.Fatalf("subject=%q", env.Subject)
	}
	if env.Payload["order_id"] != "ord-done" {
		t.Fatalf("payload=%v", env.Payload)
	}
	if env.Payload["payment_type"] != paymentPartial {
		t.Fatalf("payment_type=%v", env.Payload["payment_type"])
	}
	if paid, ok := env.Payload["amount_paid"].(float64); !ok || int64(paid) != 100000 {
		t.Fatalf("amount_paid=%v", env.Payload["amount_paid"])
	}
	items, ok := env.Payload["items"].([]any)
	if !ok || len(items) != 1 {
		t.Fatalf("items=%v", env.Payload["items"])
	}
}
