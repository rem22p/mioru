-- 028_brands_array.sql
-- KAN-14: brands become a structured TEXT[] so collaborations (stored as a
-- single legacy string like "Bape x Mastermind") are two separate brands.
-- Filtering by either brand matches the product, and facets list brands
-- individually. The display name is derived in SQL:
-- array_to_string(brands, ' x ').
--
-- B2 (PR #84 review): this migration only ADDS the new column, backfills it
-- and indexes it. DROP COLUMN brand ships as a SEPARATE migration in the next
-- release — after 028 alone the previous binary still works (the legacy column
-- is intact), so rolling back the release costs a binary rollback, not a DB
-- restore.

ALTER TABLE products ADD COLUMN IF NOT EXISTS brands TEXT[] NOT NULL DEFAULT '{}';

-- Backfill from the legacy single-string brand column. The admin used " x "
-- (spaces around x) as the collaboration separator.
-- The split is one-way (the source column is dropped in a later migration),
-- so it normalises as it goes rather than preserving whatever the legacy
-- free-text field held:
-- \s+x\s+ absorbs padded separators ("Bape  x  Mastermind"), btrim strips the
-- leading/trailing space a hand-typed value carries, and empty elements — left
-- behind by an edge separator ("Bape x ") or by an empty brand — are dropped.
-- Without this an element like 'Bape ' becomes its own facet chip that the
-- overlap filter (element equality) can never match.
UPDATE products
   SET brands = ARRAY(
       SELECT btrim(v)
         FROM unnest(regexp_split_to_array(brand, '\s+x\s+')) AS v
        WHERE btrim(v) <> ''
   );

-- Facet query does unnest(brands); the overlap filter uses brands && $n.
CREATE INDEX IF NOT EXISTS idx_products_brands_gin ON products USING GIN (brands);
