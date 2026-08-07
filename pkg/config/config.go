package config

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"strconv"
	"strings"
)

// Get returns env value or fallback.
func Get(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}

// ListenAddr returns the HTTP listen address.
// Order: primary env (e.g. GEO_ADDR) → PORT → fallback.
// PORT may be "8083" or ":8083" (PaaS-style). Bare ":port" is left as-is;
// callers should pass through httpx.NormalizeListenAddr / ListenAndServe.
func ListenAddr(primaryKey, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(primaryKey)); v != "" {
		return v
	}
	if p := strings.TrimSpace(os.Getenv("PORT")); p != "" {
		if strings.HasPrefix(p, ":") || strings.Contains(p, ":") {
			return p
		}
		return ":" + p
	}
	return fallback
}

// GetInt returns env int or fallback.
func GetInt(key string, fallback int) int {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return n
}

// GetFloat returns env float64 or fallback.
func GetFloat(key string, fallback float64) float64 {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback
	}
	n, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return fallback
	}
	return n
}

// SecretFingerprint is a short, non-reversible tag for a shared secret so two
// processes can be compared in logs without printing the secret itself.
// Services that sign and services that verify must print the same value.
func SecretFingerprint(secret string) string {
	if secret == "" {
		return "empty"
	}
	sum := sha256.Sum256([]byte(secret))
	return hex.EncodeToString(sum[:4])
}

// MustGet returns env value or panics if empty (and no fallback used).
func MustGet(key string) string {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		panic("missing required env: " + key)
	}
	return v
}
