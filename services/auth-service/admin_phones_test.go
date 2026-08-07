package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"gas-tam-de/pkg/httpx"

	"github.com/golang-jwt/jwt/v5"
)

// otpLogin drives the real request → verify pair and returns the issued tokens
// plus the user object, so tests observe exactly what the app receives.
func otpLogin(t *testing.T, svc *otpService, phone string) (access, refresh string, user map[string]any) {
	t.Helper()
	r := httpx.NewRouter("auth-test")
	r.Post("/v1/auth/otp/request", svc.handleOTPRequest)
	r.Post("/v1/auth/otp/verify", svc.handleOTPVerify)

	reqBody, _ := json.Marshal(map[string]string{"phone": phone})
	req := httptest.NewRequest(http.MethodPost, "/v1/auth/otp/request", bytes.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	req.RemoteAddr = "127.0.0.1:12345"
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("otp request status=%d body=%s", rr.Code, rr.Body.String())
	}
	var reqResp map[string]any
	_ = json.Unmarshal(rr.Body.Bytes(), &reqResp)
	code, _ := reqResp["dev_code"].(string)

	verifyBody, _ := json.Marshal(map[string]string{"phone": phone, "code": code})
	vreq := httptest.NewRequest(http.MethodPost, "/v1/auth/otp/verify", bytes.NewReader(verifyBody))
	vreq.Header.Set("Content-Type", "application/json")
	vrr := httptest.NewRecorder()
	r.ServeHTTP(vrr, vreq)
	if vrr.Code != http.StatusOK {
		t.Fatalf("otp verify status=%d body=%s", vrr.Code, vrr.Body.String())
	}

	var resp map[string]any
	_ = json.Unmarshal(vrr.Body.Bytes(), &resp)
	access, _ = resp["access_token"].(string)
	refresh, _ = resp["refresh_token"].(string)
	user, _ = resp["user"].(map[string]any)
	return access, refresh, user
}

func accessClaims(t *testing.T, token string) *AccessClaims {
	t.Helper()
	claims := &AccessClaims{}
	tok, err := jwt.ParseWithClaims(token, claims, func(*jwt.Token) (any, error) {
		return []byte("test-jwt-secret"), nil
	})
	if err != nil || !tok.Valid {
		t.Fatalf("parse jwt: %v", err)
	}
	return claims
}

func TestSeedAdminPhonesGrantsAdminOnOTPLogin(t *testing.T) {
	svc := testOTPService(t)
	if err := seedAdminPhones(svc.db, "0909777020", svc.phonePepper); err != nil {
		t.Fatal(err)
	}

	_, _, user := otpLogin(t, svc, "0909777020")
	if user["role"] != roleAdmin {
		t.Fatalf("allow-listed phone should log in as admin, got %v", user["role"])
	}

	_, _, other := otpLogin(t, svc, "0901234567")
	if other["role"] != roleCustomer {
		t.Fatalf("unlisted phone should stay a customer, got %v", other["role"])
	}
}

func TestOTPAdminAccessTokenCarriesAdminRole(t *testing.T) {
	svc := testOTPService(t)
	if err := seedAdminPhones(svc.db, "0909777020", svc.phonePepper); err != nil {
		t.Fatal(err)
	}

	access, _, _ := otpLogin(t, svc, "0909777020")
	claims := accessClaims(t, access)
	if claims.Role != roleAdmin {
		t.Fatalf("claims role=%q, want admin", claims.Role)
	}
	if claims.PhoneMasked != "090***7020" {
		t.Fatalf("claims phone_masked=%q", claims.PhoneMasked)
	}

	var role string
	if err := svc.db.QueryRow(`SELECT role FROM sessions`).Scan(&role); err != nil {
		t.Fatal(err)
	}
	if role != roleAdmin {
		t.Fatalf("session role=%q, want admin", role)
	}
}

func TestSeedAdminPhonesAcceptsMultipleFormats(t *testing.T) {
	svc := testOTPService(t)
	if err := seedAdminPhones(svc.db, "0909777020, +84912345678 , ", svc.phonePepper); err != nil {
		t.Fatal(err)
	}

	n, err := countAdminPhones(svc.db)
	if err != nil {
		t.Fatal(err)
	}
	if n != 2 {
		t.Fatalf("seeded %d entries, want 2", n)
	}

	// Re-seeding must not duplicate rows or resurrect removed ones.
	if err := seedAdminPhones(svc.db, "0909777020", svc.phonePepper); err != nil {
		t.Fatal(err)
	}
	n, _ = countAdminPhones(svc.db)
	if n != 2 {
		t.Fatalf("re-seed changed count to %d", n)
	}
}

