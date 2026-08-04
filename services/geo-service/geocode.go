package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

var errGeocodeUpstream = errors.New("geocode upstream error")

// geocoder proxies address search to Photon or Nominatim.
type geocoder interface {
	Name() string
	Search(ctx context.Context, q string, limit int) ([]geoPlace, error)
}

type httpDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

const defaultUserAgent = "GasTamDe/1.0 (local-dev; geo-service; contact:local)"

// --- Photon (Komoot) — preferred for autocomplete ---

type photonGeocoder struct {
	baseURL    string
	userAgent  string
	httpClient httpDoer
}

func newPhotonGeocoder(baseURL, userAgent string, client httpDoer) *photonGeocoder {
	if strings.TrimSpace(baseURL) == "" {
		baseURL = "https://photon.komoot.io"
	}
	if strings.TrimSpace(userAgent) == "" {
		userAgent = defaultUserAgent
	}
	if client == nil {
		client = &http.Client{Timeout: 8 * time.Second}
	}
	return &photonGeocoder{baseURL: strings.TrimRight(baseURL, "/"), userAgent: userAgent, httpClient: client}
}

func (p *photonGeocoder) Name() string { return "photon" }

func (p *photonGeocoder) Search(ctx context.Context, q string, limit int) ([]geoPlace, error) {
	u, err := url.Parse(p.baseURL + "/api/")
	if err != nil {
		return nil, err
	}
	qs := u.Query()
	qs.Set("q", q)
	qs.Set("limit", strconv.Itoa(limit))
	qs.Set("lang", "vi")
	u.RawQuery = qs.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", p.userAgent)
	req.Header.Set("Accept", "application/json")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", errGeocodeUpstream, err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, fmt.Errorf("%w: read body: %v", errGeocodeUpstream, err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("%w: status %d", errGeocodeUpstream, resp.StatusCode)
	}

	var fc photonFeatureCollection
	if err := json.Unmarshal(body, &fc); err != nil {
		return nil, fmt.Errorf("%w: decode: %v", errGeocodeUpstream, err)
	}
	items := make([]geoPlace, 0, len(fc.Features))
	for _, f := range fc.Features {
		if len(f.Geometry.Coordinates) < 2 {
			continue
		}
		lng := f.Geometry.Coordinates[0]
		lat := f.Geometry.Coordinates[1]
		label := photonLabel(f.Properties)
		if label == "" {
			continue
		}
		items = append(items, geoPlace{Label: label, Lat: lat, Lng: lng, Source: "photon"})
	}
	return items, nil
}

type photonFeatureCollection struct {
	Features []photonFeature `json:"features"`
}

type photonFeature struct {
	Geometry   photonGeometry   `json:"geometry"`
	Properties photonProperties `json:"properties"`
}

type photonGeometry struct {
	Coordinates []float64 `json:"coordinates"`
}

type photonProperties struct {
	Name        string `json:"name"`
	Street      string `json:"street"`
	Housenumber string `json:"housenumber"`
	City        string `json:"city"`
	Locality    string `json:"locality"`
	District    string `json:"district"`
	County      string `json:"county"`
	State       string `json:"state"`
	Country     string `json:"country"`
	Postcode    string `json:"postcode"`
}

func photonLabel(p photonProperties) string {
	var parts []string
	street := strings.TrimSpace(strings.TrimSpace(p.Housenumber) + " " + strings.TrimSpace(p.Street))
	if street != "" {
		parts = append(parts, street)
	}
	if n := strings.TrimSpace(p.Name); n != "" && !containsFold(parts, n) {
		parts = append(parts, n)
	}
	for _, v := range []string{p.Locality, p.District, p.City, p.County, p.State, p.Country} {
		v = strings.TrimSpace(v)
		if v == "" || containsFold(parts, v) {
			continue
		}
		parts = append(parts, v)
	}
	return strings.Join(parts, ", ")
}

func containsFold(parts []string, s string) bool {
	for _, p := range parts {
		if strings.EqualFold(p, s) {
			return true
		}
	}
	return false
}

// --- Nominatim (OSM) — requires identifying User-Agent; max ~1 req/s ---

type nominatimGeocoder struct {
	baseURL     string
	userAgent   string
	httpClient  httpDoer
	minInterval time.Duration
	lastMu      chan struct{} // mutex via channel size 1
	lastAt      time.Time
}

func newNominatimGeocoder(baseURL, userAgent string, client httpDoer) *nominatimGeocoder {
	if strings.TrimSpace(baseURL) == "" {
		baseURL = "https://nominatim.openstreetmap.org"
	}
	if strings.TrimSpace(userAgent) == "" {
		userAgent = defaultUserAgent
	}
	if client == nil {
		client = &http.Client{Timeout: 8 * time.Second}
	}
	ch := make(chan struct{}, 1)
	ch <- struct{}{}
	return &nominatimGeocoder{
		baseURL:     strings.TrimRight(baseURL, "/"),
		userAgent:   userAgent,
		httpClient:  client,
		minInterval: time.Second, // Nominatim usage policy
		lastMu:      ch,
	}
}

func (n *nominatimGeocoder) Name() string { return "nominatim" }

func (n *nominatimGeocoder) Search(ctx context.Context, q string, limit int) ([]geoPlace, error) {
	if err := n.waitSlot(ctx); err != nil {
		return nil, err
	}

	u, err := url.Parse(n.baseURL + "/search")
	if err != nil {
		return nil, err
	}
	qs := u.Query()
	qs.Set("q", q)
	qs.Set("format", "json")
	qs.Set("limit", strconv.Itoa(limit))
	qs.Set("addressdetails", "0")
	u.RawQuery = qs.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", n.userAgent)
	req.Header.Set("Accept", "application/json")

	resp, err := n.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", errGeocodeUpstream, err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, fmt.Errorf("%w: read body: %v", errGeocodeUpstream, err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("%w: status %d", errGeocodeUpstream, resp.StatusCode)
	}

	var rows []nominatimResult
	if err := json.Unmarshal(body, &rows); err != nil {
		return nil, fmt.Errorf("%w: decode: %v", errGeocodeUpstream, err)
	}
	items := make([]geoPlace, 0, len(rows))
	for _, row := range rows {
		label := strings.TrimSpace(row.DisplayName)
		if label == "" {
			continue
		}
		lat, err1 := strconv.ParseFloat(row.Lat, 64)
		lng, err2 := strconv.ParseFloat(row.Lon, 64)
		if err1 != nil || err2 != nil {
			continue
		}
		items = append(items, geoPlace{Label: label, Lat: lat, Lng: lng, Source: "nominatim"})
	}
	return items, nil
}

func (n *nominatimGeocoder) waitSlot(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-n.lastMu:
	}
	defer func() { n.lastMu <- struct{}{} }()

	wait := n.minInterval - time.Since(n.lastAt)
	if wait > 0 {
		timer := time.NewTimer(wait)
		defer timer.Stop()
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timer.C:
		}
	}
	n.lastAt = time.Now()
	return nil
}

type nominatimResult struct {
	DisplayName string `json:"display_name"`
	Lat         string `json:"lat"`
	Lon         string `json:"lon"`
}

func newGeocoder(provider, baseURL, userAgent string, client httpDoer) geocoder {
	switch strings.ToLower(strings.TrimSpace(provider)) {
	case "nominatim":
		return newNominatimGeocoder(baseURL, userAgent, client)
	default:
		return newPhotonGeocoder(baseURL, userAgent, client)
	}
}
