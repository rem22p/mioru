-- 017_order_items_measurements.sql
--
-- Replace height_cm / weight_kg with a flexible JSONB
-- measurements column for preorder items. Different
-- categories need different measurements (height+weight
-- for apparel, foot_length for shoes, etc.).

ALTER TABLE order_items
    ADD COLUMN IF NOT EXISTS measurements JSONB;

-- Migrate existing data
UPDATE order_items
SET measurements = jsonb_build_object('height', height_cm, 'weight', weight_kg)
WHERE height_cm IS NOT NULL AND weight_kg IS NOT NULL
  AND measurements IS NULL;
