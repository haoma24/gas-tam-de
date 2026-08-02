package main

import (
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"gas-tam-de/pkg/config"
)

const deliveryFeeSettingsID = "default"

// Architecture §6.4 example bands (VND) for local seed.
var defaultDeliveryFeeRules = []struct {
	ID        string
	MinKm     float64
	MaxKm     *float64
	FeeVnd    int64
	SortOrder int
}{
	{ID: "rule-0-5", MinKm: 0, MaxKm: floatPtr(5), FeeVnd: 10000, SortOrder: 0},
	{ID: "rule-5-10", MinKm: 5, MaxKm: floatPtr(10), FeeVnd: 20000, SortOrder: 1},
	{ID: "rule-10-inf", MinKm: 10, MaxKm: nil, FeeVnd: 30000, SortOrder: 2},
}

type deliveryFeeSeedConfig struct {
	Enabled bool // settings.enabled when inserting singleton
	Seed    bool // when false, skip insert entirely
}

func loadDeliveryFeeSeedConfig() deliveryFeeSeedConfig {
	seed := true
	if v := strings.TrimSpace(os.Getenv("DELIVERY_FEE_SEED")); v != "" {
		seed = strings.EqualFold(v, "1") || strings.EqualFold(v, "true")
	}
	enabled := config.GetInt("DELIVERY_FEE_ENABLED", 0) == 1
	return deliveryFeeSeedConfig{Enabled: enabled, Seed: seed}
}

// seedDeliveryFee inserts singleton settings + example bands when missing (idempotent).
func seedDeliveryFee(db *sql.DB, cfg deliveryFeeSeedConfig) error {
	if !cfg.Seed {
		slog.Info("delivery fee seed skipped", "reason", "DELIVERY_FEE_SEED disabled")
		return nil
	}

	var existing string
	err := db.QueryRow(`SELECT id FROM delivery_fee_settings WHERE id = ?`, deliveryFeeSettingsID).Scan(&existing)
	if err == nil {
		slog.Info("delivery fee seed skipped", "id", deliveryFeeSettingsID, "reason", "already exists")
		return nil
	}
	if err != sql.ErrNoRows {
		return fmt.Errorf("delivery fee seed lookup: %w", err)
	}

	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("delivery fee seed begin: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	now := time.Now().UTC().Format(time.RFC3339Nano)
	enabled := 0
	if cfg.Enabled {
		enabled = 1
	}
	if _, err := tx.Exec(`
		INSERT INTO delivery_fee_settings (id, enabled, updated_at)
		VALUES (?, ?, ?)
	`, deliveryFeeSettingsID, enabled, now); err != nil {
		return fmt.Errorf("delivery fee settings seed: %w", err)
	}

	for _, rule := range defaultDeliveryFeeRules {
		var maxKm any
		if rule.MaxKm != nil {
			maxKm = *rule.MaxKm
		}
		if _, err := tx.Exec(`
			INSERT INTO delivery_fee_rules (id, min_km, max_km, fee_vnd, sort_order, active)
			VALUES (?, ?, ?, ?, ?, 1)
		`, rule.ID, rule.MinKm, maxKm, rule.FeeVnd, rule.SortOrder); err != nil {
			return fmt.Errorf("delivery fee rule seed %s: %w", rule.ID, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("delivery fee seed commit: %w", err)
	}

	slog.Info("delivery fee seeded",
		"id", deliveryFeeSettingsID,
		"enabled", enabled,
		"rules", len(defaultDeliveryFeeRules),
	)
	return nil
}

// matchDeliveryFee returns fee_vnd for distanceKm when enabled; otherwise 0.
// Active bands are half-open [min_km, max_km) with max_km nil = +inf (PRD M4 / architecture §6.4).
// Pure / reusable for place order and future quote (T4.2.1). No matching active band → 0.
func matchDeliveryFee(enabled bool, rules []deliveryFeeRule, distanceKm float64) int64 {
	if !enabled {
		return 0
	}
	for _, r := range rules {
		if !r.Active {
			continue
		}
		if distanceKm < r.MinKm {
			continue
		}
		if r.MaxKm != nil && distanceKm >= *r.MaxKm {
			continue
		}
		return r.FeeVnd
	}
	return 0
}

// computeDeliveryFee loads settings/rules from DB and applies matchDeliveryFee.
// Missing settings (not seeded) are treated as disabled → fee 0.
func computeDeliveryFee(db *sql.DB, distanceKm float64) (int64, error) {
	cfg, err := loadDeliveryFeeConfig(db)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, nil
		}
		return 0, err
	}
	return matchDeliveryFee(cfg.Enabled, cfg.Rules, distanceKm), nil
}

func floatPtr(v float64) *float64 { return &v }
