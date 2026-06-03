-- 011_order_idempotency_user_id: add user_id column to order_idempotency.
-- Needed so idempotency keys are scoped per user — two different customers
-- with the same Idempotency-Key UUID must not collide.

ALTER TABLE order_idempotency
    ADD COLUMN IF NOT EXISTS user_id BIGINT;

-- Populate existing rows: derive user_id from the linked order's customer_id.
-- This is a best-effort backfill; rows where order_id is NULL or the order
-- was deleted get NULL (acceptable — those are expired or orphaned records).
UPDATE order_idempotency oi
   SET user_id = o.customer_id
  FROM orders o
 WHERE oi.order_id = o.id
   AND oi.user_id IS NULL;

-- From this point forward user_id is mandatory.
ALTER TABLE order_idempotency
    ALTER COLUMN user_id SET NOT NULL;

-- The primary key becomes (key, user_id) to avoid cross-user collisions.
-- Drop the old PK constraint first (tern doesn't support ALTER CONSTRAINT directly,
-- so we recreate the PK via the composite index + ADD PRIMARY KEY approach).
ALTER TABLE order_idempotency DROP CONSTRAINT IF EXISTS order_idempotency_pkey;

-- Recreate with composite PK.
ALTER TABLE order_idempotency
    ADD PRIMARY KEY (key, user_id);
