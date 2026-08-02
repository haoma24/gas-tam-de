package main

import (
	"database/sql"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"gas-tam-de/pkg/httpx"

	"github.com/google/uuid"
)

type orderService struct {
	db        *sql.DB
	geo       geoChecker
	catalog   productCatalog
	billing   billingRecorder
	inventory stockReserver
	bus       orderPublisher
}

type createOrderItemBody struct {
	ProductID string `json:"product_id"`
	Qty       int    `json:"qty"`
}

type createOrderBody struct {
	CustomerName string                `json:"customer_name"`
	AddressText  string                `json:"address_text"`
	Lat          float64               `json:"lat"`
	Lng          float64               `json:"lng"`
	Items        []createOrderItemBody `json:"items"`
}

type orderItemView struct {
	ID          string `json:"id"`
	ProductID   string `json:"product_id"`
	ProductSKU  string `json:"product_sku"`
	ProductName string `json:"product_name"`
	UnitPrice   int64  `json:"unit_price"`
	Qty         int    `json:"qty"`
	LineTotal   int64  `json:"line_total"`
}

type orderView struct {
	// Stt is 1-based FIFO desk sequence (admin list only; omitted for customer APIs).
	Stt          int             `json:"stt,omitempty"`
	ID           string          `json:"id"`
	UserID       string          `json:"user_id"`
	CustomerName string          `json:"customer_name"`
	PhoneMasked  string          `json:"phone_masked"`
	AddressText  string          `json:"address_text"`
	// Lat/Lng are the delivery destination (WGS84) captured at place time —
	// used by admin navigation (US-5.2).
	Lat          float64         `json:"lat"`
	Lng          float64         `json:"lng"`
	DistanceKm   float64         `json:"distance_km"`
	DeliveryFee  int64           `json:"delivery_fee"`
	Subtotal     int64           `json:"subtotal"`
	Total        int64           `json:"total"`
	Status       string          `json:"status"`
	CreatedAt    string          `json:"created_at"`
	Items        []orderItemView `json:"items"`
}

type preparedLine struct {
	productID   string
	productSKU  string
	productName string
	unitPrice   int64
	qty         int
	lineTotal   int64
}

