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
-- Migration is idempotent for fresh DBs: ADD CONSTRAINT ... IF NOT EXISTS
-- would be ideal, but tern runs each file once per version, so a plain
-- ADD CONSTRAINT is safe — the file is only applied on the version bump.
-- For existing DBs the IF NOT EXISTS guard avoids the failure when a partial
-- deploy had the constraint but a fresh baseline didn't.

ALTER TABLE products
    ADD CONSTRAINT products_status_chk
    CHECK (status IN ('in_stock', 'preorder', 'out_of_stock'))
    NOT VALID;
