-- 017_customer_cart_measurements: persist preorder measurements
-- in the server-side cart so they survive across devices and sessions.
-- cartStore.ts already sends measurements in the cart sync payload but
-- the backend cart schema had no column to store them, silently discarding
-- the data. This closes the gap — see PR #55 review round 29.
ALTER TABLE customer_cart ADD COLUMN IF NOT EXISTS measurements JSONB;
