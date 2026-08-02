package config

import (
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

// MustGet returns env value or panics if empty (and no fallback used).
func MustGet(key string) string {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		panic("missing required env: " + key)
	}
	return v
}
