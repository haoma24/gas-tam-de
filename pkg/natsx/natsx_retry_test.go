package natsx

import (
	"errors"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/nats-io/nats-server/v2/server"
)

func TestRetryUntilRunsOnceWhenBudgetIsZero(t *testing.T) {
	calls := 0
	err := retryUntil(0, "test", func() error {
		calls++
		return errors.New("boom")
	})
	if err == nil {
		t.Fatal("want error")
	}
	if calls != 1 {
		t.Fatalf("calls=%d want 1", calls)
	}
}

func TestRetryUntilStopsAfterSuccess(t *testing.T) {
	calls := 0
	err := retryUntil(5*time.Second, "test", func() error {
		calls++
		if calls < 3 {
			return errors.New("not yet")
		}
		return nil
	})
	if err != nil {
		t.Fatalf("retryUntil: %v", err)
	}
	if calls != 3 {
		t.Fatalf("calls=%d want 3", calls)
	}
}

// A service must survive NATS becoming reachable only after it has already
// started, which is what made compose containers exit and report unhealthy.
func TestConnectJSWaitsForBrokerStartedLate(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("reserve port: %v", err)
	}
	port := ln.Addr().(*net.TCPAddr).Port
	if err := ln.Close(); err != nil {
		t.Fatalf("release port: %v", err)
	}
	url := fmt.Sprintf("nats://127.0.0.1:%d", port)

	started := make(chan *server.Server, 1)
	go func() {
		time.Sleep(1500 * time.Millisecond)
		ns, err := server.NewServer(&server.Options{
			Host:      "127.0.0.1",
			Port:      port,
			JetStream: true,
			StoreDir:  t.TempDir(),
		})
		if err != nil {
			started <- nil
			return
		}
		go ns.Start()
		started <- ns
	}()

	t.Setenv("NATS_STARTUP_TIMEOUT_SEC", "30")
	nc, js, err := ConnectJS(url)

	ns := <-started
	if ns != nil {
		defer ns.Shutdown()
	}
	if err != nil {
		t.Fatalf("ConnectJS should wait for a late broker: %v", err)
	}
	defer nc.Close()

	if err := EnsureStreams(js); err != nil {
		t.Fatalf("EnsureStreams: %v", err)
	}
}

func TestConnectFailsAfterBudgetWhenBrokerNeverArrives(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("reserve port: %v", err)
	}
	port := ln.Addr().(*net.TCPAddr).Port
	if err := ln.Close(); err != nil {
		t.Fatalf("release port: %v", err)
	}

	t.Setenv("NATS_STARTUP_TIMEOUT_SEC", "1")
	start := time.Now()
	if _, err := Connect(fmt.Sprintf("nats://127.0.0.1:%d", port)); err == nil {
		t.Fatal("want error when broker never arrives")
	}
	if elapsed := time.Since(start); elapsed > 20*time.Second {
		t.Fatalf("gave up after %v, expected to honour the 1s budget", elapsed)
	}
}
