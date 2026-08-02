package main

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"gas-tam-de/pkg/httpx"
	"gas-tam-de/pkg/sqlite"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func testGeoService(t *testing.T, gc geocoder, limiter *geoSearchRateLimiter) (*geoService, http.Handler) {
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
	svc := &geoService{
		db:       db,
		geocoder: gc,
		limiter:  limiter,
		cacheTTL: time.Hour,
		now:      func() time.Time { return time.Date(2026, 8, 2, 10, 0, 0, 0, time.UTC) },
	}
	r := httpx.NewRouter("geo-test")
	r.Get("/v1/geo/search", svc.handleSearch)
	return svc, r
}

func TestSearchPhotonProxy(t *testing.T) {
	var gotUA string
	var gotPath string
	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		gotUA = req.Header.Get("User-Agent")
		gotPath = req.URL.Path
		body := `{
			"type":"FeatureCollection",
			"features":[{
				"type":"Feature",
				"geometry":{"type":"Point","coordinates":[106.7009,10.7769]},
				"properties":{"name":"Chợ Bến Thành","city":"Ho Chi Minh City","country":"Vietnam","street":"Lê Lợi","housenumber":"1"}
			}]
		}`
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(body)),
			Header:     make(http.Header),
		}, nil
	})}
	gc := newPhotonGeocoder("https://photon.test", "GasTamDe-Test/1.0", client)
	_, r := testGeoService(t, gc, newGeoSearchRateLimiter(30))

	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/v1/geo/search?q=ben+thanh", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	if gotUA != "GasTamDe-Test/1.0" {
		t.Fatalf("user-agent=%q", gotUA)
	}
	if gotPath != "/api/" {
		t.Fatalf("path=%q", gotPath)
	}

	var resp struct {
		Items  []geoPlace `json:"items"`
		Cached bool       `json:"cached"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp.Cached || len(resp.Items) != 1 {
		t.Fatalf("resp=%+v", resp)
	}
	item := resp.Items[0]
	if item.Source != "photon" || item.Lat != 10.7769 || item.Lng != 106.7009 {
		t.Fatalf("item=%+v", item)
	}
	if !strings.Contains(item.Label, "Lê Lợi") || !strings.Contains(item.Label, "Vietnam") {
		t.Fatalf("label=%q", item.Label)
	}
}

func TestSearchUsesCache(t *testing.T) {
	var calls int
	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		calls++
		body := `{"features":[{"geometry":{"coordinates":[106.7,10.7]},"properties":{"name":"A","city":"B"}}]}`
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(body)),
			Header:     make(http.Header),
		}, nil
	})}
	gc := newPhotonGeocoder("https://photon.test", "ua", client)
	_, r := testGeoService(t, gc, nil)

	for i, wantCached := range []bool{false, true} {
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/v1/geo/search?q=cache-me", nil))
		if rr.Code != http.StatusOK {
			t.Fatalf("iter %d status=%d body=%s", i, rr.Code, rr.Body.String())
		}
		var resp struct {
			Cached bool `json:"cached"`
		}
		if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
			t.Fatal(err)
		}
		if resp.Cached != wantCached {
			t.Fatalf("iter %d cached=%v want %v", i, resp.Cached, wantCached)
		}
	}
	if calls != 1 {
		t.Fatalf("upstream calls=%d", calls)
	}
}

func TestSearchNominatimProxy(t *testing.T) {
	var gotUA string
	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		gotUA = req.Header.Get("User-Agent")
		body := `[{"display_name":"1 Lê Lợi, Quận 1, Hồ Chí Minh","lat":"10.77","lon":"106.70"}]`
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(body)),
			Header:     make(http.Header),
		}, nil
	})}
	gc := newNominatimGeocoder("https://nominatim.test", "GasTamDe-Test/1.0", client)
	gc.minInterval = 0 // skip 1req/s wait in unit test
	_, r := testGeoService(t, gc, nil)

	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/v1/geo/search?q=le+loi", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	if gotUA != "GasTamDe-Test/1.0" {
		t.Fatalf("user-agent=%q", gotUA)
	}
	var resp struct {
		Items []geoPlace `json:"items"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if len(resp.Items) != 1 || resp.Items[0].Source != "nominatim" {
		t.Fatalf("items=%+v", resp.Items)
	}
	if resp.Items[0].Lat != 10.77 || resp.Items[0].Lng != 106.70 {
		t.Fatalf("coords=%+v", resp.Items[0])
	}
}

