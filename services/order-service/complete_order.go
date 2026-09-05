package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"gas-tam-de/pkg/httpx"

	"github.com/go-chi/chi/v5"
)

// Payment types for admin complete (PRD M6 / architecture §6.6).
const (
	paymentFull    = "FULL"
	paymentPartial = "PARTIAL"
	paymentUnpaid  = "UNPAID"
)

type completeOrderBody struct {
	PaymentType string `json:"payment_type"`
	// AmountPaid is required for PARTIAL; optional for FULL (defaults to total)
	// and UNPAID (must be 0 or omitted).
	AmountPaid *int64 `json:"amount_paid"`
}

// completeOrderView is the admin complete response: order snapshot + payment
// settlement. Billing persistence is invoked sync; order.completed is published
// after a successful COMPLETED transition.
type completeOrderView struct {
	ID           string `json:"id"`
	UserID       string `json:"user_id"`
	CustomerName string `json:"customer_name"`
	PhoneMasked  string `json:"phone_masked"`
	// Admin response — carries the callable number, like the other /v1/admin views.
	CustomerPhone string          `json:"customer_phone,omitempty"`
	AddressText   string          `json:"address_text"`
	Lat           float64         `json:"lat"`
	Lng           float64         `json:"lng"`
	DistanceKm    float64         `json:"distance_km"`
	DeliveryFee   int64           `json:"delivery_fee"`
	Subtotal      int64           `json:"subtotal"`
	Total         int64           `json:"total"`
	Status        string          `json:"status"`
	CreatedAt     string          `json:"created_at"`
	CompletedAt   string          `json:"completed_at"`
	PaymentType   string          `json:"payment_type"`
	AmountDue     int64           `json:"amount_due"`
	AmountPaid    int64           `json:"amount_paid"`
	Debt          int64           `json:"debt"`
	Items         []orderItemView `json:"items"`
}

// handleCompleteOrder serves POST /v1/admin/orders/{id}/complete.
// Validates payment payload (FULL / PARTIAL / UNPAID), transitions PENDING →
// COMPLETED, stores payment snapshot on the order, records payments/debts on
// billing-service, then publishes order.completed.
// Gateway mounts under /v1/admin/* with role=admin RBAC.
func (s *orderService) handleCompleteOrder(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSpace(chi.URLParam(r, "id"))
	if id == "" {
		httpx.Error(w, http.StatusBadRequest, "INVALID_ID", "order id is required")
		return
	}

	var body completeOrderBody
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&body); err != nil {
		httpx.Error(w, http.StatusBadRequest, "INVALID_JSON", "invalid request body")
		return
	}

	o, err := s.loadOrderByID(id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			httpx.Error(w, http.StatusNotFound, "NOT_FOUND", "order not found")
			return
		}
		slog.Error("complete order load", "err", err)
		httpx.Error(w, http.StatusInternalServerError, "INTERNAL", "could not load order")
		return
	}

	switch o.status {
	case "PENDING":
		// ok
	case "COMPLETED":
		httpx.Error(w, http.StatusConflict, "ORDER_ALREADY_COMPLETED", "order is already completed")
		return
	case "CANCELLED":
		httpx.Error(w, http.StatusConflict, "ORDER_NOT_COMPLETABLE", "cancelled order cannot be completed")
		return
	default:
		httpx.Error(w, http.StatusConflict, "ORDER_NOT_COMPLETABLE", "order status does not allow complete")
		return
	}

	paymentType, amountPaid, debt, errMsg := settlePayment(o.total, body)
	if errMsg != "" {
		httpx.Error(w, http.StatusBadRequest, "VALIDATION_ERROR", errMsg)
		return
	}

	completedAt := time.Now().UTC().Format(time.RFC3339)
	res, err := s.db.Exec(`
		UPDATE orders
		SET status = 'COMPLETED',
		    completed_at = ?,
		    payment_type = ?,
		    amount_paid = ?
		WHERE id = ? AND status = 'PENDING'`,
		completedAt, paymentType, amountPaid, id,
	)
	if err != nil {
		slog.Error("complete order update", "err", err)
		httpx.Error(w, http.StatusInternalServerError, "INTERNAL", "could not complete order")
		return
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		// Race: another complete won — treat as already completed.
		httpx.Error(w, http.StatusConflict, "ORDER_ALREADY_COMPLETED", "order is already completed")
		return
	}

	items, err := s.loadOrderItems(o.id)
	if err != nil {
		slog.Error("complete order items", "err", err)
		httpx.Error(w, http.StatusInternalServerError, "INTERNAL", "could not load order")
		return
	}

	s.recordBillingPayment(r, o, paymentType, amountPaid)

	s.publishOrderCompleted(orderCompletedEvent{
		OrderID:     o.id,
		Items:       items,
		Total:       o.total,
		PaymentType: paymentType,
		AmountPaid:  amountPaid,
	})

	httpx.JSON(w, http.StatusOK, completeOrderView{
		ID:            o.id,
		UserID:        o.userID,
		CustomerName:  o.customerName,
		PhoneMasked:   o.phoneMasked,
		CustomerPhone: displayPhone(o.customerPhone),
		AddressText:   o.addressText,
		Lat:           o.lat,
		Lng:           o.lng,
		DistanceKm:    o.distanceKm,
		DeliveryFee:   o.deliveryFee,
		Subtotal:      o.subtotal,
		Total:         o.total,
		Status:        "COMPLETED",
		CreatedAt:     o.createdAt,
		CompletedAt:   completedAt,
		PaymentType:   paymentType,
		AmountDue:     o.total,
		AmountPaid:    amountPaid,
		Debt:          debt,
		Items:         items,
	})
}

