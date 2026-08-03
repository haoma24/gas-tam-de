package main

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func testStringeeSender(url string) *StringeeSMSSender {
	return NewStringeeSMSSender(StringeeSMSConfig{
		APIKeySID:    "SKtest123",
		APIKeySecret: "secret-not-real",
		Brandname:    "GASTAMDE",
		APIURL:       url,
	})
}

func TestStringeeSendOTPSuccess(t *testing.T) {
	var (
		gotMethod string
		gotAuth   string
		gotCT     string
		gotBody   []byte
	)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotAuth = r.Header.Get("X-STRINGEE-AUTH")
		gotCT = r.Header.Get("Content-Type")
		gotBody, _ = io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"smsSent":1,"result":[{"price":"830","smsType":2,"r":0,"msg":"Success"}]}`))
	}))
	defer srv.Close()

	if err := testStringeeSender(srv.URL).SendOTP(context.Background(), "+84901234567", "123456"); err != nil {
		t.Fatalf("SendOTP: %v", err)
	}

	if gotMethod != http.MethodPost {
		t.Fatalf("method=%q", gotMethod)
	}
	if gotCT != "application/json" {
		t.Fatalf("content-type=%q", gotCT)
	}

	var req stringeeSMSRequest
	if err := json.Unmarshal(gotBody, &req); err != nil {
		t.Fatalf("decode request body: %v (%s)", err, gotBody)
	}
	if len(req.SMS) != 1 {
		t.Fatalf("sms items=%d", len(req.SMS))
	}
	item := req.SMS[0]
	if item.From != "GASTAMDE" {
		t.Fatalf("from=%q", item.From)
	}
	if item.To != "84901234567" {
		t.Fatalf("to=%q want digits without +", item.To)
	}
	if !strings.Contains(item.Text, "123456") {
		t.Fatalf("text=%q must carry the OTP", item.Text)
	}

	claims := jwt.MapClaims{}
	tok, err := jwt.ParseWithClaims(gotAuth, &claims, func(*jwt.Token) (any, error) {
		return []byte("secret-not-real"), nil
	})
	if err != nil {
		t.Fatalf("parse X-STRINGEE-AUTH: %v", err)
	}
	if tok.Method.Alg() != "HS256" {
		t.Fatalf("alg=%q", tok.Method.Alg())
	}
	if tok.Header["cty"] != "stringee-api;v=1" {
		t.Fatalf("cty=%v", tok.Header["cty"])
	}
	if claims["iss"] != "SKtest123" {
		t.Fatalf("iss=%v", claims["iss"])
	}
	if claims["rest_api"] != true {
		t.Fatalf("rest_api=%v", claims["rest_api"])
	}
	if jti, _ := claims["jti"].(string); !strings.HasPrefix(jti, "SKtest123-") {
		t.Fatalf("jti=%v", claims["jti"])
	}
	exp, err := claims.GetExpirationTime()
	if err != nil || exp == nil || !exp.After(time.Now()) {
		t.Fatalf("exp=%v err=%v", exp, err)
	}
}

func TestStringeeSendOTPVendorRejects(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"smsSent":0,"result":[{"r":2,"msg":"Brandname not approved"}]}`))
	}))
	defer srv.Close()

	err := testStringeeSender(srv.URL).SendOTP(context.Background(), "+84901234567", "123456")
	if !errors.Is(err, ErrSMSRejected) {
		t.Fatalf("err=%v want ErrSMSRejected", err)
	}
	if !strings.Contains(err.Error(), "Brandname not approved") {
		t.Fatalf("err=%v must surface the vendor message", err)
	}
}

func TestStringeeSendOTPTopLevelAuthError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"r":1,"msg":"Invalid token"}`))
	}))
	defer srv.Close()

	err := testStringeeSender(srv.URL).SendOTP(context.Background(), "+84901234567", "123456")
	if !errors.Is(err, ErrSMSRejected) {
		t.Fatalf("err=%v want ErrSMSRejected", err)
	}
	if !strings.Contains(err.Error(), "http 401") {
		t.Fatalf("err=%v must mention the status", err)
	}
}

func TestStringeeSendOTPSMSSentZero(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"smsSent":0,"result":[{"r":0,"msg":"Success"}]}`))
	}))
	defer srv.Close()

	err := testStringeeSender(srv.URL).SendOTP(context.Background(), "+84901234567", "123456")
	if !errors.Is(err, ErrSMSRejected) {
		t.Fatalf("err=%v want ErrSMSRejected when nothing was sent", err)
	}
}

func TestStringeeSendOTPMissingCredentialsNeverCallsAPI(t *testing.T) {
	called := false
	srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		called = true
	}))
	defer srv.Close()

	cases := map[string]StringeeSMSConfig{
		"no sid":       {APIKeySecret: "s", Brandname: "GASTAMDE", APIURL: srv.URL},
		"no secret":    {APIKeySID: "SK1", Brandname: "GASTAMDE", APIURL: srv.URL},
		"no brandname": {APIKeySID: "SK1", APIKeySecret: "s", APIURL: srv.URL},
	}
	for name, cfg := range cases {
		t.Run(name, func(t *testing.T) {
			err := NewStringeeSMSSender(cfg).SendOTP(context.Background(), "+84901234567", "123456")
			if !errors.Is(err, ErrSMSNotConfigured) {
				t.Fatalf("err=%v want ErrSMSNotConfigured", err)
			}
		})
	}
	if called {
		t.Fatal("must not call the vendor without full credentials")
	}
}

func TestStringeeDefaultsApplied(t *testing.T) {
	s := NewStringeeSMSSender(StringeeSMSConfig{APIKeySID: "SK1", APIKeySecret: "s", Brandname: "B"})
	if s.cfg.APIURL != stringeeDefaultAPIURL {
		t.Fatalf("api url=%q", s.cfg.APIURL)
	}
	if s.http.Timeout != 10*time.Second {
		t.Fatalf("timeout=%v", s.http.Timeout)
	}
	if s.cfg.TokenTTL != time.Hour {
		t.Fatalf("token ttl=%v", s.cfg.TokenTTL)
	}
}

func TestStringeeMSISDN(t *testing.T) {
	if got := stringeeMSISDN(" +84901234567 "); got != "84901234567" {
		t.Fatalf("got=%q", got)
	}
	if got := stringeeMSISDN("84901234567"); got != "84901234567" {
		t.Fatalf("got=%q", got)
	}
}

func TestSplitAPIKeyPair(t *testing.T) {
	sid, secret, ok := splitAPIKeyPair(" SK123 : shhh ")
	if !ok || sid != "SK123" || secret != "shhh" {
		t.Fatalf("sid=%q secret=%q ok=%v", sid, secret, ok)
	}
	if _, _, ok := splitAPIKeyPair("SK123"); ok {
		t.Fatal("single value must not parse as a pair")
	}
	if _, _, ok := splitAPIKeyPair(":shhh"); ok {
		t.Fatal("empty sid must not parse")
	}
}
