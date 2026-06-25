-- 013_telegram_messages.sql
--
-- Adds a `telegram_messages` log table so the admin workspace
-- can show *every* Telegram send attempt: which chat, which
-- text, what status, and — when the API returned a real
-- message_id — what the bot actually posted. This is the
-- "why don't orders reach the manager chat?" debugging
-- surface: an admin opens the Telegram workspace, sees
-- the last 24h summary, clicks any row, and reads the
-- exact error (Telegram API description) or the full
-- payload the bot tried to send.
--
-- Schema notes
-- ------------
--  * `order_id` is ON DELETE SET NULL because the message
--    log is audit data and must survive the order being
--    deleted. We don't want a one-line DELETE order to
--    also wipe the trail of every Telegram we sent about
--    it.
--  * `status` is a CHECK-constrained TEXT instead of a
--    Postgres enum because future status values (rate-
--    limited, retried, …) are easier to add without an
--    ALTER TYPE in production. The constraint keeps
--    today's set honest.
--  * `http_status` is nullable because a row is INSERTed
--    with status='pending' *before* the HTTP call goes
--    out; the status code only lands on UPDATE. A row
--    stuck in 'pending' for more than a few seconds
--    signals a goroutine crash.
--  * `text` is TEXT (not varchar(N)) because MarkdownV2
--    order notifications can run to ~2 KB; we don't want
--    to truncate or hit a length cap. If it ever matters
--    we can switch to TOAST compression later.
--  * Indexes are on (sent_at DESC) and (status) because
--    the admin UI lists "last N" sorted by sent_at and
--    filters by status; both queries must stay snappy
--    after months of accumulated rows.

CREATE TABLE IF NOT EXISTS telegram_messages (
    id BIGSERIAL PRIMARY KEY,
    order_id BIGINT REFERENCES orders(id) ON DELETE SET NULL,
    chat_id TEXT NOT NULL,
    text TEXT NOT NULL,
    parse_mode TEXT NOT NULL DEFAULT 'MarkdownV2',
    status TEXT NOT NULL CHECK (status IN ('pending','sent','failed')),
    http_status INT,
    error TEXT,
    telegram_message_id BIGINT,
    duration_ms INT,
    sent_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS telegram_messages_sent_at_idx
    ON telegram_messages (sent_at DESC);
CREATE INDEX IF NOT EXISTS telegram_messages_status_idx
    ON telegram_messages (status);
CREATE INDEX IF NOT EXISTS telegram_messages_order_id_idx
    ON telegram_messages (order_id) WHERE order_id IS NOT NULL;
