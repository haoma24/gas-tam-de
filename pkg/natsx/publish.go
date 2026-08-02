package natsx

import (
	"encoding/json"
	"fmt"

	"gas-tam-de/pkg/events"

	"github.com/nats-io/nats.go"
)

// PublishEnvelope marshals an events.Envelope and publishes it to JetStream.
// MsgId is set to event_id so JetStream can dedupe within the stream Duplicates window.
func PublishEnvelope(js nats.JetStreamContext, env events.Envelope) (*nats.PubAck, error) {
	if js == nil {
		return nil, fmt.Errorf("jetstream context is nil")
	}
	if env.Subject == "" {
		return nil, fmt.Errorf("envelope subject is empty")
	}
	if env.EventID == "" {
		return nil, fmt.Errorf("envelope event_id is empty")
	}
	data, err := json.Marshal(env)
	if err != nil {
		return nil, fmt.Errorf("marshal envelope: %w", err)
	}
	ack, err := js.Publish(env.Subject, data, nats.MsgId(env.EventID))
	if err != nil {
		return nil, fmt.Errorf("publish %s: %w", env.Subject, err)
	}
	return ack, nil
}
