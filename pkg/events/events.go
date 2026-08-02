package events

import "time"

// JetStream / NATS subject naming: <bounded_context>.<entity>.<verb_past>
const (
	AuthOTPVerified          = "auth.otp.verified"
	CatalogProductUpdated    = "catalog.product.updated"
	GeoStoreConfigUpdated    = "geo.store_config.updated"
	OrderPlaced              = "order.placed"
	OrderCompleted           = "order.completed"
	OrderCancelled           = "order.cancelled"
	InventoryStockAdjusted   = "inventory.stock.adjusted"
	InventoryLowStock        = "inventory.low_stock"
	BillingPaymentRecorded   = "billing.payment.recorded"
	BillingDebtUpdated       = "billing.debt.updated"
)

// Envelope is the common event wrapper (schema_version + idempotency).
type Envelope struct {
	EventID       string         `json:"event_id"`
	Subject       string         `json:"subject"`
	OccurredAt    time.Time      `json:"occurred_at"`
	SchemaVersion int            `json:"schema_version"`
	Payload       map[string]any `json:"payload"`
}

// NewEnvelope builds a v1 envelope. eventID should be ULID/UUID.
func NewEnvelope(subject, eventID string, payload map[string]any) Envelope {
	return Envelope{
		EventID:       eventID,
		Subject:       subject,
		OccurredAt:    time.Now().UTC(),
		SchemaVersion: 1,
		Payload:       payload,
	}
}
