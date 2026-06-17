-- 014_order_items_product_meta.sql
--
-- Adds product_name and product_slug to order_items so the
-- Telegram "new order" notification can render a clickable
-- "<a href="...">Product Name</a>" link even if the product
-- gets renamed or deleted later. Without this migration the
-- notifier falls back to "Товар #{id}" (which is correct but
-- unhelpful when a manager skims the chat in the middle of
-- a busy day).
--
-- The columns are nullable: legacy rows predating 014 just
-- have NULL here, and the Go-side fallback
-- (`linkText := "Товар #" + id` when ProductName is empty)
-- keeps older messages rendering correctly. New rows are
-- populated inside the CreateOrder transaction, which
-- guarantees name and slug are written atomically with
-- the order itself.
ALTER TABLE order_items
  ADD COLUMN IF NOT EXISTS product_name TEXT NOT NULL DEFAULT '',
  ADD COLUMN IF NOT EXISTS product_slug TEXT NOT NULL DEFAULT '';
