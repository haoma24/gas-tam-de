package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// geoCheckResult mirrors geo-service POST /v1/geo/check response.
type geoCheckResult struct {
	DistanceKm  float64 `json:"distance_km"`
	InRange     bool    `json:"in_range"`
	MaxRadiusKm float64 `json:"max_radius_km"`
}

// catalogProduct is a subset of catalog-service product for place-order pricing.
type catalogProduct struct {
	ID        string `json:"id"`
	SKU       string `json:"sku"`
	Name      string `json:"name"`
	SalePrice int64  `json:"sale_price"`
	Active    bool   `json:"active"`
}

type geoChecker interface {
	Check(ctx context.Context, lat, lng float64) (geoCheckResult, error)
}

type productCatalog interface {
	ListActive(ctx context.Context) ([]catalogProduct, error)
}

// billingPaymentInput mirrors billing-service POST /v1/internal/payments (T6.1.2).
type billingPaymentInput struct {
	OrderID     string `json:"order_id"`
	CustomerKey string `json:"customer_key"`
	PhoneMasked string `json:"phone_masked"`
	PaymentType string `json:"payment_type"`
	AmountDue   int64  `json:"amount_due"`
	AmountPaid  int64  `json:"amount_paid"`
	RecordedBy  string `json:"recorded_by"`
}

type billingRecorder interface {
	RecordPayment(ctx context.Context, in billingPaymentInput) error
}

type noopBillingRecorder struct{}

func (noopBillingRecorder) RecordPayment(context.Context, billingPaymentInput) error {
	return nil
}

type httpGeoClient struct {
	baseURL string
	client  *http.Client
}

func newHTTPGeoClient(baseURL string, client *http.Client) *httpGeoClient {
	if client == nil {
		client = &http.Client{Timeout: 8 * time.Second}
	}
	return &httpGeoClient{baseURL: strings.TrimRight(baseURL, "/"), client: client}
}

func (c *httpGeoClient) Check(ctx context.Context, lat, lng float64) (geoCheckResult, error) {
	body, err := json.Marshal(map[string]float64{"lat": lat, "lng": lng})
	if err != nil {
		return geoCheckResult{}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/v1/geo/check", bytes.NewReader(body))
	if err != nil {
		return geoCheckResult{}, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return geoCheckResult{}, fmt.Errorf("geo check request: %w", err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))

	if resp.StatusCode != http.StatusOK {
		return geoCheckResult{}, fmt.Errorf("geo check status %d: %s", resp.StatusCode, strings.TrimSpace(string(raw)))
	}

	var out geoCheckResult
	if err := json.Unmarshal(raw, &out); err != nil {
		return geoCheckResult{}, fmt.Errorf("geo check decode: %w", err)
	}
	return out, nil
}

type httpCatalogClient struct {
	baseURL string
	client  *http.Client
}

func newHTTPCatalogClient(baseURL string, client *http.Client) *httpCatalogClient {
	if client == nil {
		client = &http.Client{Timeout: 8 * time.Second}
	}
	return &httpCatalogClient{baseURL: strings.TrimRight(baseURL, "/"), client: client}
}

func (c *httpCatalogClient) ListActive(ctx context.Context) ([]catalogProduct, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/v1/products", nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("catalog list request: %w", err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("catalog list status %d: %s", resp.StatusCode, strings.TrimSpace(string(raw)))
	}

	var payload struct {
		Items []catalogProduct `json:"items"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, fmt.Errorf("catalog list decode: %w", err)
	}
	if payload.Items == nil {
		payload.Items = []catalogProduct{}
	}
	return payload.Items, nil
}

type httpBillingClient struct {
	baseURL string
	client  *http.Client
}

func newHTTPBillingClient(baseURL string, client *http.Client) *httpBillingClient {
	if client == nil {
		client = &http.Client{Timeout: 8 * time.Second}
	}
	return &httpBillingClient{baseURL: strings.TrimRight(baseURL, "/"), client: client}
}

func (c *httpBillingClient) RecordPayment(ctx context.Context, in billingPaymentInput) error {
	body, err := json.Marshal(in)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/v1/internal/payments", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("billing record payment request: %w", err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("billing record payment status %d: %s", resp.StatusCode, strings.TrimSpace(string(raw)))
	}
	return nil
}
