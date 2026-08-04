package natsx

import (
	"errors"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
)

// reserveFreePort returns a port nothing is listening on yet, so a test can
// point a client at a broker that starts later.
func reserveFreePort(t *testing.T) int {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("reserve port: %v", err)
	}
	port := ln.Addr().(*net.TCPAddr).Port
	if err := ln.Close(); err != nil {
		t.Fatalf("release port: %v", err)
	}
	return port
}

func startBroker(t *testing.T, port int) *server.Server {
	t.Helper()
	ns, err := server.NewServer(&server.Options{
		Host:      "127.0.0.1",
		Port:      port,
		JetStream: true,
		StoreDir:  t.TempDir(),
	})
	if err != nil {
		t.Fatalf("new server: %v", err)
	}
	go ns.Start()
	if !ns.ReadyForConnections(10 * time.Second) {
		t.Fatal("broker not ready for connections")
	}
	return ns
}

func TestBackgroundNotReadyBeforeBrokerArrives(t *testing.T) {
	t.Setenv("NATS_STARTUP_TIMEOUT_SEC", "0")

	port := reserveFreePort(t)
	b := NewBackground(fmt.Sprintf("nats://127.0.0.1:%d", port))
	b.Start(nil)
	defer b.Close()

	if _, err := b.JS(); err == nil {
		t.Fatal("JS() should fail while the broker is unreachable")
	}
	if b.Ready() {
		t.Fatal("Ready() should be false while the broker is unreachable")
	}
}

// The whole point of Background: a service keeps serving HTTP while NATS is
// down, then attaches consumers once the broker shows up.
func TestBackgroundBecomesReadyWhenBrokerStartsLate(t *testing.T) {
	t.Setenv("NATS_STARTUP_TIMEOUT_SEC", "0")

	port := reserveFreePort(t)
	b := NewBackground(fmt.Sprintf("nats://127.0.0.1:%d", port))

	onReadyRan := make(chan struct{})
	b.Start(func(js nats.JetStreamContext) error {
		if js == nil {
			return errors.New("nil jetstream context")
		}
		close(onReadyRan)
		return nil
	})
	defer b.Close()

	if b.Ready() {
		t.Fatal("Ready() should be false before the broker starts")
	}

	time.Sleep(300 * time.Millisecond)
	ns := startBroker(t, port)
	defer ns.Shutdown()

	select {
	case <-onReadyRan:
	case <-time.After(60 * time.Second):
		t.Fatalf("onReady never ran after the broker started: %v", b.Err())
	}

	deadline := time.Now().Add(10 * time.Second)
	for !b.Ready() {
		if time.Now().After(deadline) {
			t.Fatalf("provider never became ready: %v", b.Err())
		}
		time.Sleep(50 * time.Millisecond)
	}
	if _, err := b.JS(); err != nil {
		t.Fatalf("JS() after ready: %v", err)
	}
	if err := b.Err(); err != nil {
		t.Fatalf("Err() after ready: %v", err)
	}
}

func TestStaticProvider(t *testing.T) {
	if _, err := Static(nil).JS(); !errors.Is(err, ErrNotReady) {
		t.Fatalf("Static(nil).JS() = %v, want ErrNotReady", err)
	}

	ns := startBroker(t, reserveFreePort(t))
	defer ns.Shutdown()

	nc, js, err := ConnectJS(ns.ClientURL())
	if err != nil {
		t.Fatalf("ConnectJS: %v", err)
	}
	defer nc.Close()

	got, err := Static(js).JS()
	if err != nil || got == nil {
		t.Fatalf("Static(js).JS() = %v, %v", got, err)
	}
}
