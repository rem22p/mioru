-- Drop the fit column from products.  The fit (посадка) field
-- was rarely used and the admin form no longer exposes it.
ALTER TABLE products DROP COLUMN IF EXISTS fit;
