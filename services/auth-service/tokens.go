package main

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const tokenIssuer = "gas-tam-de-auth"

const (
	roleCustomer = "customer"
	roleAdmin    = "admin"
)

// AccessClaims is the customer/admin access JWT payload.
type AccessClaims struct {
	Role        string `json:"role"`
	PhoneMasked string `json:"phone_masked,omitempty"`
	SessionID   string `json:"sid"`
	jwt.RegisteredClaims
}

func issueAccessToken(secret string, userID, role, phoneMasked, sessionID string, ttl time.Duration, now time.Time) (string, error) {
	if secret == "" {
		return "", fmt.Errorf("jwt secret empty")
	}
	claims := AccessClaims{
		Role:        role,
		PhoneMasked: phoneMasked,
		SessionID:   sessionID,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			Issuer:    tokenIssuer,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
			ID:        sessionID,
		},
	}
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return t.SignedString([]byte(secret))
}

func generateRefreshToken() (raw string, hash string, err error) {
	buf := make([]byte, 32)
	if _, err = rand.Read(buf); err != nil {
		return "", "", err
	}
	raw = base64.RawURLEncoding.EncodeToString(buf)
	hash = hashRefreshToken(raw)
	return raw, hash, nil
}

func hashRefreshToken(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}
