package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
)

// ErrSMSNotConfigured is returned by the production seam when no real SMS
// provider credentials are wired yet (T1.1.3 keeps the seam; provider later).
var ErrSMSNotConfigured = errors.New("sms provider not configured")

// SMSSender delivers OTP codes to a phone number.
// Implementations must never log the raw OTP code.
type SMSSender interface {
	SendOTP(ctx context.Context, phoneE164, code string) error
}

// newSMSSenderFromEnv picks the SMS adapter from SMS_PROVIDER.
//
//	mock (default) — records/logs send without calling a vendor
//	production     — production seam for eSMS / Stringee / equivalent VN
func newSMSSenderFromEnv() SMSSender {
	provider := strings.ToLower(strings.TrimSpace(os.Getenv("SMS_PROVIDER")))
	switch provider {
	case "", "mock", "dev", "local":
		return NewMockSMSSender()
	case "production", "prod":
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

func otpSMSBody(code string) string {
	return fmt.Sprintf("Ma OTP Gas Tam De: %s. Het han trong vai phut. Khong chia se ma nay.", code)
}
