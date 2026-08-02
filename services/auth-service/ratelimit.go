package main

import (
	"sync"
	"time"
)

// otpRateLimiter limits OTP requests by phone_hash and client IP (in-process).
type otpRateLimiter struct {
	mu sync.Mutex

	cooldown time.Duration
	window   time.Duration
	maxPhone int
	maxIP    int

	phoneLast map[string]time.Time
	phoneHits map[string][]time.Time
	ipHits    map[string][]time.Time
}

func newOTPRateLimiter(cooldownSec, maxPhonePerHour, maxIPPerHour int) *otpRateLimiter {
	if cooldownSec <= 0 {
		cooldownSec = 60
	}
	if maxPhonePerHour <= 0 {
		maxPhonePerHour = 5
	}
	if maxIPPerHour <= 0 {
		maxIPPerHour = 20
	}
	return &otpRateLimiter{
		cooldown:  time.Duration(cooldownSec) * time.Second,
		window:    time.Hour,
		maxPhone:  maxPhonePerHour,
		maxIP:     maxIPPerHour,
		phoneLast: make(map[string]time.Time),
		phoneHits: make(map[string][]time.Time),
		ipHits:    make(map[string][]time.Time),
	}
}

type rateLimitResult struct {
	Allowed         bool
	RetryAfterSec   int
	Reason          string // "cooldown" | "phone_quota" | "ip_quota"
}

func (l *otpRateLimiter) Allow(phoneHash, ip string, now time.Time) rateLimitResult {
	l.mu.Lock()
	defer l.mu.Unlock()

	if last, ok := l.phoneLast[phoneHash]; ok {
		wait := l.cooldown - now.Sub(last)
		if wait > 0 {
			sec := int(wait.Seconds()) + 1
			return rateLimitResult{Allowed: false, RetryAfterSec: sec, Reason: "cooldown"}
		}
	}

	l.phoneHits[phoneHash] = pruneHits(l.phoneHits[phoneHash], now, l.window)
	if len(l.phoneHits[phoneHash]) >= l.maxPhone {
		retry := int(l.phoneHits[phoneHash][0].Add(l.window).Sub(now).Seconds()) + 1
		if retry < 1 {
			retry = 1
		}
		return rateLimitResult{Allowed: false, RetryAfterSec: retry, Reason: "phone_quota"}
	}

	if ip == "" {
		ip = "unknown"
	}
	l.ipHits[ip] = pruneHits(l.ipHits[ip], now, l.window)
	if len(l.ipHits[ip]) >= l.maxIP {
		retry := int(l.ipHits[ip][0].Add(l.window).Sub(now).Seconds()) + 1
		if retry < 1 {
			retry = 1
		}
		return rateLimitResult{Allowed: false, RetryAfterSec: retry, Reason: "ip_quota"}
	}

	l.phoneLast[phoneHash] = now
	l.phoneHits[phoneHash] = append(l.phoneHits[phoneHash], now)
	l.ipHits[ip] = append(l.ipHits[ip], now)
	return rateLimitResult{Allowed: true}
}

func pruneHits(hits []time.Time, now time.Time, window time.Duration) []time.Time {
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
