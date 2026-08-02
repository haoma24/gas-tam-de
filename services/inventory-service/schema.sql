-- inventory.db schema (architecture §6.5) — applied at process start via migrate().
-- T7.1.1: stock levels + movements + cost; processed_events for future JetStream consumers.
-- on_hand may go negative (MVP: allow + inventory.low_stock); cost_price is current VND nhập.
CREATE TABLE IF NOT EXISTS stock_items (
  product_id    TEXT PRIMARY KEY,              -- catalog product id
  sku           TEXT NOT NULL UNIQUE,
  name          TEXT NOT NULL,                 -- denormalized from catalog
  on_hand       INTEGER NOT NULL DEFAULT 0,    -- may be negative (MVP)
  cost_price    INTEGER NOT NULL DEFAULT 0 CHECK(cost_price >= 0),  -- VND giá nhập hiện tại
  reorder_level INTEGER NOT NULL DEFAULT 0 CHECK(reorder_level >= 0),
  updated_at    TEXT NOT NULL                   -- RFC3339 UTC
);

-- Append-only ledger. qty always > 0; direction via movement_type (IN/OUT/ADJUST).
-- T7.2.1: OUT (admin xuất + order.completed bán) always persists unit_cost = cost_price
-- at movement time (COGS snapshot). Later IN/ADJUST that change cost_price must NOT rewrite
-- historical OUT rows. IN requires unit_cost at API layer; ADJUST may leave unit_cost NULL.
CREATE TABLE IF NOT EXISTS stock_movements (
  id            TEXT PRIMARY KEY,
  product_id    TEXT NOT NULL,
  movement_type TEXT NOT NULL CHECK(movement_type IN ('IN','OUT','ADJUST')),
  qty           INTEGER NOT NULL CHECK(qty > 0),
  unit_cost     INTEGER CHECK(unit_cost IS NULL OR unit_cost >= 0),  -- VND; required for IN/OUT (app); NULL OK for some ADJUST
  note          TEXT,
  ref_type      TEXT,                          -- ORDER / MANUAL
  ref_id        TEXT,
  created_at    TEXT NOT NULL,                 -- RFC3339 UTC
  created_by    TEXT                           -- admin user id, optional
);
CREATE INDEX IF NOT EXISTS idx_movements_product ON stock_movements(product_id, created_at);

-- Idempotency for JetStream durable inventory-order-completed (order.completed → OUT).
CREATE TABLE IF NOT EXISTS processed_events (
  event_id     TEXT PRIMARY KEY,
  processed_at TEXT NOT NULL                   -- RFC3339 UTC
);
