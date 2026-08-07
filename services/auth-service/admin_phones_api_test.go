package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"gas-tam-de/pkg/httpx"
)

func adminPhoneRouter(svc *adminPhoneService) http.Handler {
	r := httpx.NewRouter("auth-test")
	r.Get("/v1/admin/admin-phones", svc.handleList)
	r.Post("/v1/admin/admin-phones", svc.handleCreate)
	r.Delete("/v1/admin/admin-phones/{id}", svc.handleDelete)
	return r
}

// callAdminPhones mimics the gateway, which strips client-supplied identity
// headers and re-injects them from the verified JWT.
func callAdminPhones(t *testing.T, h http.Handler, method, path, role, actorID string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var payload []byte
	if body != nil {
		payload, _ = json.Marshal(body)
	}
	req := httptest.NewRequest(method, path, bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	if actorID != "" {
		req.Header.Set("X-User-Id", actorID)
	}
	if role != "" {
		req.Header.Set("X-User-Role", role)
	}
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	return rr
}

func TestAdminPhonesRequireAdminRole(t *testing.T) {
	svc := &adminPhoneService{db: openTestDB(t), phonePepper: "pepper"}
	h := adminPhoneRouter(svc)

	if rr := callAdminPhones(t, h, http.MethodGet, "/v1/admin/admin-phones", "", "", nil); rr.Code != http.StatusUnauthorized {
		t.Fatalf("anonymous status=%d, want 401", rr.Code)
	}
	if rr := callAdminPhones(t, h, http.MethodGet, "/v1/admin/admin-phones", roleCustomer, "u1", nil); rr.Code != http.StatusForbidden {
		t.Fatalf("customer status=%d, want 403", rr.Code)
	}
}

func TestAdminPhonesAddListRemove(t *testing.T) {
	svc := &adminPhoneService{db: openTestDB(t), phonePepper: "pepper"}
	h := adminPhoneRouter(svc)

	rr := callAdminPhones(t, h, http.MethodPost, "/v1/admin/admin-phones", roleAdmin, "actor",
		map[string]string{"phone": "0909777020", "label": "Chủ cửa hàng"})
	if rr.Code != http.StatusCreated {
		t.Fatalf("create status=%d body=%s", rr.Code, rr.Body.String())
	}
	var created adminPhoneView
	_ = json.Unmarshal(rr.Body.Bytes(), &created)
	if created.PhoneMasked != "090***7020" || created.Label != "Chủ cửa hàng" {
		t.Fatalf("created=%+v", created)
	}

	// Adding the same number again is a no-op rather than a duplicate row.
	rr = callAdminPhones(t, h, http.MethodPost, "/v1/admin/admin-phones", roleAdmin, "actor",
		map[string]string{"phone": "+84909777020"})
	if rr.Code != http.StatusOK {
		t.Fatalf("re-add status=%d body=%s", rr.Code, rr.Body.String())
	}

	rr = callAdminPhones(t, h, http.MethodPost, "/v1/admin/admin-phones", roleAdmin, "actor",
		map[string]string{"phone": "0912345678"})
	if rr.Code != http.StatusCreated {
		t.Fatalf("second create status=%d body=%s", rr.Code, rr.Body.String())
	}

	rr = callAdminPhones(t, h, http.MethodGet, "/v1/admin/admin-phones", roleAdmin, "actor", nil)
	if rr.Code != http.StatusOK {
		t.Fatalf("list status=%d", rr.Code)
	}
	var listed struct {
		AdminPhones []adminPhoneView `json:"admin_phones"`
	}
	_ = json.Unmarshal(rr.Body.Bytes(), &listed)
	if len(listed.AdminPhones) != 2 {
		t.Fatalf("listed %d entries, want 2", len(listed.AdminPhones))
	}

	rr = callAdminPhones(t, h, http.MethodDelete, "/v1/admin/admin-phones/"+created.ID, roleAdmin, "actor", nil)
	if rr.Code != http.StatusOK {
		t.Fatalf("delete status=%d body=%s", rr.Code, rr.Body.String())
	}
	if n, _ := countAdminPhones(svc.db); n != 1 {
		t.Fatalf("count after delete=%d, want 1", n)
	}
}

func TestAdminPhonesRejectInvalidNumber(t *testing.T) {
	svc := &adminPhoneService{db: openTestDB(t), phonePepper: "pepper"}
	h := adminPhoneRouter(svc)

	rr := callAdminPhones(t, h, http.MethodPost, "/v1/admin/admin-phones", roleAdmin, "actor",
		map[string]string{"phone": "12345"})
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var resp map[string]any
	_ = json.Unmarshal(rr.Body.Bytes(), &resp)
	errObj, _ := resp["error"].(map[string]any)
	if errObj["code"] != "INVALID_PHONE" {
		t.Fatalf("error=%v", resp)
	}
}

// Emptying the list would lock every phone admin out, so the last entry stays.
func TestAdminPhonesKeepLastEntry(t *testing.T) {
	svc := &adminPhoneService{db: openTestDB(t), phonePepper: "pepper"}
	h := adminPhoneRouter(svc)

	rr := callAdminPhones(t, h, http.MethodPost, "/v1/admin/admin-phones", roleAdmin, "actor",
		map[string]string{"phone": "0909777020"})
	var only adminPhoneView
	_ = json.Unmarshal(rr.Body.Bytes(), &only)

	rr = callAdminPhones(t, h, http.MethodDelete, "/v1/admin/admin-phones/"+only.ID, roleAdmin, "actor", nil)
	if rr.Code != http.StatusConflict {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	if n, _ := countAdminPhones(svc.db); n != 1 {
		t.Fatalf("count=%d, want the entry kept", n)
	}
}

// is_self lets the UI warn an admin who is about to revoke their own access.
func TestAdminPhonesMarkCallersOwnEntry(t *testing.T) {
	otp := testOTPService(t)
	svc := &adminPhoneService{db: otp.db, phonePepper: otp.phonePepper}
	h := adminPhoneRouter(svc)

	if err := seedAdminPhones(otp.db, "0909777020", otp.phonePepper); err != nil {
		t.Fatal(err)
	}
	_, _, user := otpLogin(t, otp, "0909777020")
	actorID, _ := user["id"].(string)

	if rr := callAdminPhones(t, h, http.MethodPost, "/v1/admin/admin-phones", roleAdmin, actorID,
		map[string]string{"phone": "0912345678"}); rr.Code != http.StatusCreated {
		t.Fatalf("create status=%d", rr.Code)
	}

	rr := callAdminPhones(t, h, http.MethodGet, "/v1/admin/admin-phones", roleAdmin, actorID, nil)
	var listed struct {
		AdminPhones []adminPhoneView `json:"admin_phones"`
	}
	_ = json.Unmarshal(rr.Body.Bytes(), &listed)

	selfCount := 0
	for _, p := range listed.AdminPhones {
		if p.IsSelf {
			selfCount++
			if p.PhoneMasked != "090***7020" {
				t.Fatalf("is_self on the wrong entry: %+v", p)
			}
		}
	}
	if selfCount != 1 {
		t.Fatalf("is_self set on %d entries, want 1", selfCount)
	}
}
