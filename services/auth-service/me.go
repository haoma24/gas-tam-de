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
)

type meService struct {
	db *sql.DB
}

type meView struct {
	ID          string  `json:"id"`
	Role        string  `json:"role"`
	PhoneMasked string  `json:"phone_masked,omitempty"`
	FullName    *string `json:"full_name,omitempty"`
}

type patchMeBody struct {
	FullName *string `json:"full_name"`
}

// handleGetMe serves GET /v1/me — customer profile (gateway injects X-User-*).
func (s *meService) handleGetMe(w http.ResponseWriter, r *http.Request) {
	userID, role, phoneMasked, ok := requireMeIdentity(w, r)
	if !ok {
		return
	}
	if role != "customer" {
		httpx.Error(w, http.StatusForbidden, "FORBIDDEN", "customer profile only")
		return
	}

	var masked string
	var fullName sql.NullString
	err := s.db.QueryRow(
		`SELECT phone_masked, full_name FROM users WHERE id = ?`, userID,
	).Scan(&masked, &fullName)
	if errors.Is(err, sql.ErrNoRows) {
		httpx.Error(w, http.StatusNotFound, "NOT_FOUND", "user not found")
		return
	}
	if err != nil {
		slog.Error("get me", "err", err)
		httpx.Error(w, http.StatusInternalServerError, "INTERNAL", "could not load profile")
		return
	}
	if phoneMasked == "" {
		phoneMasked = masked
	}

	out := meView{
		ID:          userID,
		Role:        role,
		PhoneMasked: phoneMasked,
	}
	if fullName.Valid {
		n := strings.TrimSpace(fullName.String)
		if n != "" {
			out.FullName = &n
		}
	}
	httpx.JSON(w, http.StatusOK, out)
}

// handlePatchMe serves PATCH /v1/me — update full_name (first-time / correct name).
func (s *meService) handlePatchMe(w http.ResponseWriter, r *http.Request) {
	userID, role, _, ok := requireMeIdentity(w, r)
	if !ok {
		return
	}
	if role != "customer" {
		httpx.Error(w, http.StatusForbidden, "FORBIDDEN", "customer profile only")
		return
	}

	var body patchMeBody
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&body); err != nil {
		httpx.Error(w, http.StatusBadRequest, "INVALID_JSON", "invalid request body")
		return
	}
	if body.FullName == nil {
		httpx.Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "full_name is required")
		return
	}
	name := strings.TrimSpace(*body.FullName)
	if name == "" {
		httpx.Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "full_name must not be empty")
		return
	}
	if len([]rune(name)) > 80 {
		httpx.Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "full_name too long")
		return
	}

	now := time.Now().UTC().Format(time.RFC3339Nano)
	res, err := s.db.Exec(
		`UPDATE users SET full_name = ?, updated_at = ? WHERE id = ?`,
		name, now, userID,
	)
	if err != nil {
		slog.Error("patch me", "err", err)
		httpx.Error(w, http.StatusInternalServerError, "INTERNAL", "could not update profile")
		return
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		httpx.Error(w, http.StatusNotFound, "NOT_FOUND", "user not found")
		return
	}

	var masked string
	err = s.db.QueryRow(`SELECT phone_masked FROM users WHERE id = ?`, userID).Scan(&masked)
	if err != nil {
		slog.Error("patch me reload", "err", err)
		httpx.Error(w, http.StatusInternalServerError, "INTERNAL", "could not load profile")
		return
	}
	out := meView{
		ID:          userID,
		Role:        role,
		PhoneMasked: masked,
		FullName:    &name,
	}
	httpx.JSON(w, http.StatusOK, out)
}

func requireMeIdentity(w http.ResponseWriter, r *http.Request) (userID, role, phoneMasked string, ok bool) {
	userID = strings.TrimSpace(r.Header.Get("X-User-Id"))
	role = strings.TrimSpace(r.Header.Get("X-User-Role"))
	phoneMasked = strings.TrimSpace(r.Header.Get("X-Phone-Masked"))
	if userID == "" {
		httpx.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "missing X-User-Id (gateway JWT required)")
		return "", "", "", false
	}
	if role == "" {
		role = "customer"
	}
	return userID, role, phoneMasked, true
}