// handleCreateOrder serves POST /v1/orders — validate JWT identity headers,
// items (catalog), geo in-range, delivery fee engine, persist PENDING + publish order.placed.
func (s *orderService) handleCreateOrder(w http.ResponseWriter, r *http.Request) {
	userID, phoneMaskedRaw, ok := requireCustomerIdentity(w, r)
	if !ok {
		return
	}
	phoneMasked := ensurePhoneMasked(phoneMaskedRaw)

	var body createOrderBody
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&body); err != nil {
		httpx.Error(w, http.StatusBadRequest, "INVALID_JSON", "invalid request body")
		return
	}

	customerName := strings.TrimSpace(body.CustomerName)
	addressText := strings.TrimSpace(body.AddressText)
	if customerName == "" {
		httpx.Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "customer_name is required")
		return
	}
	if addressText == "" {
		httpx.Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "address_text is required")
		return
	}
	if err := validateCoords(body.Lat, body.Lng); err != nil {
		httpx.Error(w, http.StatusBadRequest, "INVALID_COORDS", err.Error())
		return
	}
	if len(body.Items) == 0 {
		httpx.Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "items must not be empty")
		return
	}

	qtyByProduct := make(map[string]int, len(body.Items))
	orderedIDs := make([]string, 0, len(body.Items))
	for _, it := range body.Items {
		pid := strings.TrimSpace(it.ProductID)
		if pid == "" {
			httpx.Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "items[].product_id is required")
			return
		}
		if it.Qty < 1 {
			httpx.Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "items[].qty must be >= 1")
			return
		}
		if _, seen := qtyByProduct[pid]; !seen {
			orderedIDs = append(orderedIDs, pid)
		}
		qtyByProduct[pid] += it.Qty
	}

	geo, err := s.geo.Check(r.Context(), body.Lat, body.Lng)
	if err != nil {
		slog.Error("geo check", "err", err)
		httpx.Error(w, http.StatusBadGateway, "GEO_UNAVAILABLE", "could not verify delivery range")
		return
	}
	if !geo.InRange {
		httpx.JSON(w, http.StatusUnprocessableEntity, map[string]any{
			"error": map[string]any{
				"code":          "OUT_OF_RANGE",
				"message":       "delivery address is outside the store radius",
				"distance_km":   geo.DistanceKm,
				"max_radius_km": geo.MaxRadiusKm,
			},
		})
		return
	}

	products, err := s.catalog.ListActive(r.Context())
	if err != nil {
		slog.Error("catalog list", "err", err)
		httpx.Error(w, http.StatusBadGateway, "CATALOG_UNAVAILABLE", "could not load products")
		return
	}
	byID := make(map[string]catalogProduct, len(products))
	for _, p := range products {
		byID[p.ID] = p
	}

	lines := make([]preparedLine, 0, len(orderedIDs))
	var subtotal int64
	for _, pid := range orderedIDs {
		qty := qtyByProduct[pid]
		p, found := byID[pid]
		if !found || !p.Active {
			httpx.Error(w, http.StatusBadRequest, "PRODUCT_NOT_FOUND", "product not found or inactive: "+pid)
			return
		}
		if p.SalePrice < 0 {
			httpx.Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid product price for "+pid)
			return
		}
		lineTotal := p.SalePrice * int64(qty)
		lines = append(lines, preparedLine{
			productID:   p.ID,
			productSKU:  p.SKU,
			productName: p.Name,
			unitPrice:   p.SalePrice,
			qty:         qty,
			lineTotal:   lineTotal,
		})
		subtotal += lineTotal
	}

	deliveryFee, err := computeDeliveryFee(s.db, geo.DistanceKm)
	if err != nil {
		slog.Error("compute delivery fee", "err", err)
		httpx.Error(w, http.StatusInternalServerError, "INTERNAL", "could not compute delivery fee")
		return
	}
	total := subtotal + deliveryFee

	now := time.Now().UTC().Format(time.RFC3339)
	orderID := uuid.NewString()
	// phone_hash is not on JWT; store uid-scoped placeholder (no raw phone available).
	phoneHash := "uid:" + userID

	tx, err := s.db.Begin()
	if err != nil {
		slog.Error("begin create order", "err", err)
		httpx.Error(w, http.StatusInternalServerError, "INTERNAL", "could not create order")
		return
	}
	defer func() { _ = tx.Rollback() }()

	_, err = tx.Exec(`
		INSERT INTO orders (
			id, user_id, customer_name, phone_hash, phone_masked,
			address_text, lat, lng, distance_km, delivery_fee,
			subtotal, total, status, created_at, completed_at, cancelled_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 'PENDING', ?, NULL, NULL)`,
		orderID, userID, customerName, phoneHash, phoneMasked,
		addressText, body.Lat, body.Lng, geo.DistanceKm, deliveryFee,
		subtotal, total, now,
	)
	if err != nil {
		slog.Error("insert order", "err", err)
		httpx.Error(w, http.StatusInternalServerError, "INTERNAL", "could not create order")
		return
	}

	itemViews := make([]orderItemView, 0, len(lines))
	for _, line := range lines {
		itemID := uuid.NewString()
		_, err = tx.Exec(`
			INSERT INTO order_items (
				id, order_id, product_id, product_sku, product_name, unit_price, qty, line_total
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
			itemID, orderID, line.productID, line.productSKU, line.productName,
			line.unitPrice, line.qty, line.lineTotal,
		)
		if err != nil {
			slog.Error("insert order item", "err", err)
			httpx.Error(w, http.StatusInternalServerError, "INTERNAL", "could not create order")
			return
		}
		itemViews = append(itemViews, orderItemView{
			ID:          itemID,
			ProductID:   line.productID,
			ProductSKU:  line.productSKU,
			ProductName: line.productName,
			UnitPrice:   line.unitPrice,
			Qty:         line.qty,
			LineTotal:   line.lineTotal,
		})
	}

	if err := tx.Commit(); err != nil {
		slog.Error("commit create order", "err", err)
		httpx.Error(w, http.StatusInternalServerError, "INTERNAL", "could not create order")
		return
	}

	if s.inventory != nil {
		lines := make([]stockLine, 0, len(itemViews))
		for _, it := range itemViews {
			lines = append(lines, stockLine{
				ProductID: it.ProductID,
				SKU:       it.ProductSKU,
				Qty:       int64(it.Qty),
			})
		}
		if err := s.inventory.Reserve(r.Context(), orderID, lines); err != nil {
			slog.Error("inventory reserve", "order_id", orderID, "err", err)
			// Best-effort rollback of PENDING order so stock stays consistent.
			_, _ = s.db.Exec(`UPDATE orders SET status = 'CANCELLED', cancelled_at = ? WHERE id = ?`,
				time.Now().UTC().Format(time.RFC3339Nano), orderID)
			msg := "Không đủ tồn kho cho một hoặc nhiều sản phẩm."
			if strings.Contains(err.Error(), "insufficient") || strings.Contains(err.Error(), "INSUFFICIENT") {
				httpx.Error(w, http.StatusConflict, "INSUFFICIENT_STOCK", msg)
				return
			}
			httpx.Error(w, http.StatusBadGateway, "INVENTORY_UNAVAILABLE", "Không trừ được tồn kho. Thử lại.")
			return
		}
	}

	// Publish after commit; failures are logged only (order already persisted).
	s.publishOrderPlaced(orderPlacedEvent{
		OrderID:    orderID,
		Total:      total,
		DistanceKm: geo.DistanceKm,
		CreatedAt:  now,
	})

	httpx.JSON(w, http.StatusCreated, customerOrderView(
		orderID, userID, customerName, phoneMasked, addressText,
		body.Lat, body.Lng, geo.DistanceKm,
		deliveryFee, subtotal, total,
		"PENDING", now, itemViews,
	))
}

func requireCustomerIdentity(w http.ResponseWriter, r *http.Request) (userID, phoneMasked string, ok bool) {
	userID = strings.TrimSpace(r.Header.Get("X-User-Id"))
	role := strings.TrimSpace(r.Header.Get("X-User-Role"))
	phoneMasked = strings.TrimSpace(r.Header.Get("X-Phone-Masked"))

	if userID == "" {
		httpx.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "missing X-User-Id (gateway JWT required)")
		return "", "", false
	}
	if role != "customer" {
		httpx.Error(w, http.StatusForbidden, "FORBIDDEN", "customer role required")
		return "", "", false
	}
	if phoneMasked == "" {
		httpx.Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "missing X-Phone-Masked from customer session")
		return "", "", false
	}
	return userID, phoneMasked, true
}

func validateCoords(lat, lng float64) error {
	if lat < -90 || lat > 90 {
		return errInvalidLat
	}
	if lng < -180 || lng > 180 {
		return errInvalidLng
	}
	return nil
}

var (
	errInvalidLat = errString("lat must be between -90 and 90")
	errInvalidLng = errString("lng must be between -180 and 180")
)

type errString string

func (e errString) Error() string { return string(e) }
