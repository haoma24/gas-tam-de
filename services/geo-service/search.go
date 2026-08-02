package main

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"gas-tam-de/pkg/httpx"

	"github.com/google/uuid"
)

const (
	defaultSearchLimit = 5
	maxSearchLimit     = 10
	minQueryLen        = 2
	maxQueryLen        = 200
)

// geoPlace is a normalized geocode suggestion for clients (Flutter autocomplete later).
type geoPlace struct {
	Label  string  `json:"label"`
	Lat    float64 `json:"lat"`
	Lng    float64 `json:"lng"`
	Source string  `json:"source"`
}

type geoService struct {
	db       *sql.DB
	geocoder geocoder
	limiter  *geoSearchRateLimiter
	cacheTTL time.Duration
	now      func() time.Time
}

func (s *geoService) handleSearch(w http.ResponseWriter, r *http.Request) {
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	if len(q) < minQueryLen {
		httpx.Error(w, http.StatusBadRequest, "INVALID_QUERY", "q must be at least 2 characters")
		return
	}
	if len(q) > maxQueryLen {
		httpx.Error(w, http.StatusBadRequest, "INVALID_QUERY", "q is too long")
		return
	}

	limit := defaultSearchLimit
	if raw := strings.TrimSpace(r.URL.Query().Get("limit")); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 1 || n > maxSearchLimit {
			httpx.Error(w, http.StatusBadRequest, "INVALID_LIMIT", fmt.Sprintf("limit must be 1..%d", maxSearchLimit))
			return
		}
		limit = n
	}

	ip := clientIP(r)
	now := s.now()
	if s.limiter != nil {
		if res := s.limiter.Allow(ip, now); !res.Allowed {
			w.Header().Set("Retry-After", strconv.Itoa(res.RetryAfterSec))
			httpx.Error(w, http.StatusTooManyRequests, "RATE_LIMITED", "too many geocode searches; retry later")
			return
		}
	}

	cacheKey := searchCacheKey(s.geocoder.Name(), q, limit)
	if items, ok, err := s.getCachedSearch(cacheKey, now); err != nil {
		slog.Error("geocode cache read", "err", err)
	} else if ok {
		httpx.JSON(w, http.StatusOK, map[string]any{"items": items, "cached": true})
		return
	}

	items, err := s.geocoder.Search(r.Context(), q, limit)
	if err != nil {
		if errors.Is(err, errGeocodeUpstream) {
			slog.Error("geocode upstream", "provider", s.geocoder.Name(), "err", err)
			httpx.Error(w, http.StatusBadGateway, "GEOCODE_UPSTREAM", "geocode provider unavailable")
			return
		}
		slog.Error("geocode search", "provider", s.geocoder.Name(), "err", err)
		httpx.Error(w, http.StatusInternalServerError, "INTERNAL", "could not search address")
		return
	}
	if items == nil {
		items = []geoPlace{}
	}

	if err := s.putCachedSearch(cacheKey, items, now); err != nil {
		slog.Warn("geocode cache write", "err", err)
	}

	httpx.JSON(w, http.StatusOK, map[string]any{"items": items, "cached": false})
}

func searchCacheKey(provider, q string, limit int) string {
	norm := strings.ToLower(strings.Join(strings.Fields(q), " "))
	sum := sha256.Sum256([]byte(fmt.Sprintf("%s|%s|%d", provider, norm, limit)))
	return hex.EncodeToString(sum[:])
}

func (s *geoService) getCachedSearch(queryHash string, now time.Time) ([]geoPlace, bool, error) {
	var raw, expiresAt string
	err := s.db.QueryRow(`
		SELECT result_json, expires_at FROM geocode_cache WHERE query_hash = ?`, queryHash).
		Scan(&raw, &expiresAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	exp, err := time.Parse(time.RFC3339, expiresAt)
	if err != nil || !exp.After(now) {
		_, _ = s.db.Exec(`DELETE FROM geocode_cache WHERE query_hash = ?`, queryHash)
		return nil, false, nil
	}
	var items []geoPlace
	if err := json.Unmarshal([]byte(raw), &items); err != nil {
		_, _ = s.db.Exec(`DELETE FROM geocode_cache WHERE query_hash = ?`, queryHash)
		return nil, false, nil
	}
	return items, true, nil
}

func (s *geoService) putCachedSearch(queryHash string, items []geoPlace, now time.Time) error {
	raw, err := json.Marshal(items)
	if err != nil {
		return err
	}
	ttl := s.cacheTTL
	if ttl <= 0 {
		ttl = 24 * time.Hour
	}
	expires := now.Add(ttl).UTC().Format(time.RFC3339)
	_, err = s.db.Exec(`
		INSERT INTO geocode_cache (id, query_hash, result_json, expires_at)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(query_hash) DO UPDATE SET
			result_json = excluded.result_json,
			expires_at = excluded.expires_at`,
		uuid.NewString(), queryHash, string(raw), expires)
	return err
}

func clientIP(r *http.Request) string {
	if xff := strings.TrimSpace(r.Header.Get("X-Forwarded-For")); xff != "" {
		parts := strings.Split(xff, ",")
		return strings.TrimSpace(parts[0])
	}
	if xri := strings.TrimSpace(r.Header.Get("X-Real-IP")); xri != "" {
		return xri
	}
	host := r.RemoteAddr
	if i := strings.LastIndex(host, ":"); i > 0 {
		return host[:i]
	}
	return host
}
