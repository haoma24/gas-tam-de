package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"sort"
	"strings"
	"time"

	"gas-tam-de/pkg/httpx"

	"github.com/google/uuid"
)

// deliveryFeeConfig is the admin GET/PUT payload for fee toggle + distance bands
// (architecture §4.4 GET/PUT /v1/admin/delivery-fee; PRD M4).
type deliveryFeeConfig struct {
	Enabled   bool               `json:"enabled"`
	UpdatedAt string             `json:"updated_at,omitempty"`
	Rules     []deliveryFeeRule  `json:"rules"`
}

type deliveryFeeRule struct {
	ID        string   `json:"id"`
	MinKm     float64  `json:"min_km"`
	MaxKm     *float64 `json:"max_km"` // nil = +inf; half-open [min, max)
	FeeVnd    int64    `json:"fee_vnd"`
	SortOrder int      `json:"sort_order"`
	Active    bool     `json:"active"`
}

type putDeliveryFeeBody struct {
	Enabled *bool                   `json:"enabled"`
	Rules   *[]putDeliveryFeeRule   `json:"rules"`
}

type putDeliveryFeeRule struct {
	ID        string   `json:"id"`
	MinKm     float64  `json:"min_km"`
	MaxKm     *float64 `json:"max_km"`
	FeeVnd    int64    `json:"fee_vnd"`
	SortOrder *int     `json:"sort_order"`
	Active    *bool    `json:"active"`
}

func (s *orderService) handleGetAdminDeliveryFee(w http.ResponseWriter, _ *http.Request) {
	cfg, err := loadDeliveryFeeConfig(s.db)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			httpx.Error(w, http.StatusNotFound, "FEE_NOT_CONFIGURED", "delivery fee settings not configured")
			return
		}
		slog.Error("get delivery fee", "err", err)
		httpx.Error(w, http.StatusInternalServerError, "INTERNAL", "could not load delivery fee")
		return
	}
	httpx.JSON(w, http.StatusOK, cfg)
}

func (s *orderService) handlePutAdminDeliveryFee(w http.ResponseWriter, r *http.Request) {
	var body putDeliveryFeeBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		httpx.Error(w, http.StatusBadRequest, "INVALID_BODY", "invalid JSON body")
		return
	}
	if body.Enabled == nil && body.Rules == nil {
		httpx.Error(w, http.StatusBadRequest, "INVALID_BODY", "provide enabled and/or rules")
		return
	}

	current, err := loadDeliveryFeeConfig(s.db)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			httpx.Error(w, http.StatusNotFound, "FEE_NOT_CONFIGURED", "delivery fee settings not configured; seed or create first")
			return
		}
		slog.Error("get delivery fee", "err", err)
		httpx.Error(w, http.StatusInternalServerError, "INTERNAL", "could not load delivery fee")
		return
	}

	enabled := current.Enabled
	if body.Enabled != nil {
		enabled = *body.Enabled
	}

	rules := current.Rules
	if body.Rules != nil {
		normalized, verr := normalizePutDeliveryFeeRules(*body.Rules)
		if verr != nil {
			httpx.Error(w, http.StatusBadRequest, "INVALID_RULES", verr.Error())
			return
		}
		rules = normalized
	}

	now := time.Now().UTC().Format(time.RFC3339Nano)
	if err := saveDeliveryFeeConfig(s.db, enabled, rules, now, body.Rules != nil); err != nil {
		slog.Error("update delivery fee", "err", err)
		httpx.Error(w, http.StatusInternalServerError, "INTERNAL", "could not update delivery fee")
		return
	}

	cfg, err := loadDeliveryFeeConfig(s.db)
	if err != nil {
		slog.Error("reload delivery fee", "err", err)
		httpx.Error(w, http.StatusInternalServerError, "INTERNAL", "could not load delivery fee")
		return
	}
	httpx.JSON(w, http.StatusOK, cfg)
}

func loadDeliveryFeeConfig(db *sql.DB) (deliveryFeeConfig, error) {
	var enabledInt int
	var updatedAt string
	err := db.QueryRow(`
		SELECT enabled, updated_at FROM delivery_fee_settings WHERE id = ?
	`, deliveryFeeSettingsID).Scan(&enabledInt, &updatedAt)
	if err != nil {
		return deliveryFeeConfig{}, err
	}

	rows, err := db.Query(`
		SELECT id, min_km, max_km, fee_vnd, sort_order, active
		FROM delivery_fee_rules
		ORDER BY sort_order ASC, min_km ASC, id ASC
	`)
	if err != nil {
		return deliveryFeeConfig{}, err
	}
	defer rows.Close()

	rules := make([]deliveryFeeRule, 0)
	for rows.Next() {
		var rule deliveryFeeRule
		var maxKm sql.NullFloat64
		var activeInt int
		if err := rows.Scan(&rule.ID, &rule.MinKm, &maxKm, &rule.FeeVnd, &rule.SortOrder, &activeInt); err != nil {
			return deliveryFeeConfig{}, err
		}
		if maxKm.Valid {
			v := maxKm.Float64
			rule.MaxKm = &v
		}
		rule.Active = activeInt == 1
		rules = append(rules, rule)
	}
	if err := rows.Err(); err != nil {
		return deliveryFeeConfig{}, err
	}

	return deliveryFeeConfig{
		Enabled:   enabledInt == 1,
		UpdatedAt: updatedAt,
		Rules:     rules,
	}, nil
}

