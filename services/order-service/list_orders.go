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
	// customerPhone is the full number for admin screens; empty for orders
	// placed before the column existed (backfilled lazily, see fillCustomerPhones).
	customerPhone                string
	lat, lng, distanceKm         float64
	deliveryFee, subtotal, total int64
	status, createdAt            string
	completedAt, cancelledAt     string
	paymentType                  string
	amountPaid                   int64
}

// orderListColumns is the shared SELECT list for the listing queries so the row
// scanner and the queries can never drift apart.
const orderListColumns = `id, user_id, customer_name, phone_masked, customer_phone, address_text,
	       lat, lng, distance_km, delivery_fee, subtotal, total,
	       status, created_at,
	       COALESCE(completed_at, ''), COALESCE(cancelled_at, ''),
	       COALESCE(payment_type, ''), COALESCE(amount_paid, 0)`

// handleListMyOrders serves GET /v1/orders/me — customer's own orders with PII masked.
func (s *orderService) handleListMyOrders(w http.ResponseWriter, r *http.Request) {
	userID, _, ok := requireCustomerIdentity(w, r)
	if !ok {
		return
	}

	rows, err := s.db.Query(`
		SELECT `+orderListColumns+`
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

// handleListAdminOrders serves GET /v1/admin/orders — the desk queue and the
// order history behind one endpoint.
//
// ?status= is PENDING (default, matching the Order Desk), COMPLETED, CANCELLED
// or ALL. Sort follows the intent rather than the endpoint: PENDING is a queue,
// so it stays FIFO (oldest first, numbered by `stt`); anything else is history
// and comes back newest first, unnumbered.
// Gateway mounts under /v1/admin/* with role=admin RBAC.
func (s *orderService) handleListAdminOrders(w http.ResponseWriter, r *http.Request) {
	status, ok := parseAdminOrderStatusFilter(r.URL.Query().Get("status"))
	if !ok {
		httpx.Error(w, http.StatusBadRequest, "VALIDATION_ERROR",
			"status must be PENDING, COMPLETED, CANCELLED, or ALL")
		return
	}

	query := `SELECT ` + orderListColumns + ` FROM orders`
	args := make([]any, 0, 1)
	if status != adminOrderStatusAll {
		query += ` WHERE status = ?`
		args = append(args, status)
	}
	fifo := status == orderStatusPending
	if fifo {
		query += ` ORDER BY created_at ASC`
	} else {
		query += ` ORDER BY created_at DESC`
	}

	rows, err := s.db.Query(query, args...)
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

	s.fillCustomerPhones(r.Context(), orders)

	out, err := s.adminOrderViewsFromRows(orders, fifo)
	if err != nil {
		slog.Error("list admin orders", "err", err)
		httpx.Error(w, http.StatusInternalServerError, "INTERNAL", "could not list orders")
		return
	}

	httpx.JSON(w, http.StatusOK, map[string]any{"orders": out, "status": status, "count": len(out)})
}

// handleGetAdminOrder serves GET /v1/admin/orders/{id} — single order for desk /
// navigation. Response includes the delivery destination `lat`/`lng` (WGS84)
// stored at place time, the full `customer_phone`, and the payment settlement
// once the order is completed. Gateway mounts under /v1/admin/* with role=admin RBAC.
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

	rows := []orderRow{o}
	s.fillCustomerPhones(r.Context(), rows)

	httpx.JSON(w, http.StatusOK, adminOrderView(rows[0], items))
}

func (s *orderService) loadOrderByID(id string) (orderRow, error) {
	var o orderRow
	err := s.db.QueryRow(`
		SELECT id, user_id, customer_name, phone_hash, phone_masked, customer_phone, address_text,
		       lat, lng, distance_km, delivery_fee, subtotal, total,
		       status, created_at,
		       COALESCE(completed_at, ''), COALESCE(cancelled_at, ''),
		       COALESCE(payment_type, ''), COALESCE(amount_paid, 0)
		FROM orders WHERE id = ?`, id).Scan(
		&o.id, &o.userID, &o.customerName, &o.phoneHash, &o.phoneMasked, &o.customerPhone, &o.addressText,
		&o.lat, &o.lng, &o.distanceKm, &o.deliveryFee, &o.subtotal, &o.total,
		&o.status, &o.createdAt,
		&o.completedAt, &o.cancelledAt, &o.paymentType, &o.amountPaid,
	)
	return o, err
}

// Order statuses as persisted in orders.status.
const (
	orderStatusPending   = "PENDING"
	orderStatusCompleted = "COMPLETED"
	orderStatusCancelled = "CANCELLED"
	// adminOrderStatusAll is a filter value only — never a stored status.
	adminOrderStatusAll = "ALL"
)

// parseAdminOrderStatusFilter returns the status filter. Empty → PENDING (order desk default).
func parseAdminOrderStatusFilter(raw string) (status string, ok bool) {
	s := strings.TrimSpace(strings.ToUpper(raw))
	if s == "" {
		return orderStatusPending, true
	}
	switch s {
	case orderStatusPending, orderStatusCompleted, orderStatusCancelled, adminOrderStatusAll:
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
			&o.id, &o.userID, &o.customerName, &o.phoneMasked, &o.customerPhone, &o.addressText,
			&o.lat, &o.lng, &o.distanceKm, &o.deliveryFee, &o.subtotal, &o.total,
			&o.status, &o.createdAt,
			&o.completedAt, &o.cancelledAt, &o.paymentType, &o.amountPaid,
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
		v := customerOrderView(
			o.id, o.userID, o.customerName, o.phoneMasked, o.addressText,
			o.lat, o.lng, o.distanceKm,
			o.deliveryFee, o.subtotal, o.total,
			o.status, o.createdAt, items,
		)
		v.CompletedAt = o.completedAt
		v.CancelledAt = o.cancelledAt
		out = append(out, v)
	}
	return out, nil
}

// adminOrderViewsFromRows builds Order Desk / history rows. [fifo] numbers the
// rows with `stt` (1-based, oldest first); a history listing is ordered newest
// first, where a FIFO position would be misleading.
func (s *orderService) adminOrderViewsFromRows(orders []orderRow, fifo bool) ([]orderView, error) {
	out := make([]orderView, 0, len(orders))
	for i, o := range orders {
		items, err := s.loadOrderItems(o.id)
		if err != nil {
			return nil, err
		}
		v := adminOrderView(o, items)
		if fifo {
			v.Stt = i + 1
		}
		out = append(out, v)
	}
	return out, nil
}

func (s *orderService) loadOrderItems(orderID string) ([]orderItemView, error) {
	rows, err := s.db.Query(`
		SELECT id, product_id, product_sku, product_name, unit_price, qty, line_total, unit_cost
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
			&it.UnitPrice, &it.Qty, &it.LineTotal, &it.UnitCost,
		); err != nil {
			return nil, err
		}
		items = append(items, it)
	}
	return items, rows.Err()
}
