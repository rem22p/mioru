-- Make hashed_password and email nullable so OAuth customers (who authenticate
-- through a provider and may not have a password or email) can coexist with
-- password-based customers. The UNIQUE constraint on email is preserved: multiple
-- NULLs are allowed in PostgreSQL.
ALTER TABLE customers ALTER COLUMN hashed_password DROP NOT NULL;
ALTER TABLE customers ALTER COLUMN email DROP NOT NULL;

-- Separate table for OAuth provider links. One customer can link multiple
-- providers; the UNIQUE(provider, oauth_id) constraint prevents duplicate
-- bindings. access_token / refresh_token / token_expiry are intentionally absent
-- — Telegram does not use OAuth2 tokens; they will be added in a future
-- migration when Instagram OAuth lands.
CREATE TABLE IF NOT EXISTS customer_oauth (
    id           SERIAL PRIMARY KEY,
    customer_id  INTEGER NOT NULL REFERENCES customers(id) ON DELETE CASCADE,
    provider     TEXT NOT NULL,
    oauth_id     TEXT NOT NULL,
    profile_data JSONB NOT NULL DEFAULT '{}',
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(provider, oauth_id)
);

CREATE INDEX IF NOT EXISTS idx_customer_oauth_lookup
    ON customer_oauth(provider, oauth_id);
