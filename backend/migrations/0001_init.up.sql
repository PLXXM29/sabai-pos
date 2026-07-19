-- ============================================================================
-- 0001_init — MiniMart POS initial schema
-- Money is stored as INTEGER satang (1 baht = 100 satang). NEVER float.
-- Every business table carries store_id (multi-tenant ready, single-tenant now).
-- Client-originated rows carry client_uuid as an idempotency key for sync.
--
-- NOTE: no explicit BEGIN/COMMIT — PostgreSQL runs a multi-statement migration
-- file as a single implicit transaction, so this file applies atomically.
-- ============================================================================

-- gen_random_uuid() is built into PostgreSQL 13+ (pgcrypto not required).

-- ── shared helpers ──────────────────────────────────────────────────────────

-- keep updated_at fresh on UPDATE
CREATE OR REPLACE FUNCTION set_updated_at() RETURNS trigger AS $$
BEGIN
  NEW.updated_at := now();
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- append-only guard: block UPDATE/DELETE on immutable ledger tables
CREATE OR REPLACE FUNCTION forbid_mutation() RETURNS trigger AS $$
BEGIN
  RAISE EXCEPTION 'table % is append-only (% not allowed)', TG_TABLE_NAME, TG_OP;
END;
$$ LANGUAGE plpgsql;

