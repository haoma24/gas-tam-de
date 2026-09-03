-- auth.db schema (architecture §6.1) — applied at process start via migrate().
CREATE TABLE IF NOT EXISTS users (
  id             TEXT PRIMARY KEY,
  phone_e164_enc BLOB NOT NULL,
  phone_hash     TEXT NOT NULL UNIQUE,
  phone_masked   TEXT NOT NULL,
  google_sub     TEXT,
  email          TEXT,
  picture_url    TEXT,
  contact_phone_e164_enc BLOB,
  contact_phone_masked   TEXT,
  full_name      TEXT,
  created_at     TEXT NOT NULL,
  updated_at     TEXT NOT NULL
);

-- OTP challenges (T1.1.5): store peppered SHA-256 of OTP only — never plaintext.
-- expires_at is RFC3339Nano UTC; verify rejects when now >= expires_at.
CREATE TABLE IF NOT EXISTS otp_challenges (
  id          TEXT PRIMARY KEY,
  phone_hash  TEXT NOT NULL,
  code_hash   TEXT NOT NULL,              -- hash(pepper:challenge_id:code); never raw OTP
  expires_at  TEXT NOT NULL,              -- RFC3339Nano UTC
  attempts    INTEGER NOT NULL DEFAULT 0,
  consumed_at TEXT,
  created_at  TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_otp_phone ON otp_challenges(phone_hash, created_at);
CREATE INDEX IF NOT EXISTS idx_otp_expires ON otp_challenges(expires_at);

CREATE TABLE IF NOT EXISTS sessions (
  id           TEXT PRIMARY KEY,
  user_id      TEXT NOT NULL,
  role         TEXT NOT NULL CHECK(role IN ('customer','admin')),
  refresh_hash TEXT NOT NULL,
  expires_at   TEXT NOT NULL,
  persistent   INTEGER NOT NULL DEFAULT 0,
  revoked_at   TEXT,
  created_at   TEXT NOT NULL
);

-- Admin accounts (T1.2.1): password_hash is bcrypt; seeded at process start when missing.
-- Credentials via ADMIN_USERNAME|ADMIN_EMAIL + ADMIN_PASSWORD (never commit secrets).
CREATE TABLE IF NOT EXISTS admin_accounts (
  id            TEXT PRIMARY KEY,
  username      TEXT NOT NULL UNIQUE,
  password_hash TEXT NOT NULL,             -- bcrypt (cost DefaultCost)
  display_name  TEXT,
  created_at    TEXT NOT NULL,
  disabled_at   TEXT
);

-- Phones that receive role=admin after a normal customer OTP login (T1.2.4).
-- Keyed by the same peppered hash as users.phone_hash so the plaintext number
-- is never stored; phone_masked is kept only so the admin UI can list entries.
-- Bootstrapped from ADMIN_PHONES, then managed from the admin screen.
CREATE TABLE IF NOT EXISTS admin_phones (
  id           TEXT PRIMARY KEY,
  phone_hash   TEXT NOT NULL UNIQUE,
  phone_masked TEXT NOT NULL,
  label        TEXT,
  created_at   TEXT NOT NULL,
  created_by   TEXT
);

CREATE TABLE IF NOT EXISTS audit_logs (
  id          TEXT PRIMARY KEY,
  actor_id    TEXT NOT NULL,
  action      TEXT NOT NULL,
  entity_type TEXT,
  entity_id   TEXT,
  meta_json   TEXT,
  created_at  TEXT NOT NULL
);
