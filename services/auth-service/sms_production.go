package main

import (
	"context"
	"fmt"
	"log/slog"
)

// ProductionSMSConfig holds credentials for a real VN SMS vendor.
// Fill via env (SMS_API_KEY, …); do not commit secrets.
type ProductionSMSConfig struct {
	// Vendor is a label for the intended provider, e.g. "esms" (stringee has
	// its own client in sms_stringee.go).
	Vendor string
	APIKey string
	APIURL string
	Sender string // brand / sender id when the vendor requires it
}

// ProductionSMSSender is the production adapter seam.
// It validates config and refuses to send until a concrete vendor client is
// plugged in — avoids accidental silent no-ops in production.
type ProductionSMSSender struct {
	cfg ProductionSMSConfig
}

// NewProductionSMSSender builds the production seam from config.
func NewProductionSMSSender(cfg ProductionSMSConfig) *ProductionSMSSender {
	return &ProductionSMSSender{cfg: cfg}
}

// SendOTP will call the configured VN provider once wired.
// Until then it returns ErrSMSNotConfigured (credentials may still be incomplete).
func (p *ProductionSMSSender) SendOTP(ctx context.Context, phoneE164, code string) error {
	_ = ctx
	_ = code // used only when a real HTTP client is added below

	if p.cfg.APIKey == "" {
		slog.Error("sms production not configured",
			"provider", "production",
			"vendor", p.cfg.Vendor,
			"phone_masked", maskPhoneE164(phoneE164),
		)
		return fmt.Errorf("%w: set SMS_API_KEY (and wire vendor client)", ErrSMSNotConfigured)
	}

	// Seam for eSMS / equivalent VN vendor:
	// 1. Build HTTP request to p.cfg.APIURL with API key + sender + phoneE164 + otpSMSBody(code)
	// 2. Never log raw OTP or full phone
	// 3. Map vendor errors to a stable error for the OTP handler
	slog.Error("sms production client not implemented",
		"provider", "production",
		"vendor", nonempty(p.cfg.Vendor, "unset"),
		"phone_masked", maskPhoneE164(phoneE164),
	)
	return fmt.Errorf("%w: production SMS client not implemented yet (vendor=%s)",
		ErrSMSNotConfigured, nonempty(p.cfg.Vendor, "unset"))
}

func nonempty(v, fallback string) string {
	if v == "" {
		return fallback
	}
	return v
}
