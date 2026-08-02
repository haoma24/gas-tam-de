package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func issueTestToken(t *testing.T, secret, userID, role, sessionID string, ttl time.Duration) string {
	t.Helper()
	now := time.Now()
	claims := AccessClaims{
		Role:      role,
		SessionID: sessionID,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			Issuer:    tokenIssuer,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
			ID:        sessionID,
		},
	}
	if role == roleCustomer {
		claims.PhoneMasked = "090***1234"
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	s, err := tok.SignedString([]byte(secret))
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	return s
}

func decodeErr(t *testing.T, rec *httptest.ResponseRecorder) (code, message string) {
	t.Helper()
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("json: %v body=%s", err, rec.Body.String())
	}
	errObj, _ := body["error"].(map[string]any)
	code, _ = errObj["code"].(string)
	message, _ = errObj["message"].(string)
	return code, message
}

func emptyUpstreams() upstreams { return upstreams{} }

func TestRequireJWT_MissingBearer(t *testing.T) {
	r := testRouter(t, "test-secret", emptyUpstreams())
	req := httptest.NewRequest(http.MethodPost, "/v1/orders", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	code, _ := decodeErr(t, rec)
	if code != "UNAUTHORIZED" {
		t.Fatalf("code=%q", code)
	}
}

func TestRequireJWT_InvalidToken(t *testing.T) {
	r := testRouter(t, "test-secret", emptyUpstreams())
	req := httptest.NewRequest(http.MethodPost, "/v1/orders", nil)
	req.Header.Set("Authorization", "Bearer not-a-jwt")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d", rec.Code)
	}
}

func TestAdminRoute_RequiresAdminRole(t *testing.T) {
	secret := "test-secret"
	order, _, _ := startMockUpstream(t)
	inventory, _, _ := startMockUpstream(t)
	billing, _, _ := startMockUpstream(t)
	report, _, _ := startMockUpstream(t)
	r := testRouter(t, secret, upstreams{
		order:     order.URL,
		inventory: inventory.URL,
		billing:   billing.URL,
		report:    report.URL,
	})

	customerTok := issueTestToken(t, secret, "u1", roleCustomer, "s1", time.Hour)
	req := httptest.NewRequest(http.MethodGet, "/v1/admin/orders", nil)
	req.Header.Set("Authorization", "Bearer "+customerTok)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("customer on admin: status=%d body=%s", rec.Code, rec.Body.String())
	}
	code, _ := decodeErr(t, rec)
	if code != "FORBIDDEN" {
		t.Fatalf("code=%q", code)
	}

	adminTok := issueTestToken(t, secret, "a1", roleAdmin, "s2", time.Hour)
	req2 := httptest.NewRequest(http.MethodGet, "/v1/admin/orders", nil)
	req2.Header.Set("Authorization", "Bearer "+adminTok)
	rec2 := httptest.NewRecorder()
	r.ServeHTTP(rec2, req2)
	if rec2.Code != http.StatusOK {
		t.Fatalf("admin on admin: status=%d body=%s", rec2.Code, rec2.Body.String())
	}

	req3 := httptest.NewRequest(http.MethodGet, "/v1/admin/delivery-fee", nil)
	req3.Header.Set("Authorization", "Bearer "+customerTok)
	rec3 := httptest.NewRecorder()
	r.ServeHTTP(rec3, req3)
	if rec3.Code != http.StatusForbidden {
		t.Fatalf("customer on delivery-fee: status=%d", rec3.Code)
	}

	req4 := httptest.NewRequest(http.MethodPut, "/v1/admin/delivery-fee", nil)
	req4.Header.Set("Authorization", "Bearer "+adminTok)
	rec4 := httptest.NewRecorder()
	r.ServeHTTP(rec4, req4)
	if rec4.Code != http.StatusOK {
		t.Fatalf("admin on delivery-fee: status=%d", rec4.Code)
	}

	req5 := httptest.NewRequest(http.MethodPost, "/v1/admin/orders/ord-1/complete", nil)
	req5.Header.Set("Authorization", "Bearer "+customerTok)
	rec5 := httptest.NewRecorder()
	r.ServeHTTP(rec5, req5)
	if rec5.Code != http.StatusForbidden {
		t.Fatalf("customer on complete: status=%d", rec5.Code)
	}

	req6 := httptest.NewRequest(http.MethodPost, "/v1/admin/orders/ord-1/complete", nil)
	req6.Header.Set("Authorization", "Bearer "+adminTok)
	rec6 := httptest.NewRecorder()
	r.ServeHTTP(rec6, req6)
	if rec6.Code != http.StatusOK {
		t.Fatalf("admin on complete: status=%d", rec6.Code)
	}

	req7 := httptest.NewRequest(http.MethodGet, "/v1/admin/debts", nil)
	req7.Header.Set("Authorization", "Bearer "+customerTok)
	rec7 := httptest.NewRecorder()
	r.ServeHTTP(rec7, req7)
	if rec7.Code != http.StatusForbidden {
		t.Fatalf("customer on debts: status=%d", rec7.Code)
	}

	req8 := httptest.NewRequest(http.MethodGet, "/v1/admin/debts", nil)
	req8.Header.Set("Authorization", "Bearer "+adminTok)
	rec8 := httptest.NewRecorder()
	r.ServeHTTP(rec8, req8)
	if rec8.Code != http.StatusOK {
		t.Fatalf("admin on debts: status=%d", rec8.Code)
	}

	req9 := httptest.NewRequest(http.MethodGet, "/v1/admin/inventory", nil)
	req9.Header.Set("Authorization", "Bearer "+customerTok)
	rec9 := httptest.NewRecorder()
	r.ServeHTTP(rec9, req9)
	if rec9.Code != http.StatusForbidden {
		t.Fatalf("customer on inventory: status=%d", rec9.Code)
	}

	req10 := httptest.NewRequest(http.MethodPost, "/v1/admin/inventory", nil)
	req10.Header.Set("Authorization", "Bearer "+adminTok)
	rec10 := httptest.NewRecorder()
	r.ServeHTTP(rec10, req10)
	if rec10.Code != http.StatusOK {
		t.Fatalf("admin on inventory: status=%d", rec10.Code)
	}

	req11 := httptest.NewRequest(http.MethodGet, "/v1/admin/dashboard/summary", nil)
	req11.Header.Set("Authorization", "Bearer "+customerTok)
	rec11 := httptest.NewRecorder()
	r.ServeHTTP(rec11, req11)
	if rec11.Code != http.StatusForbidden {
		t.Fatalf("customer on dashboard summary: status=%d", rec11.Code)
	}

	req12 := httptest.NewRequest(http.MethodGet, "/v1/admin/dashboard/summary", nil)
	req12.Header.Set("Authorization", "Bearer "+adminTok)
	rec12 := httptest.NewRecorder()
	r.ServeHTTP(rec12, req12)
	if rec12.Code != http.StatusOK {
		t.Fatalf("admin on dashboard summary: status=%d", rec12.Code)
	}
}

