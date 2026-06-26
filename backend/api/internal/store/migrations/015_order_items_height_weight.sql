-- 015_order_items_height_weight.sql
--
-- Add height_cm and weight_kg to order_items for preorder
-- items where the customer specifies their measurements
-- instead of a standard size.

ALTER TABLE order_items
    ADD COLUMN IF NOT EXISTS height_cm INT;

ALTER TABLE order_items
    ADD COLUMN IF NOT EXISTS weight_kg INT;
