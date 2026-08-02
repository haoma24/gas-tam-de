package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"gas-tam-de/pkg/httpx"
	"gas-tam-de/pkg/sqlite"
)

func openStoreTestDB(t *testing.T) *sql.DB {
	t.Helper()
	dir := t.TempDir()
	db, err := sqlite.Open(filepath.Join(dir, "geo.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := migrate(db); err != nil {
		t.Fatal(err)
	}
	return db
}

func testStoreService(t *testing.T) (*geoService, http.Handler) {
	t.Helper()
	db := openStoreTestDB(t)
	cfg := storeSeedConfig{
		Name:        "Gas Tam Đệ",
		Lat:         10.7769,
		Lng:         106.7009,
		MaxRadiusKm: 10,
		Enabled:     true,
	}
	if err := seedStoreSettings(db, cfg); err != nil {
		t.Fatal(err)
	}
	svc := &geoService{
		db:  db,
		now: func() time.Time { return time.Date(2026, 8, 2, 12, 0, 0, 0, time.UTC) },
	}
	r := httpx.NewRouter("geo-test")
	r.Get("/v1/geo/store", svc.handleGetStore)
	r.Put("/v1/admin/geo/store", svc.handlePutAdminStore)
	return svc, r
}

func TestSeedStoreSettingsCreatesSingleton(t *testing.T) {
	db := openStoreTestDB(t)
	cfg := storeSeedConfig{
		Name:        "Shop Test",
		Lat:         10.8,
		Lng:         106.7,
		MaxRadiusKm: 12.5,
		AddressText: "1 Lê Lợi",
		Enabled:     true,
	}
	if err := seedStoreSettings(db, cfg); err != nil {
		t.Fatal(err)
	}

	row, err := getStoreSettings(db)
	if err != nil {
		t.Fatal(err)
	}
	if row.ID != storeSettingsID || row.Name != "Shop Test" {
		t.Fatalf("id/name: %+v", row)
	}
	if row.Lat != 10.8 || row.Lng != 106.7 || row.MaxRadiusKm != 12.5 {
		t.Fatalf("coords: %+v", row)
	}
	if row.AddressText == nil || *row.AddressText != "1 Lê Lợi" {
		t.Fatalf("address: %+v", row.AddressText)
	}
	if row.UpdatedAt == "" {
		t.Fatal("expected updated_at")
	}
}

func TestSeedStoreSettingsIdempotent(t *testing.T) {
	db := openStoreTestDB(t)
	cfg := storeSeedConfig{Name: "A", Lat: 10, Lng: 106, MaxRadiusKm: 10, Enabled: true}
	if err := seedStoreSettings(db, cfg); err != nil {
		t.Fatal(err)
	}
	cfg.Lat = 11
	cfg.MaxRadiusKm = 99
	if err := seedStoreSettings(db, cfg); err != nil {
		t.Fatal(err)
	}

	row, err := getStoreSettings(db)
	if err != nil {
		t.Fatal(err)
	}
	if row.Lat != 10 || row.MaxRadiusKm != 10 {
		t.Fatalf("seed must not overwrite existing: %+v", row)
	}

	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM store_settings`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("expected 1 row, got %d", count)
	}
}

func TestSeedStoreSettingsDisabled(t *testing.T) {
	db := openStoreTestDB(t)
	cfg := storeSeedConfig{Name: "A", Lat: 10, Lng: 106, MaxRadiusKm: 10, Enabled: false}
	if err := seedStoreSettings(db, cfg); err != nil {
		t.Fatal(err)
	}
	_, err := getStoreSettings(db)
	if err != sql.ErrNoRows {
		t.Fatalf("expected no rows, got err=%v", err)
	}
}

func TestSeedStoreSettingsRejectsInvalidCoords(t *testing.T) {
	db := openStoreTestDB(t)
	err := seedStoreSettings(db, storeSeedConfig{Lat: 91, Lng: 0, MaxRadiusKm: 10, Enabled: true})
	if err == nil {
		t.Fatal("expected error for invalid lat")
	}
}

func TestHandleGetStore(t *testing.T) {
	_, h := testStoreService(t)
	req := httptest.NewRequest(http.MethodGet, "/v1/geo/store", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}

	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["lat"] != 10.7769 || body["lng"] != 106.7009 || body["max_radius_km"] != 10.0 {
		t.Fatalf("unexpected body: %v", body)
	}
	if body["name"] != "Gas Tam Đệ" {
		t.Fatalf("name=%v", body["name"])
	}
	if _, ok := body["updated_by"]; ok {
		t.Fatal("public GET must not expose updated_by")
	}
}

func TestHandleGetStoreNotConfigured(t *testing.T) {
	db := openStoreTestDB(t)
	svc := &geoService{db: db, now: time.Now}
	r := httpx.NewRouter("geo-test")
	r.Get("/v1/geo/store", svc.handleGetStore)

	req := httptest.NewRequest(http.MethodGet, "/v1/geo/store", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status=%d", rec.Code)
	}
}

func TestHandlePutAdminStore(t *testing.T) {
	_, h := testStoreService(t)
	payload := `{"lat":10.5,"lng":106.5,"max_radius_km":8,"name":"Gas Tam Đệ CN2","address_text":"Q1"}`
	req := httptest.NewRequest(http.MethodPut, "/v1/admin/geo/store", bytes.NewBufferString(payload))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}

	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["lat"] != 10.5 || body["lng"] != 106.5 || body["max_radius_km"] != 8.0 {
		t.Fatalf("body=%v", body)
	}
	if body["name"] != "Gas Tam Đệ CN2" {
		t.Fatalf("name=%v", body["name"])
	}
	if body["updated_at"] == "" {
		t.Fatal("expected updated_at")
	}

	// Public GET reflects update.
	req2 := httptest.NewRequest(http.MethodGet, "/v1/geo/store", nil)
	rec2 := httptest.NewRecorder()
	h.ServeHTTP(rec2, req2)
	if rec2.Code != http.StatusOK {
		t.Fatalf("get status=%d", rec2.Code)
	}
	var pub map[string]any
	_ = json.Unmarshal(rec2.Body.Bytes(), &pub)
	if pub["max_radius_km"] != 8.0 || pub["address_text"] != "Q1" {
		t.Fatalf("public after put: %v", pub)
	}
}

func TestHandlePutAdminStoreInvalidRadius(t *testing.T) {
	_, h := testStoreService(t)
	payload := `{"max_radius_km":0}`
	req := httptest.NewRequest(http.MethodPut, "/v1/admin/geo/store", bytes.NewBufferString(payload))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestValidateStoreCoords(t *testing.T) {
	if err := validateStoreCoords(10, 106, 10); err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		lat, lng, r float64
	}{
		{91, 0, 10},
		{-91, 0, 10},
		{0, 181, 10},
		{0, -181, 10},
		{0, 0, 0},
		{0, 0, -1},
	}
	for _, c := range cases {
		if err := validateStoreCoords(c.lat, c.lng, c.r); err == nil {
			t.Fatalf("expected error for %+v", c)
		}
	}
}
