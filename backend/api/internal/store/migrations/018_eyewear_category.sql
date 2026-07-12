-- Add "Eyewear" (очки) as a sub-category under Accessories (id=16),
-- with two leaf sub-categories: Sunglasses + Optical Glasses.
-- Follows the same pattern as 002_seed_categories.sql: explicit IDs
-- so ON CONFLICT (id) DO NOTHING is idempotent.
-- Pinned by TestSeededCategoryTree.
--
-- Irreversible: no down section.
INSERT INTO categories (id, parent_id, name, slug, criteria, sort_order) VALUES
(25, 16, 'Очки',         'eyewear',          '["brand","color"]', 6),
(26, 25, 'Солнцезащитные', 'sunglasses',       '[]', 1),
(27, 25, 'Оптические',    'optical-glasses',  '[]', 2)
ON CONFLICT (id) DO NOTHING;
