package main

import (
	"database/sql"
	"path/filepath"
	"testing"

	"gas-tam-de/pkg/sqlite"
)

func openDeliveryFeeTestDB(t *testing.T) *sql.DB {
	t.Helper()
	dir := t.TempDir()
	db, err := sqlite.Open(filepath.Join(dir, "order.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := migrate(db); err != nil {
		t.Fatal(err)
	}
	return db
}

func TestMigrateCreatesDeliveryFeeSchema(t *testing.T) {
	db := openDeliveryFeeTestDB(t)

	for _, table := range []string{"delivery_fee_settings", "delivery_fee_rules"} {
		var name string
		err := db.QueryRow(`SELECT name FROM sqlite_master WHERE type='table' AND name=?`, table).Scan(&name)
		if err != nil {
			t.Fatalf("table %s: %v", table, err)
		}
	}

	var settingsCols int
	if err := db.QueryRow(`
		SELECT COUNT(*) FROM pragma_table_info('delivery_fee_settings')
		WHERE name IN ('id','enabled','updated_at')
	`).Scan(&settingsCols); err != nil {
		t.Fatal(err)
	}
	if settingsCols != 3 {
		t.Fatalf("delivery_fee_settings columns=%d want 3", settingsCols)
	}

	var rulesCols int
	if err := db.QueryRow(`
		SELECT COUNT(*) FROM pragma_table_info('delivery_fee_rules')
		WHERE name IN ('id','min_km','max_km','fee_vnd','sort_order','active')
	`).Scan(&rulesCols); err != nil {
		t.Fatal(err)
	}
	if rulesCols != 6 {
		t.Fatalf("delivery_fee_rules columns=%d want 6", rulesCols)
	}

	var idx int
	if err := db.QueryRow(`
		SELECT COUNT(*) FROM sqlite_master
		WHERE type='index' AND name='idx_delivery_fee_rules_active'
	`).Scan(&idx); err != nil {
		t.Fatal(err)
	}
	if idx != 1 {
		t.Fatalf("idx_delivery_fee_rules_active=%d want 1", idx)
	}
}

func TestDeliveryFeeSchemaConstraints(t *testing.T) {
	db := openDeliveryFeeTestDB(t)
	now := "2026-08-02T03:00:00Z"

	_, err := db.Exec(`
		INSERT INTO delivery_fee_settings (id, enabled, updated_at)
		VALUES ('bad-enabled', 2, ?)`, now)
	if err == nil {
		t.Fatal("expected CHECK fail for enabled not in (0,1)")
	}

	_, err = db.Exec(`
		INSERT INTO delivery_fee_settings (id, enabled, updated_at)
		VALUES ('ok', 0, ?)`, now)
	if err != nil {
		t.Fatalf("valid settings insert: %v", err)
	}

	_, err = db.Exec(`
		INSERT INTO delivery_fee_rules (id, min_km, max_km, fee_vnd, sort_order, active)
		VALUES ('r-neg-fee', 0, 5, -1, 0, 1)`)
	if err == nil {
		t.Fatal("expected CHECK fail for negative fee_vnd")
	}

	_, err = db.Exec(`
		INSERT INTO delivery_fee_rules (id, min_km, max_km, fee_vnd, sort_order, active)
		VALUES ('r-bad-band', 5, 3, 1000, 0, 1)`)
	if err == nil {
		t.Fatal("expected CHECK fail for max_km <= min_km")
	}

	_, err = db.Exec(`
		INSERT INTO delivery_fee_rules (id, min_km, max_km, fee_vnd, sort_order, active)
		VALUES ('r-bad-active', 0, 5, 1000, 0, 2)`)
	if err == nil {
		t.Fatal("expected CHECK fail for active not in (0,1)")
	}

	_, err = db.Exec(`
		INSERT INTO delivery_fee_rules (id, min_km, max_km, fee_vnd, sort_order, active)
		VALUES ('r-ok', 0, 5, 10000, 0, 1)`)
	if err != nil {
		t.Fatalf("valid rule insert: %v", err)
	}

	_, err = db.Exec(`
		INSERT INTO delivery_fee_rules (id, min_km, max_km, fee_vnd, sort_order, active)
		VALUES ('r-inf', 10, NULL, 30000, 2, 1)`)
	if err != nil {
		t.Fatalf("valid open-ended rule insert: %v", err)
	}
}

func TestSeedDeliveryFeeCreatesDefaults(t *testing.T) {
	db := openDeliveryFeeTestDB(t)
	if err := seedDeliveryFee(db, deliveryFeeSeedConfig{Enabled: false, Seed: true}); err != nil {
		t.Fatal(err)
	}

	var enabled int
	var updatedAt string
	if err := db.QueryRow(`
		SELECT enabled, updated_at FROM delivery_fee_settings WHERE id = ?
	`, deliveryFeeSettingsID).Scan(&enabled, &updatedAt); err != nil {
		t.Fatal(err)
	}
	if enabled != 0 {
		t.Fatalf("enabled=%d want 0", enabled)
	}
	if updatedAt == "" {
		t.Fatal("expected updated_at")
	}

	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM delivery_fee_rules`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != len(defaultDeliveryFeeRules) {
		t.Fatalf("rules count=%d want %d", count, len(defaultDeliveryFeeRules))
	}

	var fee int64
	var maxKm sql.NullFloat64
	if err := db.QueryRow(`
		SELECT fee_vnd, max_km FROM delivery_fee_rules WHERE id = 'rule-0-5'
	`).Scan(&fee, &maxKm); err != nil {
		t.Fatal(err)
	}
	if fee != 10000 || !maxKm.Valid || maxKm.Float64 != 5 {
		t.Fatalf("rule-0-5 fee=%d max=%v", fee, maxKm)
	}

	if err := db.QueryRow(`
		SELECT fee_vnd, max_km FROM delivery_fee_rules WHERE id = 'rule-10-inf'
	`).Scan(&fee, &maxKm); err != nil {
		t.Fatal(err)
	}
	if fee != 30000 || maxKm.Valid {
		t.Fatalf("rule-10-inf fee=%d max=%v want 30000 / NULL", fee, maxKm)
	}
}

func TestSeedDeliveryFeeIdempotent(t *testing.T) {
	db := openDeliveryFeeTestDB(t)
	cfg := deliveryFeeSeedConfig{Enabled: false, Seed: true}
	if err := seedDeliveryFee(db, cfg); err != nil {
		t.Fatal(err)
	}
	cfg.Enabled = true
	if err := seedDeliveryFee(db, cfg); err != nil {
		t.Fatal(err)
	}

	var enabled int
	if err := db.QueryRow(`
		SELECT enabled FROM delivery_fee_settings WHERE id = ?
	`, deliveryFeeSettingsID).Scan(&enabled); err != nil {
		t.Fatal(err)
	}
	if enabled != 0 {
		t.Fatalf("seed must not overwrite enabled: got %d", enabled)
	}

	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM delivery_fee_rules`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != len(defaultDeliveryFeeRules) {
		t.Fatalf("rules count=%d want %d", count, len(defaultDeliveryFeeRules))
	}
}

func TestSeedDeliveryFeeDisabled(t *testing.T) {
	db := openDeliveryFeeTestDB(t)
	if err := seedDeliveryFee(db, deliveryFeeSeedConfig{Enabled: true, Seed: false}); err != nil {
		t.Fatal(err)
	}

	var settingsCount, rulesCount int
	if err := db.QueryRow(`SELECT COUNT(*) FROM delivery_fee_settings`).Scan(&settingsCount); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow(`SELECT COUNT(*) FROM delivery_fee_rules`).Scan(&rulesCount); err != nil {
		t.Fatal(err)
	}
	if settingsCount != 0 || rulesCount != 0 {
		t.Fatalf("expected empty seed when disabled; settings=%d rules=%d", settingsCount, rulesCount)
	}
}

func TestSeedDeliveryFeeEnabledFlag(t *testing.T) {
	db := openDeliveryFeeTestDB(t)
	if err := seedDeliveryFee(db, deliveryFeeSeedConfig{Enabled: true, Seed: true}); err != nil {
		t.Fatal(err)
	}
	var enabled int
	if err := db.QueryRow(`
		SELECT enabled FROM delivery_fee_settings WHERE id = ?
	`, deliveryFeeSettingsID).Scan(&enabled); err != nil {
		t.Fatal(err)
	}
	if enabled != 1 {
		t.Fatalf("enabled=%d want 1", enabled)
	}
}

func defaultMatchRules() []deliveryFeeRule {
	return []deliveryFeeRule{
		{ID: "rule-0-5", MinKm: 0, MaxKm: floatPtr(5), FeeVnd: 10000, SortOrder: 0, Active: true},
		{ID: "rule-5-10", MinKm: 5, MaxKm: floatPtr(10), FeeVnd: 20000, SortOrder: 1, Active: true},
		{ID: "rule-10-inf", MinKm: 10, MaxKm: nil, FeeVnd: 30000, SortOrder: 2, Active: true},
	}
}

func TestMatchDeliveryFeeDisabled(t *testing.T) {
	rules := defaultMatchRules()
	if got := matchDeliveryFee(false, rules, 3.2); got != 0 {
		t.Fatalf("disabled fee=%d want 0", got)
	}
	if got := matchDeliveryFee(false, rules, 12); got != 0 {
		t.Fatalf("disabled fee=%d want 0", got)
	}
}

func TestMatchDeliveryFeeBands(t *testing.T) {
	rules := defaultMatchRules()
	cases := []struct {
		km   float64
		want int64
	}{
		{0, 10000},
		{4.99, 10000},
		{5, 20000},
		{7.5, 20000},
		{9.999, 20000},
		{10, 30000},
		{25, 30000},
	}
	for _, tc := range cases {
		if got := matchDeliveryFee(true, rules, tc.km); got != tc.want {
			t.Fatalf("km=%v fee=%d want %d", tc.km, got, tc.want)
		}
	}
}

func TestMatchDeliveryFeeSkipsInactiveAndGaps(t *testing.T) {
	rules := []deliveryFeeRule{
		{ID: "a", MinKm: 0, MaxKm: floatPtr(5), FeeVnd: 10000, Active: false},
		{ID: "b", MinKm: 5, MaxKm: floatPtr(8), FeeVnd: 15000, Active: true},
		// gap [8, 10)
		{ID: "c", MinKm: 10, MaxKm: nil, FeeVnd: 30000, Active: true},
	}
	if got := matchDeliveryFee(true, rules, 3); got != 0 {
		t.Fatalf("inactive band fee=%d want 0", got)
	}
	if got := matchDeliveryFee(true, rules, 6); got != 15000 {
		t.Fatalf("active band fee=%d want 15000", got)
	}
	if got := matchDeliveryFee(true, rules, 9); got != 0 {
		t.Fatalf("gap fee=%d want 0", got)
	}
	if got := matchDeliveryFee(true, rules, 10); got != 30000 {
		t.Fatalf("open band fee=%d want 30000", got)
	}
}

func TestComputeDeliveryFeeFromDB(t *testing.T) {
	db := openDeliveryFeeTestDB(t)

	fee, err := computeDeliveryFee(db, 3.2)
	if err != nil {
		t.Fatal(err)
	}
	if fee != 0 {
		t.Fatalf("unconfigured fee=%d want 0", fee)
	}

	if err := seedDeliveryFee(db, deliveryFeeSeedConfig{Enabled: false, Seed: true}); err != nil {
		t.Fatal(err)
	}
	fee, err = computeDeliveryFee(db, 7.5)
	if err != nil {
		t.Fatal(err)
	}
	if fee != 0 {
		t.Fatalf("disabled seed fee=%d want 0", fee)
	}

	now := "2026-08-02T00:00:00Z"
	if err := saveDeliveryFeeConfig(db, true, defaultMatchRules(), now, true); err != nil {
		t.Fatal(err)
	}
	fee, err = computeDeliveryFee(db, 7.5)
	if err != nil {
		t.Fatal(err)
	}
	if fee != 20000 {
		t.Fatalf("enabled fee=%d want 20000", fee)
	}
	fee, err = computeDeliveryFee(db, 3.2)
	if err != nil {
		t.Fatal(err)
	}
	if fee != 10000 {
		t.Fatalf("enabled fee=%d want 10000", fee)
	}
}
