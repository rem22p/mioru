-- 016_product_sizes_stock.sql
--
-- Add per-size stock tracking. Previously stock_quantity
-- lived on the products table (one number for the whole
-- SKU) — that doesn't work when a size variant sells out
-- while other sizes are still available.
--
-- All sizes start at 0. Admin must set per-size stock
-- manually via the product form. The old products.stock_quantity
-- is intentionally NOT copied — copying K to N sizes would
-- inflate available stock by N× (priority #1 violation).

ALTER TABLE product_sizes
    ADD COLUMN IF NOT EXISTS stock_quantity INT NOT NULL DEFAULT 0;
