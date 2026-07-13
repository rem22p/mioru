-- Fix category criteria:
-- 1. Add "color" to Accessories (id=16) so all subcategories
--    (wallets, belts, headwear, jewelry, watches, eyewear)
--    show the color field in the admin product form.
-- 2. Remove "model" from Shoes (id=11) — the model field is
--    being deprecated entirely (see below).
UPDATE categories SET criteria = '["size","brand","color"]'      WHERE id = 16;  -- Аксессуары
UPDATE categories SET criteria = '["size","brand","color"]'      WHERE id = 11;  -- Обувь (был model)

-- Drop the model column from products.  It was only ever shown for
-- shoes and was empty for 99/163 products on prod.  The frontend
-- no longer renders or sends it.
ALTER TABLE products DROP COLUMN IF EXISTS model;
