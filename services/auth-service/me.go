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
	db       *sql.DB
	phoneKey []byte
}

type meView struct {
	ID          string  `json:"id"`
	Role        string  `json:"role"`
	PhoneMasked string  `json:"phone_masked,omitempty"`
	FullName    *string `json:"full_name,omitempty"`
	Email       *string `json:"email,omitempty"`
	PictureURL  *string `json:"picture_url,omitempty"`
}

type patchMeBody struct {
	FullName *string `json:"full_name"`
	Phone    *string `json:"phone"`
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
	var fullName, email, pictureURL sql.NullString
	err := s.db.QueryRow(
		`SELECT COALESCE(NULLIF(contact_phone_masked, ''), phone_masked), full_name, email, picture_url FROM users WHERE id = ?`, userID,
	).Scan(&masked, &fullName, &email, &pictureURL)
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
	if email.Valid && strings.TrimSpace(email.String) != "" {
		value := strings.TrimSpace(email.String)
		out.Email = &value
	}
	if pictureURL.Valid && strings.TrimSpace(pictureURL.String) != "" {
		value := strings.TrimSpace(pictureURL.String)
		out.PictureURL = &value
	}
	httpx.JSON(w, http.StatusOK, out)
}

// handlePatchMe updates customer profile fields. Phone is contact information;
// Google remains the authentication method and no OTP is sent.
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
	if body.FullName == nil && body.Phone == nil {
		httpx.Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "full_name or phone is required")
		return
	}
	var nameValue any
	if body.FullName != nil {
		name := strings.TrimSpace(*body.FullName)
		if name == "" {
			httpx.Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "full_name must not be empty")
			return
		}
		if len([]rune(name)) > 80 {
			httpx.Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "full_name too long")
			return
		}
		nameValue = name
	}
	var phoneEnc, phoneMaskedValue any
	if body.Phone != nil {
		e164, err := normalizePhoneVN(*body.Phone)
		if err != nil {
			httpx.Error(w, http.StatusBadRequest, "INVALID_PHONE", "phone must be a valid Vietnam mobile number")
			return
		}
		enc, err := encryptPhoneE164(e164, s.phoneKey)
		if err != nil {
			httpx.Error(w, http.StatusInternalServerError, "INTERNAL", "could not update profile")
			return
		}
		phoneEnc = enc
		phoneMaskedValue = maskPhoneE164(e164)
	}

	now := time.Now().UTC().Format(time.RFC3339Nano)
	res, err := s.db.Exec(
		`UPDATE users SET
			full_name = COALESCE(?, full_name),
			contact_phone_e164_enc = COALESCE(?, contact_phone_e164_enc),
			contact_phone_masked = COALESCE(?, contact_phone_masked),
			updated_at = ?
		 WHERE id = ?`,
		nameValue, phoneEnc, phoneMaskedValue, now, userID,
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
	var fullName, email, pictureURL sql.NullString
	err = s.db.QueryRow(`
		SELECT COALESCE(NULLIF(contact_phone_masked, ''), phone_masked), full_name, email, picture_url
		FROM users WHERE id = ?
	`, userID).Scan(&masked, &fullName, &email, &pictureURL)
	if err != nil {
		slog.Error("patch me reload", "err", err)
		httpx.Error(w, http.StatusInternalServerError, "INTERNAL", "could not load profile")
		return
	}
	out := meView{
		ID:          userID,
		Role:        role,
		PhoneMasked: masked,
	}
	if fullName.Valid && strings.TrimSpace(fullName.String) != "" {
		value := strings.TrimSpace(fullName.String)
		out.FullName = &value
	}
	if email.Valid && strings.TrimSpace(email.String) != "" {
		value := strings.TrimSpace(email.String)
		out.Email = &value
	}
	if pictureURL.Valid && strings.TrimSpace(pictureURL.String) != "" {
		value := strings.TrimSpace(pictureURL.String)
		out.PictureURL = &value
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
