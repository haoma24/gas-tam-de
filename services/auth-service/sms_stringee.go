package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// stringeeDefaultAPIURL is the SMS REST endpoint (docs: POST /v1/sms).
const stringeeDefaultAPIURL = "https://api.stringee.com/v1/sms"

// ErrSMSRejected means the vendor accepted the HTTP call but refused the send
// (bad brandname, out of credit, invalid number, expired token, …).
var ErrSMSRejected = errors.New("sms vendor rejected the message")

// StringeeSMSConfig holds Stringee REST credentials. Fill via env; never commit.
type StringeeSMSConfig struct {
	APIKeySID    string // SMS_API_SID — "SK..." from the Stringee dashboard
	APIKeySecret string // SMS_API_SECRET
	Brandname    string // SMS_SENDER — approved SMS brandname used as "from"
	APIURL       string
	Timeout      time.Duration
	TokenTTL     time.Duration
}

// StringeeSMSSender sends OTP codes through the Stringee SMS REST API.
type StringeeSMSSender struct {
	cfg  StringeeSMSConfig
	http *http.Client
	now  func() time.Time
}

// NewStringeeSMSSender builds the adapter, applying defaults for URL/timeouts.
func NewStringeeSMSSender(cfg StringeeSMSConfig) *StringeeSMSSender {
	if cfg.APIURL == "" {
		cfg.APIURL = stringeeDefaultAPIURL
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = 10 * time.Second
	}
	if cfg.TokenTTL <= 0 {
		cfg.TokenTTL = time.Hour
	}
	return &StringeeSMSSender{
		cfg:  cfg,
		http: &http.Client{Timeout: cfg.Timeout},
		now:  func() time.Time { return time.Now().UTC() },
	}
}

type stringeeSMSItem struct {
	From string `json:"from"`
	To   string `json:"to"`
	Text string `json:"text"`
}

type stringeeSMSRequest struct {
	SMS []stringeeSMSItem `json:"sms"`
}

type stringeeSMSResult struct {
	R   *int   `json:"r"`
	Msg string `json:"msg"`
}

type stringeeSMSResponse struct {
	SMSSent int                 `json:"smsSent"`
	Result  []stringeeSMSResult `json:"result"`
	// Auth / envelope failures answer with a top-level code instead of result[].
	R   *int   `json:"r"`
	Msg string `json:"msg"`
}

// SendOTP posts one OTP message and maps vendor codes onto stable errors.
// The raw OTP is never logged; phone numbers are logged masked only.
func (s *StringeeSMSSender) SendOTP(ctx context.Context, phoneE164, code string) error {
	if err := s.validate(); err != nil {
		return err
	}

	token, err := s.signToken()
	if err != nil {
		return fmt.Errorf("stringee: sign auth token: %w", err)
	}

	payload, err := json.Marshal(stringeeSMSRequest{SMS: []stringeeSMSItem{{
		From: s.cfg.Brandname,
		To:   stringeeMSISDN(phoneE164),
		Text: otpSMSBody(code),
	}}})
	if err != nil {
		return fmt.Errorf("stringee: encode request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.cfg.APIURL, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("stringee: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-STRINGEE-AUTH", token)

	resp, err := s.http.Do(req)
	if err != nil {
		return fmt.Errorf("stringee: call api: %w", err)
	}
	defer resp.Body.Close()

	// Vendor replies are small; cap anyway so a bad gateway cannot exhaust memory.
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<16))
	if err != nil {
		return fmt.Errorf("stringee: read response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return fmt.Errorf("%w: http %d: %s", ErrSMSRejected, resp.StatusCode, stringeeErrorSummary(body))
	}

	var parsed stringeeSMSResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return fmt.Errorf("stringee: decode response: %w", err)
	}

	if vendorCode, msg, ok := stringeeFailure(parsed); ok {
		return fmt.Errorf("%w: code=%d msg=%s", ErrSMSRejected, vendorCode, nonempty(msg, "unknown"))
	}

	slog.Info("sms sent",
		"provider", "stringee",
		"phone_masked", maskPhoneE164(phoneE164),
		"sms_sent", parsed.SMSSent,
	)
	return nil
}

func (s *StringeeSMSSender) validate() error {
	var missing []string
	if s.cfg.APIKeySID == "" {
		missing = append(missing, "SMS_API_SID")
	}
	if s.cfg.APIKeySecret == "" {
		missing = append(missing, "SMS_API_SECRET")
	}
	if s.cfg.Brandname == "" {
		missing = append(missing, "SMS_SENDER")
	}
	if len(missing) == 0 {
		return nil
	}
	slog.Error("sms stringee not configured",
		"provider", "stringee",
		"missing", strings.Join(missing, ","),
	)
	return fmt.Errorf("%w: set %s", ErrSMSNotConfigured, strings.Join(missing, ", "))
}

// signToken builds the HS256 JWT Stringee expects in X-STRINGEE-AUTH.
func (s *StringeeSMSSender) signToken() (string, error) {
	now := s.now()
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"jti":      fmt.Sprintf("%s-%d", s.cfg.APIKeySID, now.Unix()),
		"iss":      s.cfg.APIKeySID,
		"exp":      now.Add(s.cfg.TokenTTL).Unix(),
		"rest_api": true,
	})
	token.Header["cty"] = "stringee-api;v=1"
	return token.SignedString([]byte(s.cfg.APIKeySecret))
}

// stringeeFailure reports a vendor-side failure, preferring the per-message code.
func stringeeFailure(resp stringeeSMSResponse) (code int, msg string, failed bool) {
	if len(resp.Result) > 0 {
		first := resp.Result[0]
		if first.R == nil || *first.R != 0 {
			c := -1
			if first.R != nil {
				c = *first.R
			}
			return c, first.Msg, true
		}
		if resp.SMSSent < 1 {
			return 0, nonempty(first.Msg, "smsSent=0"), true
		}
		return 0, "", false
	}
	if resp.R != nil && *resp.R != 0 {
		return *resp.R, resp.Msg, true
	}
	return -1, nonempty(resp.Msg, "empty result"), true
}

// stringeeMSISDN drops the E.164 "+" — Stringee expects e.g. 84901234567.
func stringeeMSISDN(phoneE164 string) string {
	return strings.TrimPrefix(strings.TrimSpace(phoneE164), "+")
}

// stringeeErrorSummary trims a vendor error body for a single log/error line.
func stringeeErrorSummary(body []byte) string {
	s := strings.TrimSpace(string(body))
	s = strings.Join(strings.Fields(s), " ")
	if s == "" {
		return "empty body"
	}
	if len(s) > 200 {
		return s[:200] + "…"
	}
	return s
}
