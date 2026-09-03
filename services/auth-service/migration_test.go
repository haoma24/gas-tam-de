package main

import (
	"database/sql"
	"path/filepath"
	"testing"

	"gas-tam-de/pkg/sqlite"
)

func TestMigrateAddsGoogleAndPersistentColumnsToExistingDatabase(t *testing.T) {
	db, err := sqlite.Open(filepath.Join(t.TempDir(), "auth.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Exec(`
		CREATE TABLE users (
			id TEXT PRIMARY KEY, phone_e164_enc BLOB NOT NULL,
			phone_hash TEXT NOT NULL UNIQUE, phone_masked TEXT NOT NULL,
			full_name TEXT, created_at TEXT NOT NULL, updated_at TEXT NOT NULL
		);
		CREATE TABLE sessions (
			id TEXT PRIMARY KEY, user_id TEXT NOT NULL, role TEXT NOT NULL,
			refresh_hash TEXT NOT NULL, expires_at TEXT NOT NULL,
			revoked_at TEXT, created_at TEXT NOT NULL
		);
	`); err != nil {
		t.Fatal(err)
	}
	if err := migrate(db); err != nil {
		t.Fatal(err)
	}
	for table, columns := range map[string][]string{
		"users": {
			"google_sub", "email", "picture_url",
			"contact_phone_e164_enc", "contact_phone_masked",
		},
		"sessions": {"persistent"},
	} {
		for _, column := range columns {
			if !testColumnExists(t, db, table, column) {
				t.Errorf("missing %s.%s", table, column)
			}
		}
	}
}

func testColumnExists(t *testing.T, db *sql.DB, table, want string) bool {
	t.Helper()
	rows, err := db.Query("PRAGMA table_info(" + table + ")")
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	for rows.Next() {
		var cid, notNull, pk int
		var name, columnType string
		var defaultValue sql.NullString
		if err := rows.Scan(&cid, &name, &columnType, &notNull, &defaultValue, &pk); err != nil {
			t.Fatal(err)
		}
		if name == want {
			return true
		}
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	return false
}
