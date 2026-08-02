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

const (
	movementIN     = "IN"
	movementOUT    = "OUT"
	movementADJUST = "ADJUST"
	refTypeManual  = "MANUAL"
)

type inventoryService struct {
	db *sql.DB
}

type stockItem struct {
	ProductID    string `json:"product_id"`
	SKU          string `json:"sku"`
	Name         string `json:"name"`
	OnHand       int64  `json:"on_hand"`
	CostPrice    int64  `json:"cost_price"`
	ReorderLevel int64  `json:"reorder_level"`
	UpdatedAt    string `json:"updated_at"`
}

type stockMovement struct {
	ID           string  `json:"id"`
	ProductID    string  `json:"product_id"`
	MovementType string  `json:"movement_type"`
	Qty          int64   `json:"qty"`
	Delta        int64   `json:"delta"` // signed effect on on_hand
	UnitCost     *int64  `json:"unit_cost,omitempty"`
	Note         *string `json:"note,omitempty"`
	RefType      *string `json:"ref_type,omitempty"`
	RefID        *string `json:"ref_id,omitempty"`
	CreatedAt    string  `json:"created_at"`
	CreatedBy    *string `json:"created_by,omitempty"`
}

type listStockResponse struct {
	Items []stockItem `json:"items"`
	Count int         `json:"count"`
}

// postMovementBody is POST /v1/admin/inventory — admin IN / OUT / ADJUST.
type postMovementBody struct {
	MovementType string  `json:"movement_type"`
	ProductID    string  `json:"product_id"`
	Qty          *int64  `json:"qty"`           // IN/OUT: required, > 0
	Delta        *int64  `json:"delta"`         // ADJUST: required, != 0
	UnitCost     *int64  `json:"unit_cost"`     // IN: required >= 0; ADJUST: optional cost update
	SKU          string  `json:"sku"`           // required when creating stock on first IN
	Name         string  `json:"name"`          // required when creating stock on first IN
	ReorderLevel *int64  `json:"reorder_level"` // optional on first IN
	Note         *string `json:"note"`
	RefType      string  `json:"ref_type"` // default MANUAL
	RefID        *string `json:"ref_id"`
}

type postMovementResponse struct {
	Item     stockItem     `json:"item"`
	Movement stockMovement `json:"movement"`
}

// handleListStock serves GET /v1/admin/inventory — all stock rows (architecture §4.4).
func (s *inventoryService) handleListStock(w http.ResponseWriter, _ *http.Request) {
	items, err := s.listStock()
	if err != nil {
		slog.Error("list stock", "err", err)
		httpx.Error(w, http.StatusInternalServerError, "INTERNAL", "could not list inventory")
		return
	}
	httpx.JSON(w, http.StatusOK, listStockResponse{Items: items, Count: len(items)})
}

