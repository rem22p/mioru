-- 016_product_sizes_stock.sql
--
-- Add per-size stock tracking. Previously stock_quantity
-- lived on the products table (one number for the whole
-- SKU) — that doesn't work when a size variant sells out
-- while other sizes are still available.

ALTER TABLE product_sizes
    ADD COLUMN IF NOT EXISTS stock_quantity INT NOT NULL DEFAULT 0;

-- Seed: copy existing product stock to all its sizes equally
-- where the product has stock > 0 and stock_quantity on sizes
-- hasn't been set yet.
UPDATE product_sizes ps
SET stock_quantity = p.stock_quantity
FROM products p
WHERE ps.product_id = p.id
  AND p.stock_quantity > 0
  AND ps.stock_quantity = 0;
