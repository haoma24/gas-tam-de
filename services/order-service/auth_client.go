package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"
)

// maxPhoneLookupIDs bounds one internal lookup; matches the auth-service cap.
const maxPhoneLookupIDs = 500

// phoneDirectory resolves user ids to their full contact phone.
//
// auth-service is the only holder of the plaintext number (encrypted at rest in
// users.phone_e164_enc / contact_phone_e164_enc), so orders snapshot it from
// there rather than trusting anything a client sends.
type phoneDirectory interface {
	PhonesByUserID(ctx context.Context, userIDs []string) (map[string]string, error)
}

type noopPhoneDirectory struct{}

func (noopPhoneDirectory) PhonesByUserID(context.Context, []string) (map[string]string, error) {
	return nil, nil
}

type httpAuthClient struct {
	baseURL string
	client  *http.Client
}

func newHTTPAuthClient(baseURL string, client *http.Client) *httpAuthClient {
	if client == nil {
		client = &http.Client{Timeout: 5 * time.Second}
	}
	return &httpAuthClient{baseURL: strings.TrimRight(baseURL, "/"), client: client}
}

// PhonesByUserID calls POST /v1/internal/users/phones. Ids with no number on
// file (a Google account that never added a contact phone) are simply absent
// from the result — that is not an error.
func (c *httpAuthClient) PhonesByUserID(ctx context.Context, userIDs []string) (map[string]string, error) {
	ids := dedupeNonEmpty(userIDs)
	if len(ids) == 0 {
		return nil, nil
	}
	if len(ids) > maxPhoneLookupIDs {
		ids = ids[:maxPhoneLookupIDs]
	}

	body, err := json.Marshal(map[string]any{"user_ids": ids})
	if err != nil {
		return nil, err
	}
	const path = "/v1/internal/users/phones"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("auth %s%s: %w", c.baseURL, path, err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("auth %s%s status %d: %s",
			c.baseURL, path, resp.StatusCode, strings.TrimSpace(string(raw)))
	}

	var parsed struct {
		Phones map[string]string `json:"phones"`
	}
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return nil, fmt.Errorf("auth %s%s: %w", c.baseURL, path, err)
	}
	return parsed.Phones, nil
}

func dedupeNonEmpty(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, v := range values {
		v = strings.TrimSpace(v)
		if v == "" {
			continue
		}
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		out = append(out, v)
	}
	return out
}

// fillCustomerPhones backfills orders.customer_phone for rows placed before the
// column existed (or whose lookup failed at place time), then persists what it
// learned so the next listing costs nothing.
//
// One batched call per listing, only when something is actually missing.
// Best-effort throughout: an unreachable auth-service must not take down the
// Order Desk, it just leaves those rows showing the masked number.
func (s *orderService) fillCustomerPhones(ctx context.Context, orders []orderRow) {
	if s.authDir == nil {
		return
	}
	missing := make([]string, 0)
	for _, o := range orders {
		if strings.TrimSpace(o.customerPhone) == "" {
			missing = append(missing, o.userID)
		}
	}
	if len(missing) == 0 {
		return
	}

	phones, err := s.authDir.PhonesByUserID(ctx, missing)
	if err != nil {
		slog.Error("backfill customer phones", "orders", len(missing), "err", err)
		return
	}
	if len(phones) == 0 {
		return
	}

	for i := range orders {
		if strings.TrimSpace(orders[i].customerPhone) != "" {
			continue
		}
		phone := strings.TrimSpace(phones[orders[i].userID])
		if phone == "" {
			continue
		}
		orders[i].customerPhone = phone
		if _, err := s.db.Exec(
			`UPDATE orders SET customer_phone = ? WHERE id = ? AND customer_phone = ''`,
			phone, orders[i].id,
		); err != nil {
			slog.Error("persist customer phone", "order_id", orders[i].id, "err", err)
		}
	}
}
