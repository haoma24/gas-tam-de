package main

import (
	"net/http"
	"strconv"
	"strings"
)

// parseCORSOrigins splits CORS_ORIGINS (comma-separated). Empty tokens dropped.
// Patterns may end with "*" for prefix match (e.g. http://localhost:*).
func parseCORSOrigins(raw string) []string {
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		out = append(out, p)
	}
	return out
}

// CORS allows Flutter Web (and other browser) origins configured via CORS_ORIGINS.
// Handles OPTIONS preflight before JWT/RBAC.
func CORS(allowedOrigins []string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := strings.TrimSpace(r.Header.Get("Origin"))
			if origin != "" && originAllowed(origin, allowedOrigins) {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Vary", "Origin")
				w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
				// Flutter Web order/quote may send X-User-* / X-Phone-Masked (local or
				// transitional); gateway still strips spoofed identity and re-injects from JWT.
				w.Header().Set("Access-Control-Allow-Headers",
					"Authorization, Content-Type, Accept, X-Request-Id, X-User-Id, X-User-Role, X-Phone-Masked")
				w.Header().Set("Access-Control-Expose-Headers", "X-Request-Id, Retry-After")
				w.Header().Set("Access-Control-Max-Age", "600")
			}

			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func originAllowed(origin string, patterns []string) bool {
	for _, p := range patterns {
		if p == "*" {
			return true
		}
		if strings.HasSuffix(p, "*") {
			prefix := strings.TrimSuffix(p, "*")
			if strings.HasPrefix(origin, prefix) && localhostLikeOrigin(origin) {
				return true
			}
			// Non-localhost wildcard prefix (exact prefix match only).
			if !strings.Contains(p, "localhost") && !strings.Contains(p, "127.0.0.1") {
				if strings.HasPrefix(origin, prefix) {
					return true
				}
			}
			continue
		}
		if origin == p {
			return true
		}
	}
	return false
}

// localhostLikeOrigin restricts http://localhost:* / http://127.0.0.1:* wildcards
// to loopback hosts with an optional numeric port.
func localhostLikeOrigin(origin string) bool {
	const (
		local = "http://localhost"
		loop  = "http://127.0.0.1"
	)
	var rest string
	switch {
	case origin == local || origin == loop:
		return true
	case strings.HasPrefix(origin, local+":"):
		rest = origin[len(local)+1:]
	case strings.HasPrefix(origin, loop+":"):
		rest = origin[len(loop)+1:]
	default:
		return false
	}
	if rest == "" {
		return false
	}
	_, err := strconv.Atoi(rest)
	return err == nil
}
