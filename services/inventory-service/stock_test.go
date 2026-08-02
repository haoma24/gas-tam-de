package main

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	"gas-tam-de/pkg/sqlite"
)

func openInventoryTestDB(t *testing.T) *sql.DB {
	t.Helper()
	dir := t.TempDir()
	db, err := sqlite.Open(filepath.Join(dir, "inventory.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := migrate(db); err != nil {
		t.Fatal(err)
	}
	return db
}

func TestMigrateCreatesInventorySchema(t *testing.T) {
	db := openInventoryTestDB(t)

	for _, table := range []string{"stock_items", "stock_movements", "processed_events"} {
		var name string
		err := db.QueryRow(`SELECT name FROM sqlite_master WHERE type='table' AND name=?`, table).Scan(&name)
		if err != nil {
			t.Fatalf("table %s: %v", table, err)
		}
	}

	var stockCols int
	if err := db.QueryRow(`
		SELECT COUNT(*) FROM pragma_table_info('stock_items')
		WHERE name IN ('product_id','sku','name','on_hand','cost_price','reorder_level','updated_at')
	`).Scan(&stockCols); err != nil {
		t.Fatal(err)
	}
	if stockCols != 7 {
		t.Fatalf("stock_items columns=%d want 7", stockCols)
	}

	var movementCols int
	if err := db.QueryRow(`
		SELECT COUNT(*) FROM pragma_table_info('stock_movements')
		WHERE name IN ('id','product_id','movement_type','qty','unit_cost','note','ref_type','ref_id','created_at','created_by')
	`).Scan(&movementCols); err != nil {
		t.Fatal(err)
	}
	if movementCols != 10 {
		t.Fatalf("stock_movements columns=%d want 10", movementCols)
	}

	var processedCols int
	if err := db.QueryRow(`
		SELECT COUNT(*) FROM pragma_table_info('processed_events')
		WHERE name IN ('event_id','processed_at')
	`).Scan(&processedCols); err != nil {
		t.Fatal(err)
	}
	if processedCols != 2 {
		t.Fatalf("processed_events columns=%d want 2", processedCols)
	}

	var idx int
	if err := db.QueryRow(`
		SELECT COUNT(*) FROM sqlite_master
		WHERE type='index' AND name='idx_movements_product'
	`).Scan(&idx); err != nil {
		t.Fatal(err)
	}
	if idx != 1 {
		t.Fatalf("idx_movements_product=%d want 1", idx)
	}
}

func TestInventorySchemaConstraints(t *testing.T) {
	db := openInventoryTestDB(t)
	now := "2026-08-02T04:00:00Z"

	_, err := db.Exec(`
		INSERT INTO stock_items (product_id, sku, name, on_hand, cost_price, reorder_level, updated_at)
		VALUES ('p-neg-cost', 'SKU1', 'Gas', 0, -1, 0, ?)`, now)
	if err == nil {
		t.Fatal("expected CHECK fail for negative cost_price")
	}

	_, err = db.Exec(`
		INSERT INTO stock_items (product_id, sku, name, on_hand, cost_price, reorder_level, updated_at)
		VALUES ('p-neg-reorder', 'SKU2', 'Gas', 0, 0, -1, ?)`, now)
	if err == nil {
		t.Fatal("expected CHECK fail for negative reorder_level")
	}

	_, err = db.Exec(`
		INSERT INTO stock_items (product_id, sku, name, on_hand, cost_price, reorder_level, updated_at)
		VALUES ('p1', 'GAS12', 'Gas 12kg', -2, 150000, 5, ?)`, now)
	if err != nil {
		t.Fatalf("negative on_hand should be allowed (MVP): %v", err)
	}

	_, err = db.Exec(`
		INSERT INTO stock_movements (id, product_id, movement_type, qty, unit_cost, created_at)
		VALUES ('m-bad-type', 'p1', 'XFER', 1, 100, ?)`, now)
	if err == nil {
		t.Fatal("expected CHECK fail for invalid movement_type")
	}

	_, err = db.Exec(`
		INSERT INTO stock_movements (id, product_id, movement_type, qty, unit_cost, created_at)
		VALUES ('m-zero-qty', 'p1', 'IN', 0, 100, ?)`, now)
	if err == nil {
		t.Fatal("expected CHECK fail for qty <= 0")
	}

	_, err = db.Exec(`
		INSERT INTO stock_movements (id, product_id, movement_type, qty, unit_cost, created_at)
		VALUES ('m-neg-cost', 'p1', 'IN', 1, -1, ?)`, now)
	if err == nil {
		t.Fatal("expected CHECK fail for negative unit_cost")
	}

	_, err = db.Exec(`
		INSERT INTO stock_movements (id, product_id, movement_type, qty, unit_cost, ref_type, ref_id, created_at, created_by)
		VALUES ('m-ok', 'p1', 'IN', 10, 150000, 'MANUAL', NULL, ?, 'admin')`, now)
	if err != nil {
		t.Fatalf("valid IN movement: %v", err)
	}

	_, err = db.Exec(`
		INSERT INTO stock_movements (id, product_id, movement_type, qty, unit_cost, ref_type, ref_id, created_at)
		VALUES ('m-out', 'p1', 'OUT', 1, 150000, 'ORDER', 'ord-1', ?)`, now)
	if err != nil {
		t.Fatalf("valid OUT movement: %v", err)
	}

	_, err = db.Exec(`
		INSERT INTO stock_movements (id, product_id, movement_type, qty, unit_cost, created_at)
		VALUES ('m-adj', 'p1', 'ADJUST', 1, NULL, ?)`, now)
	if err != nil {
		t.Fatalf("valid ADJUST with NULL unit_cost: %v", err)
	}

	_, err = db.Exec(`
		INSERT INTO processed_events (event_id, processed_at)
		VALUES ('evt-1', ?)`, now)
	if err != nil {
		t.Fatalf("valid processed_events insert: %v", err)
	}
}

func TestSeedInventoryDefaultsEmpty(t *testing.T) {
	db := openInventoryTestDB(t)
	if err := seedInventoryDefaults(db, inventorySeedConfig{Enabled: true}); err != nil {
		t.Fatal(err)
	}

	var stockCount, movementCount int
	if err := db.QueryRow(`SELECT COUNT(*) FROM stock_items`).Scan(&stockCount); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow(`SELECT COUNT(*) FROM stock_movements`).Scan(&movementCount); err != nil {
		t.Fatal(err)
	}
	if stockCount != 0 || movementCount != 0 {
		t.Fatalf("want empty defaults stock=%d movements=%d", stockCount, movementCount)
	}
}

func TestSeedInventoryDefaultsIdempotent(t *testing.T) {
	db := openInventoryTestDB(t)
	cfg := inventorySeedConfig{Enabled: true}
	if err := seedInventoryDefaults(db, cfg); err != nil {
		t.Fatal(err)
	}
	if err := seedInventoryDefaults(db, cfg); err != nil {
		t.Fatal(err)
	}

	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM stock_items`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Fatalf("stock_items=%d want 0 after repeated seed", n)
	}
}

func TestSeedInventoryDefaultsDisabled(t *testing.T) {
	db := openInventoryTestDB(t)
	if err := seedInventoryDefaults(db, inventorySeedConfig{Enabled: false}); err != nil {
		t.Fatal(err)
	}
}

func TestLoadInventorySeedConfig(t *testing.T) {
	t.Setenv("INVENTORY_SEED", "")
	cfg := loadInventorySeedConfig()
	if !cfg.Enabled {
		t.Fatal("default INVENTORY_SEED should enable")
	}

	t.Setenv("INVENTORY_SEED", "0")
	cfg = loadInventorySeedConfig()
	if cfg.Enabled {
		t.Fatal("INVENTORY_SEED=0 should disable")
	}

	t.Setenv("INVENTORY_SEED", "true")
	cfg = loadInventorySeedConfig()
	if !cfg.Enabled {
		t.Fatal("INVENTORY_SEED=true should enable")
	}

	_ = os.Unsetenv("INVENTORY_SEED")
}
