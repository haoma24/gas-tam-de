package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"gas-tam-de/pkg/events"

	"github.com/nats-io/nats.go"
)

const (
	durableOrderPlaced    = "report-order-placed"
	durableOrderCompleted = "report-order-completed"
)

type reportService struct {
	db *sql.DB
}

// startReportConsumers registers durable JetStream consumers that upsert daily_stats
// and dashboard debt snapshot (architecture §5.1 / §5.4 / T8.1.1 / T8.1.2).
func startReportConsumers(js nats.JetStreamContext, svc *reportService) ([]*nats.Subscription, error) {
	if js == nil {
		return nil, fmt.Errorf("jetstream context is nil")
	}
	if svc == nil {
		return nil, fmt.Errorf("report service is nil")
	}

	placed, err := startOrderPlacedConsumer(js, svc)
	if err != nil {
		return nil, err
	}
	completed, err := startOrderCompletedConsumer(js, svc)
	if err != nil {
		_ = placed.Unsubscribe()
		return nil, err
	}
	debt, err := startDebtConsumer(js, svc)
	if err != nil {
		_ = placed.Unsubscribe()
		_ = completed.Unsubscribe()
		return nil, err
	}
	return []*nats.Subscription{placed, completed, debt}, nil
}

func startOrderPlacedConsumer(js nats.JetStreamContext, svc *reportService) (*nats.Subscription, error) {
	sub, err := js.Subscribe(events.OrderPlaced, func(msg *nats.Msg) {
		if err := svc.handleOrderPlacedMsg(msg.Data); err != nil {
			slog.Error("order.placed consume", "err", err)
			if nakErr := msg.Nak(); nakErr != nil {
				slog.Error("order.placed nak", "err", nakErr)
			}
			return
		}
		if err := msg.Ack(); err != nil {
			slog.Error("order.placed ack", "err", err)
		}
	},
		nats.Durable(durableOrderPlaced),
		nats.ManualAck(),
		nats.AckExplicit(),
		nats.DeliverAll(),
		nats.BindStream("ORDERS"),
	)
	if err != nil {
		return nil, fmt.Errorf("subscribe %s: %w", events.OrderPlaced, err)
	}
	slog.Info("jetstream consumer started", "durable", durableOrderPlaced, "subject", events.OrderPlaced)
	return sub, nil
}

