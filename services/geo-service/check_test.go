package main

import (
	"bytes"
	"encoding/json"
	"math"
	"net/http"
	"net/http/httptest"
	"testing"

	"gas-tam-de/pkg/httpx"
)

func TestHaversineKmSamePoint(t *testing.T) {
	d := haversineKm(10.7769, 106.7009, 10.7769, 106.7009)
	if d != 0 {
		t.Fatalf("same point distance = %v, want 0", d)
	}
}

func TestHaversineKmEquatorOneDegreeLongitude(t *testing.T) {
	// 1° longitude at equator ≈ R * π/180 km.
	want := earthRadiusKm * math.Pi / 180
	got := haversineKm(0, 0, 0, 1)
	if math.Abs(got-want) > 1e-9 {
		t.Fatalf("got %v want %v", got, want)
	}
}

func TestHaversineKmKnownCities(t *testing.T) {
	// Paris → London ≈ 343.5 km (tolerance for R=6371 convention).
	got := haversineKm(48.8566, 2.3522, 51.5074, -0.1278)
	if got < 340 || got > 350 {
		t.Fatalf("Paris–London distance = %v, expected ~343–344 km", got)
	}
}

func TestRoundDistanceKm(t *testing.T) {
	cases := []struct {
		in, want float64
	}{
		{3.14159, 3.14},
		{3.145, 3.15},
		{0, 0},
		{12.004, 12},
	}
	for _, tc := range cases {
		if got := roundDistanceKm(tc.in); got != tc.want {
			t.Fatalf("roundDistanceKm(%v)=%v want %v", tc.in, got, tc.want)
		}
	}
}

func TestInRangeInclusive(t *testing.T) {
	if !inRange(10, 10) {
		t.Fatal("boundary distance should be in_range")
	}
	if !inRange(9.99, 10) {
		t.Fatal("inside radius should be in_range")
	}
	if inRange(10.01, 10) {
		t.Fatal("outside radius should not be in_range")
	}
}

func testCheckService(t *testing.T) (*geoService, http.Handler) {
	t.Helper()
	svc, _ := testStoreService(t)
	r := httpx.NewRouter("geo-check-test")
	r.Post("/v1/geo/check", svc.handleCheck)
	return svc, r
}

func TestHandleCheckInRange(t *testing.T) {
	_, h := testCheckService(t)
	// ~3 km north of default store (1° lat ≈ 111 km → 0.027° ≈ 3 km).
	body := `{"lat":10.8039,"lng":106.7009}`
	req := httptest.NewRequest(http.MethodPost, "/v1/geo/check", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var res checkResult
	if err := json.Unmarshal(rr.Body.Bytes(), &res); err != nil {
		t.Fatal(err)
	}
	if res.MaxRadiusKm != 10 {
		t.Fatalf("max_radius_km=%v", res.MaxRadiusKm)
	}
	if res.DistanceKm < 2.5 || res.DistanceKm > 3.5 {
		t.Fatalf("distance_km=%v, expected ~3", res.DistanceKm)
	}
	if !res.InRange {
		t.Fatalf("expected in_range for ~3km: %+v", res)
	}
	// Ensure API rounds to 2 decimals.
	if res.DistanceKm != roundDistanceKm(res.DistanceKm) {
		t.Fatalf("distance_km not rounded to 2dp: %v", res.DistanceKm)
	}
}

func TestHandleCheckOutOfRange(t *testing.T) {
	_, h := testCheckService(t)
	// ~12 km north of default store.
	body := `{"lat":10.885,"lng":106.7009}`
	req := httptest.NewRequest(http.MethodPost, "/v1/geo/check", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var res checkResult
	if err := json.Unmarshal(rr.Body.Bytes(), &res); err != nil {
		t.Fatal(err)
	}
	if res.DistanceKm < 11 || res.DistanceKm > 13 {
		t.Fatalf("distance_km=%v, expected ~12", res.DistanceKm)
	}
	if res.InRange {
		t.Fatalf("expected out of range: %+v", res)
	}
}

func TestHandleCheckSameAsStore(t *testing.T) {
	_, h := testCheckService(t)
	body := `{"lat":10.7769,"lng":106.7009}`
	req := httptest.NewRequest(http.MethodPost, "/v1/geo/check", bytes.NewBufferString(body))
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var res checkResult
	if err := json.Unmarshal(rr.Body.Bytes(), &res); err != nil {
		t.Fatal(err)
	}
	if res.DistanceKm != 0 || !res.InRange {
		t.Fatalf("same point: %+v", res)
	}
}

func TestHandleCheckInvalidCoords(t *testing.T) {
	_, h := testCheckService(t)
	req := httptest.NewRequest(http.MethodPost, "/v1/geo/check", bytes.NewBufferString(`{"lat":99,"lng":0}`))
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status=%d want 400", rr.Code)
	}
}

func TestHandleCheckInvalidBody(t *testing.T) {
	_, h := testCheckService(t)
	req := httptest.NewRequest(http.MethodPost, "/v1/geo/check", bytes.NewBufferString(`not-json`))
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status=%d want 400", rr.Code)
	}
}

func TestHandleCheckStoreNotConfigured(t *testing.T) {
	db := openStoreTestDB(t)
	svc := &geoService{db: db}
	r := httpx.NewRouter("geo-check-empty")
	r.Post("/v1/geo/check", svc.handleCheck)

	req := httptest.NewRequest(http.MethodPost, "/v1/geo/check", bytes.NewBufferString(`{"lat":10,"lng":106}`))
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
}
