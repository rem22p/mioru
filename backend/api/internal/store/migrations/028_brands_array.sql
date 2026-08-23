-- 028_brands_array.sql
-- KAN-14: brands become a structured TEXT[] so collaborations (stored as a
-- single legacy string like "Bape x Mastermind") are two separate brands.
-- Filtering by either brand matches the product, and facets list brands
-- individually. The display name is derived in SQL:
-- array_to_string(brands, ' x ').

ALTER TABLE products ADD COLUMN IF NOT EXISTS brands TEXT[] NOT NULL DEFAULT '{}';

-- Backfill from the legacy single-string brand column. The admin used " x "
-- (spaces around x) as the collaboration separator.
-- array_remove drops the empty elements string_to_array leaves behind when the
-- separator sits at an edge ("Bape x " splits into {Bape,""}); an empty brand
-- would otherwise reach the facet list and the overlap filter as a real value.
UPDATE products
   SET brands = array_remove(CASE
       WHEN brand LIKE '% x %' THEN string_to_array(brand, ' x ')
       WHEN brand <> '' THEN ARRAY[brand]
       ELSE '{}'
   END, '');

ALTER TABLE products DROP COLUMN IF EXISTS brand;

-- Facet query does unnest(brands); the overlap filter uses brands && $n.
CREATE INDEX IF NOT EXISTS idx_products_brands_gin ON products USING GIN (brands);
