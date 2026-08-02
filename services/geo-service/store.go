package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"gas-tam-de/pkg/config"
	"gas-tam-de/pkg/httpx"
)

const storeSettingsID = "default"

// Local defaults: near Bến Thành (HCMC) — override via STORE_* env for real shop coords.
const (
	defaultStoreLat         = 10.7769
	defaultStoreLng         = 106.7009
	defaultStoreMaxRadiusKm = 10.0
	defaultStoreName        = "Gas Tam Đệ"
)

// storeSettings is the singleton shop geo fence config (architecture §6.3).
type storeSettings struct {
	ID           string  `json:"id"`
	Name         string  `json:"name"`
	Lat          float64 `json:"lat"`
	Lng          float64 `json:"lng"`
	MaxRadiusKm  float64 `json:"max_radius_km"`
	AddressText  *string `json:"address_text,omitempty"`
	UpdatedAt    string  `json:"updated_at,omitempty"`
	UpdatedBy    *string `json:"updated_by,omitempty"`
}

type storeSeedConfig struct {
	Name        string
	Lat         float64
	Lng         float64
	MaxRadiusKm float64
	AddressText string
	Enabled     bool
}

type putStoreBody struct {
	Name        *string  `json:"name"`
	Lat         *float64 `json:"lat"`
	Lng         *float64 `json:"lng"`
	MaxRadiusKm *float64 `json:"max_radius_km"`
	AddressText *string  `json:"address_text"`
}

func loadStoreSeedConfig() storeSeedConfig {
	enabled := true
	if v := strings.TrimSpace(os.Getenv("STORE_SEED")); v != "" {
		enabled = strings.EqualFold(v, "1") || strings.EqualFold(v, "true")
	}
	return storeSeedConfig{
		Name:        config.Get("STORE_NAME", defaultStoreName),
		Lat:         config.GetFloat("STORE_LAT", defaultStoreLat),
		Lng:         config.GetFloat("STORE_LNG", defaultStoreLng),
		MaxRadiusKm: config.GetFloat("STORE_MAX_RADIUS_KM", defaultStoreMaxRadiusKm),
		AddressText: config.Get("STORE_ADDRESS_TEXT", ""),
		Enabled:     enabled,
	}
}

