package natsx

import (
	"fmt"
	"time"

	"github.com/nats-io/nats.go"
)

// Connect opens a NATS connection (core). Prefer ConnectJS for JetStream work.
func Connect(url string) (*nats.Conn, error) {
	if url == "" {
		url = nats.DefaultURL
	}
	nc, err := nats.Connect(url,
		nats.Name("gas-tam-de"),
		nats.Timeout(5*time.Second),
		nats.MaxReconnects(-1),
	)
	if err != nil {
		return nil, fmt.Errorf("nats connect: %w", err)
	}
	return nc, nil
}

// ConnectJS opens NATS and returns a JetStream context.
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

// EnsureStreams creates or updates domain streams (idempotent).
func EnsureStreams(js nats.JetStreamContext) error {
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
