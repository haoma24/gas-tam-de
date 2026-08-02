package main

import (
	"context"
	"net/http"

	"gas-tam-de/pkg/httpx"
)

type ctxKey int

const claimsCtxKey ctxKey = 1

// ClaimsFromContext returns JWT claims set by RequireJWT (if any).
func ClaimsFromContext(ctx context.Context) (*AccessClaims, bool) {
	c, ok := ctx.Value(claimsCtxKey).(*AccessClaims)
	return c, ok
}

// RequireJWT validates Authorization Bearer access token and stores claims on the request.
func RequireJWT(secret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			raw, ok := bearerToken(r.Header.Get("Authorization"))
			if !ok {
				httpx.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "missing or invalid Authorization Bearer token")
				return
			}
			claims, err := parseAccessToken(secret, raw)
			if err != nil {
				httpx.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "invalid or expired access token")
				return
			}
			ctx := context.WithValue(r.Context(), claimsCtxKey, claims)
			r = r.WithContext(ctx)
			// Forward identity for upstream services (proxy wiring later).
			r.Header.Set("X-User-Id", claims.Subject)
			r.Header.Set("X-User-Role", claims.Role)
			r.Header.Set("X-Session-Id", claims.SessionID)
			if claims.PhoneMasked != "" {
				r.Header.Set("X-Phone-Masked", claims.PhoneMasked)
			}
			next.ServeHTTP(w, r)
		})
	}
}

// RequireRole rejects requests whose JWT role is not in allowed.
func RequireRole(allowed ...string) func(http.Handler) http.Handler {
	set := make(map[string]struct{}, len(allowed))
	for _, role := range allowed {
		set[role] = struct{}{}
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims, ok := ClaimsFromContext(r.Context())
			if !ok {
				httpx.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "missing authentication")
				return
			}
			if _, ok := set[claims.Role]; !ok {
				httpx.Error(w, http.StatusForbidden, "FORBIDDEN", "insufficient role for this route")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
