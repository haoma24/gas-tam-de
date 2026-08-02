package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"gas-tam-de/pkg/httpx"
)

const deskSettingsID = "default"

type deskSettings struct {
	ID                string `json:"id"`
	WaitBlueMaxMin    int    `json:"wait_blue_max_min"`
	WaitOrangeMaxMin  int    `json:"wait_orange_max_min"`
	WaitRedMaxMin     int    `json:"wait_red_max_min"`
	AlertEnabled      bool   `json:"alert_enabled"`
	AlertIntervalSec  int    `json:"alert_interval_sec"`
	UpdatedAt         string `json:"updated_at,omitempty"`
}

type putDeskSettingsBody struct {
	WaitBlueMaxMin   *int  `json:"wait_blue_max_min"`
	WaitOrangeMaxMin *int  `json:"wait_orange_max_min"`
	WaitRedMaxMin    *int  `json:"wait_red_max_min"`
	AlertEnabled     *bool `json:"alert_enabled"`
	AlertIntervalSec *int  `json:"alert_interval_sec"`
}

func seedDeskSettings(db *sql.DB) error {
	var id string
	err := db.QueryRow(`SELECT id FROM desk_settings WHERE id = ?`, deskSettingsID).Scan(&id)
	if err == nil {
		return nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return err
	}
	// Ensure table exists for DBs created before this migration.
	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS desk_settings (
		  id                   TEXT PRIMARY KEY,
		  wait_blue_max_min    INTEGER NOT NULL DEFAULT 5,
		  wait_orange_max_min  INTEGER NOT NULL DEFAULT 15,
		  wait_red_max_min     INTEGER NOT NULL DEFAULT 30,
		  alert_enabled        INTEGER NOT NULL DEFAULT 1,
		  alert_interval_sec   INTEGER NOT NULL DEFAULT 300,
		  updated_at           TEXT NOT NULL
		)`); err != nil {
		return err
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	_, err = db.Exec(`
		INSERT INTO desk_settings (
			id, wait_blue_max_min, wait_orange_max_min, wait_red_max_min,
			alert_enabled, alert_interval_sec, updated_at
		) VALUES (?, 5, 15, 30, 1, 300, ?)
	`, deskSettingsID, now)
	return err
}

func (s *orderService) handleGetDeskSettings(w http.ResponseWriter, _ *http.Request) {
	row, err := getDeskSettings(s.db)
	if err != nil {
		slog.Error("get desk settings", "err", err)
		httpx.Error(w, http.StatusInternalServerError, "INTERNAL", "could not load desk settings")
		return
	}
	httpx.JSON(w, http.StatusOK, row)
}

func (s *orderService) handlePutDeskSettings(w http.ResponseWriter, r *http.Request) {
	var body putDeskSettingsBody
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&body); err != nil {
		httpx.Error(w, http.StatusBadRequest, "INVALID_JSON", "invalid request body")
		return
	}
	cur, err := getDeskSettings(s.db)
	if err != nil {
		slog.Error("get desk settings", "err", err)
		httpx.Error(w, http.StatusInternalServerError, "INTERNAL", "could not load desk settings")
		return
	}
	blue := cur.WaitBlueMaxMin
	orange := cur.WaitOrangeMaxMin
	red := cur.WaitRedMaxMin
	alertOn := cur.AlertEnabled
	interval := cur.AlertIntervalSec
	if body.WaitBlueMaxMin != nil {
		blue = *body.WaitBlueMaxMin
	}
	if body.WaitOrangeMaxMin != nil {
		orange = *body.WaitOrangeMaxMin
	}
	if body.WaitRedMaxMin != nil {
		red = *body.WaitRedMaxMin
	}
	if body.AlertEnabled != nil {
		alertOn = *body.AlertEnabled
	}
	if body.AlertIntervalSec != nil {
		interval = *body.AlertIntervalSec
	}
	if err := validateDeskSettings(blue, orange, red, interval); err != nil {
		httpx.Error(w, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	enabled := 0
	if alertOn {
		enabled = 1
	}
	_, err = s.db.Exec(`
		UPDATE desk_settings SET
			wait_blue_max_min = ?, wait_orange_max_min = ?, wait_red_max_min = ?,
			alert_enabled = ?, alert_interval_sec = ?, updated_at = ?
		WHERE id = ?
	`, blue, orange, red, enabled, interval, now, deskSettingsID)
	if err != nil {
		slog.Error("put desk settings", "err", err)
		httpx.Error(w, http.StatusInternalServerError, "INTERNAL", "could not update desk settings")
		return
	}
	row, err := getDeskSettings(s.db)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "INTERNAL", "could not load desk settings")
		return
	}
	httpx.JSON(w, http.StatusOK, row)
}

func getDeskSettings(db *sql.DB) (deskSettings, error) {
	_ = seedDeskSettings(db)
	var s deskSettings
	var enabled int
	err := db.QueryRow(`
		SELECT id, wait_blue_max_min, wait_orange_max_min, wait_red_max_min,
		       alert_enabled, alert_interval_sec, updated_at
		FROM desk_settings WHERE id = ?
	`, deskSettingsID).Scan(
		&s.ID, &s.WaitBlueMaxMin, &s.WaitOrangeMaxMin, &s.WaitRedMaxMin,
		&enabled, &s.AlertIntervalSec, &s.UpdatedAt,
	)
	if err != nil {
		return deskSettings{}, err
	}
	s.AlertEnabled = enabled == 1
	return s, nil
}

func validateDeskSettings(blue, orange, red, intervalSec int) error {
	if blue <= 0 || orange <= blue || red <= orange {
		return fmt.Errorf("wait thresholds must satisfy 0 < blue < orange < red (minutes)")
	}
	if intervalSec < 30 {
		return fmt.Errorf("alert_interval_sec must be >= 30")
	}
	return nil
}
