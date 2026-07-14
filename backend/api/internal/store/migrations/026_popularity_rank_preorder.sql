-- Add separate popularity rank for preorder products.
-- popularity_rank = in_stock ranking, popularity_rank_preorder = preorder ranking.
-- When sort=popular and status filter is set, the backend picks the right column.
ALTER TABLE products ADD COLUMN IF NOT EXISTS popularity_rank_preorder INTEGER;
