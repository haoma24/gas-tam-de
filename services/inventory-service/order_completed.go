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

	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
)

const (
	durableOrderCompleted = "inventory-order-completed"
	refTypeOrder          = "ORDER"
)

// orderCompletedItem is one line from order.completed payload (architecture §5.1).
type orderCompletedItem struct {
	ProductID string
	SKU       string
	Qty       int64
}

// orderCompletedPayload is the decoded order.completed envelope payload.
type orderCompletedPayload struct {
	OrderID string
	Items   []orderCompletedItem
}

// startOrderCompletedConsumer registers durable JetStream consumer inventory-order-completed.
// Stock is deducted only on order.completed (not order.placed) — architecture §3 / §10.
func startOrderCompletedConsumer(js nats.JetStreamContext, svc *inventoryService) (*nats.Subscription, error) {
	if js == nil {
		return nil, fmt.Errorf("jetstream context is nil")
	}
	if svc == nil {
		return nil, fmt.Errorf("inventory service is nil")
	}

	sub, err := js.Subscribe(events.OrderCompleted, func(msg *nats.Msg) {
		if err := svc.handleOrderCompletedMsg(msg.Data); err != nil {
			slog.Error("order.completed consume", "err", err)
			// Nak so JetStream redelivers; processed_events keeps successful applies idempotent.
			if nakErr := msg.Nak(); nakErr != nil {
				slog.Error("order.completed nak", "err", nakErr)
			}
			return
		}
		if err := msg.Ack(); err != nil {
			slog.Error("order.completed ack", "err", err)
		}
	},
		nats.Durable(durableOrderCompleted),
		nats.ManualAck(),
		nats.AckExplicit(),
		nats.DeliverAll(),
		nats.BindStream("ORDERS"),
	)
	if err != nil {
		return nil, fmt.Errorf("subscribe %s: %w", events.OrderCompleted, err)
	}
	slog.Info("jetstream consumer started", "durable", durableOrderCompleted, "subject", events.OrderCompleted)
	return sub, nil
}

func (s *inventoryService) handleOrderCompletedMsg(data []byte) error {
	var env events.Envelope
	if err := json.Unmarshal(data, &env); err != nil {
		return fmt.Errorf("unmarshal envelope: %w", err)
	}
	if env.EventID == "" {
		return fmt.Errorf("missing event_id")
	}
	if env.Subject != "" && env.Subject != events.OrderCompleted {
		return fmt.Errorf("unexpected subject %q", env.Subject)
	}

	payload, err := parseOrderCompletedPayload(env.Payload)
	if err != nil {
		return err
	}
	return s.applyOrderCompleted(env.EventID, payload)
}

func parseOrderCompletedPayload(payload map[string]any) (orderCompletedPayload, error) {
	var out orderCompletedPayload
	if payload == nil {
		return out, fmt.Errorf("missing payload")
	}
	out.OrderID = strings.TrimSpace(asString(payload["order_id"]))
	if out.OrderID == "" {
		return out, fmt.Errorf("order_id is required")
	}

	rawItems, ok := payload["items"]
	if !ok || rawItems == nil {
		return out, nil
	}
	list, ok := rawItems.([]any)
	if !ok {
		return out, fmt.Errorf("items must be an array")
	}
	out.Items = make([]orderCompletedItem, 0, len(list))
	for i, raw := range list {
		m, ok := raw.(map[string]any)
		if !ok {
			return out, fmt.Errorf("items[%d] must be an object", i)
		}
		it := orderCompletedItem{
			ProductID: strings.TrimSpace(asString(m["product_id"])),
			SKU:       strings.TrimSpace(asString(m["sku"])),
			Qty:       asInt64(m["qty"]),
		}
		if it.ProductID == "" {
			return out, fmt.Errorf("items[%d].product_id is required", i)
		}
		if it.Qty <= 0 {
			return out, fmt.Errorf("items[%d].qty must be > 0", i)
		}
		out.Items = append(out.Items, it)
	}
	return out, nil
}

