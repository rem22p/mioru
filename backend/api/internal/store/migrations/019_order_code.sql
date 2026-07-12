-- Add a human-friendly, unique order code to every order.
-- Format: two uppercase latin letters + hyphen + three digits (e.g. AB-017, KX-429).
-- This replaces the raw numeric order id exposed on the storefront profile,
-- which leaked the total number of orders and looked unpolished.
--
-- The code is generated server-side at order creation.  Collisions are
-- guarded by a partial unique index (excluding the empty-string default
-- for legacy rows) and a retry loop in generateOrderCode that catches
-- 23505 (duplicate key) and regenerates.
--
-- Codespace: 26² × 1,000 = 676,000 unique codes.  Existing orders keep
-- the empty string — only new orders receive a code.  A future backfill
-- migration can populate legacy orders if desired.
ALTER TABLE orders ADD COLUMN order_code TEXT NOT NULL DEFAULT '';
CREATE UNIQUE INDEX orders_order_code_unique ON orders (order_code) WHERE order_code != '';
