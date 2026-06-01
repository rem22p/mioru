-- 008_category_perf.sql — indexes for hot /api/categories cover_image query
--
-- /api/categories uses a correlated subquery:
--   SELECT pi.url FROM products p JOIN product_images pi ...
--   WHERE p.category_id = c.id OR p.category_id IN (children)
--   ORDER BY p.stock_quantity DESC, pi.sort_order LIMIT 1
--
-- Without indexes this seq-scans products and product_images on every
-- request (the hottest public endpoint — every visitor hits the homepage).
-- These two indexes turn the subquery into index scans + limited heap fetches.

CREATE INDEX IF NOT EXISTS idx_products_category_id ON products(category_id);
CREATE INDEX IF NOT EXISTS idx_product_images_product_id ON product_images(product_id);
