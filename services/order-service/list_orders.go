package main

import (
	"database/sql"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"gas-tam-de/pkg/httpx"

	"github.com/go-chi/chi/v5"
)

type orderRow struct {
	id, userID, customerName, phoneHash, phoneMasked, addressText string
	lat, lng, distanceKm                                          float64
	deliveryFee, subtotal, total                                  int64
	status, createdAt                                             string
}

// handleListMyOrders serves GET /v1/orders/me — customer's own orders with PII masked.
func (s *orderService) handleListMyOrders(w http.ResponseWriter, r *http.Request) {
	userID, _, ok := requireCustomerIdentity(w, r)
	if !ok {
		return
	}

	rows, err := s.db.Query(`
		SELECT id, user_id, customer_name, phone_masked, address_text,
		       lat, lng, distance_km, delivery_fee, subtotal, total,
		       status, created_at
		FROM orders
		WHERE user_id = ?
		ORDER BY created_at DESC`, userID)
	if err != nil {
		slog.Error("list my orders", "err", err)
		httpx.Error(w, http.StatusInternalServerError, "INTERNAL", "could not list orders")
		return
	}

	orders, err := scanOrderRows(rows)
	if err != nil {
		slog.Error("list my orders", "err", err)
		httpx.Error(w, http.StatusInternalServerError, "INTERNAL", "could not list orders")
		return
	}

	out, err := s.orderViewsFromRows(orders)
	if err != nil {
		slog.Error("list my orders", "err", err)
		httpx.Error(w, http.StatusInternalServerError, "INTERNAL", "could not list orders")
		return
	}

	httpx.JSON(w, http.StatusOK, map[string]any{"orders": out})
}

// handleListAdminOrders serves GET /v1/admin/orders — FIFO desk list (oldest first).
// Optional ?status= filters; omitted defaults to PENDING (chờ giao). Sort: created_at ASC.
// Gateway mounts under /v1/admin/* with role=admin RBAC.
func (s *orderService) handleListAdminOrders(w http.ResponseWriter, r *http.Request) {
	status, ok := parseAdminOrderStatusFilter(r.URL.Query().Get("status"))
	if !ok {
		httpx.Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "status must be PENDING, COMPLETED, or CANCELLED")
		return
	}

	rows, err := s.db.Query(`
		SELECT id, user_id, customer_name, phone_masked, address_text,
		       lat, lng, distance_km, delivery_fee, subtotal, total,
		       status, created_at
		FROM orders
		WHERE status = ?
		ORDER BY created_at ASC`, status)
	if err != nil {
		slog.Error("list admin orders", "err", err)
		httpx.Error(w, http.StatusInternalServerError, "INTERNAL", "could not list orders")
		return
	}

	orders, err := scanOrderRows(rows)
	if err != nil {
		slog.Error("list admin orders", "err", err)
		httpx.Error(w, http.StatusInternalServerError, "INTERNAL", "could not list orders")
		return
	}

	out, err := s.adminOrderViewsFromRows(orders)
	if err != nil {
		slog.Error("list admin orders", "err", err)
		httpx.Error(w, http.StatusInternalServerError, "INTERNAL", "could not list orders")
		return
	}

	httpx.JSON(w, http.StatusOK, map[string]any{"orders": out})
}

// handleGetAdminOrder serves GET /v1/admin/orders/{id} — single order for desk /
// navigation. Response includes delivery destination `lat`/`lng` (WGS84) stored
// at place time. Gateway mounts under /v1/admin/* with role=admin RBAC.
func (s *orderService) handleGetAdminOrder(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSpace(chi.URLParam(r, "id"))
	if id == "" {
		httpx.Error(w, http.StatusBadRequest, "INVALID_ID", "order id is required")
		return
	}

	o, err := s.loadOrderByID(id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			httpx.Error(w, http.StatusNotFound, "NOT_FOUND", "order not found")
			return
		}
		slog.Error("get admin order", "err", err)
		httpx.Error(w, http.StatusInternalServerError, "INTERNAL", "could not load order")
		return
	}

	items, err := s.loadOrderItems(o.id)
	if err != nil {
		slog.Error("get admin order items", "err", err)
		httpx.Error(w, http.StatusInternalServerError, "INTERNAL", "could not load order")
		return
	}

	httpx.JSON(w, http.StatusOK, customerOrderView(
		o.id, o.userID, o.customerName, o.phoneMasked, o.addressText,
		o.lat, o.lng, o.distanceKm,
		o.deliveryFee, o.subtotal, o.total,
		o.status, o.createdAt, items,
	))
}

