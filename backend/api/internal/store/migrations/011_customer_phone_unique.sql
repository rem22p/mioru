-- 011_customer_phone_unique: partial unique index on non-empty phone numbers.
-- Two customers may both have NULL/empty phone (default), but two customers
-- with the same non-empty phone are rejected at the DB level.

CREATE UNIQUE INDEX IF NOT EXISTS idx_customers_phone_unique
    ON customers (phone)
    WHERE phone IS NOT NULL AND phone != '';