func (s *inventoryService) listStock() ([]stockItem, error) {
	rows, err := s.db.Query(`
		SELECT product_id, sku, name, on_hand, cost_price, reorder_level, updated_at
		FROM stock_items
		ORDER BY name ASC, product_id ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]stockItem, 0)
	for rows.Next() {
		var it stockItem
		if err := rows.Scan(&it.ProductID, &it.SKU, &it.Name, &it.OnHand, &it.CostPrice, &it.ReorderLevel, &it.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, it)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

// handlePostMovement serves POST /v1/admin/inventory — IN / OUT / ADJUST + persist movement.
func (s *inventoryService) handlePostMovement(w http.ResponseWriter, r *http.Request) {
	var body postMovementBody
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&body); err != nil {
		httpx.Error(w, http.StatusBadRequest, "INVALID_JSON", "invalid request body")
		return
	}

	createdBy := optionalHeader(r, "X-User-Id")
	out, err := s.applyMovement(body, createdBy)
	if err != nil {
		var apiErr *movementAPIError
		if errors.As(err, &apiErr) {
			httpx.Error(w, apiErr.status, apiErr.code, apiErr.message)
			return
		}
		slog.Error("apply stock movement", "err", err)
		httpx.Error(w, http.StatusInternalServerError, "INTERNAL", "could not apply stock movement")
		return
	}
	httpx.JSON(w, http.StatusCreated, out)
}

type movementAPIError struct {
	status  int
	code    string
	message string
}

func (e *movementAPIError) Error() string {
	return e.message
}

func movementErr(status int, code, message string) error {
	return &movementAPIError{status: status, code: code, message: message}
}

func (s *inventoryService) applyMovement(body postMovementBody, createdBy string) (postMovementResponse, error) {
	mt := strings.ToUpper(strings.TrimSpace(body.MovementType))
	productID := strings.TrimSpace(body.ProductID)
	if productID == "" {
		return postMovementResponse{}, movementErr(http.StatusBadRequest, "INVALID_PRODUCT", "product_id is required")
	}
	switch mt {
	case movementIN, movementOUT, movementADJUST:
	default:
		return postMovementResponse{}, movementErr(http.StatusBadRequest, "INVALID_TYPE", "movement_type must be IN, OUT, or ADJUST")
	}

	refType := strings.TrimSpace(body.RefType)
	if refType == "" {
		refType = refTypeManual
	}

	var (
		qty      int64
		delta    int64
		unitCost *int64
	)

	switch mt {
	case movementIN:
		if body.Qty == nil || *body.Qty <= 0 {
			return postMovementResponse{}, movementErr(http.StatusBadRequest, "INVALID_QTY", "qty must be > 0 for IN")
		}
		if body.UnitCost == nil || *body.UnitCost < 0 {
			return postMovementResponse{}, movementErr(http.StatusBadRequest, "INVALID_UNIT_COST", "unit_cost is required and must be >= 0 for IN")
		}
		qty = *body.Qty
		delta = qty
		uc := *body.UnitCost
		unitCost = &uc

	case movementOUT:
		if body.Qty == nil || *body.Qty <= 0 {
			return postMovementResponse{}, movementErr(http.StatusBadRequest, "INVALID_QTY", "qty must be > 0 for OUT")
		}
		qty = *body.Qty
		delta = -qty
		// T7.2.1: client unit_cost on OUT is ignored; COGS is snapshotted from stock_items.cost_price.

	case movementADJUST:
		if body.Delta == nil || *body.Delta == 0 {
			return postMovementResponse{}, movementErr(http.StatusBadRequest, "INVALID_DELTA", "delta is required and must be non-zero for ADJUST")
		}
		delta = *body.Delta
		if delta > 0 {
			qty = delta
		} else {
			qty = -delta
		}
		if body.UnitCost != nil {
			if *body.UnitCost < 0 {
				return postMovementResponse{}, movementErr(http.StatusBadRequest, "INVALID_UNIT_COST", "unit_cost must be >= 0")
			}
			uc := *body.UnitCost
			unitCost = &uc
		}
	}

	tx, err := s.db.Begin()
	if err != nil {
		return postMovementResponse{}, err
	}
	defer func() { _ = tx.Rollback() }()

	item, err := loadStockItemTx(tx, productID)
	exists := err == nil
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return postMovementResponse{}, err
	}

	now := time.Now().UTC().Format(time.RFC3339Nano)

	switch mt {
	case movementIN:
		if !exists {
			sku := strings.TrimSpace(body.SKU)
			name := strings.TrimSpace(body.Name)
			if sku == "" || name == "" {
				return postMovementResponse{}, movementErr(http.StatusBadRequest, "INVALID_PRODUCT", "sku and name are required when creating stock via IN")
			}
			reorder := int64(0)
			if body.ReorderLevel != nil {
				if *body.ReorderLevel < 0 {
					return postMovementResponse{}, movementErr(http.StatusBadRequest, "INVALID_REORDER", "reorder_level must be >= 0")
				}
				reorder = *body.ReorderLevel
			}
			item = stockItem{
				ProductID:    productID,
				SKU:          sku,
				Name:         name,
				OnHand:       qty,
				CostPrice:    *unitCost,
				ReorderLevel: reorder,
				UpdatedAt:    now,
			}
			if err := insertStockItemTx(tx, item); err != nil {
				if isUniqueViolation(err) {
					return postMovementResponse{}, movementErr(http.StatusConflict, "SKU_CONFLICT", "sku already exists")
				}
				return postMovementResponse{}, err
			}
		} else {
			// Current nhập cost = latest IN unit_cost (architecture: giá nhập hiện tại).
			item.OnHand += qty
			item.CostPrice = *unitCost
			item.UpdatedAt = now
			if err := updateStockItemTx(tx, item); err != nil {
				return postMovementResponse{}, err
			}
		}

	case movementOUT:
		if !exists {
			return postMovementResponse{}, movementErr(http.StatusNotFound, "NOT_FOUND", "stock item not found")
		}
		// T7.2.1: persist COGS snapshot at movement time; later cost_price changes must not rewrite it.
		snap := snapshotOUTCost(item.CostPrice)
		unitCost = &snap
		item.OnHand -= qty
		item.UpdatedAt = now
		if err := updateStockItemTx(tx, item); err != nil {
			return postMovementResponse{}, err
		}

	case movementADJUST:
		if !exists {
			return postMovementResponse{}, movementErr(http.StatusNotFound, "NOT_FOUND", "stock item not found; use IN to create")
		}
		item.OnHand += delta
		if unitCost != nil {
			item.CostPrice = *unitCost
		}
		item.UpdatedAt = now
		if err := updateStockItemTx(tx, item); err != nil {
			return postMovementResponse{}, err
		}
	}

	movID := uuid.NewString()
	note := trimOpt(body.Note)
	refID := trimOpt(body.RefID)
	var createdByPtr *string
	if createdBy != "" {
		createdByPtr = &createdBy
	}
	refTypePtr := refType

	if err := insertMovementTx(tx, movID, productID, mt, qty, unitCost, note, &refTypePtr, refID, now, createdByPtr); err != nil {
		return postMovementResponse{}, err
	}

	if err := tx.Commit(); err != nil {
		return postMovementResponse{}, err
	}

	return postMovementResponse{
		Item: item,
		Movement: stockMovement{
			ID:           movID,
			ProductID:    productID,
			MovementType: mt,
			Qty:          qty,
			Delta:        delta,
			UnitCost:     unitCost,
			Note:         note,
			RefType:      &refTypePtr,
			RefID:        refID,
			CreatedAt:    now,
			CreatedBy:    createdByPtr,
		},
	}, nil
}

func loadStockItemTx(tx *sql.Tx, productID string) (stockItem, error) {
	var it stockItem
	err := tx.QueryRow(`
		SELECT product_id, sku, name, on_hand, cost_price, reorder_level, updated_at
		FROM stock_items WHERE product_id = ?`, productID).
		Scan(&it.ProductID, &it.SKU, &it.Name, &it.OnHand, &it.CostPrice, &it.ReorderLevel, &it.UpdatedAt)
	return it, err
}

func insertStockItemTx(tx *sql.Tx, it stockItem) error {
	_, err := tx.Exec(`
		INSERT INTO stock_items (product_id, sku, name, on_hand, cost_price, reorder_level, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		it.ProductID, it.SKU, it.Name, it.OnHand, it.CostPrice, it.ReorderLevel, it.UpdatedAt)
	return err
}

func updateStockItemTx(tx *sql.Tx, it stockItem) error {
	res, err := tx.Exec(`
		UPDATE stock_items
		SET on_hand = ?, cost_price = ?, reorder_level = ?, updated_at = ?
		WHERE product_id = ?`,
		it.OnHand, it.CostPrice, it.ReorderLevel, it.UpdatedAt, it.ProductID)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return fmt.Errorf("stock item %s not updated", it.ProductID)
	}
	return nil
}

