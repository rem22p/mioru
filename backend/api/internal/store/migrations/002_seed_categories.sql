-- Seed the canonical category tree (the storefront/admin taxonomy). This is
-- reference data the app depends on; ON CONFLICT (id) DO NOTHING keeps it
-- idempotent and lets the migration adopt a database that was already seeded by
-- the pre-tern code path. Future taxonomy changes ship as new migrations, not
-- edits to this file. Pinned by TestSeededCategoryTree in the store package.
-- Irreversible: no down section.

INSERT INTO categories (id, parent_id, name, slug, criteria, sort_order) VALUES
(1, NULL, 'Одежда', 'clothing', '["size","brand","color"]', 1),
(2, 1, 'Футболки / поло', 'tshirts-polo', '[]', 1),
(3, 1, 'Шорты', 'shorts', '[]', 2),
(4, 1, 'Худи / зип-худи', 'hoodies-zip', '[]', 3),
(5, 1, 'Свитшоты / свитера', 'sweatshirts-sweaters', '[]', 4),
(6, 1, 'Джинсы', 'jeans', '[]', 5),
(7, 1, 'Штаны', 'pants', '[]', 6),
(8, 1, 'Куртки', 'jackets', '[]', 7),
(9, 1, 'Жилетки', 'vests', '[]', 8),
(10, 1, 'Нижнее бельё', 'underwear', '[]', 9),
(11, NULL, 'Обувь', 'shoes', '["size","brand","model","color"]', 2),
(12, 11, 'Кроссовки', 'sneakers', '[]', 1),
(13, 11, 'Тапки', 'slides', '[]', 2),
(14, 11, 'Ботинки', 'boots', '[]', 3),
(15, NULL, 'Сумки', 'bags', '["brand","color"]', 3),
(16, NULL, 'Аксессуары', 'accessories', '["size","brand"]', 4),
(17, 16, 'Кошельки / кардхолдеры', 'wallets-cardholders', '[]', 1),
(18, 16, 'Ремни', 'belts', '[]', 2),
(19, 16, 'Головные уборы', 'headwear', '[]', 3),
(20, 16, 'Ювелирные украшения', 'jewelry', '[]', 4),
(21, 20, 'Браслеты', 'bracelets', '[]', 1),
(22, 20, 'Подвески', 'pendants', '[]', 2),
(23, 20, 'Кольца', 'rings', '[]', 3),
(24, 16, 'Часы', 'watches', '[]', 5)
ON CONFLICT (id) DO NOTHING;
