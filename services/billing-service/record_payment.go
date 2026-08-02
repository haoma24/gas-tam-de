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

	"github.com/google/uuid"
)

const (
	paymentFull    = "FULL"
	paymentPartial = "PARTIAL"
	paymentUnpaid  = "UNPAID"
)

// recordPaymentInput is the settlement snapshot from order complete (T6.1.2).
type recordPaymentInput struct {
	OrderID     string `json:"order_id"`
	CustomerKey string `json:"customer_key"`
	PhoneMasked string `json:"phone_masked"`
	PaymentType string `json:"payment_type"`
	AmountDue   int64  `json:"amount_due"`
	AmountPaid  int64  `json:"amount_paid"`
	RecordedBy  string `json:"recorded_by"`
}

type recordPaymentResult struct {
	PaymentID   string `json:"payment_id"`
	OrderID     string `json:"order_id"`
	PaymentType string `json:"payment_type"`
	AmountDue   int64  `json:"amount_due"`
	AmountPaid  int64  `json:"amount_paid"`
	DebtDelta   int64  `json:"debt_delta"`
	CustomerKey string `json:"customer_key"`
	PhoneMasked string `json:"phone_masked"`
	Balance     int64  `json:"balance"`
	RecordedAt  string `json:"recorded_at"`
	RecordedBy  string `json:"recorded_by"`
	Idempotent  bool   `json:"idempotent,omitempty"`
}

// handleRecordPayment serves POST /v1/internal/payments — called by order-service
// after admin complete (sync path; also publishes billing.* events).
func (s *billingService) handleRecordPayment(w http.ResponseWriter, r *http.Request) {
	var body recordPaymentInput
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&body); err != nil {
		httpx.Error(w, http.StatusBadRequest, "INVALID_JSON", "invalid request body")
		return
	}

	out, err := s.recordPayment(body)
	if err != nil {
		var ve *validationError
		if errors.As(err, &ve) {
			httpx.Error(w, http.StatusBadRequest, "VALIDATION_ERROR", ve.msg)
			return
		}
		slog.Error("record payment", "order_id", body.OrderID, "err", err)
		httpx.Error(w, http.StatusInternalServerError, "INTERNAL", "could not record payment")
		return
	}

	httpx.JSON(w, http.StatusOK, out)
}

type validationError struct{ msg string }

func (e *validationError) Error() string { return e.msg }

func validateRecordPayment(in recordPaymentInput) (recordPaymentInput, error) {
	in.OrderID = strings.TrimSpace(in.OrderID)
	in.CustomerKey = strings.TrimSpace(in.CustomerKey)
	in.PhoneMasked = strings.TrimSpace(in.PhoneMasked)
	in.PaymentType = strings.TrimSpace(strings.ToUpper(in.PaymentType))
	in.RecordedBy = strings.TrimSpace(in.RecordedBy)

	if in.OrderID == "" {
		return in, &validationError{"order_id is required"}
	}
	if in.CustomerKey == "" {
		return in, &validationError{"customer_key is required"}
	}
	if in.PhoneMasked == "" {
		return in, &validationError{"phone_masked is required"}
	}
	if in.AmountDue <= 0 {
		return in, &validationError{"amount_due must be > 0"}
	}
	if in.RecordedBy == "" {
		in.RecordedBy = "system"
	}

	switch in.PaymentType {
	case paymentFull:
		if in.AmountPaid != in.AmountDue {
			return in, &validationError{"amount_paid must equal amount_due for FULL"}
		}
	case paymentPartial:
		if in.AmountPaid <= 0 || in.AmountPaid >= in.AmountDue {
			return in, &validationError{"amount_paid must be > 0 and < amount_due for PARTIAL"}
		}
	case paymentUnpaid:
		if in.AmountPaid != 0 {
			return in, &validationError{"amount_paid must be 0 for UNPAID"}
		}
	default:
		return in, &validationError{"payment_type must be FULL, PARTIAL, or UNPAID"}
	}
	return in, nil
}

