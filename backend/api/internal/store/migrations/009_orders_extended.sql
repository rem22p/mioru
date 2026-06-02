-- 009_orders_extended: checkout data, order items, idempotency.
-- Extends the base orders table (006_orders.sql) with delivery/payment/address
-- fields for both cart orders and individual orders, plus line items and
-- idempotency guard for financial safety.

-- Extend orders with checkout fields
ALTER TABLE orders
    ADD COLUMN IF NOT EXISTS type             TEXT NOT NULL DEFAULT 'cart',
    ADD COLUMN IF NOT EXISTS city             TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS delivery_method  TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS payment_method   TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS street           TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS house            TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS apartment        TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS comment          TEXT NOT NULL DEFAULT '';

-- Individual order fields (nullable — only set when type='individual')
ALTER TABLE orders
    ADD COLUMN IF NOT EXISTS height           REAL,
    ADD COLUMN IF NOT EXISTS weight           REAL,
    ADD COLUMN IF NOT EXISTS delivery_time    TEXT[] NOT NULL DEFAULT '{}',
    ADD COLUMN IF NOT EXISTS photos           TEXT[] NOT NULL DEFAULT '{}';

-- Order line items (one row per product in the order)
CREATE TABLE IF NOT EXISTS order_items (
    id            BIGSERIAL PRIMARY KEY,
    order_id      BIGINT NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    product_id    BIGINT NOT NULL REFERENCES products(id),
    size_label    TEXT   NOT NULL DEFAULT '',
    quantity      INT    NOT NULL DEFAULT 1,
    price_minor   BIGINT NOT NULL DEFAULT 0
);

CREATE INDEX IF NOT EXISTS idx_order_items_order_id ON order_items(order_id);

-- Idempotency guard — prevents double-charge on network retry.
CREATE TABLE IF NOT EXISTS order_idempotency (
    key           TEXT PRIMARY KEY,
    order_id      BIGINT,
    request_hash  TEXT NOT NULL,
    status        INT  NOT NULL,
    response_body TEXT,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at    TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_order_idempotency_expires ON order_idempotency(expires_at);
