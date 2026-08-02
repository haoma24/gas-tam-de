package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"log/slog"
	"math"
	"net/http"

	"gas-tam-de/pkg/httpx"
)

// Mean Earth radius in km (WGS84 / common Haversine convention).
const earthRadiusKm = 6371.0

type checkBody struct {
	Lat float64 `json:"lat"`
	Lng float64 `json:"lng"`
}

type checkResult struct {
	DistanceKm  float64 `json:"distance_km"`
	InRange     bool    `json:"in_range"`
	MaxRadiusKm float64 `json:"max_radius_km"`
}

// haversineKm returns great-circle distance in kilometres between two WGS84 points.
func haversineKm(lat1, lng1, lat2, lng2 float64) float64 {
	φ1 := lat1 * math.Pi / 180
	φ2 := lat2 * math.Pi / 180
	Δφ := (lat2 - lat1) * math.Pi / 180
	Δλ := (lng2 - lng1) * math.Pi / 180

	sinΔφ := math.Sin(Δφ / 2)
	sinΔλ := math.Sin(Δλ / 2)
	a := sinΔφ*sinΔφ + math.Cos(φ1)*math.Cos(φ2)*sinΔλ*sinΔλ
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	return earthRadiusKm * c
}

// roundDistanceKm rounds to 2 decimal places for API display (architecture §6.3).
func roundDistanceKm(km float64) float64 {
	return math.Round(km*100) / 100
}

// inRange reports whether distance is within the delivery radius (inclusive).
func inRange(distanceKm, maxRadiusKm float64) bool {
	return distanceKm <= maxRadiusKm
}

func (s *geoService) handleCheck(w http.ResponseWriter, r *http.Request) {
	var body checkBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		httpx.Error(w, http.StatusBadRequest, "INVALID_BODY", "invalid JSON body")
		return
	}
	if err := validateCustomerCoords(body.Lat, body.Lng); err != nil {
		httpx.Error(w, http.StatusBadRequest, "INVALID_COORDS", err.Error())
		return
	}

	store, err := getStoreSettings(s.db)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			httpx.Error(w, http.StatusNotFound, "STORE_NOT_CONFIGURED", "store settings not configured")
			return
		}
		slog.Error("get store settings", "err", err)
		httpx.Error(w, http.StatusInternalServerError, "INTERNAL", "could not load store settings")
		return
	}

	raw := haversineKm(store.Lat, store.Lng, body.Lat, body.Lng)
	dist := roundDistanceKm(raw)
	httpx.JSON(w, http.StatusOK, checkResult{
		DistanceKm:  dist,
		InRange:     inRange(dist, store.MaxRadiusKm),
		MaxRadiusKm: store.MaxRadiusKm,
	})
}

func validateCustomerCoords(lat, lng float64) error {
	if lat < -90 || lat > 90 {
		return errors.New("lat must be between -90 and 90")
	}
	if lng < -180 || lng > 180 {
		return errors.New("lng must be between -180 and 180")
	}
	return nil
}
