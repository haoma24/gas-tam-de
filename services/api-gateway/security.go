package main

import (
	"net/http"
)

// SecurityHeaders sets baseline browser / client hardening headers on every response.
// Matches architecture §7.2 (nosniff, frame deny); CSP limited to frame-ancestors
// because the gateway serves JSON APIs, not static HTML.
func SecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("X-Frame-Options", "DENY")
		h.Set("Referrer-Policy", "no-referrer")
		h.Set("Permissions-Policy", "geolocation=(), microphone=(), camera=()")
		h.Set("Content-Security-Policy", "frame-ancestors 'none'")
		next.ServeHTTP(w, r)
	})
}
