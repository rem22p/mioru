-- 029_order_category_foot_length.sql
-- KAN-52: the custom-order form gains a mandatory category choice
-- (clothing | shoes | accessories). Shoes carry the insole length.
-- Old orders keep NULL category / NULL foot_length and render as before.
ALTER TABLE orders ADD COLUMN IF NOT EXISTS category TEXT;
ALTER TABLE orders ADD COLUMN IF NOT EXISTS foot_length NUMERIC;

ALTER TABLE orders ADD CONSTRAINT orders_category_check
    CHECK (category IS NULL OR category IN ('clothing', 'shoes', 'accessories'));
ALTER TABLE orders ADD CONSTRAINT orders_foot_length_check
    CHECK (foot_length IS NULL OR (foot_length >= 10 AND foot_length <= 40));
