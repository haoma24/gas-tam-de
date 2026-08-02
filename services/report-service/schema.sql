-- report.db schema (architecture §6.7)
-- profit_vnd = revenue_vnd - cogs_vnd (T7.2.2); delivery_fee_vnd tracked separately.
CREATE TABLE IF NOT EXISTS daily_stats (
  day               TEXT PRIMARY KEY,           -- YYYY-MM-DD (timezone VN)
  revenue_vnd       INTEGER NOT NULL DEFAULT 0, -- product subtotal (sale lines)
  cogs_vnd          INTEGER NOT NULL DEFAULT 0, -- Σ(qty×unit_cost) OUT/ORDER snapshots
  delivery_fee_vnd  INTEGER NOT NULL DEFAULT 0, -- not included in profit (MVP)
  orders_completed  INTEGER NOT NULL DEFAULT 0,
  orders_placed     INTEGER NOT NULL DEFAULT 0,
  profit_vnd        INTEGER NOT NULL DEFAULT 0  -- revenue_vnd - cogs_vnd (T7.2.2)
);

CREATE TABLE IF NOT EXISTS dashboard_snapshot (
  id            TEXT PRIMARY KEY,           -- 'current'
  revenue_today INTEGER NOT NULL,
  revenue_month INTEGER NOT NULL,
  debt_total    INTEGER NOT NULL,            -- SUM outstanding balances (T8.1.2)
  profit_month  INTEGER NOT NULL,
  updated_at    TEXT NOT NULL
);

-- Per-customer debt read-model for dashboard_snapshot.debt_total (T8.1.2).
-- Absolute balance from billing.debt.updated (not a delta).
CREATE TABLE IF NOT EXISTS customer_debt_balances (
  customer_key TEXT PRIMARY KEY,
  balance      INTEGER NOT NULL DEFAULT 0,
  updated_at   TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS processed_events (
  event_id     TEXT PRIMARY KEY,
  processed_at TEXT NOT NULL
);
