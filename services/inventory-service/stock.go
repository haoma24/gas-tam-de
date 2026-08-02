package main

import (
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"strings"
)

// inventorySeedConfig controls local bootstrap for inventory.db.
// Empty stock_items / stock_movements is the intentional default (T7.1.1):
// products appear via catalog sync / stock IN APIs (T7.1.2+), not hardcoded seed rows.
type inventorySeedConfig struct {
	Enabled bool // when false, skip seed log/check entirely
}

func loadInventorySeedConfig() inventorySeedConfig {
	enabled := true
	if v := strings.TrimSpace(os.Getenv("INVENTORY_SEED")); v != "" {
		enabled = strings.EqualFold(v, "1") || strings.EqualFold(v, "true")
	}
	return inventorySeedConfig{Enabled: enabled}
}

// seedInventoryDefaults confirms schema is usable and leaves stock empty (idempotent).
func seedInventoryDefaults(db *sql.DB, cfg inventorySeedConfig) error {
	if !cfg.Enabled {
		slog.Info("inventory seed skipped", "reason", "INVENTORY_SEED disabled")
		return nil
	}

	var stockCount, movementCount, processedCount int
	if err := db.QueryRow(`SELECT COUNT(*) FROM stock_items`).Scan(&stockCount); err != nil {
		return fmt.Errorf("inventory seed stock_items: %w", err)
	}
	if err := db.QueryRow(`SELECT COUNT(*) FROM stock_movements`).Scan(&movementCount); err != nil {
		return fmt.Errorf("inventory seed stock_movements: %w", err)
	}
	if err := db.QueryRow(`SELECT COUNT(*) FROM processed_events`).Scan(&processedCount); err != nil {
		return fmt.Errorf("inventory seed processed_events: %w", err)
	}

	slog.Info("inventory ready",
		"stock_items", stockCount,
		"stock_movements", movementCount,
		"processed_events", processedCount,
		"note", "empty defaults until catalog sync or stock IN",
	)
	return nil
}
