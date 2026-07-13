-- Enable pg_trgm extension for fuzzy (trigram) product search.
-- GIN index on products.name supports similarity-based queries
-- that handle typos, partial matches, and transliteration without
-- a language-specific dictionary.  The index is used by:
--   WHERE name % $1                          (trigram match)
--   WHERE similarity(name, $1) > 0.3         (threshold query)
--   ORDER BY similarity(name, $1) DESC       (relevance sort)
CREATE EXTENSION IF NOT EXISTS pg_trgm;
CREATE INDEX IF NOT EXISTS idx_products_name_trgm ON products USING GIN (name gin_trgm_ops);
