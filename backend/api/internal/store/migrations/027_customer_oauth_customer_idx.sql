-- The Telegram order gate put customer_oauth on the checkout path: POST /api/store/orders,
-- GET /me and login all read it by customer_id, and the table's only index was
-- (provider, oauth_id) — Postgres does not index the FK on its own, so every one of
-- those reads was a sequential scan.
CREATE INDEX IF NOT EXISTS idx_customer_oauth_customer ON customer_oauth (customer_id);
