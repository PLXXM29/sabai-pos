-- Payment intents for auto-confirming PromptPay transfers. A phone forwards the
-- bank's "money received" notification to /payments/notify; we match it to a
-- pending intent by amount and mark it paid.
CREATE TABLE payments (
  id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  store_id         UUID NOT NULL REFERENCES stores(id) ON DELETE CASCADE,
  bill_client_uuid UUID,                                   -- links to the sale
  amount           BIGINT NOT NULL CHECK (amount > 0),     -- satang, expected
  status           TEXT NOT NULL DEFAULT 'pending'
                     CHECK (status IN ('pending', 'paid', 'expired', 'cancelled')),
  ref              TEXT,
  raw_note         TEXT,
  created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
  paid_at          TIMESTAMPTZ,
  expires_at       TIMESTAMPTZ NOT NULL
);
CREATE INDEX idx_payments_match ON payments (status, amount, created_at);
CREATE INDEX idx_payments_store ON payments (store_id, created_at);
