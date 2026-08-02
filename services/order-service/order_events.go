package main

import (
	"fmt"
	"log/slog"

	"gas-tam-de/pkg/events"
	"gas-tam-de/pkg/natsx"

	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
)

// orderPlacedEvent is the JetStream payload for order.placed (architecture §5.1).
type orderPlacedEvent struct {
	OrderID    string
	Total      int64
	DistanceKm float64
	CreatedAt  string
}

// orderCompletedEvent is the JetStream payload for order.completed (architecture §5.1).
type orderCompletedEvent struct {
	OrderID     string
	Items       []orderItemView
	Total       int64
	PaymentType string
	AmountPaid  int64
}

// orderPublisher emits order.* events after successful commits.
type orderPublisher interface {
	PublishOrderPlaced(e orderPlacedEvent) error
	PublishOrderCompleted(e orderCompletedEvent) error
}

type noopOrderPublisher struct{}

func (noopOrderPublisher) PublishOrderPlaced(orderPlacedEvent) error       { return nil }
func (noopOrderPublisher) PublishOrderCompleted(orderCompletedEvent) error { return nil }

type jsOrderPublisher struct {
	js nats.JetStreamContext
}

func newJSOrderPublisher(js nats.JetStreamContext) *jsOrderPublisher {
	return &jsOrderPublisher{js: js}
}

func (j *jsOrderPublisher) PublishOrderPlaced(e orderPlacedEvent) error {
	if j == nil || j.js == nil {
		return fmt.Errorf("jetstream publisher not configured")
	}
	env := events.NewEnvelope(events.OrderPlaced, uuid.NewString(), map[string]any{
		"order_id":    e.OrderID,
		"total":       e.Total,
		"distance_km": e.DistanceKm,
		"created_at":  e.CreatedAt,
	})
	_, err := natsx.PublishEnvelope(j.js, env)
	return err
}

func (j *jsOrderPublisher) PublishOrderCompleted(e orderCompletedEvent) error {
	if j == nil || j.js == nil {
		return fmt.Errorf("jetstream publisher not configured")
	}
	items := make([]map[string]any, 0, len(e.Items))
	for _, it := range e.Items {
		items = append(items, map[string]any{
			"product_id": it.ProductID,
			"qty":        it.Qty,
			"unit_price": it.UnitPrice,
			"sku":        it.ProductSKU,
		})
	}
	env := events.NewEnvelope(events.OrderCompleted, uuid.NewString(), map[string]any{
		"order_id":     e.OrderID,
		"items":        items,
		"total":        e.Total,
		"payment_type": e.PaymentType,
		"amount_paid":  e.AmountPaid,
	})
	_, err := natsx.PublishEnvelope(j.js, env)
	return err
}

func (s *orderService) publishOrderPlaced(e orderPlacedEvent) {
	if s.bus == nil {
		return
	}
	if err := s.bus.PublishOrderPlaced(e); err != nil {
		slog.Error("publish order.placed", "order_id", e.OrderID, "err", err)
	}
}

func (s *orderService) publishOrderCompleted(e orderCompletedEvent) {
	if s.bus == nil {
		return
	}
	if err := s.bus.PublishOrderCompleted(e); err != nil {
		slog.Error("publish order.completed", "order_id", e.OrderID, "err", err)
	}
}