func insertMovementTx(
	tx *sql.Tx,
	id, productID, mt string,
	qty int64,
	unitCost *int64,
	note, refType, refID *string,
	createdAt string,
	createdBy *string,
) error {
	// OUT (sale/export) must always carry a non-null COGS snapshot for report profit (T7.2.1).
	if mt == movementOUT && unitCost == nil {
		return fmt.Errorf("OUT movement requires unit_cost snapshot")
	}
	_, err := tx.Exec(`
		INSERT INTO stock_movements
			(id, product_id, movement_type, qty, unit_cost, note, ref_type, ref_id, created_at, created_by)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		id, productID, mt, qty, unitCost, note, refType, refID, createdAt, createdBy)
	return err
}

// snapshotOUTCost returns the VND unit_cost to persist on an OUT movement (T7.2.1).
// Taken from stock_items.cost_price at movement time; never from client body or later updates.
func snapshotOUTCost(costPrice int64) int64 {
	if costPrice < 0 {
		return 0
	}
	return costPrice
}

func optionalHeader(r *http.Request, name string) string {
	return strings.TrimSpace(r.Header.Get(name))
}

func trimOpt(s *string) *string {
	if s == nil {
		return nil
	}
	v := strings.TrimSpace(*s)
	if v == "" {
		return nil
	}
	return &v
}

func isUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "unique") || strings.Contains(msg, "constraint")
}
