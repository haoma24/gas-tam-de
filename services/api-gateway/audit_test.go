package main

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"gas-tam-de/pkg/sqlite"
)

func TestAuditAdminMutations_RecordsMutatingAdmin(t *testing.T) {
	secret := "secret"
	order, _, _ := startMockUpstream(t)
	mem := NewMemoryAuditRecorder()
	r := testRouterWithAudit(t, secret, upstreams{order: order.URL}, mem)

	tok := issueTestToken(t, secret, "admin-1", roleAdmin, "sess-1", time.Hour)
	req := httptest.NewRequest(http.MethodPost, "/v1/admin/orders/ord-9/complete", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}

	entries := mem.Entries()
	if len(entries) != 1 {
		t.Fatalf("entries=%d want 1", len(entries))
	}
	e := entries[0]
	if e.ActorID != "admin-1" {
		t.Fatalf("actor_id=%q", e.ActorID)
	}
	if e.Method != http.MethodPost {
		t.Fatalf("method=%q", e.Method)
	}
	if e.Path != "/v1/admin/orders/ord-9/complete" {
		t.Fatalf("path=%q", e.Path)
	}
	if e.Status != http.StatusOK {
		t.Fatalf("status=%d", e.Status)
	}
	if e.At.IsZero() {
		t.Fatal("at zero")
	}
}

func TestAuditAdminMutations_SkipsGet(t *testing.T) {
	secret := "secret"
	order, _, _ := startMockUpstream(t)
	mem := NewMemoryAuditRecorder()
	r := testRouterWithAudit(t, secret, upstreams{order: order.URL}, mem)

	tok := issueTestToken(t, secret, "admin-1", roleAdmin, "sess-1", time.Hour)
	req := httptest.NewRequest(http.MethodGet, "/v1/admin/orders", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d", rec.Code)
	}
	if n := len(mem.Entries()); n != 0 {
		t.Fatalf("entries=%d want 0 for GET", n)
	}
}

func TestAuditAdminMutations_RecordsUpstreamErrorStatus(t *testing.T) {
	secret := "secret"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusConflict)
		_, _ = w.Write([]byte(`{"error":"conflict"}`))
	}))
	t.Cleanup(srv.Close)

	mem := NewMemoryAuditRecorder()
	r := testRouterWithAudit(t, secret, upstreams{order: srv.URL}, mem)

	tok := issueTestToken(t, secret, "admin-2", roleAdmin, "sess-2", time.Hour)
	req := httptest.NewRequest(http.MethodPatch, "/v1/admin/delivery-fee", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusConflict {
		t.Fatalf("status=%d", rec.Code)
	}
	entries := mem.Entries()
	if len(entries) != 1 {
		t.Fatalf("entries=%d", len(entries))
	}
	if entries[0].Status != http.StatusConflict {
		t.Fatalf("audit status=%d", entries[0].Status)
	}
	if entries[0].ActorID != "admin-2" {
		t.Fatalf("actor=%q", entries[0].ActorID)
	}
}

func TestAuditAdminMutations_NoRecordWhenUnauthorized(t *testing.T) {
	mem := NewMemoryAuditRecorder()
	r := testRouterWithAudit(t, "secret", upstreams{}, mem)
	req := httptest.NewRequest(http.MethodPost, "/v1/admin/orders/x/complete", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d", rec.Code)
	}
	if n := len(mem.Entries()); n != 0 {
		t.Fatalf("entries=%d want 0", n)
	}
}

func TestSQLiteAuditRecorder_Persists(t *testing.T) {
	db := openTestGatewayDB(t)
	store := newSQLiteAuditRecorder(db)
	store.Record(AuditEntry{
		ActorID:   "admin-db",
		Method:    http.MethodDelete,
		Path:      "/v1/admin/products/p1",
		Status:    http.StatusNoContent,
		RequestID: "req-1",
		At:        time.Date(2026, 8, 2, 12, 0, 0, 0, time.UTC),
	})

	var actor, method, path, reqID, created string
	var status int
	err := db.QueryRow(
		`SELECT actor_id, method, path, status, request_id, created_at FROM admin_audit_logs`,
	).Scan(&actor, &method, &path, &status, &reqID, &created)
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if actor != "admin-db" || method != http.MethodDelete || path != "/v1/admin/products/p1" {
		t.Fatalf("row=%s %s %s", actor, method, path)
	}
	if status != http.StatusNoContent || reqID != "req-1" {
		t.Fatalf("status=%d req=%q", status, reqID)
	}
	if created == "" {
		t.Fatal("created_at empty")
	}
}

func openTestGatewayDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sqlite.Open(filepath.Join(t.TempDir(), "gateway.db"))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := migrateGateway(db); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}
