-- geo.db schema (architecture §6.3) — applied at process start via migrate().
CREATE TABLE IF NOT EXISTS store_settings (
  id            TEXT PRIMARY KEY,
  name          TEXT NOT NULL DEFAULT 'Gas Tam Đệ',
  lat           REAL NOT NULL,
  lng           REAL NOT NULL,
  max_radius_km REAL NOT NULL DEFAULT 10,
  address_text  TEXT,
  updated_at    TEXT NOT NULL,
  updated_by    TEXT
);

CREATE TABLE IF NOT EXISTS geocode_cache (
  id          TEXT PRIMARY KEY,
  query_hash  TEXT NOT NULL UNIQUE,
  result_json TEXT NOT NULL,
  expires_at  TEXT NOT NULL
);
