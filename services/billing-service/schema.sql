-- billing.db schema (architecture §6.6)
CREATE TABLE IF NOT EXISTS payments (
  id           TEXT PRIMARY KEY,
  order_id     TEXT NOT NULL UNIQUE,
  payment_type TEXT NOT NULL CHECK(payment_type IN ('FULL','PARTIAL','UNPAID')),
  amount_due   INTEGER NOT NULL,
  amount_paid  INTEGER NOT NULL,
  recorded_at  TEXT NOT NULL,
  recorded_by  TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS debts (
  customer_key TEXT PRIMARY KEY,
  phone_masked TEXT NOT NULL,
  balance      INTEGER NOT NULL DEFAULT 0,
  updated_at   TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS debt_ledger (
  id            TEXT PRIMARY KEY,
  customer_key  TEXT NOT NULL,
  order_id      TEXT,
  delta         INTEGER NOT NULL,
  balance_after INTEGER NOT NULL,
  note          TEXT,
  created_at    TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS processed_events (
  event_id     TEXT PRIMARY KEY,
  processed_at TEXT NOT NULL
);
