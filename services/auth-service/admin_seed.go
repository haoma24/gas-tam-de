package main

import (
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"gas-tam-de/pkg/config"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// adminSeedConfig holds local/bootstrap credentials for the default admin row.
// Secrets come from env — never commit real passwords.
type adminSeedConfig struct {
	Username    string
	Password    string
	DisplayName string
	Enabled     bool
}

func loadAdminSeedConfig() adminSeedConfig {
	// ADMIN_SEED=0|false disables bootstrap entirely (e.g. production with external provision).
	enabled := true
	if v := strings.TrimSpace(os.Getenv("ADMIN_SEED")); v != "" {
		enabled = strings.EqualFold(v, "1") || strings.EqualFold(v, "true")
	}

	// Prefer ADMIN_USERNAME; ADMIN_EMAIL is an accepted alias (username may be an email).
	username := config.Get("ADMIN_USERNAME", "")
	if username == "" {
		username = config.Get("ADMIN_EMAIL", "admin")
	}

	return adminSeedConfig{
		Username:    username,
		Password:    config.Get("ADMIN_PASSWORD", "admin-change-me"),
		DisplayName: config.Get("ADMIN_DISPLAY_NAME", "Admin"),
		Enabled:     enabled,
	}
}

// seedAdminAccount inserts a default admin with bcrypt password hash when missing.
// Idempotent: existing username is left unchanged (password not reset on restart).
func seedAdminAccount(db *sql.DB, cfg adminSeedConfig) error {
	if !cfg.Enabled {
		slog.Info("admin seed skipped", "reason", "ADMIN_SEED disabled")
		return nil
	}
	username := strings.TrimSpace(cfg.Username)
	password := cfg.Password
	if username == "" {
		return fmt.Errorf("admin seed: empty username")
	}
	if password == "" {
		return fmt.Errorf("admin seed: empty password")
	}

	var existingID string
	err := db.QueryRow(`SELECT id FROM admin_accounts WHERE username = ?`, username).Scan(&existingID)
	if err == nil {
		slog.Info("admin seed skipped", "username", username, "reason", "already exists")
		return nil
	}
	if err != sql.ErrNoRows {
		return fmt.Errorf("admin seed lookup: %w", err)
	}

	hash, err := hashAdminPassword(password)
	if err != nil {
		return fmt.Errorf("admin seed hash: %w", err)
	}

	id := uuid.NewString()
	now := time.Now().UTC().Format(time.RFC3339Nano)
	display := strings.TrimSpace(cfg.DisplayName)
	var displayArg any
	if display != "" {
		displayArg = display
	}

	_, err = db.Exec(`
		INSERT INTO admin_accounts (id, username, password_hash, display_name, created_at, disabled_at)
		VALUES (?, ?, ?, ?, ?, NULL)
	`, id, username, hash, displayArg, now)
	if err != nil {
		return fmt.Errorf("admin seed insert: %w", err)
	}

	slog.Info("admin account seeded",
		"id", id,
		"username", username,
		"default_password", password == "admin-change-me",
	)
	return nil
}

func hashAdminPassword(password string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func verifyAdminPassword(passwordHash, password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(password)) == nil
}