-- ── stores (tenant root) ────────────────────────────────────────────────────
CREATE TABLE stores (
  id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  name       TEXT NOT NULL,
  config     JSONB NOT NULL DEFAULT '{}'::jsonb,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE TRIGGER trg_stores_updated BEFORE UPDATE ON stores
  FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- ── users ───────────────────────────────────────────────────────────────────
CREATE TABLE users (
  id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  store_id      UUID NOT NULL REFERENCES stores(id) ON DELETE CASCADE,
  username      TEXT NOT NULL,
  password_hash TEXT NOT NULL,
  role          TEXT NOT NULL CHECK (role IN ('superadmin', 'manager', 'cashier')),
  pin_hash      TEXT,            -- optional hashed PIN for fast cashier switch
  is_active     BOOLEAN NOT NULL DEFAULT TRUE,
  created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (store_id, username)
);
CREATE TRIGGER trg_users_updated BEFORE UPDATE ON users
  FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- ── products ────────────────────────────────────────────────────────────────
CREATE TABLE products (
  id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  store_id   UUID NOT NULL REFERENCES stores(id) ON DELETE CASCADE,
  name       TEXT NOT NULL,
  barcode    TEXT,
  category   TEXT NOT NULL DEFAULT '',
  cost_price BIGINT NOT NULL DEFAULT 0 CHECK (cost_price >= 0),  -- satang
  sell_price BIGINT NOT NULL CHECK (sell_price >= 0),            -- satang
  is_active  BOOLEAN NOT NULL DEFAULT TRUE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE TRIGGER trg_products_updated BEFORE UPDATE ON products
  FOR EACH ROW EXECUTE FUNCTION set_updated_at();
CREATE INDEX idx_products_store ON products (store_id);
-- barcode unique per store when present
CREATE UNIQUE INDEX uq_products_store_barcode ON products (store_id, barcode)
  WHERE barcode IS NOT NULL AND barcode <> '';

-- ── inventory (fast on-hand cache; source of truth is stock_movements) ───────
CREATE TABLE inventory (
  product_id    UUID PRIMARY KEY REFERENCES products(id) ON DELETE CASCADE,
  store_id      UUID NOT NULL REFERENCES stores(id) ON DELETE CASCADE,
  qty_on_hand   INTEGER NOT NULL DEFAULT 0,
  reorder_point INTEGER NOT NULL DEFAULT 0,
  updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE TRIGGER trg_inventory_updated BEFORE UPDATE ON inventory
  FOR EACH ROW EXECUTE FUNCTION set_updated_at();
CREATE INDEX idx_inventory_store ON inventory (store_id);

-- ── stock_movements (immutable, append-only ledger) ──────────────────────────
CREATE TABLE stock_movements (
  id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  store_id    UUID NOT NULL REFERENCES stores(id) ON DELETE CASCADE,
  product_id  UUID NOT NULL REFERENCES products(id),
  type        TEXT NOT NULL CHECK (type IN ('sale', 'receive', 'adjust', 'void')),
  qty_delta   INTEGER NOT NULL,          -- +receive / -sale
  ref_type    TEXT,                      -- e.g. 'bill'
  ref_id      UUID,
  reason      TEXT,
  created_by  UUID REFERENCES users(id),
  client_uuid UUID,                      -- idempotency for client-originated ops
  created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_movements_store_product ON stock_movements (store_id, product_id, created_at);
CREATE UNIQUE INDEX uq_movements_client_uuid ON stock_movements (store_id, client_uuid)
  WHERE client_uuid IS NOT NULL;
CREATE TRIGGER trg_movements_append_only
  BEFORE UPDATE OR DELETE ON stock_movements
  FOR EACH ROW EXECUTE FUNCTION forbid_mutation();

-- ── bill_counters (gap-free bill numbers per store, assigned server-side) ─────
CREATE TABLE bill_counters (
  store_id UUID PRIMARY KEY REFERENCES stores(id) ON DELETE CASCADE,
  next_seq BIGINT NOT NULL DEFAULT 1
);

-- ── bills (immutable financial document) ─────────────────────────────────────
CREATE TABLE bills (
  id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  store_id          UUID NOT NULL REFERENCES stores(id) ON DELETE CASCADE,
  bill_no           TEXT NOT NULL,             -- gap-free per store
  client_uuid       UUID NOT NULL,             -- idempotency key from client
  cashier_id        UUID REFERENCES users(id),
  subtotal          BIGINT NOT NULL CHECK (subtotal >= 0),   -- satang
  discount          BIGINT NOT NULL DEFAULT 0 CHECK (discount >= 0),
  total             BIGINT NOT NULL CHECK (total >= 0),
  paid              BIGINT NOT NULL DEFAULT 0 CHECK (paid >= 0),
  change            BIGINT NOT NULL DEFAULT 0 CHECK (change >= 0),
  payment_method    TEXT NOT NULL CHECK (payment_method IN ('cash', 'transfer')),
  status            TEXT NOT NULL DEFAULT 'completed' CHECK (status IN ('completed', 'void')),
  voids_bill_id     UUID REFERENCES bills(id), -- if this bill voids/refunds another
  created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
  synced_at         TIMESTAMPTZ,
  UNIQUE (store_id, bill_no),
  UNIQUE (store_id, client_uuid)
);
CREATE INDEX idx_bills_store_created ON bills (store_id, created_at);
CREATE INDEX idx_bills_store_status ON bills (store_id, status);

-- ── bill_items (append-only, price/name snapshot at sale time) ────────────────
CREATE TABLE bill_items (
  id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  bill_id        UUID NOT NULL REFERENCES bills(id) ON DELETE CASCADE,
  product_id     UUID REFERENCES products(id),
  name_snapshot  TEXT NOT NULL,
  price_snapshot BIGINT NOT NULL CHECK (price_snapshot >= 0),  -- satang, frozen at sale
  qty            INTEGER NOT NULL CHECK (qty > 0),
  line_total     BIGINT NOT NULL CHECK (line_total >= 0)
);
CREATE INDEX idx_bill_items_bill ON bill_items (bill_id);
CREATE TRIGGER trg_bill_items_append_only
  BEFORE UPDATE OR DELETE ON bill_items
  FOR EACH ROW EXECUTE FUNCTION forbid_mutation();

-- ── sync_log ─────────────────────────────────────────────────────────────────
CREATE TABLE sync_log (
  id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  store_id   UUID NOT NULL REFERENCES stores(id) ON DELETE CASCADE,
  device_id  TEXT,
  ops_count  INTEGER NOT NULL DEFAULT 0,
  conflicts  INTEGER NOT NULL DEFAULT 0,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_sync_log_store ON sync_log (store_id, created_at);

-- ── audit_log (who did what to money/stock/price) ────────────────────────────
CREATE TABLE audit_log (
  id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  store_id   UUID NOT NULL REFERENCES stores(id) ON DELETE CASCADE,
  actor_id   UUID REFERENCES users(id),
  action     TEXT NOT NULL,
  entity     TEXT,
  entity_id  UUID,
  detail     JSONB NOT NULL DEFAULT '{}'::jsonb,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_audit_store_created ON audit_log (store_id, created_at);
