-- Add popularity_rank column for manual product ordering.
-- NULL means not ranked (falls to end of list).  Managers set
-- ranks via the admin "Sort" workspace.  The storefront uses
-- ?sort=popular which orders by popularity_rank ASC NULLS LAST.
ALTER TABLE products ADD COLUMN IF NOT EXISTS popularity_rank INTEGER;
