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
	durableBillingDebtUpdated = "report-billing-debt-updated"
	dashboardSnapshotID       = "current"
)

// startDebtConsumer registers durable JetStream consumer for billing.debt.updated
// → customer_debt_balances + dashboard_snapshot.debt_total (T8.1.2 / architecture §6.7).
func startDebtConsumer(js nats.JetStreamContext, svc *reportService) (*nats.Subscription, error) {
	if js == nil {
		return nil, fmt.Errorf("jetstream context is nil")
	}
	if svc == nil {
		return nil, fmt.Errorf("report service is nil")
	}

	sub, err := js.Subscribe(events.BillingDebtUpdated, func(msg *nats.Msg) {
		if err := svc.handleBillingDebtUpdatedMsg(msg.Data); err != nil {
			slog.Error("billing.debt.updated consume", "err", err)
			if nakErr := msg.Nak(); nakErr != nil {
				slog.Error("billing.debt.updated nak", "err", nakErr)
			}
			return
		}
		if err := msg.Ack(); err != nil {
			slog.Error("billing.debt.updated ack", "err", err)
		}
	},
		nats.Durable(durableBillingDebtUpdated),
		nats.ManualAck(),
		nats.AckExplicit(),
		nats.DeliverAll(),
		nats.BindStream("BILLING"),
	)
	if err != nil {
		return nil, fmt.Errorf("subscribe %s: %w", events.BillingDebtUpdated, err)
	}
	slog.Info("jetstream consumer started",
		"durable", durableBillingDebtUpdated, "subject", events.BillingDebtUpdated)
	return sub, nil
}

func (s *reportService) handleBillingDebtUpdatedMsg(data []byte) error {
	var env events.Envelope
	if err := json.Unmarshal(data, &env); err != nil {
		return fmt.Errorf("unmarshal envelope: %w", err)
	}
	if env.EventID == "" {
		return fmt.Errorf("missing event_id")
	}
	if env.Subject != "" && env.Subject != events.BillingDebtUpdated {
		return fmt.Errorf("unexpected subject %q", env.Subject)
	}

	customerKey, balance, err := parseBillingDebtUpdated(env)
	if err != nil {
		return err
	}
	return s.applyBillingDebtUpdated(env.EventID, customerKey, balance)
}

func parseBillingDebtUpdated(env events.Envelope) (customerKey string, balance int64, err error) {
	if env.Payload == nil {
		return "", 0, fmt.Errorf("missing payload")
	}
	customerKey = strings.TrimSpace(asString(env.Payload["customer_key"]))
	if customerKey == "" {
		return "", 0, fmt.Errorf("customer_key is required")
	}
	balance = asInt64(env.Payload["balance"])
	return customerKey, balance, nil
}

func (s *reportService) applyBillingDebtUpdated(eventID, customerKey string, balance int64) error {
	eventID = strings.TrimSpace(eventID)
	customerKey = strings.TrimSpace(customerKey)
	if eventID == "" {
		return fmt.Errorf("event_id is required")
	}
	if customerKey == "" {
		return fmt.Errorf("customer_key is required")
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
		slog.Info("billing.debt.updated already processed",
			"event_id", eventID, "customer_key", customerKey)
		return nil
	}

	now := time.Now().UTC().Format(time.RFC3339Nano)
	if _, err := tx.Exec(`
		INSERT INTO customer_debt_balances (customer_key, balance, updated_at)
		VALUES (?, ?, ?)
		ON CONFLICT(customer_key) DO UPDATE SET
			balance = excluded.balance,
			updated_at = excluded.updated_at`,
		customerKey, balance, now,
	); err != nil {
		return err
	}

	debtTotal, err := sumCustomerDebtBalancesTx(tx)
	if err != nil {
		return err
	}
	if err := upsertDashboardSnapshotDebtTx(tx, debtTotal, now); err != nil {
		return err
	}
	if err := insertProcessedEventTx(tx, eventID, now); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	slog.Info("billing.debt.updated snapshot upserted",
		"event_id", eventID,
		"customer_key", customerKey,
		"balance", balance,
		"debt_total", debtTotal,
	)
	return nil
}

func sumCustomerDebtBalancesTx(tx *sql.Tx) (int64, error) {
	var total int64
	err := tx.QueryRow(`
		SELECT COALESCE(SUM(balance), 0)
		FROM customer_debt_balances
		WHERE balance > 0`).Scan(&total)
	return total, err
}

func upsertDashboardSnapshotDebtTx(tx *sql.Tx, debtTotal int64, updatedAt string) error {
	_, err := tx.Exec(`
		INSERT INTO dashboard_snapshot (
			id, revenue_today, revenue_month, debt_total, profit_month, updated_at
		) VALUES (?, 0, 0, ?, 0, ?)
		ON CONFLICT(id) DO UPDATE SET
			debt_total = excluded.debt_total,
			updated_at = excluded.updated_at`,
		dashboardSnapshotID, debtTotal, updatedAt,
	)
	return err
}

// loadDebtTotal returns outstanding customer debt for the dashboard summary.
// Prefers dashboard_snapshot; falls back to SUM(customer_debt_balances) if missing.
func loadDebtTotal(db *sql.DB) (int64, error) {
	var total int64
	err := db.QueryRow(`
		SELECT debt_total FROM dashboard_snapshot WHERE id = ?`, dashboardSnapshotID,
	).Scan(&total)
	if err == nil {
		return total, nil
	}
	if err != sql.ErrNoRows {
		return 0, err
	}
	err = db.QueryRow(`
		SELECT COALESCE(SUM(balance), 0)
		FROM customer_debt_balances
		WHERE balance > 0`).Scan(&total)
	return total, err
}
