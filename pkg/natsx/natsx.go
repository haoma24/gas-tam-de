package natsx

import (
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"time"

	"github.com/nats-io/nats.go"
)

// startupTimeout bounds how long Connect/ConnectJS keep retrying before giving
// up. In compose the broker can accept TCP before JetStream finishes recovering
// its store, so a single attempt at boot is not enough on a cold or slow host.
// Override with NATS_STARTUP_TIMEOUT_SEC (0 disables retrying).
func startupTimeout() time.Duration {
	if raw := os.Getenv("NATS_STARTUP_TIMEOUT_SEC"); raw != "" {
		if sec, err := strconv.Atoi(raw); err == nil && sec >= 0 {
			return time.Duration(sec) * time.Second
		}
	}
	return 60 * time.Second
}

// retryUntil calls fn until it succeeds or the budget runs out, backing off
// between attempts. It always runs fn at least once and returns the last error.
func retryUntil(budget time.Duration, what string, fn func() error) error {
	deadline := time.Now().Add(budget)
	backoff := 500 * time.Millisecond
	for attempt := 1; ; attempt++ {
		err := fn()
		if err == nil {
			return nil
		}
		if budget <= 0 {
			return err
		}
		if time.Now().After(deadline) {
			return err
		}
		slog.Warn("waiting for nats", "what", what, "attempt", attempt, "err", err, "retry_in", backoff)
		time.Sleep(backoff)
		if backoff < 5*time.Second {
			backoff *= 2
		}
	}
}

// Connect opens a NATS connection (core). Prefer ConnectJS for JetStream work.
// Retries until the broker accepts the connection (see startupTimeout).
func Connect(url string) (*nats.Conn, error) {
	if url == "" {
		url = nats.DefaultURL
	}
	var nc *nats.Conn
	err := retryUntil(startupTimeout(), "connect", func() error {
		var err error
		nc, err = nats.Connect(url,
			nats.Name("gas-tam-de"),
			nats.Timeout(5*time.Second),
			nats.MaxReconnects(-1),
		)
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("nats connect: %w", err)
	}
	return nc, nil
}

// ConnectJS opens NATS and returns a JetStream context, waiting until JetStream
// itself answers so callers do not have to treat a cold broker as fatal.
func ConnectJS(url string) (*nats.Conn, nats.JetStreamContext, error) {
	nc, err := Connect(url)
	if err != nil {
		return nil, nil, err
	}
	js, err := nc.JetStream()
	if err != nil {
		nc.Close()
		return nil, nil, fmt.Errorf("jetstream context: %w", err)
	}
	if err := retryUntil(startupTimeout(), "jetstream", func() error { return PingJS(js) }); err != nil {
		nc.Close()
		return nil, nil, err
	}
	return nc, js, nil
}

// StreamDef is a JetStream stream bound to one bounded-context subject prefix.
type StreamDef struct {
	Name     string
	Subjects []string
}

// DomainStreams are the local JetStream streams for Gas Tam Đệ events
// (architecture §5.1 subject naming).
func DomainStreams() []StreamDef {
	return []StreamDef{
		{Name: "AUTH", Subjects: []string{"auth.>"}},
		{Name: "CATALOG", Subjects: []string{"catalog.>"}},
		{Name: "GEO", Subjects: []string{"geo.>"}},
		{Name: "ORDERS", Subjects: []string{"order.>"}},
		{Name: "INVENTORY", Subjects: []string{"inventory.>"}},
		{Name: "BILLING", Subjects: []string{"billing.>"}},
	}
}

// EnsureStreams creates or updates domain streams (idempotent), retrying while
// JetStream is still coming up.
func EnsureStreams(js nats.JetStreamContext) error {
	return retryUntil(startupTimeout(), "ensure streams", func() error { return ensureStreamsOnce(js) })
}

func ensureStreamsOnce(js nats.JetStreamContext) error {
	for _, def := range DomainStreams() {
		cfg := &nats.StreamConfig{
			Name:       def.Name,
			Subjects:   def.Subjects,
			Storage:    nats.FileStorage,
			Retention:  nats.LimitsPolicy,
			Discard:    nats.DiscardOld,
			MaxAge:     7 * 24 * time.Hour,
			Duplicates: 2 * time.Minute,
		}
		if _, err := js.StreamInfo(def.Name); err == nats.ErrStreamNotFound {
			if _, err := js.AddStream(cfg); err != nil {
				return fmt.Errorf("add stream %s: %w", def.Name, err)
			}
			continue
		} else if err != nil {
			return fmt.Errorf("stream info %s: %w", def.Name, err)
		}
		if _, err := js.UpdateStream(cfg); err != nil {
			return fmt.Errorf("update stream %s: %w", def.Name, err)
		}
	}
	return nil
}

// PingJS verifies JetStream account info is reachable.
func PingJS(js nats.JetStreamContext) error {
	if _, err := js.AccountInfo(); err != nil {
		return fmt.Errorf("jetstream account info: %w", err)
	}
	return nil
}
