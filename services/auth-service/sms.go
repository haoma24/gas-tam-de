package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"gas-tam-de/pkg/config"
)

// ErrSMSNotConfigured is returned when the selected provider is missing
// credentials, or when the generic production seam has no vendor client yet.
var ErrSMSNotConfigured = errors.New("sms provider not configured")

// SMSSender delivers OTP codes to a phone number.
// Implementations must never log the raw OTP code.
type SMSSender interface {
	SendOTP(ctx context.Context, phoneE164, code string) error
}

// newSMSSenderFromEnv picks the SMS adapter from SMS_PROVIDER.
//
//	mock (default) — records/logs send without calling a vendor
//	stringee       — Stringee SMS REST API client
//	production     — SMS_VENDOR decides the client (stringee wired; others seam)
func newSMSSenderFromEnv() SMSSender {
	provider := strings.ToLower(strings.TrimSpace(os.Getenv("SMS_PROVIDER")))
	vendor := strings.ToLower(strings.TrimSpace(os.Getenv("SMS_VENDOR")))

	switch provider {
	case "", "mock", "dev", "local":
		return NewMockSMSSender()
	case "stringee":
		return NewStringeeSMSSender(stringeeConfigFromEnv())
	case "production", "prod":
		if vendor == "stringee" {
			return NewStringeeSMSSender(stringeeConfigFromEnv())
		}
		return NewProductionSMSSender(ProductionSMSConfig{
			Vendor: strings.TrimSpace(os.Getenv("SMS_VENDOR")),
			APIKey: strings.TrimSpace(os.Getenv("SMS_API_KEY")),
			APIURL: strings.TrimSpace(os.Getenv("SMS_API_URL")),
			Sender: strings.TrimSpace(os.Getenv("SMS_SENDER")),
		})
	default:
		// Unknown provider → safe mock so local boot never hangs on SMS.
		return NewMockSMSSender()
	}
}

func stringeeConfigFromEnv() StringeeSMSConfig {
	sid := strings.TrimSpace(os.Getenv("SMS_API_SID"))
	secret := strings.TrimSpace(os.Getenv("SMS_API_SECRET"))
	// Convenience for deployments carrying a single secret: SMS_API_KEY="sid:secret".
	if sid == "" || secret == "" {
		if pairSID, pairSecret, ok := splitAPIKeyPair(os.Getenv("SMS_API_KEY")); ok {
			if sid == "" {
				sid = pairSID
			}
			if secret == "" {
				secret = pairSecret
			}
		}
	}

	return StringeeSMSConfig{
		APIKeySID:    sid,
		APIKeySecret: secret,
		Brandname:    strings.TrimSpace(os.Getenv("SMS_SENDER")),
		APIURL:       config.Get("SMS_API_URL", stringeeDefaultAPIURL),
		Timeout:      time.Duration(config.GetInt("SMS_TIMEOUT_SEC", 10)) * time.Second,
		TokenTTL:     time.Duration(config.GetInt("SMS_JWT_TTL_SEC", 3600)) * time.Second,
	}
}

// splitAPIKeyPair parses "sid:secret" (Stringee secrets never contain ":").
func splitAPIKeyPair(raw string) (sid, secret string, ok bool) {
	sid, secret, found := strings.Cut(strings.TrimSpace(raw), ":")
	sid = strings.TrimSpace(sid)
	secret = strings.TrimSpace(secret)
	if !found || sid == "" || secret == "" {
		return "", "", false
	}
	return sid, secret, true
}

func otpSMSBody(code string) string {
	return fmt.Sprintf("Ma OTP Gas Tam De: %s. Het han trong vai phut. Khong chia se ma nay.", code)
}
