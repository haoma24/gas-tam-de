package main

import (
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"gas-tam-de/pkg/httpx"
)

// rateLimitConfig holds per-bucket sliding-window quotas (gateway edge).
type rateLimitConfig struct {
	OTPPerIPPerMinute     int
	LoginPerIPPerMinute   int
	OrderPerIPPerMinute   int
	OrderPerUserPerMinute int
}

func defaultRateLimitConfig() rateLimitConfig {
	return rateLimitConfig{
		OTPPerIPPerMinute:     10,
		LoginPerIPPerMinute:   10,
		OrderPerIPPerMinute:   30,
		OrderPerUserPerMinute: 10,
	}
}

type slidingWindowLimiter struct {
	mu      sync.Mutex
	window  time.Duration
	maxHits int
	hits    map[string][]time.Time
}

func newSlidingWindowLimiter(maxPerMinute int) *slidingWindowLimiter {
	if maxPerMinute <= 0 {
		maxPerMinute = 10
	}
	return &slidingWindowLimiter{
		window:  time.Minute,
		maxHits: maxPerMinute,
		hits:    make(map[string][]time.Time),
	}
}

type rateLimitResult struct {
	Allowed       bool
	RetryAfterSec int
}

func (l *slidingWindowLimiter) Allow(key string, now time.Time) rateLimitResult {
	l.mu.Lock()
	defer l.mu.Unlock()

	if key == "" {
		key = "unknown"
	}
	l.hits[key] = pruneRateHits(l.hits[key], now, l.window)
	if len(l.hits[key]) >= l.maxHits {
		retry := int(l.hits[key][0].Add(l.window).Sub(now).Seconds()) + 1
		if retry < 1 {
			retry = 1
		}
		return rateLimitResult{Allowed: false, RetryAfterSec: retry}
	}
	l.hits[key] = append(l.hits[key], now)
	return rateLimitResult{Allowed: true}
}

func pruneRateHits(hits []time.Time, now time.Time, window time.Duration) []time.Time {
	if len(hits) == 0 {
		return hits
	}
	cut := now.Add(-window)
	i := 0
	for i < len(hits) && !hits[i].After(cut) {
		i++
	}
	if i == 0 {
		return hits
	}
	out := make([]time.Time, len(hits)-i)
	copy(out, hits[i:])
	return out
}

// rateLimiters bundles gateway limiters for OTP, admin login, and place-order.
type rateLimiters struct {
	otpIP     *slidingWindowLimiter
	loginIP   *slidingWindowLimiter
	orderIP   *slidingWindowLimiter
	orderUser *slidingWindowLimiter
}

func newRateLimiters(cfg rateLimitConfig) *rateLimiters {
	return &rateLimiters{
		otpIP:     newSlidingWindowLimiter(cfg.OTPPerIPPerMinute),
		loginIP:   newSlidingWindowLimiter(cfg.LoginPerIPPerMinute),
		orderIP:   newSlidingWindowLimiter(cfg.OrderPerIPPerMinute),
		orderUser: newSlidingWindowLimiter(cfg.OrderPerUserPerMinute),
	}
}

// RateLimitOTPAndLogin limits POST /v1/auth/otp/request and POST /v1/auth/admin/login by client IP.
func RateLimitOTPAndLogin(rl *rateLimiters) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodPost {
				switch r.URL.Path {
				case "/v1/auth/otp/request":
					if !enforceLimit(w, rl.otpIP, "ip:"+clientIP(r), "too many OTP requests; retry later") {
						return
					}
				case "/v1/auth/admin/login":
					if !enforceLimit(w, rl.loginIP, "ip:"+clientIP(r), "too many login attempts; retry later") {
						return
					}
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}

// RateLimitPlaceOrder limits POST /v1/orders by IP and authenticated user (after JWT).
func RateLimitPlaceOrder(rl *rateLimiters) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodPost && r.URL.Path == "/v1/orders" {
				ipKey := "ip:" + clientIP(r)
				if !enforceLimit(w, rl.orderIP, ipKey, "too many place-order requests; retry later") {
					return
				}
				userKey := "user:anonymous"
				if claims, ok := ClaimsFromContext(r.Context()); ok && claims.Subject != "" {
					userKey = "user:" + claims.Subject
				}
				if !enforceLimit(w, rl.orderUser, userKey, "too many place-order requests; retry later") {
					return
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}

func enforceLimit(w http.ResponseWriter, lim *slidingWindowLimiter, key, message string) bool {
	res := lim.Allow(key, time.Now())
	if res.Allowed {
		return true
	}
	w.Header().Set("Retry-After", fmt.Sprintf("%d", res.RetryAfterSec))
	httpx.Error(w, http.StatusTooManyRequests, "RATE_LIMITED", message)
	return false
}

func clientIP(r *http.Request) string {
	if xff := strings.TrimSpace(r.Header.Get("X-Forwarded-For")); xff != "" {
		parts := strings.Split(xff, ",")
		return strings.TrimSpace(parts[0])
	}
	if xri := strings.TrimSpace(r.Header.Get("X-Real-IP")); xri != "" {
		return xri
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
