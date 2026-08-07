package main

import (
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
)

// adminPhoneRow is one entry of the phone allow-list that grants role=admin
// after a normal customer OTP login.
type adminPhoneRow struct {
	ID          string
	PhoneHash   string
	PhoneMasked string
	Label       sql.NullString
	CreatedAt   string
}

// queryer covers *sql.DB and *sql.Tx so lookups work inside the OTP verify and
// refresh transactions as well as from plain handlers.
type queryer interface {
	Query(query string, args ...any) (*sql.Rows, error)
	QueryRow(query string, args ...any) *sql.Row
	Exec(query string, args ...any) (sql.Result, error)
}

// isAdminPhone reports whether a peppered phone hash is on the allow-list.
func isAdminPhone(q queryer, phoneHash string) (bool, error) {
	var one int
	err := q.QueryRow(`SELECT 1 FROM admin_phones WHERE phone_hash = ?`, phoneHash).Scan(&one)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

// roleForPhone maps a phone hash to the role its session should carry.
func roleForPhone(q queryer, phoneHash string) (string, error) {
	admin, err := isAdminPhone(q, phoneHash)
	if err != nil {
		return "", err
	}
	if admin {
		return roleAdmin, nil
	}
	return roleCustomer, nil
}

func listAdminPhones(q queryer) ([]adminPhoneRow, error) {
	rows, err := q.Query(`
		SELECT id, phone_hash, phone_masked, label, created_at
		FROM admin_phones
		ORDER BY created_at ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []adminPhoneRow
	for rows.Next() {
		var r adminPhoneRow
		if err := rows.Scan(&r.ID, &r.PhoneHash, &r.PhoneMasked, &r.Label, &r.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func countAdminPhones(q queryer) (int, error) {
	var n int
	err := q.QueryRow(`SELECT COUNT(*) FROM admin_phones`).Scan(&n)
	return n, err
}

// insertAdminPhone adds one entry. Returns the row plus whether it was created;
// an existing phone is returned unchanged so the endpoint stays idempotent.
func insertAdminPhone(q queryer, phoneHash, phoneMasked, label, createdBy string, now time.Time) (adminPhoneRow, bool, error) {
	existing := adminPhoneRow{}
	err := q.QueryRow(`
		SELECT id, phone_hash, phone_masked, label, created_at
		FROM admin_phones WHERE phone_hash = ?
	`, phoneHash).Scan(
		&existing.ID, &existing.PhoneHash, &existing.PhoneMasked,
		&existing.Label, &existing.CreatedAt,
	)
	if err == nil {
		return existing, false, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return adminPhoneRow{}, false, err
	}

	row := adminPhoneRow{
		ID:          uuid.NewString(),
		PhoneHash:   phoneHash,
		PhoneMasked: phoneMasked,
		CreatedAt:   now.UTC().Format(time.RFC3339Nano),
	}
	var labelArg any
	if l := strings.TrimSpace(label); l != "" {
		row.Label = sql.NullString{String: l, Valid: true}
		labelArg = l
	}
	var createdByArg any
	if createdBy != "" {
		createdByArg = createdBy
	}

	_, err = q.Exec(`
		INSERT INTO admin_phones (id, phone_hash, phone_masked, label, created_at, created_by)
		VALUES (?, ?, ?, ?, ?, ?)
	`, row.ID, row.PhoneHash, row.PhoneMasked, labelArg, row.CreatedAt, createdByArg)
	if err != nil {
		return adminPhoneRow{}, false, err
	}
	return row, true, nil
}

func deleteAdminPhone(q queryer, id string) (adminPhoneRow, error) {
	var row adminPhoneRow
	err := q.QueryRow(`
		SELECT id, phone_hash, phone_masked, label, created_at
		FROM admin_phones WHERE id = ?
	`, id).Scan(&row.ID, &row.PhoneHash, &row.PhoneMasked, &row.Label, &row.CreatedAt)
	if err != nil {
		return adminPhoneRow{}, err
	}
	if _, err := q.Exec(`DELETE FROM admin_phones WHERE id = ?`, id); err != nil {
		return adminPhoneRow{}, err
	}
	return row, nil
}

// seedAdminPhones bootstraps the allow-list from ADMIN_PHONES (comma-separated).
// Idempotent: entries added or removed from the admin screen are never undone by
// a restart, so the env var only ever adds the numbers it lists.
func seedAdminPhones(db *sql.DB, raw, pepper string) error {
	for _, item := range strings.Split(raw, ",") {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		e164, err := normalizePhoneVN(item)
		if err != nil {
			return fmt.Errorf("admin phone seed %q: %w", item, err)
		}
		hash := hashPhone(e164, pepper)
		row, created, err := insertAdminPhone(db, hash, maskPhoneE164(e164), "ADMIN_PHONES", "", time.Now())
		if err != nil {
			return fmt.Errorf("admin phone seed %q: %w", item, err)
		}
		if created {
			slog.Info("admin phone seeded", "id", row.ID, "phone_masked", row.PhoneMasked)
		}
	}
	return nil
}
