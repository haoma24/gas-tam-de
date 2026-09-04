package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"gas-tam-de/pkg/httpx"
	"gas-tam-de/pkg/sqlite"
)

func openTestDB(t *testing.T) *sql.DB {
	t.Helper()
	dir := t.TempDir()
	db, err := sqlite.Open(filepath.Join(dir, "catalog.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := migrate(db); err != nil {
		t.Fatal(err)
	}
	return db
}

func testCatalogRouter(t *testing.T) (*catalogService, http.Handler) {
	t.Helper()
	svc := &catalogService{db: openTestDB(t), bus: noopProductPublisher{}}
	r := httpx.NewRouter("catalog-test")
	r.Get("/v1/products", svc.handleListActiveProducts)
	r.Get("/v1/admin/products", svc.handleListAdminProducts)
	r.Post("/v1/admin/products", svc.handleCreateProduct)
	r.Get("/v1/admin/products/{id}", svc.handleGetAdminProduct)
	r.Patch("/v1/admin/products/{id}", svc.handlePatchProduct)
	return svc, r
}

type recordingProductPublisher struct {
	events []product
}

func (r *recordingProductPublisher) PublishProductUpdated(p product) error {
	r.events = append(r.events, p)
	return nil
}

func TestCreateListGetPatchProduct(t *testing.T) {
	svc, r := testCatalogRouter(t)

	createBody, _ := json.Marshal(map[string]any{
		"sku":         "GAS12",
		"name":        "Gas 12kg",
		"description": "Bình 12kg",
		"unit":        "binh",
		"sale_price":  450000,
		"active":      true,
		"image_urls": []string{
			"https://img.example/cover.jpg",
			"https://img.example/detail.jpg",
		},
	})
	req := httptest.NewRequest(http.MethodPost, "/v1/admin/products", bytes.NewReader(createBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-Id", "admin-1")
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("create status=%d body=%s", rr.Code, rr.Body.String())
	}

	var created product
	if err := json.Unmarshal(rr.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}
	if created.ID == "" || created.SKU != "GAS12" || created.SalePrice != 450000 || !created.Active {
		t.Fatalf("created=%+v", created)
	}
	if created.Description == nil || *created.Description != "Bình 12kg" {
		t.Fatalf("description=%v", created.Description)
	}
	if created.ImageURL == nil || *created.ImageURL != "https://img.example/cover.jpg" || len(created.ImageURLs) != 2 {
		t.Fatalf("images primary=%v gallery=%v", created.ImageURL, created.ImageURLs)
	}

	var histCount int
	if err := svc.db.QueryRow(`SELECT COUNT(*) FROM product_price_history WHERE product_id = ?`, created.ID).Scan(&histCount); err != nil {
		t.Fatal(err)
	}
	if histCount != 1 {
		t.Fatalf("price history after create=%d", histCount)
	}
	var changedBy sql.NullString
	if err := svc.db.QueryRow(`SELECT changed_by FROM product_price_history WHERE product_id = ?`, created.ID).Scan(&changedBy); err != nil {
		t.Fatal(err)
	}
	if !changedBy.Valid || changedBy.String != "admin-1" {
		t.Fatalf("changed_by=%v", changedBy)
	}

	rr = httptest.NewRecorder()
	r.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/v1/admin/products", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("list status=%d body=%s", rr.Code, rr.Body.String())
	}
	var list map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &list); err != nil {
		t.Fatal(err)
	}
	items, _ := list["items"].([]any)
	if len(items) != 1 {
		t.Fatalf("items=%v", list["items"])
	}

	rr = httptest.NewRecorder()
	r.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/v1/admin/products/"+created.ID, nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("get status=%d body=%s", rr.Code, rr.Body.String())
	}

	patchBody, _ := json.Marshal(map[string]any{
		"sale_price": 460000,
		"active":     false,
		"name":       "Gas 12kg (ẩn)",
	})
	req = httptest.NewRequest(http.MethodPatch, "/v1/admin/products/"+created.ID, bytes.NewReader(patchBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-Id", "admin-2")
	rr = httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("patch status=%d body=%s", rr.Code, rr.Body.String())
	}
	var patched product
	if err := json.Unmarshal(rr.Body.Bytes(), &patched); err != nil {
		t.Fatal(err)
	}
	if patched.SalePrice != 460000 || patched.Active || patched.Name != "Gas 12kg (ẩn)" {
		t.Fatalf("patched=%+v", patched)
	}

	if err := svc.db.QueryRow(`SELECT COUNT(*) FROM product_price_history WHERE product_id = ?`, created.ID).Scan(&histCount); err != nil {
		t.Fatal(err)
	}
	if histCount != 2 {
		t.Fatalf("price history after patch=%d", histCount)
	}
}