// recordBillingPayment sync-writes payments + debts via billing-service after
// the order is already COMPLETED. Failures are logged only (billing may later
// retry from order.completed); complete response still succeeds.
func (s *orderService) recordBillingPayment(r *http.Request, o orderRow, paymentType string, amountPaid int64) {
	if s.billing == nil {
		return
	}
	customerKey := strings.TrimSpace(o.phoneHash)
	if customerKey == "" {
		customerKey = "uid:" + o.userID
	}
	recordedBy := strings.TrimSpace(r.Header.Get("X-User-Id"))
	if recordedBy == "" {
		recordedBy = "admin"
	}
	err := s.billing.RecordPayment(r.Context(), billingPaymentInput{
		OrderID:     o.id,
		CustomerKey: customerKey,
		PhoneMasked: o.phoneMasked,
		PaymentType: paymentType,
		AmountDue:   o.total,
		AmountPaid:  amountPaid,
		RecordedBy:  recordedBy,
	})
	if err != nil {
		slog.Error("billing record payment", "order_id", o.id, "err", err)
	}
}

// settlePayment applies PRD M6 rules against amount_due (= order.total).
// Returns paymentType, amountPaid, debt, or a validation message.
func settlePayment(amountDue int64, body completeOrderBody) (paymentType string, amountPaid, debt int64, errMsg string) {
	paymentType = strings.TrimSpace(strings.ToUpper(body.PaymentType))
	switch paymentType {
	case paymentFull:
		if body.AmountPaid != nil && *body.AmountPaid != amountDue {
			return "", 0, 0, "amount_paid must equal total for FULL payment"
		}
		return paymentFull, amountDue, 0, ""

	case paymentPartial:
		if body.AmountPaid == nil {
			return "", 0, 0, "amount_paid is required for PARTIAL payment"
		}
		paid := *body.AmountPaid
		if paid <= 0 || paid >= amountDue {
			return "", 0, 0, "amount_paid must be > 0 and < total for PARTIAL payment"
		}
		return paymentPartial, paid, amountDue - paid, ""

	case paymentUnpaid:
		if body.AmountPaid != nil && *body.AmountPaid != 0 {
			return "", 0, 0, "amount_paid must be 0 for UNPAID"
		}
		return paymentUnpaid, 0, amountDue, ""

	default:
		return "", 0, 0, "payment_type must be FULL, PARTIAL, or UNPAID"
	}
}
