-- 011_product_status_check.sql — CHECK constraint on products.status
--
-- The catalog "В наличии / Под заказ" toggle filters by status. Until now
-- the column was unconstrained text — a buggy handler or admin form could
-- insert a typo ("preorderr", "in stock", "") and the storefront filter
-- would silently treat it as "everything" (empty string = no filter).
--
-- This constraint makes the allowed set a single source of truth at the DB
-- level: in_stock | preorder | out_of_stock. The handler also validates
-- before insert, so the wire format is consistent; this constraint is the
-- last line of defence for any path that bypasses the handler.
--
-- Backfill BEFORE the constraint is added: the legacy admin SPA wrote
-- non-canonical values that the storefront filter (`WHERE p.status = $n`
-- in product_postgres.go) would silently drop. We translate each legacy
-- value to its semantic equivalent, matching the live mapping in
-- handler/product_form.go:normalizeProductStatus — same function, same
-- input → output rules, so data on disk after the migration is
-- consistent with what the admin SPA would have written if the
-- constraint had existed all along.
--
--   pre_order (legacy underscore)  →  preorder       ("made to order")
--   none       (legacy sold-out)   →  out_of_stock   ("not available")
--   ''         (empty)             →  out_of_stock   (safe default;
--                                                      product_form.go
--                                                      inStock=false branch)
--   NULL       (if any)            →  out_of_stock   (safe default)
--   <other>    (typo / future)     →  out_of_stock   (safe default)
--
-- tern applies each file once per version, so plain ADD CONSTRAINT is
-- safe — the file is only applied on the version bump.

UPDATE products
SET    status = 'preorder'
WHERE  status = 'pre_order';

UPDATE products
SET    status = 'out_of_stock'
WHERE  status = 'none';

-- Catch-all for NULLs, empty strings, and any other non-canonical value
-- (typos, future legacy values that pre-date this constraint).
UPDATE products
SET    status = 'out_of_stock'
WHERE  status IS NULL
   OR  status = ''
   OR  status NOT IN ('in_stock', 'preorder', 'out_of_stock');

ALTER TABLE products
    ADD CONSTRAINT products_status_chk
    CHECK (status IN ('in_stock', 'preorder', 'out_of_stock'))
    NOT VALID;
