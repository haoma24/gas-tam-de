package main

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"gas-tam-de/pkg/httpx"
)

// upstreamProbeTimeout bounds one /readyz dependency probe. Short on purpose:
// /readyz is a debugging and load-balancer gate, not a customer-facing call.
const upstreamProbeTimeout = 3 * time.Second

// upstreamHealth probes GET <baseURL>/healthz for one synchronous dependency.
//
// order-service calls geo, catalog, billing and inventory over HTTP while
// serving a checkout, so a wrong URL or a dead container only surfaced as a
// failed order ("Không trừ được tồn kho"). Reporting each dependency on
// /readyz — with the URL actually configured — turns that into something
// `curl /readyz` answers before a customer hits it.
type upstreamHealth struct {
	name    string
	baseURL string
	client  *http.Client
}

func newUpstreamHealth(name, baseURL string) *upstreamHealth {
	return &upstreamHealth{
		name:    name,
		baseURL: strings.TrimRight(strings.TrimSpace(baseURL), "/"),
		client:  &http.Client{Timeout: upstreamProbeTimeout},
	}
}

// Check reports the dependency as unusable, naming the configured URL. The URL
// is the diagnosis: an in-container 127.0.0.1 means the service never received
// its *_SERVICE_URL from compose and is dialling itself.
func (u *upstreamHealth) Check() error {
	if u.baseURL == "" {
		return fmt.Errorf("%s url is empty", u.name)
	}

	ctx, cancel := context.WithTimeout(context.Background(), upstreamProbeTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.baseURL+"/healthz", nil)
	if err != nil {
		return fmt.Errorf("%s %s: %w", u.name, u.baseURL, err)
	}
	resp, err := u.client.Do(req)
	if err != nil {
		return fmt.Errorf("%s %s: %w", u.name, u.baseURL, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%s %s/healthz status %d", u.name, u.baseURL, resp.StatusCode)
	}
	return nil
}

// upstreamReadyCheck adapts one dependency to httpx.MountReady.
func upstreamReadyCheck(name, baseURL string) httpx.ReadyCheck {
	return httpx.ReadyCheck{Name: name, Check: newUpstreamHealth(name, baseURL).Check}
}
