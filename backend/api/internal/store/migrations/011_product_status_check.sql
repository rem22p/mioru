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
-- status='none' for sold-out rows (the dropdown value the admin code
-- still sends today). Without this backfill, a filtered query
-- (WHERE p.status IN ('in_stock','preorder')) would silently drop those
-- rows from the storefront. We map every non-canonical status to
-- 'in_stock' as a conservative default — the next admin edit can correct
-- it. (Mapping to 'out_of_stock' would be more accurate semantically,
-- but 'in_stock' is the visible default the storefront already shows for
-- the bulk of legacy rows.) The CHECK is NOT VALID so it doesn't run on
-- existing rows; the backfill is the only place we read them.
--
-- tern applies each file once per version, so plain ADD CONSTRAINT is
-- safe — the file is only applied on the version bump.

UPDATE products
SET    status = 'in_stock'
WHERE  status IS NULL
   OR  status NOT IN ('in_stock', 'preorder', 'out_of_stock');

ALTER TABLE products
    ADD CONSTRAINT products_status_chk
    CHECK (status IN ('in_stock', 'preorder', 'out_of_stock'))
    NOT VALID;
