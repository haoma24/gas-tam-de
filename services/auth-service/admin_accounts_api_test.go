package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"gas-tam-de/pkg/httpx"
)

func adminAccountsTestHandler(t *testing.T) (*adminAccountService, http.Handler) {
	t.Helper()
	svc := &adminAccountService{db: openTestDB(t)}
	r := httpx.NewRouter("auth-test")
	r.Get("/v1/admin/admin-accounts", svc.handleList)
	r.Post("/v1/admin/admin-accounts", svc.handleCreate)
	r.Patch("/v1/admin/admin-accounts/{id}", svc.handleUpdate)
	return svc, r
}

func callAdminAccounts(t *testing.T, h http.Handler, method, path, actor string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var payload []byte
	if body != nil {
		payload, _ = json.Marshal(body)
	}
	req := httptest.NewRequest(method, path, bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	if actor != "" {
		req.Header.Set("X-User-Id", actor)
		req.Header.Set("X-User-Role", roleAdmin)
	}
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	return rr
}

func TestAdminAccountsCreateListAndLogin(t *testing.T) {
	svc, h := adminAccountsTestHandler(t)
	rr := callAdminAccounts(t, h, http.MethodPost, "/v1/admin/admin-accounts", "phone-admin", map[string]string{
		"username": "manager01", "password": "strong-pass-123", "display_name": "Quản lý ca sáng",
	})
	if rr.Code != http.StatusCreated {
		t.Fatalf("create status=%d body=%s", rr.Code, rr.Body.String())
	}
	var created adminAccountView
	if err := json.Unmarshal(rr.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}
	if created.ID == "" || created.Username != "manager01" || created.DisplayName != "Quản lý ca sáng" {
		t.Fatalf("created=%+v", created)
	}

	rr = callAdminAccounts(t, h, http.MethodGet, "/v1/admin/admin-accounts", "phone-admin", nil)
	if rr.Code != http.StatusOK || !bytes.Contains(rr.Body.Bytes(), []byte("manager01")) {
		t.Fatalf("list status=%d body=%s", rr.Code, rr.Body.String())
	}

	admin, err := loadAdminByUsername(svc.db, "manager01")
	if err != nil {
		t.Fatal(err)
	}
	if !verifyAdminPassword(admin.PasswordHash, "strong-pass-123") {
		t.Fatal("created password does not verify")
	}
	if bytes.Contains(rr.Body.Bytes(), []byte("password")) {
		t.Fatal("list response must not expose password data")
	}
}

func TestAdminAccountsRejectDuplicateAndWeakPassword(t *testing.T) {
	_, h := adminAccountsTestHandler(t)
	body := map[string]string{"username": "manager01", "password": "strong-pass-123"}
	if rr := callAdminAccounts(t, h, http.MethodPost, "/v1/admin/admin-accounts", "actor", body); rr.Code != http.StatusCreated {
		t.Fatalf("first create status=%d body=%s", rr.Code, rr.Body.String())
	}
	if rr := callAdminAccounts(t, h, http.MethodPost, "/v1/admin/admin-accounts", "actor", body); rr.Code != http.StatusConflict {
		t.Fatalf("duplicate status=%d body=%s", rr.Code, rr.Body.String())
	}
	body["username"], body["password"] = "manager02", "short"
	if rr := callAdminAccounts(t, h, http.MethodPost, "/v1/admin/admin-accounts", "actor", body); rr.Code != http.StatusBadRequest {
		t.Fatalf("weak password status=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestAdminAccountsSelfUpdateRequiresCurrentPassword(t *testing.T) {
	svc, h := adminAccountsTestHandler(t)
	if err := seedAdminAccount(svc.db, adminSeedConfig{
		Username: "owner", Password: "old-password", DisplayName: "Owner", Enabled: true,
	}); err != nil {
		t.Fatal(err)
	}
	admin, err := loadAdminByUsername(svc.db, "owner")
	if err != nil {
		t.Fatal(err)
	}
	path := "/v1/admin/admin-accounts/" + admin.ID
	body := map[string]string{"username": "new-owner", "new_password": "new-password-123"}
	if rr := callAdminAccounts(t, h, http.MethodPatch, path, admin.ID, body); rr.Code != http.StatusUnauthorized {
		t.Fatalf("missing current password status=%d body=%s", rr.Code, rr.Body.String())
	}
	body["current_password"] = "old-password"
	rr := callAdminAccounts(t, h, http.MethodPatch, path, admin.ID, body)
	if rr.Code != http.StatusOK {
		t.Fatalf("update status=%d body=%s", rr.Code, rr.Body.String())
	}
	updated, err := loadAdminByUsername(svc.db, "new-owner")
	if err != nil || !verifyAdminPassword(updated.PasswordHash, "new-password-123") {
		t.Fatalf("updated credentials invalid: %v", err)
	}
}

func TestAdminAccountsRequireAdminIdentity(t *testing.T) {
	_, h := adminAccountsTestHandler(t)
	rr := callAdminAccounts(t, h, http.MethodGet, "/v1/admin/admin-accounts", "", nil)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
}
