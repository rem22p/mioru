-- 030_drop_brand_column.sql
-- Issue #86 (B2 follow-up): one full release cycle has passed since 028
-- added the structured brands[] column and backfilled it — the legacy
-- single-string brand column is dead (no code reads or writes it) and can
-- now be dropped. NOTE: after this migration, binaries older than PR #84
-- cannot read products (their SELECT references p.brand) — rollback is a
-- DB restore, not a binary rollback.

ALTER TABLE products DROP COLUMN IF EXISTS brand;
