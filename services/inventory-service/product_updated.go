package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"gas-tam-de/pkg/events"

	"github.com/nats-io/nats.go"
)

const durableProductUpdated = "inventory-catalog-product-updated"

// productUpdatedPayload is the decoded catalog.product.updated payload
// (architecture §5.1). `name` was added later, so it may be absent on events
// still sitting in the stream — callers fall back to the SKU.
type productUpdatedPayload struct {
	ProductID string
	SKU       string
	Name      string
}

// startProductUpdatedConsumer registers durable JetStream consumer
// inventory-catalog-product-updated.
//
// Without it, a stock row only ever came from an admin typing a product_id by
// hand; a typo produced a row checkout could never match, so orders failed with
// «Không đủ tồn kho» while the inventory screen showed stock. Creating the row
// (on_hand=0) from catalog makes the two sides share one id by construction.
func startProductUpdatedConsumer(js nats.JetStreamContext, svc *inventoryService) (*nats.Subscription, error) {
	if js == nil {
		return nil, fmt.Errorf("jetstream context is nil")
	}
	if svc == nil {
		return nil, fmt.Errorf("inventory service is nil")
	}

	sub, err := js.Subscribe(events.CatalogProductUpdated, func(msg *nats.Msg) {
		if err := svc.handleProductUpdatedMsg(msg.Data); err != nil {
			slog.Error("catalog.product.updated consume", "err", err)
			if nakErr := msg.Nak(); nakErr != nil {
				slog.Error("catalog.product.updated nak", "err", nakErr)
			}
			return
		}
		if err := msg.Ack(); err != nil {
			slog.Error("catalog.product.updated ack", "err", err)
		}
	},
		nats.Durable(durableProductUpdated),
		nats.ManualAck(),
		nats.AckExplicit(),
		// DeliverAll backfills stock rows for products created before this
		// consumer existed; processed_events keeps the replay idempotent.
		nats.DeliverAll(),
		nats.BindStream("CATALOG"),
	)
	if err != nil {
		return nil, fmt.Errorf("subscribe %s: %w", events.CatalogProductUpdated, err)
	}
	slog.Info("jetstream consumer started", "durable", durableProductUpdated, "subject", events.CatalogProductUpdated)
	return sub, nil
}

func (s *inventoryService) handleProductUpdatedMsg(data []byte) error {
	var env events.Envelope
	if err := json.Unmarshal(data, &env); err != nil {
		return fmt.Errorf("unmarshal envelope: %w", err)
	}
	if env.EventID == "" {
		return fmt.Errorf("missing event_id")
	}
	if env.Subject != "" && env.Subject != events.CatalogProductUpdated {
		return fmt.Errorf("unexpected subject %q", env.Subject)
	}

	payload, err := parseProductUpdatedPayload(env.Payload)
	if err != nil {
		return err
	}
	return s.applyProductUpdated(env.EventID, payload)
}

func parseProductUpdatedPayload(payload map[string]any) (productUpdatedPayload, error) {
	var out productUpdatedPayload
	if payload == nil {
		return out, fmt.Errorf("missing payload")
	}
	out.ProductID = strings.TrimSpace(asString(payload["product_id"]))
	if out.ProductID == "" {
		return out, fmt.Errorf("product_id is required")
	}
	out.SKU = strings.TrimSpace(asString(payload["sku"]))
	if out.SKU == "" {
		// stock_items.sku is NOT NULL UNIQUE; the id is unique too, so it is a
		// safe last resort for an event published without a SKU.
		out.SKU = out.ProductID
	}
	out.Name = strings.TrimSpace(asString(payload["name"]))
	if out.Name == "" {
		out.Name = out.SKU
	}
	return out, nil
}

// applyProductUpdated creates the stock row at on_hand=0 for a product it has
// never seen, or refreshes the denormalized sku/name for one it has.
// Idempotent via processed_events PK. Quantities and cost are never touched:
// catalog owns identity, the stock ledger owns quantity.
func (s *inventoryService) applyProductUpdated(eventID string, payload productUpdatedPayload) error {
	eventID = strings.TrimSpace(eventID)
	if eventID == "" {
		return fmt.Errorf("event_id is required")
	}
	if strings.TrimSpace(payload.ProductID) == "" {
		return fmt.Errorf("product_id is required")
	}

	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	already, err := isEventProcessedTx(tx, eventID)
	if err != nil {
		return err
	}
	if already {
		return nil
	}

	now := time.Now().UTC().Format(time.RFC3339Nano)
	item, err := loadStockItemTx(tx, payload.ProductID)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		created := stockItem{
			ProductID:    payload.ProductID,
			SKU:          payload.SKU,
			Name:         payload.Name,
			OnHand:       0,
			CostPrice:    0,
			ReorderLevel: 0,
			UpdatedAt:    now,
		}
		if err := insertStockItemTx(tx, created); err != nil {
			if !isUniqueViolation(err) {
				return err
			}
			// Another product already owns this SKU. Retrying can never clear
			// that, so record the event and shout instead of looping forever.
			slog.Error("catalog.product.updated sku conflict; stock row not created",
				"event_id", eventID, "product_id", payload.ProductID, "sku", payload.SKU)
		} else {
			slog.Info("catalog.product.updated stock row created",
				"product_id", payload.ProductID, "sku", payload.SKU, "on_hand", 0)
		}

	case err != nil:
		return err

	default:
		if item.SKU != payload.SKU || item.Name != payload.Name {
			if err := updateStockIdentityTx(tx, payload.ProductID, payload.SKU, payload.Name, now); err != nil {
				if !isUniqueViolation(err) {
					return err
				}
				slog.Error("catalog.product.updated sku conflict; identity not synced",
					"event_id", eventID, "product_id", payload.ProductID, "sku", payload.SKU)
			} else {
				slog.Info("catalog.product.updated stock identity synced",
					"product_id", payload.ProductID, "sku", payload.SKU, "name", payload.Name)
			}
		}
	}

	if err := insertProcessedEventTx(tx, eventID, now); err != nil {
		return err
	}
	return tx.Commit()
}

// updateStockIdentityTx syncs only the fields catalog owns. on_hand, cost_price
// and reorder_level belong to the stock ledger and must survive a rename.
func updateStockIdentityTx(tx *sql.Tx, productID, sku, name, now string) error {
	_, err := tx.Exec(`
		UPDATE stock_items
		SET sku = ?, name = ?, updated_at = ?
		WHERE product_id = ?`,
		sku, name, now, productID)
	return err
}
