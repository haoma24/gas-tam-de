package main

import (
	"fmt"
	"log/slog"

	"gas-tam-de/pkg/events"
	"gas-tam-de/pkg/natsx"

	"github.com/google/uuid"
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
	PublishOrderCancelled(e orderCancelledEvent) error
}

type orderCancelledEvent struct {
	OrderID string
	Items   []orderItemView
}

type noopOrderPublisher struct{}

func (noopOrderPublisher) PublishOrderPlaced(orderPlacedEvent) error       { return nil }
func (noopOrderPublisher) PublishOrderCompleted(orderCompletedEvent) error { return nil }
func (noopOrderPublisher) PublishOrderCancelled(orderCancelledEvent) error { return nil }

type jsOrderPublisher struct {
	provider natsx.JSProvider
}

func newJSOrderPublisher(provider natsx.JSProvider) *jsOrderPublisher {
	return &jsOrderPublisher{provider: provider}
}

func (j *jsOrderPublisher) PublishOrderPlaced(e orderPlacedEvent) error {
	if j == nil || j.provider == nil {
		return fmt.Errorf("jetstream publisher not configured")
	}
	js, err := j.provider.JS()
	if err != nil {
		return err
	}
	env := events.NewEnvelope(events.OrderPlaced, uuid.NewString(), map[string]any{
		"order_id":    e.OrderID,
		"total":       e.Total,
		"distance_km": e.DistanceKm,
		"created_at":  e.CreatedAt,
	})
	_, err = natsx.PublishEnvelope(js, env)
	return err
}

func (j *jsOrderPublisher) PublishOrderCompleted(e orderCompletedEvent) error {
	if j == nil || j.provider == nil {
		return fmt.Errorf("jetstream publisher not configured")
	}
	js, err := j.provider.JS()
	if err != nil {
		return err
	}
	items := make([]map[string]any, 0, len(e.Items))
	for _, it := range e.Items {
		items = append(items, map[string]any{
			"product_id": it.ProductID,
			"qty":        it.Qty,
			"unit_price": it.UnitPrice,
			// COGS snapshot; report-service sums qty×unit_cost into cogs_vnd so
			// profit stops equalling revenue (architecture §6.7).
			"unit_cost": it.UnitCost,
			"sku":       it.ProductSKU,
		})
	}
	env := events.NewEnvelope(events.OrderCompleted, uuid.NewString(), map[string]any{
		"order_id":     e.OrderID,
		"items":        items,
		"total":        e.Total,
		"payment_type": e.PaymentType,
		"amount_paid":  e.AmountPaid,
	})
	_, err = natsx.PublishEnvelope(js, env)
	return err
}

func (j *jsOrderPublisher) PublishOrderCancelled(e orderCancelledEvent) error {
	if j == nil || j.provider == nil {
		return fmt.Errorf("jetstream publisher not configured")
	}
	js, err := j.provider.JS()
	if err != nil {
		return err
	}
	items := make([]map[string]any, 0, len(e.Items))
	for _, it := range e.Items {
		items = append(items, map[string]any{
			"product_id": it.ProductID,
			"qty":        it.Qty,
			"sku":        it.ProductSKU,
		})
	}
	env := events.NewEnvelope(events.OrderCancelled, uuid.NewString(), map[string]any{
		"order_id": e.OrderID,
		"items":    items,
	})
	_, err = natsx.PublishEnvelope(js, env)
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

func (s *orderService) publishOrderCancelled(e orderCancelledEvent) {
	if s.bus == nil {
		return
	}
	if err := s.bus.PublishOrderCancelled(e); err != nil {
		slog.Error("publish order.cancelled", "order_id", e.OrderID, "err", err)
	}
}