func TestListActiveProductsPublic(t *testing.T) {
	_, r := testCatalogRouter(t)

	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/v1/products", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("empty list status=%d body=%s", rr.Code, rr.Body.String())
	}
	var empty map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &empty); err != nil {
		t.Fatal(err)
	}
	if items, _ := empty["items"].([]any); len(items) != 0 {
		t.Fatalf("expected empty items, got %v", empty["items"])
	}

	create := func(sku, name string, price int64, active bool) {
		t.Helper()
		body, _ := json.Marshal(map[string]any{
			"sku": sku, "name": name, "sale_price": price, "active": active,
		})
		req := httptest.NewRequest(http.MethodPost, "/v1/admin/products", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusCreated {
			t.Fatalf("create %s status=%d body=%s", sku, w.Code, w.Body.String())
		}
	}
	create("GAS12", "Gas 12kg", 450000, true)
	create("GAS45", "Gas 45kg", 900000, true)
	create("HIDDEN", "Ẩn", 1000, false)

	rr = httptest.NewRecorder()
	r.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/v1/products", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("list status=%d body=%s", rr.Code, rr.Body.String())
	}
	var list struct {
		Items []product `json:"items"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &list); err != nil {
		t.Fatal(err)
	}
	if len(list.Items) != 2 {
		t.Fatalf("active items=%d want 2 body=%s", len(list.Items), rr.Body.String())
	}
	for _, p := range list.Items {
		if !p.Active {
			t.Fatalf("inactive product leaked: %+v", p)
		}
		if p.SKU == "HIDDEN" {
			t.Fatalf("hidden sku in public list: %+v", p)
		}
	}

	// Admin list still includes inactive.
	rr = httptest.NewRecorder()
	r.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/v1/admin/products", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("admin list status=%d", rr.Code)
	}
	var adminList struct {
		Items []product `json:"items"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &adminList); err != nil {
		t.Fatal(err)
	}
	if len(adminList.Items) != 3 {
		t.Fatalf("admin items=%d want 3", len(adminList.Items))
	}
}

