package main

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"

	"gas-tam-de/pkg/httpx"
)

// maxPhoneLookupIDs bounds one lookup so a single call cannot walk the whole
// user table.
const maxPhoneLookupIDs = 500

type internalPhonesBody struct {
	UserIDs []string `json:"user_ids"`
}

// handleInternalUserPhones serves POST /v1/internal/users/phones.
//
// auth-service is the only place the plaintext number exists (AES-GCM at rest,
// see phone_crypto.go), so order-service asks here for the number an admin can
// actually dial, and snapshots it on the order.
//
// Like /v1/internal/stock/* and /v1/internal/payments this route is NOT mounted
// on the gateway — it is reachable only from inside the compose network.
func (s *meService) handleInternalUserPhones(w http.ResponseWriter, r *http.Request) {
	var body internalPhonesBody
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&body); err != nil {
		httpx.Error(w, http.StatusBadRequest, "INVALID_JSON", "invalid request body")
		return
	}

	ids := make([]string, 0, len(body.UserIDs))
	seen := make(map[string]struct{}, len(body.UserIDs))
	for _, raw := range body.UserIDs {
		id := strings.TrimSpace(raw)
		if id == "" {
			continue
		}
		if _, dup := seen[id]; dup {
			continue
		}
		seen[id] = struct{}{}
		ids = append(ids, id)
	}
	if len(ids) == 0 {
		httpx.Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "user_ids must not be empty")
		return
	}
	if len(ids) > maxPhoneLookupIDs {
		httpx.Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "too many user_ids")
		return
	}

	phones, err := s.phonesByUserID(ids)
	if err != nil {
		slog.Error("internal user phones", "err", err)
		httpx.Error(w, http.StatusInternalServerError, "INTERNAL", "could not load phones")
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"phones": phones})
}

// phonesByUserID decrypts the contact phone (falling back to the login phone)
// for each id. Users with no number on file — a Google account that never added
// a contact phone — are simply absent from the map.
func (s *meService) phonesByUserID(ids []string) (map[string]string, error) {
	placeholders := strings.TrimSuffix(strings.Repeat("?,", len(ids)), ",")
	args := make([]any, 0, len(ids))
	for _, id := range ids {
		args = append(args, id)
	}

	rows, err := s.db.Query(`
		SELECT id, contact_phone_e164_enc, phone_e164_enc
		FROM users WHERE id IN (`+placeholders+`)`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make(map[string]string, len(ids))
	for rows.Next() {
		var id string
		var contactEnc, loginEnc []byte
		if err := rows.Scan(&id, &contactEnc, &loginEnc); err != nil {
			return nil, err
		}
		if phone := s.decryptFirstPhone(id, contactEnc, loginEnc); phone != "" {
			out[id] = localPhoneVN(phone)
		}
	}
	return out, rows.Err()
}

// decryptFirstPhone prefers the profile contact number over the login number.
// A blob that fails to decrypt is logged and skipped rather than failing the
// whole batch — one unreadable row must not hide every other customer's number.
func (s *meService) decryptFirstPhone(userID string, blobs ...[]byte) string {
	for _, blob := range blobs {
		if len(blob) == 0 {
			continue
		}
		phone, err := decryptPhoneE164(blob, s.phoneKey)
		if err != nil {
			slog.Error("decrypt phone", "user_id", userID, "err", err)
			continue
		}
		if phone = strings.TrimSpace(phone); phone != "" {
			return phone
		}
	}
	return ""
}

// localPhoneVN renders +84901234567 as 0901234567 — the form a Vietnamese shop
// dials and the form the admin UI shows.
func localPhoneVN(e164 string) string {
	s := strings.TrimSpace(e164)
	if strings.HasPrefix(s, "+84") && len(s) == 12 {
		return "0" + s[3:]
	}
	return s
}
