package main

import (
	"database/sql"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
)

// AuditEntry is one admin mutating action observed at the gateway.
type AuditEntry struct {
	ActorID   string
	Method    string
	Path      string
	Status    int
	RequestID string
	At        time.Time
}

// AuditRecorder persists or emits admin audit entries.
type AuditRecorder interface {
	Record(entry AuditEntry)
}

func isMutatingMethod(method string) bool {
	switch method {
	case http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
		return true
	default:
		return false
	}
}

// AuditAdminMutations records who/what/when/outcome for admin mutating requests.
// Mount after RequireJWT + RequireRole(admin). Non-mutating methods pass through.
func AuditAdminMutations(rec AuditRecorder) func(http.Handler) http.Handler {
	if rec == nil {
		rec = slogAuditRecorder{}
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !isMutatingMethod(r.Method) {
				next.ServeHTTP(w, r)
				return
			}
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
			next.ServeHTTP(ww, r)

			actorID := ""
			if c, ok := ClaimsFromContext(r.Context()); ok {
				actorID = c.Subject
			}
			status := ww.Status()
			if status == 0 {
				status = http.StatusOK
			}
			rec.Record(AuditEntry{
				ActorID:   actorID,
				Method:    r.Method,
				Path:      r.URL.Path,
				Status:    status,
				RequestID: middleware.GetReqID(r.Context()),
				At:        time.Now().UTC(),
			})
		})
	}
}

// slogAuditRecorder emits structured admin_audit logs (always-on sink).
type slogAuditRecorder struct{}

func (slogAuditRecorder) Record(e AuditEntry) {
	slog.Info("admin_audit",
		"actor_id", e.ActorID,
		"method", e.Method,
		"path", e.Path,
		"status", e.Status,
		"request_id", e.RequestID,
		"at", e.At.Format(time.RFC3339Nano),
	)
}

// sqliteAuditRecorder appends rows to admin_audit_logs. Failures are logged; never fail the request.
type sqliteAuditRecorder struct {
	db *sql.DB
}

func newSQLiteAuditRecorder(db *sql.DB) *sqliteAuditRecorder {
	return &sqliteAuditRecorder{db: db}
}

func (s *sqliteAuditRecorder) Record(e AuditEntry) {
	if s == nil || s.db == nil {
		return
	}
	at := e.At
	if at.IsZero() {
		at = time.Now().UTC()
	}
	_, err := s.db.Exec(
		`INSERT INTO admin_audit_logs (id, actor_id, method, path, status, request_id, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		uuid.NewString(),
		e.ActorID,
		e.Method,
		e.Path,
		e.Status,
		nullIfEmpty(e.RequestID),
		at.Format(time.RFC3339Nano),
	)
	if err != nil {
		slog.Error("admin_audit persist failed",
			"err", err,
			"actor_id", e.ActorID,
			"method", e.Method,
			"path", e.Path,
			"status", e.Status,
		)
	}
}

func nullIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}

// multiAuditRecorder fans out to multiple sinks.
type multiAuditRecorder struct {
	sinks []AuditRecorder
}

func newMultiAuditRecorder(sinks ...AuditRecorder) AuditRecorder {
	out := make([]AuditRecorder, 0, len(sinks))
	for _, s := range sinks {
		if s != nil {
			out = append(out, s)
		}
	}
	if len(out) == 0 {
		return slogAuditRecorder{}
	}
	return &multiAuditRecorder{sinks: out}
}

func (m *multiAuditRecorder) Record(e AuditEntry) {
	for _, s := range m.sinks {
		s.Record(e)
	}
}

// MemoryAuditRecorder stores entries in memory (tests).
type MemoryAuditRecorder struct {
	mu      sync.Mutex
	entries []AuditEntry
}

func NewMemoryAuditRecorder() *MemoryAuditRecorder {
	return &MemoryAuditRecorder{}
}

func (m *MemoryAuditRecorder) Record(e AuditEntry) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.entries = append(m.entries, e)
}

func (m *MemoryAuditRecorder) Entries() []AuditEntry {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]AuditEntry, len(m.entries))
	copy(out, m.entries)
	return out
}
