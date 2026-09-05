package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func newPhoneLookupService(t *testing.T) *meService {
	t.Helper()
	db := openTestDB(t)
	t.Cleanup(func() { _ = db.Close() })
	if err := migrate(db); err != nil {
		t.Fatal(err)
	}
	return &meService{db: db, phoneKey: derivePhoneKey("test-phone-key")}
}

// insertPhoneUser writes a user with an encrypted login phone and, optionally,
// an encrypted profile contact phone.
func insertPhoneUser(t *testing.T, s *meService, id, loginE164, contactE164 string) {
	t.Helper()
	now := time.Now().UTC().Format(time.RFC3339Nano)

	var loginEnc []byte
	if loginE164 != "" {
		enc, err := encryptPhoneE164(loginE164, s.phoneKey)
		if err != nil {
			t.Fatal(err)
		}
		loginEnc = enc
	} else {
		loginEnc = []byte{}
	}

	var contactEnc any
	if contactE164 != "" {
		enc, err := encryptPhoneE164(contactE164, s.phoneKey)
		if err != nil {
			t.Fatal(err)
		}
		contactEnc = enc
	}

	_, err := s.db.Exec(`
		INSERT INTO users (id, phone_e164_enc, phone_hash, phone_masked,
		                   contact_phone_e164_enc, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		id, loginEnc, "hash-"+id, "090***0000", contactEnc, now, now)
	if err != nil {
		t.Fatal(err)
	}
}

func postPhoneLookup(t *testing.T, s *meService, ids []string) *httptest.ResponseRecorder {
	t.Helper()
	body, _ := json.Marshal(map[string]any{"user_ids": ids})
	req := httptest.NewRequest(http.MethodPost, "/v1/internal/users/phones", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	s.handleInternalUserPhones(rr, req)
	return rr
}

func phonesFrom(t *testing.T, rr *httptest.ResponseRecorder) map[string]string {
	t.Helper()
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var payload struct {
		Phones map[string]string `json:"phones"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	return payload.Phones
}

// TestInternalPhonesDecryptsAndLocalises — order-service needs the number the
// shop dials, decrypted from the at-rest blob and in 09… form.
func TestInternalPhonesDecryptsAndLocalises(t *testing.T) {
	s := newPhoneLookupService(t)
	insertPhoneUser(t, s, "user-a", "+84909777020", "")

	got := phonesFrom(t, postPhoneLookup(t, s, []string{"user-a"}))
	if got["user-a"] != "0909777020" {
		t.Fatalf("phone=%q, want 0909777020", got["user-a"])
	}
}

// TestInternalPhonesPrefersContactPhone — a customer who set a different
// contact number in their profile wants to be called on that one.
func TestInternalPhonesPrefersContactPhone(t *testing.T) {
	s := newPhoneLookupService(t)
	insertPhoneUser(t, s, "user-a", "+84909777020", "+84912345678")

	got := phonesFrom(t, postPhoneLookup(t, s, []string{"user-a"}))
	if got["user-a"] != "0912345678" {
		t.Fatalf("phone=%q, want the contact number 0912345678", got["user-a"])
	}
}

// TestInternalPhonesSkipsUsersWithoutNumber — a Google account that never added
// a phone is absent from the map, not an error for the whole batch.
func TestInternalPhonesSkipsUsersWithoutNumber(t *testing.T) {
	s := newPhoneLookupService(t)
	insertPhoneUser(t, s, "user-a", "+84909777020", "")
	insertPhoneUser(t, s, "user-google", "", "")

	got := phonesFrom(t, postPhoneLookup(t, s, []string{"user-a", "user-google", "user-missing"}))
	if len(got) != 1 || got["user-a"] == "" {
		t.Fatalf("phones=%v, want only user-a", got)
	}
}

func TestInternalPhonesRejectsEmptyAndOversizedBatches(t *testing.T) {
	s := newPhoneLookupService(t)

	if rr := postPhoneLookup(t, s, nil); rr.Code != http.StatusBadRequest {
		t.Fatalf("empty batch status=%d, want 400", rr.Code)
	}

	tooMany := make([]string, maxPhoneLookupIDs+1)
	for i := range tooMany {
		tooMany[i] = "user-" + string(rune('a'+i%26)) + string(rune('a'+i/26))
	}
	if rr := postPhoneLookup(t, s, tooMany); rr.Code != http.StatusBadRequest {
		t.Fatalf("oversized batch status=%d, want 400", rr.Code)
	}
}
