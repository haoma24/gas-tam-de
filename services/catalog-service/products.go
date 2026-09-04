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
	"github.com/google/uuid"
)

type catalogService struct {
	db  *sql.DB
	bus productUpdatedPublisher
}

type product struct {
	ID          string   `json:"id"`
	SKU         string   `json:"sku"`
	Name        string   `json:"name"`
	Description *string  `json:"description,omitempty"`
	Unit        string   `json:"unit"`
	SalePrice   int64    `json:"sale_price"`
	Active      bool     `json:"active"`
	ImageURL    *string  `json:"image_url,omitempty"`
	ImageURLs   []string `json:"image_urls,omitempty"`
	CreatedAt   string   `json:"created_at"`
	UpdatedAt   string   `json:"updated_at"`
}

type createProductBody struct {
	SKU         string   `json:"sku"`
	Name        string   `json:"name"`
	Description *string  `json:"description"`
	Unit        string   `json:"unit"`
	SalePrice   *int64   `json:"sale_price"`
	Active      *bool    `json:"active"`
	ImageURL    *string  `json:"image_url"`
	ImageURLs   []string `json:"image_urls"`
}

type patchProductBody struct {
	SKU         *string   `json:"sku"`
	Name        *string   `json:"name"`
	Description *string   `json:"description"`
	Unit        *string   `json:"unit"`
	SalePrice   *int64    `json:"sale_price"`
	Active      *bool     `json:"active"`
	ImageURL    *string   `json:"image_url"`
	ImageURLs   *[]string `json:"image_urls"`
}

type productListResponse struct {
	Items []product `json:"items"`
}

// handleListActiveProducts serves GET /v1/products — active catalog for customers
// (public or authenticated; authz optional at gateway).
// @Summary List active products
// @Description Returns products currently available to customers.
// @Tags Products
// @Success 200 {object} productListResponse
// @Failure 500 {object} httpx.ErrorResponse
// @Router /products [get]
func (s *catalogService) handleListActiveProducts(w http.ResponseWriter, r *http.Request) {
	rows, err := s.db.Query(`
		SELECT id, sku, name, description, unit, sale_price, active, image_url, created_at, updated_at
		FROM products
		WHERE active = 1
		ORDER BY created_at DESC`)
	if err != nil {
		slog.Error("list active products", "err", err)
		httpx.Error(w, http.StatusInternalServerError, "INTERNAL", "could not list products")
		return
	}
	defer rows.Close()

	items, err := collectProducts(rows)
	if err != nil {
		slog.Error("list active products", "err", err)
		httpx.Error(w, http.StatusInternalServerError, "INTERNAL", "could not list products")
		return
	}
	httpx.JSON(w, http.StatusOK, productListResponse{Items: items})
}

// handleListAdminProducts lists the complete catalog for administrators.
// @Summary List all products
// @Tags Admin - Products
// @Security BearerAuth
// @Success 200 {object} productListResponse
// @Failure 401 {object} httpx.ErrorResponse
// @Failure 403 {object} httpx.ErrorResponse
// @Failure 500 {object} httpx.ErrorResponse
// @Router /admin/products [get]
func (s *catalogService) handleListAdminProducts(w http.ResponseWriter, r *http.Request) {
	rows, err := s.db.Query(`
		SELECT id, sku, name, description, unit, sale_price, active, image_url, created_at, updated_at
		FROM products
		ORDER BY created_at DESC`)
	if err != nil {
		slog.Error("list products", "err", err)
		httpx.Error(w, http.StatusInternalServerError, "INTERNAL", "could not list products")
		return
	}
	defer rows.Close()

	items, err := collectProducts(rows)
	if err != nil {
		slog.Error("list products", "err", err)
		httpx.Error(w, http.StatusInternalServerError, "INTERNAL", "could not list products")
		return
	}
	httpx.JSON(w, http.StatusOK, productListResponse{Items: items})
}

