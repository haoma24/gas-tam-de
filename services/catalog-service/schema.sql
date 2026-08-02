-- catalog.db schema (architecture §6.2) — applied at process start via migrate().
-- T2.1.2: products + price history (PRD "product_prices" ≡ product_price_history).
-- sale_price is VND integer; active 1=selling / 0=hidden (soft delete for admin).
CREATE TABLE IF NOT EXISTS products (
  id          TEXT PRIMARY KEY,
  sku         TEXT NOT NULL UNIQUE,
  name        TEXT NOT NULL,
  description TEXT,
  unit        TEXT NOT NULL DEFAULT 'binh',
  sale_price  INTEGER NOT NULL CHECK(sale_price >= 0),  -- VND
  active      INTEGER NOT NULL DEFAULT 1 CHECK(active IN (0, 1)),
  image_url   TEXT,
  created_at  TEXT NOT NULL,                            -- RFC3339 UTC
  updated_at  TEXT NOT NULL                             -- RFC3339 UTC
);
CREATE INDEX IF NOT EXISTS idx_products_active ON products(active, created_at DESC);

-- Price change audit (PRD product_prices). Append-only; current price lives on products.sale_price.
CREATE TABLE IF NOT EXISTS product_price_history (
  id          TEXT PRIMARY KEY,
  product_id  TEXT NOT NULL REFERENCES products(id),
  sale_price  INTEGER NOT NULL CHECK(sale_price >= 0),  -- VND at change time
  changed_at  TEXT NOT NULL,                            -- RFC3339 UTC
  changed_by  TEXT                                      -- admin user id (X-User-Id), optional
);
CREATE INDEX IF NOT EXISTS idx_price_history_product ON product_price_history(product_id, changed_at);
