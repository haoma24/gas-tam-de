package natsx

import (
	"encoding/json"
	"testing"
	"time"

	"gas-tam-de/pkg/events"

	"github.com/nats-io/nats-server/v2/server"
)

func TestPublishEnvelopeToCatalogStream(t *testing.T) {
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

	nc, js, err := ConnectJS(ns.ClientURL())
	if err != nil {
		t.Fatalf("ConnectJS: %v", err)
	}
	defer nc.Close()
	if err := EnsureStreams(js); err != nil {
		t.Fatalf("EnsureStreams: %v", err)
	}

	sub, err := js.SubscribeSync(events.CatalogProductUpdated)
	if err != nil {
		t.Fatalf("SubscribeSync: %v", err)
	}
	defer func() { _ = sub.Unsubscribe() }()

	env := events.NewEnvelope(events.CatalogProductUpdated, "evt-catalog-1", map[string]any{
		"product_id": "p1",
		"sku":        "GAS12",
		"sale_price": int64(450000),
		"active":     true,
	})
	ack, err := PublishEnvelope(js, env)
	if err != nil {
		t.Fatalf("PublishEnvelope: %v", err)
	}
	if ack.Stream != "CATALOG" {
		t.Fatalf("ack stream=%s want CATALOG", ack.Stream)
	}

	msg, err := sub.NextMsg(2 * time.Second)
	if err != nil {
		t.Fatalf("NextMsg: %v", err)
	}
	var got events.Envelope
	if err := json.Unmarshal(msg.Data, &got); err != nil {
		t.Fatal(err)
	}
	if got.EventID != "evt-catalog-1" || got.Subject != events.CatalogProductUpdated {
		t.Fatalf("got=%+v", got)
	}
}

func TestPublishEnvelopeValidation(t *testing.T) {
	if _, err := PublishEnvelope(nil, events.Envelope{Subject: "x", EventID: "y"}); err == nil {
		t.Fatal("expected nil js error")
	}
	// Need a real js for empty subject — use nil already covered; empty fields:
	opts := &server.Options{Port: -1, JetStream: true, StoreDir: t.TempDir()}
	ns, err := server.NewServer(opts)
	if err != nil {
		t.Fatal(err)
	}
	go ns.Start()
	if !ns.ReadyForConnections(5 * time.Second) {
		t.Fatal("not ready")
	}
	defer ns.Shutdown()
	nc, js, err := ConnectJS(ns.ClientURL())
	if err != nil {
		t.Fatal(err)
	}
	defer nc.Close()

	if _, err := PublishEnvelope(js, events.Envelope{EventID: "e1"}); err == nil {
		t.Fatal("expected empty subject error")
	}
	if _, err := PublishEnvelope(js, events.Envelope{Subject: "catalog.product.updated"}); err == nil {
		t.Fatal("expected empty event_id error")
	}
}
