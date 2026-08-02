package main

import (
	"database/sql"
	"strings"
	"testing"
)

func TestSeedAdminAccountCreatesBcryptHash(t *testing.T) {
	db := openTestDB(t)

	cfg := adminSeedConfig{
		Username:    "boss@example.com",
		Password:    "s3cret-local-only",
		DisplayName: "Chủ cửa hàng",
		Enabled:     true,
	}
	if err := seedAdminAccount(db, cfg); err != nil {
		t.Fatal(err)
	}

	var id, username, hash, display, created string
	var disabled sql.NullString
	err := db.QueryRow(`
		SELECT id, username, password_hash, display_name, created_at, disabled_at
		FROM admin_accounts WHERE username = ?
	`, cfg.Username).Scan(&id, &username, &hash, &display, &created, &disabled)
	if err != nil {
		t.Fatal(err)
	}
	if id == "" || created == "" {
		t.Fatalf("missing id/created_at: id=%q created=%q", id, created)
	}
	if username != cfg.Username || display != cfg.DisplayName {
		t.Fatalf("row username=%q display=%q", username, display)
	}
	if disabled.Valid {
		t.Fatalf("disabled_at should be NULL, got %q", disabled.String)
	}
	if hash == "" || hash == cfg.Password || strings.Contains(hash, cfg.Password) {
		t.Fatalf("password_hash must be bcrypt, not plaintext: %q", hash)
	}
	if !strings.HasPrefix(hash, "$2a$") && !strings.HasPrefix(hash, "$2b$") {
		t.Fatalf("expected bcrypt prefix, got %q", hash)
	}
	if !verifyAdminPassword(hash, cfg.Password) {
		t.Fatal("bcrypt verify failed for seeded password")
	}
	if verifyAdminPassword(hash, "wrong-password") {
		t.Fatal("bcrypt verify should fail for wrong password")
	}
}

func TestSeedAdminAccountIdempotent(t *testing.T) {
	db := openTestDB(t)
	cfg := adminSeedConfig{
		Username:    "admin",
		Password:    "first-password",
		DisplayName: "Admin",
		Enabled:     true,
	}
	if err := seedAdminAccount(db, cfg); err != nil {
		t.Fatal(err)
	}

	var firstID, firstHash string
	if err := db.QueryRow(`SELECT id, password_hash FROM admin_accounts WHERE username = ?`, cfg.Username).
		Scan(&firstID, &firstHash); err != nil {
		t.Fatal(err)
	}

	cfg.Password = "second-password-should-not-apply"
	if err := seedAdminAccount(db, cfg); err != nil {
		t.Fatal(err)
	}

	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM admin_accounts`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("count=%d want 1", count)
	}

	var secondID, secondHash string
	if err := db.QueryRow(`SELECT id, password_hash FROM admin_accounts WHERE username = ?`, cfg.Username).
		Scan(&secondID, &secondHash); err != nil {
		t.Fatal(err)
	}
	if secondID != firstID || secondHash != firstHash {
		t.Fatal("seed must not overwrite existing admin")
	}
	if !verifyAdminPassword(secondHash, "first-password") {
		t.Fatal("original password should still verify")
	}
}

func TestSeedAdminAccountDisabled(t *testing.T) {
	db := openTestDB(t)
	cfg := adminSeedConfig{
		Username: "admin",
		Password: "unused",
		Enabled:  false,
	}
	if err := seedAdminAccount(db, cfg); err != nil {
		t.Fatal(err)
	}
	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM admin_accounts`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("count=%d want 0 when seed disabled", count)
	}
}

func TestLoadAdminSeedConfigPrefersUsernameOverEmail(t *testing.T) {
	t.Setenv("ADMIN_USERNAME", "shopadmin")
	t.Setenv("ADMIN_EMAIL", "ignored@example.com")
	t.Setenv("ADMIN_PASSWORD", "pw-from-env")
	t.Setenv("ADMIN_DISPLAY_NAME", "Shop Admin")
	t.Setenv("ADMIN_SEED", "1")

	cfg := loadAdminSeedConfig()
	if cfg.Username != "shopadmin" {
		t.Fatalf("username=%q", cfg.Username)
	}
	if cfg.Password != "pw-from-env" || cfg.DisplayName != "Shop Admin" || !cfg.Enabled {
		t.Fatalf("cfg=%+v", cfg)
	}
}

func TestLoadAdminSeedConfigEmailAlias(t *testing.T) {
	t.Setenv("ADMIN_USERNAME", "")
	t.Setenv("ADMIN_EMAIL", "admin@gastamde.local")
	t.Setenv("ADMIN_PASSWORD", "")
	t.Setenv("ADMIN_SEED", "")

	cfg := loadAdminSeedConfig()
	if cfg.Username != "admin@gastamde.local" {
		t.Fatalf("username=%q want email alias", cfg.Username)
	}
	if cfg.Password != "admin-change-me" {
		t.Fatalf("password default=%q", cfg.Password)
	}
	if !cfg.Enabled {
		t.Fatal("seed should be enabled by default")
	}
}

func TestLoadAdminSeedConfigCanDisable(t *testing.T) {
	t.Setenv("ADMIN_SEED", "false")
	cfg := loadAdminSeedConfig()
	if cfg.Enabled {
		t.Fatal("ADMIN_SEED=false should disable")
	}
}
