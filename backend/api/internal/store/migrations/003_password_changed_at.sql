-- Add password_changed_at to users and customers to support session revocation
-- on password change. The auth middleware rejects any JWT whose "iat" (issued-at)
-- predates this timestamp, so changing or resetting a password invalidates every
-- token minted before the change. Existing rows adopt NOW() at migration time,
-- which logs out all live sessions exactly once on deploy.
ALTER TABLE users ADD COLUMN IF NOT EXISTS password_changed_at TIMESTAMPTZ NOT NULL DEFAULT NOW();
ALTER TABLE customers ADD COLUMN IF NOT EXISTS password_changed_at TIMESTAMPTZ NOT NULL DEFAULT NOW();
