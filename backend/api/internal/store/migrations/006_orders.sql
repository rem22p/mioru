-- 006_orders: customer order history.
-- total_minor is in minor currency units (kopecks/cents).
-- FK defaults to RESTRICT — orders preserved when customer deleted.

CREATE TABLE IF NOT EXISTS orders (
    id          BIGSERIAL PRIMARY KEY,
    customer_id BIGINT NOT NULL REFERENCES customers(id),
    total_minor BIGINT NOT NULL DEFAULT 0,
    status      TEXT NOT NULL DEFAULT 'pending',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_orders_customer_id ON orders(customer_id);
