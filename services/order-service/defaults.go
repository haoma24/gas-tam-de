package main

import (
	"database/sql"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"gas-tam-de/pkg/httpx"
)

// handleGetMyOrderDefaults serves GET /v1/orders/me/defaults —
// last delivery name+address for returning customers (by JWT user_id).
func (s *orderService) handleGetMyOrderDefaults(w http.ResponseWriter, r *http.Request) {
	userID, _, ok := requireCustomerIdentity(w, r)
	if !ok {
		return
	}

	var (
		customerName, addressText string
		lat, lng                  float64
		createdAt                 string
	)
	err := s.db.QueryRow(`
		SELECT customer_name, address_text, lat, lng, created_at
		FROM orders
		WHERE user_id = ?
		ORDER BY created_at DESC
		LIMIT 1
	`, userID).Scan(&customerName, &addressText, &lat, &lng, &createdAt)
	if errors.Is(err, sql.ErrNoRows) {
		httpx.JSON(w, http.StatusOK, map[string]any{
			"has_defaults": false,
		})
		return
	}
	if err != nil {
		slog.Error("get my order defaults", "err", err)
		httpx.Error(w, http.StatusInternalServerError, "INTERNAL", "could not load defaults")
		return
	}

	httpx.JSON(w, http.StatusOK, map[string]any{
		"has_defaults":  true,
		"customer_name": strings.TrimSpace(customerName),
		"address_text":  strings.TrimSpace(addressText),
		"lat":           lat,
		"lng":           lng,
		"ordered_at":    createdAt,
	})
}
