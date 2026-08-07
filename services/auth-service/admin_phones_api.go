package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"gas-tam-de/pkg/httpx"

	"github.com/go-chi/chi/v5"
)

// adminPhoneService serves `/v1/admin/admin-phones` — the list of numbers that
// get an admin session after a normal OTP login. The gateway has already
// enforced role=admin and injected the X-User-* identity headers.
type adminPhoneService struct {
	db          *sql.DB
	phonePepper string
}

type adminPhoneView struct {
	ID          string `json:"id"`
	PhoneMasked string `json:"phone_masked"`
	Label       string `json:"label,omitempty"`
	CreatedAt   string `json:"created_at"`
	IsSelf      bool   `json:"is_self"`
}

type createAdminPhoneBody struct {
	Phone string `json:"phone"`
	Label string `json:"label"`
}

func (s *adminPhoneService) handleList(w http.ResponseWriter, r *http.Request) {
	actor, ok := requireAdminIdentity(w, r)
	if !ok {
		return
	}
	rows, err := listAdminPhones(s.db)
	if err != nil {
		slog.Error("list admin phones", "err", err)
		httpx.Error(w, http.StatusInternalServerError, "INTERNAL", "could not load admin phones")
		return
	}
	out := make([]adminPhoneView, 0, len(rows))
	for _, r := range rows {
		out = append(out, s.view(r, actor))
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"admin_phones": out})
}

func (s *adminPhoneService) handleCreate(w http.ResponseWriter, r *http.Request) {
	actor, ok := requireAdminIdentity(w, r)
	if !ok {
		return
	}

	var body createAdminPhoneBody
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&body); err != nil {
		httpx.Error(w, http.StatusBadRequest, "INVALID_JSON", "invalid request body")
		return
	}
	e164, err := normalizePhoneVN(body.Phone)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "INVALID_PHONE", "phone must be a valid Vietnam mobile number")
		return
	}
	label := strings.TrimSpace(body.Label)
	if len([]rune(label)) > 60 {
		httpx.Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "label too long")
		return
	}

	row, created, err := insertAdminPhone(
		s.db, hashPhone(e164, s.phonePepper), maskPhoneE164(e164), label, actor, time.Now(),
	)
	if err != nil {
		slog.Error("add admin phone", "err", err)
		httpx.Error(w, http.StatusInternalServerError, "INTERNAL", "could not add admin phone")
		return
	}
	if created {
		slog.Info("admin phone added", "id", row.ID, "phone_masked", row.PhoneMasked, "actor_id", actor)
	}

	status := http.StatusOK
	if created {
		status = http.StatusCreated
	}
	httpx.JSON(w, status, s.view(row, actor))
}

func (s *adminPhoneService) handleDelete(w http.ResponseWriter, r *http.Request) {
	actor, ok := requireAdminIdentity(w, r)
	if !ok {
		return
	}
	id := strings.TrimSpace(chi.URLParam(r, "id"))
	if id == "" {
		httpx.Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "admin phone id is required")
		return
	}

	// Removing the last entry would leave the phone login with no admin at all,
	// so keep one. The username/password admin is unaffected either way.
	n, err := countAdminPhones(s.db)
	if err != nil {
		slog.Error("count admin phones", "err", err)
		httpx.Error(w, http.StatusInternalServerError, "INTERNAL", "could not remove admin phone")
		return
	}
	if n <= 1 {
		httpx.Error(w, http.StatusConflict, "LAST_ADMIN_PHONE", "cannot remove the last admin phone")
		return
	}

	row, err := deleteAdminPhone(s.db, id)
	if errors.Is(err, sql.ErrNoRows) {
		httpx.Error(w, http.StatusNotFound, "NOT_FOUND", "admin phone not found")
		return
	}
	if err != nil {
		slog.Error("remove admin phone", "err", err)
		httpx.Error(w, http.StatusInternalServerError, "INTERNAL", "could not remove admin phone")
		return
	}

	slog.Info("admin phone removed", "id", row.ID, "phone_masked", row.PhoneMasked, "actor_id", actor)
	httpx.JSON(w, http.StatusOK, map[string]any{"ok": true, "id": row.ID})
}

// view marks the caller's own entry so the UI can warn before self-removal.
func (s *adminPhoneService) view(row adminPhoneRow, actorID string) adminPhoneView {
	out := adminPhoneView{
		ID:          row.ID,
		PhoneMasked: row.PhoneMasked,
		CreatedAt:   row.CreatedAt,
	}
	if row.Label.Valid {
		out.Label = row.Label.String
	}
	if actorID != "" {
		var hash string
		err := s.db.QueryRow(`SELECT phone_hash FROM users WHERE id = ?`, actorID).Scan(&hash)
		out.IsSelf = err == nil && hash == row.PhoneHash
	}
	return out
}

func requireAdminIdentity(w http.ResponseWriter, r *http.Request) (actorID string, ok bool) {
	actorID = strings.TrimSpace(r.Header.Get("X-User-Id"))
	role := strings.TrimSpace(r.Header.Get("X-User-Role"))
	if actorID == "" {
		httpx.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "missing X-User-Id (gateway JWT required)")
		return "", false
	}
	if role != roleAdmin {
		httpx.Error(w, http.StatusForbidden, "FORBIDDEN", "admin role required")
		return "", false
	}
	return actorID, true
}