// saveDeliveryFeeConfig updates settings; when replaceRules, deletes all rules and inserts the provided set.
func saveDeliveryFeeConfig(db *sql.DB, enabled bool, rules []deliveryFeeRule, now string, replaceRules bool) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	enabledInt := 0
	if enabled {
		enabledInt = 1
	}
	res, err := tx.Exec(`
		UPDATE delivery_fee_settings SET enabled = ?, updated_at = ? WHERE id = ?
	`, enabledInt, now, deliveryFeeSettingsID)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return sql.ErrNoRows
	}

	if replaceRules {
		if _, err := tx.Exec(`DELETE FROM delivery_fee_rules`); err != nil {
			return err
		}
		for _, rule := range rules {
			var maxKm any
			if rule.MaxKm != nil {
				maxKm = *rule.MaxKm
			}
			activeInt := 0
			if rule.Active {
				activeInt = 1
			}
			if _, err := tx.Exec(`
				INSERT INTO delivery_fee_rules (id, min_km, max_km, fee_vnd, sort_order, active)
				VALUES (?, ?, ?, ?, ?, ?)
			`, rule.ID, rule.MinKm, maxKm, rule.FeeVnd, rule.SortOrder, activeInt); err != nil {
				return err
			}
		}
	}

	return tx.Commit()
}

func normalizePutDeliveryFeeRules(in []putDeliveryFeeRule) ([]deliveryFeeRule, error) {
	out := make([]deliveryFeeRule, 0, len(in))
	seenIDs := make(map[string]struct{}, len(in))

	for i, raw := range in {
		id := strings.TrimSpace(raw.ID)
		if id == "" {
			id = uuid.NewString()
		}
		if _, dup := seenIDs[id]; dup {
			return nil, fmt.Errorf("duplicate rule id %q", id)
		}
		seenIDs[id] = struct{}{}

		if raw.MinKm < 0 {
			return nil, fmt.Errorf("rules[%d].min_km must be >= 0", i)
		}
		if raw.MaxKm != nil && *raw.MaxKm <= raw.MinKm {
			return nil, fmt.Errorf("rules[%d].max_km must be > min_km", i)
		}
		if raw.FeeVnd < 0 {
			return nil, fmt.Errorf("rules[%d].fee_vnd must be >= 0", i)
		}

		sortOrder := i
		if raw.SortOrder != nil {
			sortOrder = *raw.SortOrder
		}
		active := true
		if raw.Active != nil {
			active = *raw.Active
		}

		out = append(out, deliveryFeeRule{
			ID:        id,
			MinKm:     raw.MinKm,
			MaxKm:     raw.MaxKm,
			FeeVnd:    raw.FeeVnd,
			SortOrder: sortOrder,
			Active:    active,
		})
	}

	if err := validateActiveRuleBands(out); err != nil {
		return nil, err
	}
	return out, nil
}

// validateActiveRuleBands ensures active half-open bands [min,max) do not overlap
// (max NULL = +inf; only one such open-ended active rule, and it must be last by min_km).
func validateActiveRuleBands(rules []deliveryFeeRule) error {
	active := make([]deliveryFeeRule, 0, len(rules))
	for _, r := range rules {
		if r.Active {
			active = append(active, r)
		}
	}
	sort.SliceStable(active, func(i, j int) bool {
		if active[i].MinKm != active[j].MinKm {
			return active[i].MinKm < active[j].MinKm
		}
		return active[i].SortOrder < active[j].SortOrder
	})

	for i, r := range active {
		if r.MaxKm == nil && i != len(active)-1 {
			return fmt.Errorf("open-ended rule (max_km null) must be the last active band by min_km")
		}
		if i == 0 {
			continue
		}
		prev := active[i-1]
		if prev.MaxKm == nil {
			return fmt.Errorf("active bands overlap: open-ended rule before min_km=%.2f", r.MinKm)
		}
		if *prev.MaxKm > r.MinKm {
			return fmt.Errorf("active bands overlap: [%.2f,%.2f) and [%.2f,...)", prev.MinKm, *prev.MaxKm, r.MinKm)
		}
	}
	return nil
}
