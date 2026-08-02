package main

import (
	"database/sql"
	"embed"
	"log/slog"
	"os"
	"strings"
	"time"

	"gas-tam-de/pkg/config"
	"gas-tam-de/pkg/httpx"
	"gas-tam-de/pkg/sqlite"
)

const serviceName = "geo-service"

//go:embed schema.sql
var schemaFS embed.FS

func main() {
	addr := config.Get("GEO_ADDR", ":8083")
	dbPath := config.Get("GEO_DB", "data/geo.db")

	db, err := sqlite.Open(dbPath)
	if err != nil {
		slog.Error("open db", "err", err)
		os.Exit(1)
	}
	defer db.Close()

	if err := migrate(db); err != nil {
		slog.Error("migrate", "err", err)
		os.Exit(1)
	}

	if err := seedStoreSettings(db, loadStoreSeedConfig()); err != nil {
		slog.Error("seed store settings", "err", err)
		os.Exit(1)
	}

	svc := newGeoServiceFromEnv(db)

	r := httpx.NewRouter(serviceName)
	httpx.MountHealth(r, serviceName)

	r.Get("/v1/geo/store", svc.handleGetStore)
	r.Get("/v1/geo/search", svc.handleSearch)
	r.Post("/v1/geo/check", svc.handleCheck)
	r.Put("/v1/admin/geo/store", svc.handlePutAdminStore)

	if err := httpx.ListenAndServe(addr, serviceName, r); err != nil {
		slog.Error("server stopped", "err", err)
		os.Exit(1)
	}
}

func migrate(db *sql.DB) error {
	sqlBytes, err := schemaFS.ReadFile("schema.sql")
	if err != nil {
		return err
	}
	_, err = db.Exec(string(sqlBytes))
	return err
}

func newGeoServiceFromEnv(db *sql.DB) *geoService {
	provider := config.Get("GEOCODE_PROVIDER", "photon")
	baseURL := config.Get("GEOCODE_BASE_URL", "")
	userAgent := config.Get("GEOCODE_USER_AGENT", defaultUserAgent)
	cacheTTLHours := config.GetInt("GEOCODE_CACHE_TTL_HOURS", 24)
	maxPerMin := config.GetInt("GEOCODE_MAX_PER_IP_MINUTE", 30)

	if strings.EqualFold(provider, "nominatim") && strings.TrimSpace(baseURL) == "" {
		baseURL = "https://nominatim.openstreetmap.org"
	}

	return &geoService{
		db:       db,
		geocoder: newGeocoder(provider, baseURL, userAgent, nil),
		limiter:  newGeoSearchRateLimiter(maxPerMin),
		cacheTTL: time.Duration(cacheTTLHours) * time.Hour,
		now:      time.Now,
	}
}
