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

type stockLine struct {
	ProductID string `json:"product_id"`
	SKU       string `json:"sku"`
	Qty       int64  `json:"qty"`
}

type stockReserver interface {
	// Reserve deducts stock and returns the COGS snapshot per product_id
	// (VND giá nhập at deduct time), which the caller persists on order_items.
	Reserve(ctx context.Context, orderID string, items []stockLine) (map[string]int64, error)
	Release(ctx context.Context, orderID string, items []stockLine) error
}

type noopStockReserver struct{}

func (noopStockReserver) Reserve(context.Context, string, []stockLine) (map[string]int64, error) {
	return nil, nil
}
func (noopStockReserver) Release(context.Context, string, []stockLine) error {
	return nil
}

type httpInventoryClient struct {
	baseURL string
	client  *http.Client
}

func newHTTPInventoryClient(baseURL string, client *http.Client) *httpInventoryClient {
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}
	return &httpInventoryClient{baseURL: strings.TrimRight(baseURL, "/"), client: client}
}

func (c *httpInventoryClient) Reserve(ctx context.Context, orderID string, items []stockLine) (map[string]int64, error) {
	return c.postStock(ctx, "/v1/internal/stock/reserve", orderID, items)
}

func (c *httpInventoryClient) Release(ctx context.Context, orderID string, items []stockLine) error {
	_, err := c.postStock(ctx, "/v1/internal/stock/release", orderID, items)
	return err
}

// stockOpResponse is the reserve/release reply; `items` carries the COGS
// snapshot inventory wrote on each OUT movement (empty for release).
type stockOpResponse struct {
	Items []struct {
		ProductID string `json:"product_id"`
		UnitCost  int64  `json:"unit_cost"`
	} `json:"items"`
}

func (c *httpInventoryClient) postStock(ctx context.Context, path, orderID string, items []stockLine) (map[string]int64, error) {
	body, err := json.Marshal(map[string]any{
		"order_id": orderID,
		"items":    items,
	})
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	// Errors carry the base URL: a 127.0.0.1 here inside a container is the
	// signature of a missing INVENTORY_SERVICE_URL, which otherwise looks
	// identical to inventory-service being down.
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("inventory %s%s: %w", c.baseURL, path, err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("inventory %s%s status %d: %s", c.baseURL, path, resp.StatusCode, strings.TrimSpace(string(raw)))
	}

	// An older inventory build answers without `items`; treat that as "no cost
	// known" rather than an error so placing an order never breaks on a rollout skew.
	var parsed stockOpResponse
	if err := json.Unmarshal(raw, &parsed); err != nil || len(parsed.Items) == 0 {
		return nil, nil
	}
	costs := make(map[string]int64, len(parsed.Items))
	for _, it := range parsed.Items {
		if pid := strings.TrimSpace(it.ProductID); pid != "" {
			costs[pid] = it.UnitCost
		}
	}
	return costs, nil
}