func TestCustomerRoute_RequiresCustomerRole(t *testing.T) {
	secret := "test-secret"
	order, _, _ := startMockUpstream(t)
	r := testRouter(t, secret, upstreams{order: order.URL})

	adminTok := issueTestToken(t, secret, "a1", roleAdmin, "s1", time.Hour)
	req := httptest.NewRequest(http.MethodGet, "/v1/orders/me", nil)
	req.Header.Set("Authorization", "Bearer "+adminTok)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("admin on customer route: status=%d body=%s", rec.Code, rec.Body.String())
	}

	customerTok := issueTestToken(t, secret, "u1", roleCustomer, "s2", time.Hour)
	req2 := httptest.NewRequest(http.MethodGet, "/v1/orders/me", nil)
	req2.Header.Set("Authorization", "Bearer "+customerTok)
	rec2 := httptest.NewRecorder()
	r.ServeHTTP(rec2, req2)
	if rec2.Code != http.StatusOK {
		t.Fatalf("customer on orders: status=%d body=%s", rec2.Code, rec2.Body.String())
	}
}

func TestPublicRoutes_NoJWT(t *testing.T) {
	auth, _, _ := startMockUpstream(t)
	catalog, _, _ := startMockUpstream(t)
	geo, _, _ := startMockUpstream(t)
	r := testRouter(t, "test-secret", upstreams{auth: auth.URL, catalog: catalog.URL, geo: geo.URL})
	cases := []struct {
		method, path string
		want         int
	}{
		{http.MethodGet, "/healthz", http.StatusOK},
		{http.MethodGet, "/v1/hello", http.StatusOK},
		{http.MethodPost, "/v1/auth/otp/request", http.StatusOK},
		{http.MethodPost, "/v1/auth/admin/login", http.StatusOK},
		{http.MethodGet, "/v1/products", http.StatusOK},
		{http.MethodGet, "/v1/geo/store", http.StatusOK},
		{http.MethodGet, "/v1/geo/search", http.StatusOK},
	}
	for _, tc := range cases {
		req := httptest.NewRequest(tc.method, tc.path, nil)
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)
		if rec.Code != tc.want {
			t.Fatalf("%s %s: status=%d want=%d body=%s", tc.method, tc.path, rec.Code, tc.want, rec.Body.String())
		}
	}
}

func TestParseAccessToken_Expired(t *testing.T) {
	secret := "test-secret"
	now := time.Now().Add(-2 * time.Hour)
	claims := AccessClaims{
		Role:      roleCustomer,
		SessionID: "s1",
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   "u1",
			Issuer:    tokenIssuer,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(time.Minute)),
			ID:        "s1",
		},
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	raw, err := tok.SignedString([]byte(secret))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := parseAccessToken(secret, raw); err == nil {
		t.Fatal("expected expiry error")
	}
}

func TestParseAccessToken_MissingExp(t *testing.T) {
	secret := "test-secret"
	claims := AccessClaims{
		Role:      roleCustomer,
		SessionID: "s1",
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:  "u1",
			Issuer:   tokenIssuer,
			IssuedAt: jwt.NewNumericDate(time.Now()),
			ID:       "s1",
		},
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	raw, err := tok.SignedString([]byte(secret))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := parseAccessToken(secret, raw); err == nil {
		t.Fatal("expected missing exp error")
	}
}

func TestParseAccessToken_WrongIssuer(t *testing.T) {
	secret := "test-secret"
	now := time.Now()
	claims := AccessClaims{
		Role:      roleCustomer,
		SessionID: "s1",
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   "u1",
			Issuer:    "other-issuer",
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(time.Hour)),
			ID:        "s1",
		},
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	raw, err := tok.SignedString([]byte(secret))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := parseAccessToken(secret, raw); err == nil {
		t.Fatal("expected issuer error")
	}
}

func TestBearerToken(t *testing.T) {
	if _, ok := bearerToken(""); ok {
		t.Fatal("empty")
	}
	if _, ok := bearerToken("Basic x"); ok {
		t.Fatal("basic")
	}
	tok, ok := bearerToken("Bearer abc.def")
	if !ok || tok != "abc.def" {
		t.Fatalf("got %q ok=%v", tok, ok)
	}
}