func (s *orderService) loadOrderByID(id string) (orderRow, error) {
	var o orderRow
	err := s.db.QueryRow(`
		SELECT id, user_id, customer_name, phone_hash, phone_masked, address_text,
		       lat, lng, distance_km, delivery_fee, subtotal, total,
		       status, created_at
		FROM orders WHERE id = ?`, id).Scan(
		&o.id, &o.userID, &o.customerName, &o.phoneHash, &o.phoneMasked, &o.addressText,
		&o.lat, &o.lng, &o.distanceKm, &o.deliveryFee, &o.subtotal, &o.total,
		&o.status, &o.createdAt,
	)
	return o, err
}

// parseAdminOrderStatusFilter returns the status filter. Empty → PENDING (order desk default).
func parseAdminOrderStatusFilter(raw string) (status string, ok bool) {
	s := strings.TrimSpace(strings.ToUpper(raw))
	if s == "" {
		return "PENDING", true
	}
	switch s {
	case "PENDING", "COMPLETED", "CANCELLED":
		return s, true
	default:
		return "", false
	}
}

func scanOrderRows(rows *sql.Rows) ([]orderRow, error) {
	orders := make([]orderRow, 0)
	for rows.Next() {
		var o orderRow
		if err := rows.Scan(
			&o.id, &o.userID, &o.customerName, &o.phoneMasked, &o.addressText,
			&o.lat, &o.lng, &o.distanceKm, &o.deliveryFee, &o.subtotal, &o.total,
			&o.status, &o.createdAt,
		); err != nil {
			_ = rows.Close()
			return nil, err
		}
		orders = append(orders, o)
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return nil, err
	}
	_ = rows.Close()
	return orders, nil
}

func (s *orderService) orderViewsFromRows(orders []orderRow) ([]orderView, error) {
	out := make([]orderView, 0, len(orders))
	for _, o := range orders {
		items, err := s.loadOrderItems(o.id)
		if err != nil {
			return nil, err
		}
		out = append(out, customerOrderView(
			o.id, o.userID, o.customerName, o.phoneMasked, o.addressText,
			o.lat, o.lng, o.distanceKm,
			o.deliveryFee, o.subtotal, o.total,
			o.status, o.createdAt, items,
		))
	}
	return out, nil
}

// adminOrderViewsFromRows builds Order Desk rows: STT (1-based FIFO), tên, SĐT (masked),
// địa chỉ, km, thời gian. Orders store phone_masked only — no full phone_e164 to expose.
func (s *orderService) adminOrderViewsFromRows(orders []orderRow) ([]orderView, error) {
	out := make([]orderView, 0, len(orders))
	for i, o := range orders {
		items, err := s.loadOrderItems(o.id)
		if err != nil {
			return nil, err
		}
		v := customerOrderView(
			o.id, o.userID, o.customerName, o.phoneMasked, o.addressText,
			o.lat, o.lng, o.distanceKm,
			o.deliveryFee, o.subtotal, o.total,
			o.status, o.createdAt, items,
		)
		v.Stt = i + 1
		out = append(out, v)
	}
	return out, nil
}

func (s *orderService) loadOrderItems(orderID string) ([]orderItemView, error) {
	rows, err := s.db.Query(`
		SELECT id, product_id, product_sku, product_name, unit_price, qty, line_total
		FROM order_items WHERE order_id = ? ORDER BY id`, orderID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]orderItemView, 0)
	for rows.Next() {
		var it orderItemView
		if err := rows.Scan(
			&it.ID, &it.ProductID, &it.ProductSKU, &it.ProductName,
			&it.UnitPrice, &it.Qty, &it.LineTotal,
		); err != nil {
			return nil, err
		}
		items = append(items, it)
	}
	return items, rows.Err()
}
