package main

import (
	"database/sql"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"gas-tam-de/pkg/httpx"

	"github.com/go-chi/chi/v5"
)

// handleCancelMyOrder serves POST /v1/orders/{id}/cancel — customer cancels own PENDING.
func (s *orderService) handleCancelMyOrder(w http.ResponseWriter, r *http.Request) {
	userID, _, ok := requireCustomerIdentity(w, r)
	if !ok {
		return
	}
	orderID := strings.TrimSpace(chi.URLParam(r, "id"))
	if orderID == "" {
		httpx.Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "order id is required")
		return
	}

	tx, err := s.db.Begin()
	if err != nil {
		slog.Error("begin cancel", "err", err)
		httpx.Error(w, http.StatusInternalServerError, "INTERNAL", "could not cancel order")
		return
	}
	defer func() { _ = tx.Rollback() }()

	var ownerID, status string
	err = tx.QueryRow(`SELECT user_id, status FROM orders WHERE id = ?`, orderID).Scan(&ownerID, &status)
	if errors.Is(err, sql.ErrNoRows) {
		httpx.Error(w, http.StatusNotFound, "NOT_FOUND", "order not found")
		return
	}
	if err != nil {
		slog.Error("load order cancel", "err", err)
		httpx.Error(w, http.StatusInternalServerError, "INTERNAL", "could not cancel order")
		return
	}
	if ownerID != userID {
		httpx.Error(w, http.StatusForbidden, "FORBIDDEN", "not your order")
		return
	}
	switch status {
	case "COMPLETED":
		httpx.Error(w, http.StatusConflict, "CONFLICT", "order already completed")
		return
	case "CANCELLED":
		httpx.Error(w, http.StatusConflict, "CONFLICT", "order already cancelled")
		return
	}

	items, err := loadOrderItemsTx(tx, orderID)
	if err != nil {
		slog.Error("load items cancel", "err", err)
		httpx.Error(w, http.StatusInternalServerError, "INTERNAL", "could not cancel order")
		return
	}

	now := time.Now().UTC().Format(time.RFC3339Nano)
	_, err = tx.Exec(`UPDATE orders SET status = 'CANCELLED', cancelled_at = ? WHERE id = ?`, now, orderID)
	if err != nil {
		slog.Error("update cancel", "err", err)
		httpx.Error(w, http.StatusInternalServerError, "INTERNAL", "could not cancel order")
		return
	}
	if err := tx.Commit(); err != nil {
		slog.Error("commit cancel", "err", err)
		httpx.Error(w, http.StatusInternalServerError, "INTERNAL", "could not cancel order")
		return
	}

	if s.inventory != nil {
		lines := make([]stockLine, 0, len(items))
		for _, it := range items {
			lines = append(lines, stockLine{
				ProductID: it.ProductID,
				SKU:       it.ProductSKU,
				Qty:       int64(it.Qty),
			})
		}
		if err := s.inventory.Release(r.Context(), orderID, lines); err != nil {
			slog.Error("inventory release after cancel", "order_id", orderID, "err", err)
		}
	}

	s.publishOrderCancelled(orderCancelledEvent{
		OrderID: orderID,
		Items:   items,
	})

	httpx.JSON(w, http.StatusOK, map[string]any{
		"ok":           true,
		"order_id":     orderID,
		"status":       "CANCELLED",
		"cancelled_at": now,
	})
}

func loadOrderItemsTx(tx *sql.Tx, orderID string) ([]orderItemView, error) {
	rows, err := tx.Query(`
		SELECT id, product_id, product_sku, product_name, unit_price, qty, line_total
		FROM order_items WHERE order_id = ?
	`, orderID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]orderItemView, 0)
	for rows.Next() {
		var it orderItemView
		if err := rows.Scan(&it.ID, &it.ProductID, &it.ProductSKU, &it.ProductName, &it.UnitPrice, &it.Qty, &it.LineTotal); err != nil {
			return nil, err
		}
		out = append(out, it)
	}
	return out, rows.Err()
}
