package main

import (
	"fmt"
	"log/slog"

	"gas-tam-de/pkg/events"
	"gas-tam-de/pkg/natsx"

	"github.com/google/uuid"
)

// productUpdatedPublisher emits catalog.product.updated after product mutations.
type productUpdatedPublisher interface {
	PublishProductUpdated(p product) error
}

type noopProductPublisher struct{}

func (noopProductPublisher) PublishProductUpdated(product) error { return nil }

type jsProductPublisher struct {
	provider natsx.JSProvider
}

func newJSProductPublisher(provider natsx.JSProvider) *jsProductPublisher {
	return &jsProductPublisher{provider: provider}
}

func (j *jsProductPublisher) PublishProductUpdated(p product) error {
	if j == nil || j.provider == nil {
		return fmt.Errorf("jetstream publisher not configured")
	}
	js, err := j.provider.JS()
	if err != nil {
		return err
	}
	// `name` is additive to the §5.1 payload: inventory is documented as the
	// consumer that syncs "tên/sku", which it cannot do without the name.
	// Existing consumers ignore unknown fields, so schema_version stays 1.
	env := events.NewEnvelope(events.CatalogProductUpdated, uuid.NewString(), map[string]any{
		"product_id": p.ID,
		"sku":        p.SKU,
		"name":       p.Name,
		"sale_price": p.SalePrice,
		"active":     p.Active,
	})
	_, err = natsx.PublishEnvelope(js, env)
	return err
}

func (s *catalogService) publishProductUpdated(p product) {
	if s.bus == nil {
		return
	}
	if err := s.bus.PublishProductUpdated(p); err != nil {
		slog.Error("publish catalog.product.updated", "product_id", p.ID, "err", err)
	}
}
