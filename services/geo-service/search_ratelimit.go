package main

import (
	"sync"
	"time"
)

// geoSearchRateLimiter limits geocode searches per client IP (abuse protection).
type geoSearchRateLimiter struct {
	mu      sync.Mutex
	window  time.Duration
	maxHits int
	hits    map[string][]time.Time
}

func newGeoSearchRateLimiter(maxPerMinute int) *geoSearchRateLimiter {
	if maxPerMinute <= 0 {
		maxPerMinute = 30
	}
	return &geoSearchRateLimiter{
		window:  time.Minute,
		maxHits: maxPerMinute,
		hits:    make(map[string][]time.Time),
	}
}

type geoRateLimitResult struct {
	Allowed       bool
	RetryAfterSec int
}

func (l *geoSearchRateLimiter) Allow(ip string, now time.Time) geoRateLimitResult {
	l.mu.Lock()
	defer l.mu.Unlock()

	if ip == "" {
		ip = "unknown"
	}
	l.hits[ip] = pruneGeoHits(l.hits[ip], now, l.window)
	if len(l.hits[ip]) >= l.maxHits {
		retry := int(l.hits[ip][0].Add(l.window).Sub(now).Seconds()) + 1
		if retry < 1 {
			retry = 1
		}
		return geoRateLimitResult{Allowed: false, RetryAfterSec: retry}
	}
	l.hits[ip] = append(l.hits[ip], now)
	return geoRateLimitResult{Allowed: true}
}

func pruneGeoHits(hits []time.Time, now time.Time, window time.Duration) []time.Time {
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