// recordPayment inserts payments (unique per order_id) and bumps debts / debt_ledger
// when debt_delta = amount_due − amount_paid > 0. Idempotent on order_id.
func (s *billingService) recordPayment(in recordPaymentInput) (recordPaymentResult, error) {
	in, err := validateRecordPayment(in)
	if err != nil {
		return recordPaymentResult{}, err
	}

	debtDelta := in.AmountDue - in.AmountPaid
	now := time.Now().UTC().Format(time.RFC3339)

	tx, err := s.db.Begin()
	if err != nil {
		return recordPaymentResult{}, err
	}
	defer func() { _ = tx.Rollback() }()

	if existing, ok, err := loadPaymentByOrderID(tx, in.OrderID); err != nil {
		return recordPaymentResult{}, err
	} else if ok {
		existing.CustomerKey = in.CustomerKey
		existing.PhoneMasked = in.PhoneMasked
		bal, err := loadDebtBalance(tx, in.CustomerKey)
		if err != nil {
			return recordPaymentResult{}, err
		}
		existing.Balance = bal
		existing.Idempotent = true
		if err := tx.Commit(); err != nil {
			return recordPaymentResult{}, err
		}
		return existing, nil
	}

	paymentID := uuid.NewString()
	_, err = tx.Exec(`
		INSERT INTO payments (
			id, order_id, payment_type, amount_due, amount_paid, recorded_at, recorded_by
		) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		paymentID, in.OrderID, in.PaymentType, in.AmountDue, in.AmountPaid, now, in.RecordedBy,
	)
	if err != nil {
		return recordPaymentResult{}, err
	}

	balance := int64(0)
	if debtDelta != 0 {
		balance, err = applyDebtDelta(tx, in.CustomerKey, in.PhoneMasked, in.OrderID, debtDelta, now)
		if err != nil {
			return recordPaymentResult{}, err
		}
	} else {
		balance, err = loadDebtBalance(tx, in.CustomerKey)
		if err != nil {
			return recordPaymentResult{}, err
		}
	}

	if err := tx.Commit(); err != nil {
		return recordPaymentResult{}, err
	}

	out := recordPaymentResult{
		PaymentID:   paymentID,
		OrderID:     in.OrderID,
		PaymentType: in.PaymentType,
		AmountDue:   in.AmountDue,
		AmountPaid:  in.AmountPaid,
		DebtDelta:   debtDelta,
		CustomerKey: in.CustomerKey,
		PhoneMasked: in.PhoneMasked,
		Balance:     balance,
		RecordedAt:  now,
		RecordedBy:  in.RecordedBy,
	}
	s.publishPaymentRecorded(out.OrderID, out.PaymentType, out.AmountPaid)
	s.publishDebtUpdated(out.CustomerKey, out.Balance)
	return out, nil
}

func loadPaymentByOrderID(tx *sql.Tx, orderID string) (recordPaymentResult, bool, error) {
	var out recordPaymentResult
	err := tx.QueryRow(`
		SELECT id, order_id, payment_type, amount_due, amount_paid, recorded_at, recorded_by
		FROM payments WHERE order_id = ?`, orderID).Scan(
		&out.PaymentID, &out.OrderID, &out.PaymentType, &out.AmountDue, &out.AmountPaid,
		&out.RecordedAt, &out.RecordedBy,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return recordPaymentResult{}, false, nil
	}
	if err != nil {
		return recordPaymentResult{}, false, err
	}
	out.DebtDelta = out.AmountDue - out.AmountPaid
	return out, true, nil
}

func loadDebtBalance(tx *sql.Tx, customerKey string) (int64, error) {
	var bal int64
	err := tx.QueryRow(`SELECT balance FROM debts WHERE customer_key = ?`, customerKey).Scan(&bal)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, nil
	}
	return bal, err
}

func applyDebtDelta(tx *sql.Tx, customerKey, phoneMasked, orderID string, delta int64, now string) (int64, error) {
	var bal int64
	var existingMasked string
	err := tx.QueryRow(`SELECT balance, phone_masked FROM debts WHERE customer_key = ?`, customerKey).
		Scan(&bal, &existingMasked)
	if errors.Is(err, sql.ErrNoRows) {
		bal = delta
		_, err = tx.Exec(`
			INSERT INTO debts (customer_key, phone_masked, balance, updated_at)
			VALUES (?, ?, ?, ?)`, customerKey, phoneMasked, bal, now)
		if err != nil {
			return 0, err
		}
	} else if err != nil {
		return 0, err
	} else {
		bal += delta
		masked := existingMasked
		if phoneMasked != "" {
			masked = phoneMasked
		}
		_, err = tx.Exec(`
			UPDATE debts SET balance = ?, phone_masked = ?, updated_at = ?
			WHERE customer_key = ?`, bal, masked, now, customerKey)
		if err != nil {
			return 0, err
		}
	}

	_, err = tx.Exec(`
		INSERT INTO debt_ledger (id, customer_key, order_id, delta, balance_after, note, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		uuid.NewString(), customerKey, orderID, delta, bal, "order complete", now,
	)
	if err != nil {
		return 0, err
	}
	return bal, nil
}