// seedStoreSettings inserts the singleton row when missing (idempotent).
func seedStoreSettings(db *sql.DB, cfg storeSeedConfig) error {
	if !cfg.Enabled {
		slog.Info("store seed skipped", "reason", "STORE_SEED disabled")
		return nil
	}
	if err := validateStoreCoords(cfg.Lat, cfg.Lng, cfg.MaxRadiusKm); err != nil {
		return fmt.Errorf("store seed: %w", err)
	}
	name := strings.TrimSpace(cfg.Name)
	if name == "" {
		name = defaultStoreName
	}

	var existing string
	err := db.QueryRow(`SELECT id FROM store_settings WHERE id = ?`, storeSettingsID).Scan(&existing)
	if err == nil {
		slog.Info("store seed skipped", "id", storeSettingsID, "reason", "already exists")
		return nil
	}
	if err != sql.ErrNoRows {
		return fmt.Errorf("store seed lookup: %w", err)
	}

	now := time.Now().UTC().Format(time.RFC3339Nano)
	var address any
	if t := strings.TrimSpace(cfg.AddressText); t != "" {
		address = t
	}

	_, err = db.Exec(`
		INSERT INTO store_settings (id, name, lat, lng, max_radius_km, address_text, updated_at, updated_by)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, storeSettingsID, name, cfg.Lat, cfg.Lng, cfg.MaxRadiusKm, address, now, "seed")
	if err != nil {
		return fmt.Errorf("store seed insert: %w", err)
	}

	slog.Info("store settings seeded",
		"id", storeSettingsID,
		"lat", cfg.Lat,
		"lng", cfg.Lng,
		"max_radius_km", cfg.MaxRadiusKm,
	)
	return nil
}

func getStoreSettings(db *sql.DB) (storeSettings, error) {
	var s storeSettings
	var address, updatedBy sql.NullString
	err := db.QueryRow(`
		SELECT id, name, lat, lng, max_radius_km, address_text, updated_at, updated_by
		FROM store_settings WHERE id = ?
	`, storeSettingsID).Scan(
		&s.ID, &s.Name, &s.Lat, &s.Lng, &s.MaxRadiusKm, &address, &s.UpdatedAt, &updatedBy,
	)
	if err != nil {
		return storeSettings{}, err
	}
	if address.Valid {
		s.AddressText = &address.String
	}
	if updatedBy.Valid {
		s.UpdatedBy = &updatedBy.String
	}
	return s, nil
}

// publicStoreView strips internal fields for GET /v1/geo/store (architecture §4.4).
func publicStoreView(s storeSettings) map[string]any {
	out := map[string]any{
		"name":          s.Name,
		"lat":           s.Lat,
		"lng":           s.Lng,
		"max_radius_km": s.MaxRadiusKm,
	}
	if s.AddressText != nil && strings.TrimSpace(*s.AddressText) != "" {
		out["address_text"] = *s.AddressText
	}
	return out
}

func (s *geoService) handleGetStore(w http.ResponseWriter, _ *http.Request) {
	row, err := getStoreSettings(s.db)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			httpx.Error(w, http.StatusNotFound, "STORE_NOT_CONFIGURED", "store settings not configured")
			return
		}
		slog.Error("get store settings", "err", err)
		httpx.Error(w, http.StatusInternalServerError, "INTERNAL", "could not load store settings")
		return
	}
	httpx.JSON(w, http.StatusOK, publicStoreView(row))
}

func (s *geoService) handlePutAdminStore(w http.ResponseWriter, r *http.Request) {
	var body putStoreBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		httpx.Error(w, http.StatusBadRequest, "INVALID_BODY", "invalid JSON body")
		return
	}

	current, err := getStoreSettings(s.db)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			httpx.Error(w, http.StatusNotFound, "STORE_NOT_CONFIGURED", "store settings not configured; seed or create first")
			return
		}
		slog.Error("get store settings", "err", err)
		httpx.Error(w, http.StatusInternalServerError, "INTERNAL", "could not load store settings")
		return
	}

	name := current.Name
	if body.Name != nil {
		name = strings.TrimSpace(*body.Name)
		if name == "" {
			httpx.Error(w, http.StatusBadRequest, "INVALID_NAME", "name must not be empty")
			return
		}
	}
	lat := current.Lat
	if body.Lat != nil {
		lat = *body.Lat
	}
	lng := current.Lng
	if body.Lng != nil {
		lng = *body.Lng
	}
	radius := current.MaxRadiusKm
	if body.MaxRadiusKm != nil {
		radius = *body.MaxRadiusKm
	}
	if err := validateStoreCoords(lat, lng, radius); err != nil {
		httpx.Error(w, http.StatusBadRequest, "INVALID_STORE", err.Error())
		return
	}

	var address any
	switch {
	case body.AddressText != nil:
		if t := strings.TrimSpace(*body.AddressText); t != "" {
			address = t
		} else {
			address = nil
		}
	case current.AddressText != nil:
		address = *current.AddressText
	default:
		address = nil
	}

	now := s.now().UTC().Format(time.RFC3339Nano)
	_, err = s.db.Exec(`
		UPDATE store_settings
		SET name = ?, lat = ?, lng = ?, max_radius_km = ?, address_text = ?, updated_at = ?, updated_by = ?
		WHERE id = ?
	`, name, lat, lng, radius, address, now, "admin", storeSettingsID)
	if err != nil {
		slog.Error("update store settings", "err", err)
		httpx.Error(w, http.StatusInternalServerError, "INTERNAL", "could not update store settings")
		return
	}

	row, err := getStoreSettings(s.db)
	if err != nil {
		slog.Error("reload store settings", "err", err)
		httpx.Error(w, http.StatusInternalServerError, "INTERNAL", "could not load store settings")
		return
	}
	// Admin response may include updated_at; still omit updated_by noise.
	httpx.JSON(w, http.StatusOK, map[string]any{
		"id":            row.ID,
		"name":          row.Name,
		"lat":           row.Lat,
		"lng":           row.Lng,
		"max_radius_km": row.MaxRadiusKm,
		"address_text":  row.AddressText,
		"updated_at":    row.UpdatedAt,
	})
}

func validateStoreCoords(lat, lng, maxRadiusKm float64) error {
	if lat < -90 || lat > 90 {
		return fmt.Errorf("lat must be between -90 and 90")
	}
	if lng < -180 || lng > 180 {
		return fmt.Errorf("lng must be between -180 and 180")
	}
	if maxRadiusKm <= 0 {
		return fmt.Errorf("max_radius_km must be > 0")
	}
	return nil
}