func collectProducts(rows *sql.Rows) ([]product, error) {
	items := make([]product, 0)
	for rows.Next() {
		p, err := scanProduct(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, p)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

// handleGetAdminProduct returns one catalog product.
// @Summary Get a product
// @Tags Admin - Products
// @Security BearerAuth
// @Param id path string true "Product ID"
// @Success 200 {object} product
// @Failure 401 {object} httpx.ErrorResponse
// @Failure 403 {object} httpx.ErrorResponse
// @Failure 404 {object} httpx.ErrorResponse
// @Router /admin/products/{id} [get]
func (s *catalogService) handleGetAdminProduct(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSpace(chi.URLParam(r, "id"))
	if id == "" {
		httpx.Error(w, http.StatusBadRequest, "INVALID_ID", "product id is required")
		return
	}

	p, err := s.loadProduct(id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			httpx.Error(w, http.StatusNotFound, "NOT_FOUND", "product not found")
			return
		}
		slog.Error("get product", "err", err)
		httpx.Error(w, http.StatusInternalServerError, "INTERNAL", "could not load product")
		return
	}
	httpx.JSON(w, http.StatusOK, p)
}

// handleCreateProduct creates a catalog product.
// @Summary Create a product
// @Tags Admin - Products
// @Security BearerAuth
// @Param body body createProductBody true "Product"
// @Success 201 {object} product
// @Failure 400 {object} httpx.ErrorResponse
// @Failure 401 {object} httpx.ErrorResponse
// @Failure 403 {object} httpx.ErrorResponse
// @Failure 409 {object} httpx.ErrorResponse
// @Router /admin/products [post]
func (s *catalogService) handleCreateProduct(w http.ResponseWriter, r *http.Request) {
	var body createProductBody
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&body); err != nil {
		httpx.Error(w, http.StatusBadRequest, "INVALID_JSON", "invalid request body")
		return
	}

	sku := strings.TrimSpace(body.SKU)
	name := strings.TrimSpace(body.Name)
	unit := strings.TrimSpace(body.Unit)
	if unit == "" {
		unit = "binh"
	}
	if sku == "" || name == "" {
		httpx.Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "sku and name are required")
		return
	}
	if body.SalePrice == nil {
		httpx.Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "sale_price is required")
		return
	}
	if *body.SalePrice < 0 {
		httpx.Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "sale_price must be >= 0")
		return
	}

	active := true
	if body.Active != nil {
		active = *body.Active
	}

	now := time.Now().UTC().Format(time.RFC3339)
	id := uuid.NewString()
	changedBy := optionalHeader(r, "X-User-Id")

	tx, err := s.db.Begin()
	if err != nil {
		slog.Error("begin create product", "err", err)
		httpx.Error(w, http.StatusInternalServerError, "INTERNAL", "could not create product")
		return
	}
	defer func() { _ = tx.Rollback() }()

	_, err = tx.Exec(`
		INSERT INTO products (id, sku, name, description, unit, sale_price, active, image_url, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		id, sku, name, nullStringPtr(body.Description), unit, *body.SalePrice, boolToInt(active),
		nullString(imageStorage(body.ImageURL, body.ImageURLs)), now, now,
	)
	if err != nil {
		if isUniqueViolation(err) {
			httpx.Error(w, http.StatusConflict, "SKU_EXISTS", "sku already exists")
			return
		}
		slog.Error("insert product", "err", err)
		httpx.Error(w, http.StatusInternalServerError, "INTERNAL", "could not create product")
		return
	}

	if err := insertPriceHistory(tx, id, *body.SalePrice, now, changedBy); err != nil {
		slog.Error("insert price history", "err", err)
		httpx.Error(w, http.StatusInternalServerError, "INTERNAL", "could not create product")
		return
	}

	if err := tx.Commit(); err != nil {
		slog.Error("commit create product", "err", err)
		httpx.Error(w, http.StatusInternalServerError, "INTERNAL", "could not create product")
		return
	}

	p, err := s.loadProduct(id)
	if err != nil {
		slog.Error("reload created product", "err", err)
		httpx.Error(w, http.StatusInternalServerError, "INTERNAL", "product created but could not reload")
		return
	}
	s.publishProductUpdated(p)
	httpx.JSON(w, http.StatusCreated, p)
}

// handlePatchProduct updates selected product fields.
// @Summary Update a product
// @Tags Admin - Products
// @Security BearerAuth
// @Param id path string true "Product ID"
// @Param body body patchProductBody true "Fields to update"
// @Success 200 {object} product
// @Failure 400 {object} httpx.ErrorResponse
// @Failure 401 {object} httpx.ErrorResponse
// @Failure 403 {object} httpx.ErrorResponse
// @Failure 404 {object} httpx.ErrorResponse
// @Failure 409 {object} httpx.ErrorResponse
// @Router /admin/products/{id} [patch]
func (s *catalogService) handlePatchProduct(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSpace(chi.URLParam(r, "id"))
	if id == "" {
		httpx.Error(w, http.StatusBadRequest, "INVALID_ID", "product id is required")
		return
	}

	var body patchProductBody
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&body); err != nil {
		httpx.Error(w, http.StatusBadRequest, "INVALID_JSON", "invalid request body")
		return
	}
	if body.SKU == nil && body.Name == nil && body.Description == nil && body.Unit == nil &&
		body.SalePrice == nil && body.Active == nil && body.ImageURL == nil && body.ImageURLs == nil {
		httpx.Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "at least one field is required")
		return
	}
	if body.SalePrice != nil && *body.SalePrice < 0 {
		httpx.Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "sale_price must be >= 0")
		return
	}
	if body.SKU != nil && strings.TrimSpace(*body.SKU) == "" {
		httpx.Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "sku cannot be empty")
		return
	}
	if body.Name != nil && strings.TrimSpace(*body.Name) == "" {
		httpx.Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "name cannot be empty")
		return
	}
	if body.Unit != nil && strings.TrimSpace(*body.Unit) == "" {
		httpx.Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "unit cannot be empty")
		return
	}

	existing, err := s.loadProduct(id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			httpx.Error(w, http.StatusNotFound, "NOT_FOUND", "product not found")
			return
		}
		slog.Error("load product for patch", "err", err)
		httpx.Error(w, http.StatusInternalServerError, "INTERNAL", "could not update product")
		return
	}

	sku := existing.SKU
	name := existing.Name
	unit := existing.Unit
	salePrice := existing.SalePrice
	active := existing.Active
	description := existing.Description
	imageURL := existing.ImageURL
	imageURLs := existing.ImageURLs

	if body.SKU != nil {
		sku = strings.TrimSpace(*body.SKU)
	}
	if body.Name != nil {
		name = strings.TrimSpace(*body.Name)
	}
	if body.Unit != nil {
		unit = strings.TrimSpace(*body.Unit)
	}
	if body.SalePrice != nil {
		salePrice = *body.SalePrice
	}
	if body.Active != nil {
		active = *body.Active
	}
	if body.Description != nil {
		description = body.Description
	}
	if body.ImageURL != nil {
		imageURL = body.ImageURL
		imageURLs = []string{strings.TrimSpace(*body.ImageURL)}
	}
	if body.ImageURLs != nil {
		imageURLs = *body.ImageURLs
		imageURL = nil
		if normalized := normalizeImageURLs(imageURLs); len(normalized) > 0 {
			imageURL = &normalized[0]
			imageURLs = normalized
		}
	}

	now := time.Now().UTC().Format(time.RFC3339)
	changedBy := optionalHeader(r, "X-User-Id")
	priceChanged := body.SalePrice != nil && *body.SalePrice != existing.SalePrice

	tx, err := s.db.Begin()
	if err != nil {
		slog.Error("begin patch product", "err", err)
		httpx.Error(w, http.StatusInternalServerError, "INTERNAL", "could not update product")
		return
	}
	defer func() { _ = tx.Rollback() }()

	res, err := tx.Exec(`
		UPDATE products
		SET sku = ?, name = ?, description = ?, unit = ?, sale_price = ?, active = ?, image_url = ?, updated_at = ?
		WHERE id = ?`,
		sku, name, nullStringPtr(description), unit, salePrice, boolToInt(active), nullString(imageStorage(imageURL, imageURLs)), now, id,
	)
	if err != nil {
		if isUniqueViolation(err) {
			httpx.Error(w, http.StatusConflict, "SKU_EXISTS", "sku already exists")
			return
		}
		slog.Error("update product", "err", err)
		httpx.Error(w, http.StatusInternalServerError, "INTERNAL", "could not update product")
		return
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		httpx.Error(w, http.StatusNotFound, "NOT_FOUND", "product not found")
		return
	}

	if priceChanged {
		if err := insertPriceHistory(tx, id, salePrice, now, changedBy); err != nil {
			slog.Error("insert price history", "err", err)
			httpx.Error(w, http.StatusInternalServerError, "INTERNAL", "could not update product")
			return
		}
	}

	if err := tx.Commit(); err != nil {
		slog.Error("commit patch product", "err", err)
		httpx.Error(w, http.StatusInternalServerError, "INTERNAL", "could not update product")
		return
	}

	p, err := s.loadProduct(id)
	if err != nil {
		slog.Error("reload patched product", "err", err)
		httpx.Error(w, http.StatusInternalServerError, "INTERNAL", "product updated but could not reload")
		return
	}
	s.publishProductUpdated(p)
	httpx.JSON(w, http.StatusOK, p)
}

func (s *catalogService) loadProduct(id string) (product, error) {
	row := s.db.QueryRow(`
		SELECT id, sku, name, description, unit, sale_price, active, image_url, created_at, updated_at
		FROM products WHERE id = ?`, id)
	return scanProduct(row)
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanProduct(row rowScanner) (product, error) {
	var (
		p           product
		description sql.NullString
		imageURL    sql.NullString
		activeInt   int
	)
	err := row.Scan(
		&p.ID, &p.SKU, &p.Name, &description, &p.Unit, &p.SalePrice, &activeInt, &imageURL, &p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		return product{}, err
	}
	p.Active = activeInt != 0
	if description.Valid {
		v := description.String
		p.Description = &v
	}
	if imageURL.Valid {
		p.ImageURLs = splitImageURLs(imageURL.String)
		if len(p.ImageURLs) > 0 {
			v := p.ImageURLs[0]
			p.ImageURL = &v
		}
	}
	return p, nil
}

func normalizeImageURLs(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}

func splitImageURLs(raw string) []string {
	raw = strings.ReplaceAll(raw, "\r\n", "\n")
	return normalizeImageURLs(strings.Split(raw, "\n"))
}

func imageStorage(legacy *string, images []string) *string {
	values := normalizeImageURLs(images)
	if len(values) == 0 && legacy != nil {
		values = normalizeImageURLs([]string{*legacy})
	}
	if len(values) == 0 {
		return nil
	}
	joined := strings.Join(values, "\n")
	return &joined
}

func nullString(value *string) any {
	if value == nil || strings.TrimSpace(*value) == "" {
		return nil
	}
	return *value
}

func insertPriceHistory(tx *sql.Tx, productID string, salePrice int64, changedAt, changedBy string) error {
	var by any
	if changedBy == "" {
		by = nil
	} else {
		by = changedBy
	}
	_, err := tx.Exec(`
		INSERT INTO product_price_history (id, product_id, sale_price, changed_at, changed_by)
		VALUES (?, ?, ?, ?, ?)`,
		uuid.NewString(), productID, salePrice, changedAt, by,
	)
	return err
}

func nullStringPtr(s *string) any {
	if s == nil {
		return nil
	}
	v := strings.TrimSpace(*s)
	if v == "" {
		return nil
	}
	return v
}

func boolToInt(v bool) int {
	if v {
		return 1
	}
	return 0
}

func optionalHeader(r *http.Request, name string) string {
	return strings.TrimSpace(r.Header.Get(name))
}

func isUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "unique") || strings.Contains(msg, "constraint failed")
}
