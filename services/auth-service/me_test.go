package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestMeGetAndPatch(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	if err := migrate(db); err != nil {
		t.Fatal(err)
	}

	now := time.Now().UTC().Format(time.RFC3339Nano)
	_, err := db.Exec(`
		INSERT INTO users (id, phone_e164_enc, phone_hash, phone_masked, full_name, created_at, updated_at)
		VALUES (?, ?, ?, ?, NULL, ?, ?)
	`, "user-1", []byte("x"), "hash-1", "090***4567", now, now)
	if err != nil {
		t.Fatal(err)
	}

	me := &meService{db: db}
	r := httpxNewTestRouter(me)

	getReq := httptest.NewRequest(http.MethodGet, "/v1/me", nil)
	getReq.Header.Set("X-User-Id", "user-1")
	getReq.Header.Set("X-User-Role", "customer")
	getReq.Header.Set("X-Phone-Masked", "090***4567")
	getRec := httptest.NewRecorder()
	r.ServeHTTP(getRec, getReq)
	if getRec.Code != http.StatusOK {
		t.Fatalf("get status=%d body=%s", getRec.Code, getRec.Body.String())
	}
	var got map[string]any
	if err := json.Unmarshal(getRec.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got["full_name"] != nil {
		t.Fatalf("expected null full_name, got %v", got["full_name"])
	}

	body, _ := json.Marshal(map[string]string{"full_name": " Nguyễn Văn A "})
	patchReq := httptest.NewRequest(http.MethodPatch, "/v1/me", bytes.NewReader(body))
	patchReq.Header.Set("Content-Type", "application/json")
	patchReq.Header.Set("X-User-Id", "user-1")
	patchReq.Header.Set("X-User-Role", "customer")
	patchRec := httptest.NewRecorder()
	r.ServeHTTP(patchRec, patchReq)
	if patchRec.Code != http.StatusOK {
		t.Fatalf("patch status=%d body=%s", patchRec.Code, patchRec.Body.String())
	}
	var patched map[string]any
	if err := json.Unmarshal(patchRec.Body.Bytes(), &patched); err != nil {
		t.Fatal(err)
	}
	if patched["full_name"] != "Nguyễn Văn A" {
		t.Fatalf("full_name=%v", patched["full_name"])
	}
}

func httpxNewTestRouter(me *meService) http.Handler {
	mux := http.NewServeMux()
	// Minimal mux — auth tests usually use chi via httpx; keep simple:
	mux.HandleFunc("/v1/me", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			me.handleGetMe(w, r)
		case http.MethodPatch:
			me.handlePatchMe(w, r)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})
	return mux
}
