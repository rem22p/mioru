-- 010_product_sizes_quantity: per-size stock quantity
ALTER TABLE product_sizes ADD COLUMN IF NOT EXISTS quantity INTEGER NOT NULL DEFAULT 1;
