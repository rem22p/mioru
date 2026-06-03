-- 011_customer_phone_unique: partial unique index on non-empty phone numbers.
-- Two customers may both have NULL/empty phone (default), but two customers
-- with the same non-empty phone are rejected at the DB level.
--
-- Clean up any pre-existing duplicates: keep the oldest customer per phone,
-- nullify phone on the rest (they're test accounts / data entry noise).

DO $$
DECLARE
    rec RECORD;
BEGIN
    FOR rec IN
        SELECT phone, array_agg(id ORDER BY created_at) AS ids
          FROM customers
         WHERE phone IS NOT NULL AND phone != ''
         GROUP BY phone
        HAVING COUNT(*) > 1
    LOOP
        -- Keep the first (oldest) customer's phone, clear the rest
        UPDATE customers
           SET phone = ''
         WHERE id = ANY(rec.ids[2:]);
    END LOOP;
END $$;

CREATE UNIQUE INDEX IF NOT EXISTS idx_customers_phone_unique
    ON customers (phone)
    WHERE phone IS NOT NULL AND phone != '';
