package main

import (
	"log/slog"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	"gas-tam-de/pkg/httpx"

	"github.com/go-chi/chi/v5/middleware"
)

// reverseProxy forwards the request to upstream, keeping path and query.
// Upstream services expect the full /v1/... path (same as gateway).
func reverseProxy(upstream string) http.Handler {
	target, err := url.Parse(strings.TrimSpace(upstream))
	if err != nil || target.Scheme == "" || target.Host == "" {
		return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			// Generic message — never echo the misconfigured URL to clients.
			httpx.Error(w, http.StatusBadGateway, "BAD_GATEWAY", "upstream unavailable")
		})
	}

	proxy := httputil.NewSingleHostReverseProxy(target)
	originalDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		originalDirector(req)
		req.Host = target.Host
		req.URL.Host = target.Host
		req.URL.Scheme = target.Scheme
		// Path + RawQuery already set from the incoming request.
	}
	proxy.ModifyResponse = stripUpstreamSensitiveHeaders
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		// Log dial/timeout details server-side only (may include internal host:port).
		slog.Error("upstream proxy error",
			"err", err,
			"path", r.URL.Path,
			"request_id", middleware.GetReqID(r.Context()),
		)
		httpx.Error(w, http.StatusBadGateway, "BAD_GATEWAY", "upstream unavailable")
	}
	return proxy
}

// stripUpstreamSensitiveHeaders removes headers that can fingerprint internals.
func stripUpstreamSensitiveHeaders(resp *http.Response) error {
	if resp == nil {
		return nil
	}
	resp.Header.Del("Server")
	resp.Header.Del("X-Powered-By")
	return nil
}

// stripInboundIdentityHeaders removes client-spoofable identity / internal headers
// before JWT middleware re-attaches trusted values.
func stripInboundIdentityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.Header.Del("X-User-Id")
		r.Header.Del("X-User-Role")
		r.Header.Del("X-Session-Id")
		r.Header.Del("X-Phone-Masked")
		r.Header.Del("X-Internal-Token")
		next.ServeHTTP(w, r)
	})
}
