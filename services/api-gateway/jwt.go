package main

import (
	"fmt"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

const tokenIssuer = "gas-tam-de-auth"

const (
	roleCustomer = "customer"
	roleAdmin    = "admin"
)

// AccessClaims mirrors auth-service JWT payload (HS256).
type AccessClaims struct {
	Role        string `json:"role"`
	PhoneMasked string `json:"phone_masked,omitempty"`
	SessionID   string `json:"sid"`
	jwt.RegisteredClaims
}

func parseAccessToken(secret, raw string) (*AccessClaims, error) {
	raw = strings.TrimSpace(raw)
	if secret == "" {
		return nil, fmt.Errorf("jwt secret empty")
	}
	if raw == "" {
		return nil, fmt.Errorf("token empty")
	}

	claims := &AccessClaims{}
	tok, err := jwt.ParseWithClaims(raw, claims, func(t *jwt.Token) (any, error) {
		if t.Method != jwt.SigningMethodHS256 {
			return nil, fmt.Errorf("unexpected signing method %v", t.Header["alg"])
		}
		return []byte(secret), nil
	}, jwt.WithIssuer(tokenIssuer), jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}))
	if err != nil {
		return nil, err
	}
	if !tok.Valid {
		return nil, fmt.Errorf("token invalid")
	}
	if claims.Subject == "" {
		return nil, fmt.Errorf("subject empty")
	}
	if claims.ExpiresAt == nil {
		return nil, fmt.Errorf("exp missing")
	}
	if claims.Role != roleCustomer && claims.Role != roleAdmin {
		return nil, fmt.Errorf("role invalid")
	}
	if claims.SessionID == "" {
		return nil, fmt.Errorf("session empty")
	}
	return claims, nil
}

func bearerToken(header string) (string, bool) {
	header = strings.TrimSpace(header)
	if header == "" {
		return "", false
	}
	const prefix = "Bearer "
	if len(header) < len(prefix) || !strings.EqualFold(header[:len(prefix)], prefix) {
		return "", false
	}
	tok := strings.TrimSpace(header[len(prefix):])
	if tok == "" {
		return "", false
	}
	return tok, true
}