func TestSearchValidationAndRateLimit(t *testing.T) {
	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(bytes.NewReader([]byte(`{"features":[]}`))),
			Header:     make(http.Header),
		}, nil
	})}
	gc := newPhotonGeocoder("https://photon.test", "ua", client)
	limiter := newGeoSearchRateLimiter(2)
	_, r := testGeoService(t, gc, limiter)

	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/v1/geo/search?q=a", nil))
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("short q status=%d", rr.Code)
	}

	rr = httptest.NewRecorder()
	r.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/v1/geo/search?q=ok&limit=99", nil))
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("limit status=%d", rr.Code)
	}

	reqOK := func() *http.Request {
		req := httptest.NewRequest(http.MethodGet, "/v1/geo/search?q=quan+1", nil)
		req.RemoteAddr = "203.0.113.10:1234"
		return req
	}
	for i := 0; i < 2; i++ {
		rr = httptest.NewRecorder()
		r.ServeHTTP(rr, reqOK())
		if rr.Code != http.StatusOK {
			t.Fatalf("ok %d status=%d body=%s", i, rr.Code, rr.Body.String())
		}
	}
	rr = httptest.NewRecorder()
	r.ServeHTTP(rr, reqOK())
	if rr.Code != http.StatusTooManyRequests {
		t.Fatalf("rate limit status=%d body=%s", rr.Code, rr.Body.String())
	}
	if rr.Header().Get("Retry-After") == "" {
		t.Fatal("missing Retry-After")
	}
}

func TestSearchUpstreamError(t *testing.T) {
	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusServiceUnavailable,
			Body:       io.NopCloser(strings.NewReader("down")),
			Header:     make(http.Header),
		}, nil
	})}
	gc := newPhotonGeocoder("https://photon.test", "ua", client)
	_, r := testGeoService(t, gc, nil)

	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/v1/geo/search?q=fail", nil))
	if rr.Code != http.StatusBadGateway {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestPhotonLabel(t *testing.T) {
	label := photonLabel(photonProperties{
		Name: "Chợ", Street: "Lê Lợi", Housenumber: "1", City: "HCM", Country: "Vietnam",
	})
	if label != "1 Lê Lợi, Chợ, HCM, Vietnam" {
		t.Fatalf("label=%q", label)
	}
}

func TestNewGeocoderProvider(t *testing.T) {
	if newGeocoder("photon", "", "", nil).Name() != "photon" {
		t.Fatal("photon")
	}
	if newGeocoder("nominatim", "", "", nil).Name() != "nominatim" {
		t.Fatal("nominatim")
	}
	if newGeocoder("", "", "", nil).Name() != "photon" {
		t.Fatal("default")
	}
}

func TestNominatimRateSerializes(t *testing.T) {
	var mu sync.Mutex
	var starts []time.Time
	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		mu.Lock()
		starts = append(starts, time.Now())
		mu.Unlock()
		body := `[{"display_name":"X","lat":"1","lon":"2"}]`
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(body)),
			Header:     make(http.Header),
		}, nil
	})}
	gc := newNominatimGeocoder("https://nominatim.test", "ua", client)
	gc.minInterval = 50 * time.Millisecond

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		_, _ = gc.Search(httptest.NewRequest(http.MethodGet, "/", nil).Context(), "a", 1)
	}()
	go func() {
		defer wg.Done()
		_, _ = gc.Search(httptest.NewRequest(http.MethodGet, "/", nil).Context(), "b", 1)
	}()
	wg.Wait()
	if len(starts) != 2 {
		t.Fatalf("starts=%d", len(starts))
	}
	delta := starts[1].Sub(starts[0])
	if delta < 0 {
		delta = -delta
	}
	if delta < 40*time.Millisecond {
		t.Fatalf("expected spacing, delta=%v", delta)
	}
}