func TestCreateProductValidationAndConflict(t *testing.T) {
	_, r := testCatalogRouter(t)

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/admin/products", bytes.NewReader([]byte(`{"sku":"X"}`)))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("missing fields status=%d", rr.Code)
	}

	createBody, _ := json.Marshal(map[string]any{
		"sku":        "DUP",
		"name":       "A",
		"sale_price": 1000,
	})
	req = httptest.NewRequest(http.MethodPost, "/v1/admin/products", bytes.NewReader(createBody))
	req.Header.Set("Content-Type", "application/json")
	rr = httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("first create status=%d body=%s", rr.Code, rr.Body.String())
	}

	req = httptest.NewRequest(http.MethodPost, "/v1/admin/products", bytes.NewReader(createBody))
	req.Header.Set("Content-Type", "application/json")
	rr = httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusConflict {
		t.Fatalf("dup sku status=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestGetPatchNotFound(t *testing.T) {
	_, r := testCatalogRouter(t)

	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/v1/admin/products/missing-id", nil))
	if rr.Code != http.StatusNotFound {
		t.Fatalf("get missing status=%d", rr.Code)
	}

	body, _ := json.Marshal(map[string]any{"active": false})
	req := httptest.NewRequest(http.MethodPatch, "/v1/admin/products/missing-id", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr = httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("patch missing status=%d", rr.Code)
	}
}

func TestMigrateCreatesCatalogSchema(t *testing.T) {
	db := openTestDB(t)

	for _, table := range []string{"products", "product_price_history"} {
		var name string
		err := db.QueryRow(`SELECT name FROM sqlite_master WHERE type='table' AND name=?`, table).Scan(&name)
		if err != nil {
			t.Fatalf("table %s: %v", table, err)
		}
	}

	var productCols int
	if err := db.QueryRow(`
		SELECT COUNT(*) FROM pragma_table_info('products')
		WHERE name IN ('id','sku','name','description','unit','sale_price','active','image_url','created_at','updated_at')
	`).Scan(&productCols); err != nil {
		t.Fatal(err)
	}
	if productCols != 10 {
		t.Fatalf("products columns=%d want 10", productCols)
	}

	var histCols int
	if err := db.QueryRow(`
		SELECT COUNT(*) FROM pragma_table_info('product_price_history')
		WHERE name IN ('id','product_id','sale_price','changed_at','changed_by')
	`).Scan(&histCols); err != nil {
		t.Fatal(err)
	}
	if histCols != 5 {
		t.Fatalf("product_price_history columns=%d want 5", histCols)
	}

	var idxActive, idxHist int
	if err := db.QueryRow(`
		SELECT COUNT(*) FROM sqlite_master
		WHERE type='index' AND name='idx_products_active'
	`).Scan(&idxActive); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow(`
		SELECT COUNT(*) FROM sqlite_master
		WHERE type='index' AND name='idx_price_history_product'
	`).Scan(&idxHist); err != nil {
		t.Fatal(err)
	}
	if idxActive != 1 || idxHist != 1 {
		t.Fatalf("indexes active=%d price_history=%d", idxActive, idxHist)
	}
}

func TestProductsSchemaConstraints(t *testing.T) {
	db := openTestDB(t)
	now := "2026-08-02T01:00:00Z"

	_, err := db.Exec(`
		INSERT INTO products (id, sku, name, unit, sale_price, active, created_at, updated_at)
		VALUES ('p1', 'GAS12', 'Gas 12kg', 'binh', -1, 1, ?, ?)`, now, now)
	if err == nil {
		t.Fatal("expected CHECK fail for negative sale_price")
	}

	_, err = db.Exec(`
		INSERT INTO products (id, sku, name, unit, sale_price, active, created_at, updated_at)
		VALUES ('p2', 'GAS12B', 'Gas', 'binh', 1000, 2, ?, ?)`, now, now)
	if err == nil {
		t.Fatal("expected CHECK fail for active not in (0,1)")
	}

	_, err = db.Exec(`
		INSERT INTO products (id, sku, name, unit, sale_price, active, created_at, updated_at)
		VALUES ('p3', 'GAS45', 'Gas 45kg', 'binh', 900000, 1, ?, ?)`, now, now)
	if err != nil {
		t.Fatalf("valid product insert: %v", err)
	}

	_, err = db.Exec(`
		INSERT INTO product_price_history (id, product_id, sale_price, changed_at)
		VALUES ('h-orphan', 'missing-product', 1000, ?)`, now)
	if err == nil {
		t.Fatal("expected FK fail for unknown product_id")
	}

	_, err = db.Exec(`
		INSERT INTO product_price_history (id, product_id, sale_price, changed_at, changed_by)
		VALUES ('h1', 'p3', 900000, ?, 'admin-1')`, now)
	if err != nil {
		t.Fatalf("valid history insert: %v", err)
	}
}
