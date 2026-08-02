package main

import (
	"context"
	"log/slog"
	"sync"
	"time"
)

// MockSMSRecord is an in-memory capture of a mock send (tests / local debug).
// Code is kept only in process memory — never written to slog.
type MockSMSRecord struct {
	PhoneE164 string
	Code      string
	Body      string
	At        time.Time
}

// MockSMSSender pretends to send SMS. Used for local/dev and unit tests.
type MockSMSSender struct {
	mu   sync.Mutex
	Sent []MockSMSRecord
}

// NewMockSMSSender returns an empty mock adapter.
func NewMockSMSSender() *MockSMSSender {
	return &MockSMSSender{}
}

// SendOTP records the message and logs a masked phone (never the OTP).
func (m *MockSMSSender) SendOTP(_ context.Context, phoneE164, code string) error {
	rec := MockSMSRecord{
		PhoneE164: phoneE164,
		Code:      code,
		Body:      otpSMSBody(code),
		At:        time.Now().UTC(),
	}
	m.mu.Lock()
	m.Sent = append(m.Sent, rec)
	n := len(m.Sent)
	m.mu.Unlock()

	slog.Info("sms mock sent",
		"provider", "mock",
		"phone_masked", maskPhoneE164(phoneE164),
		"sent_count", n,
	)
	return nil
}

// Last returns the most recent mock send, or false if none.
func (m *MockSMSSender) Last() (MockSMSRecord, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(m.Sent) == 0 {
		return MockSMSRecord{}, false
	}
	return m.Sent[len(m.Sent)-1], true
}

// Reset clears recorded sends (tests).
func (m *MockSMSSender) Reset() {
	m.mu.Lock()
	m.Sent = nil
	m.mu.Unlock()
}
