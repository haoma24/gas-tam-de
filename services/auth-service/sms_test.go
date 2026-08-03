package main

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestMockSMSSenderRecordsSend(t *testing.T) {
	m := NewMockSMSSender()
	if err := m.SendOTP(context.Background(), "+84901234567", "123456"); err != nil {
		t.Fatal(err)
	}
	rec, ok := m.Last()
	if !ok {
		t.Fatal("expected record")
	}
	if rec.PhoneE164 != "+84901234567" || rec.Code != "123456" {
		t.Fatalf("record=%+v", rec)
	}
	if !strings.Contains(rec.Body, "123456") {
		t.Fatalf("body=%q", rec.Body)
	}
}

func TestProductionSMSSenderNotConfigured(t *testing.T) {
	p := NewProductionSMSSender(ProductionSMSConfig{})
	err := p.SendOTP(context.Background(), "+84901234567", "123456")
	if !errors.Is(err, ErrSMSNotConfigured) {
		t.Fatalf("err=%v", err)
	}
}

func TestProductionSMSSenderSeamWithKey(t *testing.T) {
	p := NewProductionSMSSender(ProductionSMSConfig{
		Vendor: "esms",
		APIKey: "test-key-not-real",
		APIURL: "https://example.invalid/sms",
		Sender: "GasTamDe",
	})
	err := p.SendOTP(context.Background(), "+84901234567", "654321")
	if !errors.Is(err, ErrSMSNotConfigured) {
		t.Fatalf("expected seam ErrSMSNotConfigured, got %v", err)
	}
}

func TestNewSMSSenderFromEnvDefaultMock(t *testing.T) {
	t.Setenv("SMS_PROVIDER", "")
	s := newSMSSenderFromEnv()
	if _, ok := s.(*MockSMSSender); !ok {
		t.Fatalf("want *MockSMSSender got %T", s)
	}
}

func TestNewSMSSenderFromEnvProduction(t *testing.T) {
	t.Setenv("SMS_PROVIDER", "production")
	t.Setenv("SMS_VENDOR", "esms")
	s := newSMSSenderFromEnv()
	if _, ok := s.(*ProductionSMSSender); !ok {
		t.Fatalf("want *ProductionSMSSender got %T", s)
	}
}

func TestNewSMSSenderFromEnvStringee(t *testing.T) {
	for _, env := range []struct{ provider, vendor string }{
		{"stringee", ""},
		{"production", "stringee"},
		{"prod", "Stringee"},
	} {
		t.Setenv("SMS_PROVIDER", env.provider)
		t.Setenv("SMS_VENDOR", env.vendor)
		s := newSMSSenderFromEnv()
		if _, ok := s.(*StringeeSMSSender); !ok {
			t.Fatalf("provider=%q vendor=%q: want *StringeeSMSSender got %T", env.provider, env.vendor, s)
		}
	}
}

func TestStringeeConfigFromEnvSplitsAPIKeyPair(t *testing.T) {
	t.Setenv("SMS_API_SID", "")
	t.Setenv("SMS_API_SECRET", "")
	t.Setenv("SMS_API_KEY", "SKpair:secretpair")
	t.Setenv("SMS_SENDER", "GASTAMDE")
	t.Setenv("SMS_API_URL", "")
	t.Setenv("SMS_TIMEOUT_SEC", "7")

	cfg := stringeeConfigFromEnv()
	if cfg.APIKeySID != "SKpair" || cfg.APIKeySecret != "secretpair" {
		t.Fatalf("cfg=%+v", cfg)
	}
	if cfg.Brandname != "GASTAMDE" {
		t.Fatalf("brandname=%q", cfg.Brandname)
	}
	if cfg.APIURL != stringeeDefaultAPIURL {
		t.Fatalf("api url=%q", cfg.APIURL)
	}
	if cfg.Timeout != 7*time.Second {
		t.Fatalf("timeout=%v", cfg.Timeout)
	}
}

func TestStringeeConfigFromEnvExplicitPairWins(t *testing.T) {
	t.Setenv("SMS_API_SID", "SKexplicit")
	t.Setenv("SMS_API_SECRET", "secretexplicit")
	t.Setenv("SMS_API_KEY", "SKpair:secretpair")

	cfg := stringeeConfigFromEnv()
	if cfg.APIKeySID != "SKexplicit" || cfg.APIKeySecret != "secretexplicit" {
		t.Fatalf("cfg=%+v", cfg)
	}
}
