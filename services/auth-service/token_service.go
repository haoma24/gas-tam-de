package main

import (
	"database/sql"
	"fmt"
	"time"
)

// tokenService issues access/refresh tokens for admin login and refresh rotation.
type tokenService struct {
	db         *sql.DB
	jwtSecret  string
	accessTTL  time.Duration
	refreshTTL time.Duration
}

func newTokenService(db *sql.DB, jwtSecret string, accessTTL, refreshTTL time.Duration) *tokenService {
	return &tokenService{
		db:         db,
		jwtSecret:  jwtSecret,
		accessTTL:  accessTTL,
		refreshTTL: refreshTTL,
	}
}

func parseRFC3339Flexible(raw string) (time.Time, error) {
	if t, err := time.Parse(time.RFC3339Nano, raw); err == nil {
		return t, nil
	}
	if t, err := time.Parse(time.RFC3339, raw); err == nil {
		return t, nil
	}
	return time.Time{}, fmt.Errorf("parse time %q", raw)
}
