package main

import (
	"context"
	"errors"
	"strings"
	"testing"
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
	t.Setenv("SMS_VENDOR", "stringee")
	s := newSMSSenderFromEnv()
	if _, ok := s.(*ProductionSMSSender); !ok {
		t.Fatalf("want *ProductionSMSSender got %T", s)
	}
}
