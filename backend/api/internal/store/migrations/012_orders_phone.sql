-- 012_orders_phone.sql
--
-- Add a `phone` column to `orders` so that the phone number the
-- customer typed at checkout is preserved with the order itself,
-- independent of the customer's profile row.
--
-- The customer is still asked for their phone at every checkout
-- (and can one-click-fill it from their profile). Whatever they
-- type is what we record — even if the profile is later edited.
--
-- Existing rows are backfilled from `customers.phone` for orders
-- that belong to a registered customer. The default for new
-- inserts is the empty string so the column is NOT NULL without
-- forcing a destructive rewrite of legacy data.

ALTER TABLE orders
    ADD COLUMN IF NOT EXISTS phone TEXT NOT NULL DEFAULT '';

-- Backfill: copy the customer's phone into every existing order
-- that doesn't have one yet. The JOIN is INNER to keep the
-- behavior simple — orphaned orders (deleted customer) keep
-- the empty default.
UPDATE orders o
SET phone = c.phone
FROM customers c
WHERE o.customer_id = c.id
  AND o.phone = ''
  AND c.phone <> '';
