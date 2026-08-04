package main

import (
	"fmt"
	"log/slog"

	"gas-tam-de/pkg/events"
	"gas-tam-de/pkg/natsx"

	"github.com/google/uuid"
)

// billingEventPublisher emits billing.* events after a successful payment write.
type billingEventPublisher interface {
	PublishPaymentRecorded(orderID, paymentType string, amountPaid int64) error
	PublishDebtUpdated(customerKey string, balance int64) error
}

type noopBillingPublisher struct{}

func (noopBillingPublisher) PublishPaymentRecorded(string, string, int64) error { return nil }
func (noopBillingPublisher) PublishDebtUpdated(string, int64) error            { return nil }

type jsBillingPublisher struct {
	provider natsx.JSProvider
}

func newJSBillingPublisher(provider natsx.JSProvider) *jsBillingPublisher {
	return &jsBillingPublisher{provider: provider}
}

func (j *jsBillingPublisher) PublishPaymentRecorded(orderID, paymentType string, amountPaid int64) error {
	if j == nil || j.provider == nil {
		return fmt.Errorf("jetstream publisher not configured")
	}
	js, err := j.provider.JS()
	if err != nil {
		return err
	}
	env := events.NewEnvelope(events.BillingPaymentRecorded, uuid.NewString(), map[string]any{
		"order_id":     orderID,
		"amount_paid":  amountPaid,
		"payment_type": paymentType,
	})
	_, err = natsx.PublishEnvelope(js, env)
	return err
}

func (j *jsBillingPublisher) PublishDebtUpdated(customerKey string, balance int64) error {
	if j == nil || j.provider == nil {
		return fmt.Errorf("jetstream publisher not configured")
	}
	js, err := j.provider.JS()
	if err != nil {
		return err
	}
	env := events.NewEnvelope(events.BillingDebtUpdated, uuid.NewString(), map[string]any{
		"customer_key": customerKey,
		"balance":      balance,
	})
	_, err = natsx.PublishEnvelope(js, env)
	return err
}

func (s *billingService) publishPaymentRecorded(orderID, paymentType string, amountPaid int64) {
	if s.bus == nil {
		return
	}
	if err := s.bus.PublishPaymentRecorded(orderID, paymentType, amountPaid); err != nil {
		slog.Error("publish billing.payment.recorded", "order_id", orderID, "err", err)
	}
}

func (s *billingService) publishDebtUpdated(customerKey string, balance int64) {
	if s.bus == nil {
		return
	}
	if err := s.bus.PublishDebtUpdated(customerKey, balance); err != nil {
		slog.Error("publish billing.debt.updated", "customer_key", customerKey, "err", err)
	}
}
