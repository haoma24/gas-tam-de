-- order.db schema (architecture §6.4) — applied at process start via migrate().
-- T4.1.1: delivery_fee_settings (singleton toggle) + delivery_fee_rules (distance bands).
-- T4.1.2: GET/PUT /v1/admin/delivery-fee. T4.1.3: fee engine on place order (quote = T4.2.1).

CREATE TABLE IF NOT EXISTS delivery_fee_settings (
  id         TEXT PRIMARY KEY,                              -- singleton row 'default'
  enabled    INTEGER NOT NULL DEFAULT 0 CHECK(enabled IN (0, 1)),
  updated_at TEXT NOT NULL                                  -- RFC3339 UTC
);

-- Distance bands: min_km inclusive, max_km exclusive (NULL = +inf). fee_vnd is VND integer.
CREATE TABLE IF NOT EXISTS delivery_fee_rules (
  id         TEXT PRIMARY KEY,
  min_km     REAL NOT NULL CHECK(min_km >= 0),
  max_km     REAL CHECK(max_km IS NULL OR max_km > min_km),
  fee_vnd    INTEGER NOT NULL CHECK(fee_vnd >= 0),
  sort_order INTEGER NOT NULL DEFAULT 0,
  active     INTEGER NOT NULL DEFAULT 1 CHECK(active IN (0, 1))
);
CREATE INDEX IF NOT EXISTS idx_delivery_fee_rules_active
  ON delivery_fee_rules(active, sort_order);

CREATE TABLE IF NOT EXISTS orders (
  id            TEXT PRIMARY KEY,
  user_id       TEXT NOT NULL,
  customer_name TEXT NOT NULL,
  phone_hash    TEXT NOT NULL,
  phone_masked  TEXT NOT NULL,
  address_text  TEXT NOT NULL,
  lat           REAL NOT NULL,
  lng           REAL NOT NULL,
  distance_km   REAL NOT NULL,
  delivery_fee  INTEGER NOT NULL CHECK(delivery_fee >= 0),
  subtotal      INTEGER NOT NULL CHECK(subtotal >= 0),
  total         INTEGER NOT NULL CHECK(total >= 0),
  status        TEXT NOT NULL CHECK(status IN ('PENDING','COMPLETED','CANCELLED')),
  created_at    TEXT NOT NULL,
  completed_at  TEXT,
  cancelled_at  TEXT,
  -- Payment snapshot at complete (T6.1.1); billing.db mirror is T6.1.2.
  payment_type  TEXT CHECK(payment_type IS NULL OR payment_type IN ('FULL','PARTIAL','UNPAID')),
  amount_paid   INTEGER CHECK(amount_paid IS NULL OR amount_paid >= 0)
);
CREATE INDEX IF NOT EXISTS idx_orders_admin_fifo ON orders(status, created_at);
CREATE INDEX IF NOT EXISTS idx_orders_user ON orders(user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_orders_created ON orders(created_at);

-- Admin Order Desk UX settings (wait badge thresholds + TTS interval).
CREATE TABLE IF NOT EXISTS desk_settings (
  id                   TEXT PRIMARY KEY,                              -- singleton 'default'
  wait_blue_max_min    INTEGER NOT NULL DEFAULT 5 CHECK(wait_blue_max_min > 0),
  wait_orange_max_min  INTEGER NOT NULL DEFAULT 15 CHECK(wait_orange_max_min > wait_blue_max_min),
  wait_red_max_min     INTEGER NOT NULL DEFAULT 30 CHECK(wait_red_max_min > wait_orange_max_min),
  alert_enabled        INTEGER NOT NULL DEFAULT 1 CHECK(alert_enabled IN (0, 1)),
  alert_interval_sec   INTEGER NOT NULL DEFAULT 300 CHECK(alert_interval_sec >= 30),
  updated_at           TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS order_items (
  id           TEXT PRIMARY KEY,
  order_id     TEXT NOT NULL,
  product_id   TEXT NOT NULL,
  product_sku  TEXT NOT NULL,
  product_name TEXT NOT NULL,
  unit_price   INTEGER NOT NULL CHECK(unit_price >= 0),
  qty          INTEGER NOT NULL CHECK(qty >= 1),
  line_total   INTEGER NOT NULL CHECK(line_total >= 0)
);
CREATE INDEX IF NOT EXISTS idx_order_items_order ON order_items(order_id);
CREATE INDEX IF NOT EXISTS idx_order_items_product ON order_items(product_id);

CREATE TABLE IF NOT EXISTS processed_events (
  event_id     TEXT PRIMARY KEY,
  processed_at TEXT NOT NULL
);
