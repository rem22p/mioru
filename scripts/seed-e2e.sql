-- Deterministic catalog fixture for the storefront E2E + visual-regression
-- suite (apps/store/e2e/*.spec.ts). Unlike scripts/seed-300.sh (random data,
-- unsable for pixel baselines) every value here is fixed, so screenshots are
-- reproducible across runs and environments.
--
-- Required by the specs:
--   * a product with slug `midnight-runner` that has size "42" and is in stock
--     (product/cart/touch-target tests hardcode that slug and click size 42),
--   * enough in-stock, sized products to fill the catalog grid and feed the
--     storefront order journey (user-flow.spec.ts findInStockSlug).
--
-- Categories are already seeded by migration 002_seed_categories.sql; this only
-- touches the product tables. Re-runnable: it resets them first.

BEGIN;

TRUNCATE products, product_sizes RESTART IDENTITY CASCADE;

-- Fixed timestamps keep any "newest first" ordering stable. Higher id = later.
INSERT INTO products
  (id, slug, category_id, brand, name, price, color, material, description,
   xp_reward, in_stock, status, stock_quantity, created_by, created_at, updated_at)
VALUES
  (1,  'midnight-runner',   12, 'Nike',        'Midnight Runner',        7990, 'Чёрный',   'Текстиль/резина', 'Беговые кроссовки для города. Лёгкая подошва, дышащий верх.',        79, 1, 'in_stock', 25, 'e2e', TIMESTAMPTZ '2024-01-01 00:01:00+00', TIMESTAMPTZ '2024-01-01 00:01:00+00'),
  (2,  'urban-tee',          2, 'Adidas',      'Urban Tee',              1990, 'Белый',    'Хлопок',          'Базовая футболка прямого кроя из плотного хлопка.',                 19, 1, 'in_stock', 40, 'e2e', TIMESTAMPTZ '2024-01-01 00:02:00+00', TIMESTAMPTZ '2024-01-01 00:02:00+00'),
  (3,  'classic-polo',       2, 'Lacoste',     'Classic Polo',           3490, 'Синий',    'Пике',            'Поло из пике с фирменным воротником.',                              34, 1, 'in_stock', 30, 'e2e', TIMESTAMPTZ '2024-01-01 00:03:00+00', TIMESTAMPTZ '2024-01-01 00:03:00+00'),
  (4,  'summer-shorts',      3, 'Puma',        'Summer Shorts',          2490, 'Бежевый',  'Хлопок',          'Лёгкие шорты на каждый день с карманами.',                          24, 1, 'in_stock', 35, 'e2e', TIMESTAMPTZ '2024-01-01 00:04:00+00', TIMESTAMPTZ '2024-01-01 00:04:00+00'),
  (5,  'zip-hoodie',         4, 'Champion',    'Zip Hoodie',             4990, 'Серый',    'Футер',           'Зип-худи из плотного футера с начёсом.',                            49, 1, 'in_stock', 28, 'e2e', TIMESTAMPTZ '2024-01-01 00:05:00+00', TIMESTAMPTZ '2024-01-01 00:05:00+00'),
  (6,  'pullover-hoodie',    4, 'Nike',        'Pullover Hoodie',        4590, 'Чёрный',   'Футер',           'Классическое худи-кенгуру с капюшоном на кулиске.',                 45, 1, 'in_stock', 32, 'e2e', TIMESTAMPTZ '2024-01-01 00:06:00+00', TIMESTAMPTZ '2024-01-01 00:06:00+00'),
  (7,  'court-sneaker',     12, 'Reebok',      'Court Sneaker',          6490, 'Белый',    'Кожа',            'Минималистичные кеды в ретро-стиле.',                               64, 1, 'in_stock', 22, 'e2e', TIMESTAMPTZ '2024-01-01 00:07:00+00', TIMESTAMPTZ '2024-01-01 00:07:00+00'),
  (8,  'trail-boot',        14, 'Asics',       'Trail Boot',             8990, 'Коричневый','Нубук',          'Демисезонные ботинки с протектором для трейла.',                    89, 1, 'in_stock', 18, 'e2e', TIMESTAMPTZ '2024-01-01 00:08:00+00', TIMESTAMPTZ '2024-01-01 00:08:00+00'),
  (9,  'crew-sweatshirt',    5, 'New Balance', 'Crew Sweatshirt',        3990, 'Зелёный',  'Футер',           'Свитшот с круглым вырезом и рукавом реглан.',                       39, 1, 'in_stock', 26, 'e2e', TIMESTAMPTZ '2024-01-01 00:09:00+00', TIMESTAMPTZ '2024-01-01 00:09:00+00'),
  (10, 'slim-jeans',         6, 'Vans',        'Slim Jeans',             5490, 'Синий',    'Деним',           'Зауженные джинсы из эластичного денима.',                           54, 1, 'in_stock', 24, 'e2e', TIMESTAMPTZ '2024-01-01 00:10:00+00', TIMESTAMPTZ '2024-01-01 00:10:00+00'),
  (11, 'track-pants',        7, 'Fila',        'Track Pants',            3290, 'Чёрный',   'Полиэстер',       'Спортивные штаны с лампасами и манжетами.',                         32, 1, 'in_stock', 27, 'e2e', TIMESTAMPTZ '2024-01-01 00:11:00+00', TIMESTAMPTZ '2024-01-01 00:11:00+00'),
  (12, 'winter-parka',       8, 'Adidas',      'Winter Parka',          12990, 'Оливковый','Нейлон',          'Утеплённая парка с капюшоном и ветрозащитой.',                     129, 1, 'in_stock', 15, 'e2e', TIMESTAMPTZ '2024-01-01 00:12:00+00', TIMESTAMPTZ '2024-01-01 00:12:00+00');

SELECT setval(pg_get_serial_sequence('products', 'id'), 12, true);

-- Sizes with per-size stock_quantity. midnight-runner MUST have size "42".
INSERT INTO product_sizes (product_id, size_label, stock_quantity) VALUES
  (1,  '41', 5), (1,  '42', 5), (1,  '43', 5),
  (2,  'S',  10), (2,  'M',  10), (2,  'L',  10),
  (3,  'M',  8), (3,  'L',  8), (3,  'XL', 8),
  (4,  'S',  8), (4,  'M',  8), (4,  'L',  8),
  (5,  'M',  6), (5,  'L',  6), (5,  'XL', 6),
  (6,  'S',  8), (6,  'M',  8), (6,  'L',  8),
  (7,  '41', 4), (7,  '42', 4), (7,  '43', 4),
  (8,  '42', 3), (8,  '43', 3), (8,  '44', 3),
  (9,  'M',  5), (9,  'L',  5), (9,  'XL', 5),
  (10, '30/32', 6), (10, '32/32', 6), (10, '34/32', 6),
  (11, 'S', 5), (11, 'M', 5), (11, 'L', 5),
  (12, 'M', 4), (12, 'L', 4), (12, 'XL', 4);

COMMIT;