func TestSeedAdminPhonesRejectsInvalidNumber(t *testing.T) {
	svc := testOTPService(t)
	if err := seedAdminPhones(svc.db, "not-a-phone", svc.phonePepper); err == nil {
		t.Fatal("expected an error for an unparseable ADMIN_PHONES entry")
	}
}

func TestRefreshKeepsPhoneAdminRole(t *testing.T) {
	otp := testOTPService(t)
	if err := seedAdminPhones(otp.db, "0909777020", otp.phonePepper); err != nil {
		t.Fatal(err)
	}
	_, refresh, _ := otpLogin(t, otp, "0909777020")

	tokens := newTokenService(otp.db, "test-jwt-secret", otp.accessTTL, otp.refreshTTL)
	resp := postRefresh(t, tokens, refresh, http.StatusOK)

	user, _ := resp["user"].(map[string]any)
	if user["role"] != roleAdmin {
		t.Fatalf("rotated session role=%v, want admin", user["role"])
	}
	if user["phone_masked"] != "090***7020" {
		t.Fatalf("rotated session phone_masked=%v", user["phone_masked"])
	}
	access, _ := resp["access_token"].(string)
	if claims := accessClaims(t, access); claims.Role != roleAdmin {
		t.Fatalf("rotated claims role=%q", claims.Role)
	}
}

// Removing a number from the allow-list must take effect without waiting for
// the admin to log out — the next rotation downgrades them to customer.
func TestRefreshDowngradesRemovedPhoneAdmin(t *testing.T) {
	otp := testOTPService(t)
	if err := seedAdminPhones(otp.db, "0909777020", otp.phonePepper); err != nil {
		t.Fatal(err)
	}
	_, refresh, _ := otpLogin(t, otp, "0909777020")

	rows, err := listAdminPhones(otp.db)
	if err != nil || len(rows) != 1 {
		t.Fatalf("rows=%v err=%v", rows, err)
	}
	if _, err := deleteAdminPhone(otp.db, rows[0].ID); err != nil {
		t.Fatal(err)
	}

	tokens := newTokenService(otp.db, "test-jwt-secret", otp.accessTTL, otp.refreshTTL)
	resp := postRefresh(t, tokens, refresh, http.StatusOK)
	user, _ := resp["user"].(map[string]any)
	if user["role"] != roleCustomer {
		t.Fatalf("removed admin phone kept role=%v", user["role"])
	}
}

// The mirror case: a customer already signed in is promoted on the next
// rotation once their number is added, without needing a fresh login.
func TestRefreshPromotesNewlyAddedPhoneAdmin(t *testing.T) {
	otp := testOTPService(t)
	_, refresh, user := otpLogin(t, otp, "0909777020")
	if user["role"] != roleCustomer {
		t.Fatalf("precondition: role=%v", user["role"])
	}

	if err := seedAdminPhones(otp.db, "0909777020", otp.phonePepper); err != nil {
		t.Fatal(err)
	}

	tokens := newTokenService(otp.db, "test-jwt-secret", otp.accessTTL, otp.refreshTTL)
	resp := postRefresh(t, tokens, refresh, http.StatusOK)
	rotated, _ := resp["user"].(map[string]any)
	if rotated["role"] != roleAdmin {
		t.Fatalf("newly allow-listed phone kept role=%v", rotated["role"])
	}
}

func postRefresh(t *testing.T, svc *tokenService, refreshToken string, wantStatus int) map[string]any {
	t.Helper()
	r := httpx.NewRouter("auth-test")
	r.Post("/v1/auth/refresh", svc.handleRefresh)

	body, _ := json.Marshal(map[string]string{"refresh_token": refreshToken})
	req := httptest.NewRequest(http.MethodPost, "/v1/auth/refresh", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != wantStatus {
		t.Fatalf("refresh status=%d want=%d body=%s", rr.Code, wantStatus, rr.Body.String())
	}
	var resp map[string]any
	_ = json.Unmarshal(rr.Body.Bytes(), &resp)
	return resp
}
