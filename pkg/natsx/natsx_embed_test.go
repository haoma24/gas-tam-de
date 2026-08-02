package natsx

import (
	"testing"
	"time"

	"github.com/nats-io/nats-server/v2/server"
)

func TestEnsureStreamsWithEmbeddedJetStream(t *testing.T) {
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

	if err := PingJS(js); err != nil {
		t.Fatalf("PingJS: %v", err)
	}
	if err := EnsureStreams(js); err != nil {
		t.Fatalf("EnsureStreams: %v", err)
	}
	// Idempotent second pass
	if err := EnsureStreams(js); err != nil {
		t.Fatalf("EnsureStreams again: %v", err)
	}

	for _, def := range DomainStreams() {
		info, err := js.StreamInfo(def.Name)
		if err != nil {
			t.Fatalf("StreamInfo %s: %v", def.Name, err)
		}
		if info.Config.Subjects[0] != def.Subjects[0] {
			t.Fatalf("stream %s subject=%v want %v", def.Name, info.Config.Subjects, def.Subjects)
		}
	}

	// Smoke: publish retained by JetStream ORDERS stream
	ack, err := js.Publish("order.placed", []byte(`{"ok":true}`))
	if err != nil {
		t.Fatalf("publish order.placed: %v", err)
	}
	if ack.Stream != "ORDERS" {
		t.Fatalf("ack stream=%s want ORDERS", ack.Stream)
	}
	if _, err := js.Publish("no.such.subject", []byte("x")); err == nil {
		t.Fatal("expected publish to unmatched subject to fail")
	}
}