func startOrderCompletedConsumer(js nats.JetStreamContext, svc *reportService) (*nats.Subscription, error) {
	sub, err := js.Subscribe(events.OrderCompleted, func(msg *nats.Msg) {
		if err := svc.handleOrderCompletedMsg(msg.Data); err != nil {
			slog.Error("order.completed consume", "err", err)
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

func (s *reportService) handleOrderPlacedMsg(data []byte) error {
	var env events.Envelope
	if err := json.Unmarshal(data, &env); err != nil {
		return fmt.Errorf("unmarshal envelope: %w", err)
	}
	if env.EventID == "" {
		return fmt.Errorf("missing event_id")
	}
	if env.Subject != "" && env.Subject != events.OrderPlaced {
		return fmt.Errorf("unexpected subject %q", env.Subject)
	}

	orderID, day, err := parseOrderPlaced(env)
	if err != nil {
		return err
	}
	return s.applyOrderPlaced(env.EventID, orderID, day)
}

func parseOrderPlaced(env events.Envelope) (orderID, day string, err error) {
	if env.Payload == nil {
		return "", "", fmt.Errorf("missing payload")
	}
	orderID = strings.TrimSpace(asString(env.Payload["order_id"]))
	if orderID == "" {
		return "", "", fmt.Errorf("order_id is required")
	}
	day = dayFromPayloadOrOccurred(env.Payload["created_at"], env.OccurredAt)
	return orderID, day, nil
}

func (s *reportService) applyOrderPlaced(eventID, orderID, day string) error {
	eventID = strings.TrimSpace(eventID)
	if eventID == "" {
		return fmt.Errorf("event_id is required")
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
		slog.Info("order.placed already processed", "event_id", eventID, "order_id", orderID)
		return nil
	}

	if err := applyDailyStatsDeltaTx(tx, day, DailyStatsDelta{OrdersPlaced: 1}); err != nil {
		return err
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	if err := insertProcessedEventTx(tx, eventID, now); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	slog.Info("order.placed daily_stats upserted",
		"event_id", eventID, "order_id", orderID, "day", day)
	return nil
}

func (s *reportService) handleOrderCompletedMsg(data []byte) error {
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

	payload, day, err := parseOrderCompleted(env)
	if err != nil {
		return err
	}
	return s.applyOrderCompleted(env.EventID, day, payload)
}

type orderCompletedStats struct {
	OrderID string
	Amounts DailyStatsAmounts
}

func parseOrderCompleted(env events.Envelope) (orderCompletedStats, string, error) {
	var out orderCompletedStats
	if env.Payload == nil {
		return out, "", fmt.Errorf("missing payload")
	}
	out.OrderID = strings.TrimSpace(asString(env.Payload["order_id"]))
	if out.OrderID == "" {
		return out, "", fmt.Errorf("order_id is required")
	}

	saleLines, cogsLines, err := parseOrderCompletedItems(env.Payload["items"])
	if err != nil {
		return out, "", err
	}

	revenue := SumSaleRevenue(saleLines)
	total := asInt64(env.Payload["total"])
	deliveryFee := deliveryFeeFromPayload(env.Payload, revenue, total)
	out.Amounts = BuildDailyStatsAmounts(saleLines, cogsLines, deliveryFee)

	day := dayFromPayloadOrOccurred(env.Payload["completed_at"], env.OccurredAt)
	return out, day, nil
}

func parseOrderCompletedItems(raw any) ([]SaleLine, []COGSLine, error) {
	if raw == nil {
		return nil, nil, nil
	}
	list, ok := raw.([]any)
	if !ok {
		return nil, nil, fmt.Errorf("items must be an array")
	}
	sales := make([]SaleLine, 0, len(list))
	cogs := make([]COGSLine, 0, len(list))
	for i, item := range list {
		m, ok := item.(map[string]any)
		if !ok {
			return nil, nil, fmt.Errorf("items[%d] must be an object", i)
		}
		qty := asInt64(m["qty"])
		if qty <= 0 {
			return nil, nil, fmt.Errorf("items[%d].qty must be > 0", i)
		}
		sales = append(sales, SaleLine{
			Qty:       qty,
			UnitPrice: asInt64(m["unit_price"]),
		})
		// Optional unit_cost (COGS snapshot). Absent → 0 until event enrichment /
		// inventory publishes cost on the line (architecture §6.7 / T7.2.1).
		cogs = append(cogs, COGSLine{
			Qty:      qty,
			UnitCost: asInt64(m["unit_cost"]),
		})
	}
	return sales, cogs, nil
}

// deliveryFeeFromPayload prefers explicit delivery_fee; else derives max(0, total − revenue).
func deliveryFeeFromPayload(payload map[string]any, revenue, total int64) int64 {
	if payload == nil {
		return 0
	}
	if _, ok := payload["delivery_fee"]; ok {
		fee := asInt64(payload["delivery_fee"])
		if fee < 0 {
			return 0
		}
		return fee
	}
	fee := total - revenue
	if fee < 0 {
		return 0
	}
	return fee
}

func (s *reportService) applyOrderCompleted(eventID, day string, payload orderCompletedStats) error {
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

	delta := DailyStatsDelta{
		RevenueVnd:      payload.Amounts.RevenueVnd,
		CogsVnd:         payload.Amounts.CogsVnd,
		DeliveryFeeVnd:  payload.Amounts.DeliveryFeeVnd,
		OrdersCompleted: 1,
	}
	if err := applyDailyStatsDeltaTx(tx, day, delta); err != nil {
		return err
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	if err := insertProcessedEventTx(tx, eventID, now); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	slog.Info("order.completed daily_stats upserted",
		"event_id", eventID,
		"order_id", payload.OrderID,
		"day", day,
		"revenue", payload.Amounts.RevenueVnd,
		"cogs", payload.Amounts.CogsVnd,
		"profit", payload.Amounts.ProfitVnd,
	)
	return nil
}

func dayFromPayloadOrOccurred(raw any, occurredAt time.Time) string {
	if s := strings.TrimSpace(asString(raw)); s != "" {
		if t, err := time.Parse(time.RFC3339Nano, s); err == nil {
			return DayKeyVN(t)
		}
		if t, err := time.Parse(time.RFC3339, s); err == nil {
			return DayKeyVN(t)
		}
	}
	return DayKeyVN(occurredAt)
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
