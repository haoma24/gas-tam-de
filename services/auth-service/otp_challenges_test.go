package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"gas-tam-de/pkg/httpx"
)

func TestMigrateCreatesOTPChallengesSchema(t *testing.T) {
	db := openTestDB(t)

	var cols int
	if err := db.QueryRow(`
		SELECT COUNT(*) FROM pragma_table_info('otp_challenges')
		WHERE name IN ('id','phone_hash','code_hash','expires_at','attempts','consumed_at','created_at')
	`).Scan(&cols); err != nil {
		t.Fatal(err)
	}
	if cols != 7 {
		t.Fatalf("otp_challenges columns=%d want 7", cols)
	}

	var idxPhone, idxExpires int
	if err := db.QueryRow(`
		SELECT COUNT(*) FROM sqlite_master
		WHERE type='index' AND name='idx_otp_phone'
	`).Scan(&idxPhone); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow(`
		SELECT COUNT(*) FROM sqlite_master
		WHERE type='index' AND name='idx_otp_expires'
	`).Scan(&idxExpires); err != nil {
		t.Fatal(err)
	}
	if idxPhone != 1 || idxExpires != 1 {
		t.Fatalf("indexes phone=%d expires=%d", idxPhone, idxExpires)
	}
}

func TestOTPChallengePersistsHashAndExpiry(t *testing.T) {
	svc := testOTPService(t)
	svc.ttl = 3 * time.Minute
	r := httpx.NewRouter("auth-test")
	r.Post("/v1/auth/otp/request", svc.handleOTPRequest)

	before := time.Now().UTC()
	body, _ := json.Marshal(map[string]string{"phone": "0901234567"})
	req := httptest.NewRequest(http.MethodPost, "/v1/auth/otp/request", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.RemoteAddr = "127.0.0.1:12345"
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	after := time.Now().UTC()

	var resp map[string]any
	_ = json.Unmarshal(rr.Body.Bytes(), &resp)
	code, _ := resp["dev_code"].(string)
	if len(code) != 6 {
		t.Fatalf("dev_code=%v", resp["dev_code"])
	}

	var id, phoneHash, codeHash, expiresRaw, createdRaw string
	var attempts int
	var consumed any
	err := svc.db.QueryRow(`
		SELECT id, phone_hash, code_hash, expires_at, attempts, consumed_at, created_at
		FROM otp_challenges
	`).Scan(&id, &phoneHash, &codeHash, &expiresRaw, &attempts, &consumed, &createdRaw)
	if err != nil {
		t.Fatal(err)
	}
	if id == "" || phoneHash == "" {
		t.Fatalf("id=%q phone_hash=%q", id, phoneHash)
	}
	if attempts != 0 || consumed != nil {
		t.Fatalf("attempts=%d consumed=%v", attempts, consumed)
	}

	// Never persist plaintext OTP (or trivially related forms).
	if codeHash == code || strings.Contains(codeHash, code) {
		t.Fatalf("code_hash must not contain raw OTP: hash=%q code=%q", codeHash, code)
	}
	wantHash := hashOTPCode(code, id, svc.otpPepper)
	if codeHash != wantHash {
		t.Fatalf("code_hash=%q want %q", codeHash, wantHash)
	}
	if len(codeHash) != 64 {
		t.Fatalf("expected sha256 hex length 64, got %d", len(codeHash))
	}

	expiresAt, err := time.Parse(time.RFC3339Nano, expiresRaw)
	if err != nil {
		t.Fatalf("parse expires_at: %v", err)
	}
	minExp := before.Add(svc.ttl - time.Second)
	maxExp := after.Add(svc.ttl + time.Second)
	if expiresAt.Before(minExp) || expiresAt.After(maxExp) {
		t.Fatalf("expires_at=%s outside [%s, %s]", expiresAt, minExp, maxExp)
	}

	createdAt, err := time.Parse(time.RFC3339Nano, createdRaw)
	if err != nil {
		t.Fatalf("parse created_at: %v", err)
	}
	if createdAt.Before(before.Add(-time.Second)) || createdAt.After(after.Add(time.Second)) {
		t.Fatalf("created_at=%s outside request window", createdAt)
	}
}

func TestOTPVerifyExpiredChallenge(t *testing.T) {
	svc := testOTPService(t)
	svc.ttl = 50 * time.Millisecond
	r := httpx.NewRouter("auth-test")
	r.Post("/v1/auth/otp/request", svc.handleOTPRequest)
	r.Post("/v1/auth/otp/verify", svc.handleOTPVerify)

	reqBody, _ := json.Marshal(map[string]string{"phone": "0909999888"})
	req := httptest.NewRequest(http.MethodPost, "/v1/auth/otp/request", bytes.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	req.RemoteAddr = "10.0.0.9:1"
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("request status=%d body=%s", rr.Code, rr.Body.String())
	}
	var reqResp map[string]any
	_ = json.Unmarshal(rr.Body.Bytes(), &reqResp)
	code, _ := reqResp["dev_code"].(string)
	if code == "" {
		t.Fatal("missing dev_code")
	}

	time.Sleep(80 * time.Millisecond)

	verifyBody, _ := json.Marshal(map[string]string{"phone": "0909999888", "code": code})
	vreq := httptest.NewRequest(http.MethodPost, "/v1/auth/otp/verify", bytes.NewReader(verifyBody))
	vreq.Header.Set("Content-Type", "application/json")
	vrr := httptest.NewRecorder()
	r.ServeHTTP(vrr, vreq)
	if vrr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 got %d body=%s", vrr.Code, vrr.Body.String())
	}
	var resp map[string]any
	_ = json.Unmarshal(vrr.Body.Bytes(), &resp)
	errObj, _ := resp["error"].(map[string]any)
	if errObj["code"] != "OTP_EXPIRED" {
		t.Fatalf("error=%v", resp)
	}
}

func TestInsertChallengeRoundTrip(t *testing.T) {
	svc := testOTPService(t)
	now := time.Date(2026, 8, 2, 7, 0, 0, 0, time.UTC)
	exp := now.Add(5 * time.Minute)
	id := "chal-1"
	code := "123456"
	hash := hashOTPCode(code, id, svc.otpPepper)

	if err := svc.insertChallenge(id, "phonehash", hash, exp, now); err != nil {
		t.Fatal(err)
	}

	var gotHash, gotExp string
	if err := svc.db.QueryRow(`
		SELECT code_hash, expires_at FROM otp_challenges WHERE id = ?
	`, id).Scan(&gotHash, &gotExp); err != nil {
		t.Fatal(err)
	}
	if gotHash != hash {
		t.Fatalf("hash=%q want %q", gotHash, hash)
	}
	parsed, err := time.Parse(time.RFC3339Nano, gotExp)
	if err != nil {
		t.Fatal(err)
	}
	if !parsed.Equal(exp) {
		t.Fatalf("expires_at=%s want %s", parsed, exp)
	}
}
