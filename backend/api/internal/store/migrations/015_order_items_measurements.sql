-- 015_order_items_measurements.sql
--
-- Add flexible measurements JSONB column for preorder items.
-- Replaces the earlier approach of separate height_cm/weight_kg
-- columns — different categories need different measurement
-- fields (foot_length, head_circumference, etc.).
--
-- The old height_cm/weight_kg are dropped — they were never
-- populated in production (015 was reverted before deploy).

ALTER TABLE order_items
    ADD COLUMN IF NOT EXISTS measurements JSONB;

-- Clean up dead columns if they exist from earlier migration attempts
ALTER TABLE order_items
    DROP COLUMN IF EXISTS height_cm;

ALTER TABLE order_items
    DROP COLUMN IF EXISTS weight_kg;