// applyOrderCompleted deducts stock for each line (OUT + COGS snapshot) and records event_id.
// Idempotent via processed_events PK. MVP: missing stock rows are created (cost 0) so on_hand may go negative.
func (s *inventoryService) applyOrderCompleted(eventID string, payload orderCompletedPayload) error {
	eventID = strings.TrimSpace(eventID)
	if eventID == "" {
		return fmt.Errorf("event_id is required")
	}
	if strings.TrimSpace(payload.OrderID) == "" {
		return fmt.Errorf("order_id is required")
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
		slog.Info("order.completed already processed", "event_id", eventID, "order_id", payload.OrderID)
		return nil
	}

	// Stock already reserved on place — skip OUT to avoid double-deduct.
	var prior int
	if err := tx.QueryRow(`
		SELECT COUNT(1) FROM stock_movements
		WHERE ref_type = ? AND ref_id = ? AND movement_type = ?
	`, refTypeOrder, payload.OrderID, movementOUT).Scan(&prior); err != nil {
		return err
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	if prior > 0 {
		if err := insertProcessedEventTx(tx, eventID, now); err != nil {
			return err
		}
		if err := tx.Commit(); err != nil {
			return err
		}
		slog.Info("order.completed stock already reserved on place",
			"event_id", eventID, "order_id", payload.OrderID)
		return nil
	}

	refType := refTypeOrder
	orderID := payload.OrderID

	for _, it := range payload.Items {
		if err := applyOrderOUTTx(tx, it, orderID, refType, now); err != nil {
			return err
		}
	}

	if err := insertProcessedEventTx(tx, eventID, now); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}

	slog.Info("order.completed stock deducted",
		"event_id", eventID,
		"order_id", orderID,
		"lines", len(payload.Items),
	)
	return nil
}

func applyOrderOUTTx(tx *sql.Tx, it orderCompletedItem, orderID, refType, now string) error {
	item, err := loadStockItemTx(tx, it.ProductID)
	exists := err == nil
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return err
	}

	if !exists {
		sku := it.SKU
		if sku == "" {
			sku = it.ProductID
		}
		item = stockItem{
			ProductID:    it.ProductID,
			SKU:          sku,
			Name:         sku, // placeholder until catalog sync; MVP allow negative on_hand
			OnHand:       0,
			CostPrice:    0,
			ReorderLevel: 0,
			UpdatedAt:    now,
		}
		if err := insertStockItemTx(tx, item); err != nil {
			if isUniqueViolation(err) {
				// Concurrent create race: reload and continue OUT.
				item, err = loadStockItemTx(tx, it.ProductID)
				if err != nil {
					return err
				}
			} else {
				return err
			}
		}
	}

	// T7.2.1: COGS snapshot from current cost_price; immutable on the movement row.
	snap := snapshotOUTCost(item.CostPrice)
	item.OnHand -= it.Qty
	item.UpdatedAt = now
	if err := updateStockItemTx(tx, item); err != nil {
		return err
	}

	note := "order.completed"
	refID := orderID
	return insertMovementTx(tx, uuid.NewString(), it.ProductID, movementOUT, it.Qty, &snap, &note, &refType, &refID, now, nil)
}

func isEventProcessedTx(tx *sql.Tx, eventID string) (bool, error) {
	var n int
	err := tx.QueryRow(`SELECT COUNT(1) FROM processed_events WHERE event_id = ?`, eventID).Scan(&n)
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

func insertProcessedEventTx(tx *sql.Tx, eventID, processedAt string) error {
	_, err := tx.Exec(`
		INSERT INTO processed_events (event_id, processed_at) VALUES (?, ?)`,
		eventID, processedAt)
	return err
}

func asString(v any) string {
	switch t := v.(type) {
	case string:
		return t
	case fmt.Stringer:
		return t.String()
	default:
		if v == nil {
			return ""
		}
		return fmt.Sprint(v)
	}
}

func asInt64(v any) int64 {
	switch t := v.(type) {
	case int64:
		return t
	case int:
		return int64(t)
	case float64:
		return int64(t)
	case json.Number:
		n, _ := t.Int64()
		return n
	default:
		return 0
	}
}
