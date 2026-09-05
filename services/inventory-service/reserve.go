package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"gas-tam-de/pkg/httpx"

	"github.com/google/uuid"
)

type stockOpBody struct {
	OrderID string `json:"order_id"`
	Items   []struct {
		ProductID string `json:"product_id"`
		SKU       string `json:"sku"`
		Qty       int64  `json:"qty"`
	} `json:"items"`
}

type stockLevel struct {
	ProductID string `json:"product_id"`
	OnHand    int64  `json:"on_hand"`
}

// handleListStockLevels serves GET /v1/stock/levels — public on_hand for catalog UI.
func (s *inventoryService) handleListStockLevels(w http.ResponseWriter, _ *http.Request) {
	rows, err := s.db.Query(`SELECT product_id, on_hand FROM stock_items`)
	if err != nil {
		slog.Error("list stock levels", "err", err)
		httpx.Error(w, http.StatusInternalServerError, "INTERNAL", "could not list stock")
		return
	}
	defer rows.Close()
	items := make([]stockLevel, 0)
	for rows.Next() {
		var it stockLevel
		if err := rows.Scan(&it.ProductID, &it.OnHand); err != nil {
			httpx.Error(w, http.StatusInternalServerError, "INTERNAL", "could not list stock")
			return
		}
		items = append(items, it)
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"items": items, "count": len(items)})
}

// handleReserveStock serves POST /v1/internal/stock/reserve — OUT for PENDING order.
func (s *inventoryService) handleReserveStock(w http.ResponseWriter, r *http.Request) {
	s.handleStockOp(w, r, true)
}

// handleReleaseStock serves POST /v1/internal/stock/release — IN restore after cancel.
func (s *inventoryService) handleReleaseStock(w http.ResponseWriter, r *http.Request) {
	s.handleStockOp(w, r, false)
}

// stockOpLine is one line of the reserve response: the COGS snapshot the OUT
// movement was written with. order-service persists it on order_items.unit_cost
// so report-service can compute profit = revenue − COGS (architecture §6.7).
type stockOpLine struct {
	ProductID string `json:"product_id"`
	UnitCost  int64  `json:"unit_cost"`
}

func (s *inventoryService) handleStockOp(w http.ResponseWriter, r *http.Request, reserve bool) {
	var body stockOpBody
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&body); err != nil {
		httpx.Error(w, http.StatusBadRequest, "INVALID_JSON", "invalid request body")
		return
	}
	orderID := strings.TrimSpace(body.OrderID)
	if orderID == "" {
		httpx.Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "order_id is required")
		return
	}
	if len(body.Items) == 0 {
		httpx.Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "items required")
		return
	}

	tx, err := s.db.Begin()
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "INTERNAL", "could not update stock")
		return
	}
	defer func() { _ = tx.Rollback() }()

	now := time.Now().UTC().Format(time.RFC3339Nano)
	refType := refTypeOrder
	lines := make([]stockOpLine, 0, len(body.Items))

	for _, it := range body.Items {
		pid := strings.TrimSpace(it.ProductID)
		if pid == "" || it.Qty <= 0 {
			httpx.Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "each item needs product_id and qty > 0")
			return
		}
		if reserve {
			unitCost, err := reserveLineTx(tx, pid, strings.TrimSpace(it.SKU), it.Qty, orderID, refType, now)
			if err != nil {
				if errors.Is(err, errInsufficientStock) {
					httpx.Error(w, http.StatusConflict, "INSUFFICIENT_STOCK", err.Error())
					return
				}
				slog.Error("reserve line", "err", err)
				httpx.Error(w, http.StatusInternalServerError, "INTERNAL", "could not reserve stock")
				return
			}
			lines = append(lines, stockOpLine{ProductID: pid, UnitCost: unitCost})
		} else {
			if err := releaseLineTx(tx, pid, strings.TrimSpace(it.SKU), it.Qty, orderID, refType, now); err != nil {
				slog.Error("release line", "err", err)
				httpx.Error(w, http.StatusInternalServerError, "INTERNAL", "could not release stock")
				return
			}
		}
	}

	if err := tx.Commit(); err != nil {
		httpx.Error(w, http.StatusInternalServerError, "INTERNAL", "could not update stock")
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"ok": true, "order_id": orderID, "items": lines})
}

var errInsufficientStock = fmt.Errorf("insufficient stock")

// reserveLineTx deducts one line and returns the COGS snapshot written on the
// OUT movement, so the caller can hand it back to order-service.
func reserveLineTx(tx *sql.Tx, productID, sku string, qty int64, orderID, refType, now string) (int64, error) {
	item, err := loadStockItemTx(tx, productID)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, fmt.Errorf("%w for product %s (on_hand=0)", errInsufficientStock, productID)
	}
	if err != nil {
		return 0, err
	}
	if item.OnHand < qty {
		return 0, fmt.Errorf("%w for product %s (on_hand=%d, need=%d)", errInsufficientStock, productID, item.OnHand, qty)
	}
	snap := snapshotOUTCost(item.CostPrice)
	item.OnHand -= qty
	item.UpdatedAt = now
	if err := updateStockItemTx(tx, item); err != nil {
		return 0, err
	}
	uc := snap
	note := "reserve order.placed"
	refID := orderID
	rt := refType
	if err := insertMovementTx(tx, uuid.NewString(), productID, movementOUT, qty, &uc, &note, &rt, &refID, now, nil); err != nil {
		return 0, err
	}
	return snap, nil
}

func releaseLineTx(tx *sql.Tx, productID, sku string, qty int64, orderID, refType, now string) error {
	item, err := loadStockItemTx(tx, productID)
	if errors.Is(err, sql.ErrNoRows) {
		if sku == "" {
			sku = productID
		}
		item = stockItem{
			ProductID: productID, SKU: sku, Name: sku,
			OnHand: 0, CostPrice: 0, UpdatedAt: now,
		}
		if err := insertStockItemTx(tx, item); err != nil {
			return err
		}
	} else if err != nil {
		return err
	}
	item.OnHand += qty
	item.UpdatedAt = now
	if err := updateStockItemTx(tx, item); err != nil {
		return err
	}
	uc := item.CostPrice
	note := "release order.cancelled"
	refID := orderID
	rt := refType
	return insertMovementTx(tx, uuid.NewString(), productID, movementIN, qty, &uc, &note, &rt, &refID, now, nil)
}
