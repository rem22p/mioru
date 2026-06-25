package store

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// TelegramMessageRow is one row of `telegram_messages` as the
// admin workspace needs to display it: human-friendly text
// (already MarkdownV2-rendered), chat_id (kept as text because
// Telegram channel IDs are negative strings like "-1001234567890"
// that don't fit in an INT), HTTP outcome, and timing.
//
// `Text` is the *exact* payload the bot tried to send — not a
// prettified version — so when a manager sees a 400 from
// Telegram they can read the raw text and figure out which
// character broke the MarkdownV2 parser.
type TelegramMessageRow struct {
	ID                int64   `json:"id"`
	OrderID           *int64  `json:"order_id,omitempty"`
	ChatID            string  `json:"chat_id"`
	Text              string  `json:"text"`
	ParseMode         string  `json:"parse_mode"`
	Status            string  `json:"status"`
	HTTPStatus        *int32  `json:"http_status,omitempty"`
	Error             string  `json:"error,omitempty"`
	TelegramMessageID *int64  `json:"telegram_message_id,omitempty"`
	DurationMs        *int32  `json:"duration_ms,omitempty"`
	SentAt            string  `json:"sent_at"`
}

// TelegramStats is the rolled-up counters the admin Status card
// shows: how many sends the bot attempted in the last 24h, how
// many succeeded, how many failed, and the timestamp of the
// most recent send (regardless of outcome). The latter is the
// "is the bot still doing anything?" signal — if `last_send_at`
// is days old, the notifier is just not being called at all
// (a different bug from the one this table exists to debug).
type TelegramStats struct {
	Last24hTotal  int    `json:"last_24h_total"`
	Last24hSent   int    `json:"last_24h_sent"`
	Last24hFailed int    `json:"last_24h_failed"`
	LastSendAt    string `json:"last_send_at,omitempty"`
	LastSendError string `json:"last_send_error,omitempty"`
}

// RecordTelegramSend creates a `pending` row before the HTTP
// call goes out and returns the row's id so the caller can
// UPDATE it once the response (success or error) is in. The
// `pending` status is what makes a stuck goroutine visible:
// any row in `pending` for more than a few seconds is a panic
// that the deferred recover() in the handler missed.
func (s *PostgresStore) RecordTelegramSend(ctx context.Context, orderID *int64, chatID, text, parseMode string) (int64, error) {
	var id int64
	err := s.pool.QueryRow(ctx, `
		INSERT INTO telegram_messages
		    (order_id, chat_id, text, parse_mode, status)
		VALUES ($1, $2, $3, $4, 'pending')
		RETURNING id`,
		orderID, chatID, text, parseMode,
	).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("record telegram send: %w", err)
	}
	return id, nil
}

// MarkTelegramSent is the happy-path UPDATE: HTTP 2xx,
// telegram_message_id parsed out of the response. We don't
// touch `sent_at` because the INSERT already set it to now()
// and the call went out within milliseconds of that.
func (s *PostgresStore) MarkTelegramSent(ctx context.Context, id int64, httpStatus int, telegramMsgID *int64, durationMs int) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE telegram_messages
		   SET status = 'sent',
		       http_status = $2,
		       telegram_message_id = $3,
		       duration_ms = $4
		 WHERE id = $1`,
		id, httpStatus, telegramMsgID, durationMs,
	)
	if err != nil {
		return fmt.Errorf("mark telegram sent: %w", err)
	}
	return nil
}

// MarkTelegramFailed is the failure UPDATE: any non-2xx (400
// MarkdownV2, 401 revoked, 403 kicked, 429 rate-limited, or a
// transport-level error from the http client). We keep the
// row in `failed` so the admin workspace can list and click
// through to see the exact description Telegram gave us.
func (s *PostgresStore) MarkTelegramFailed(ctx context.Context, id int64, httpStatus *int32, errMsg string, durationMs int) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE telegram_messages
		   SET status = 'failed',
		       http_status = $2,
		       error = $3,
		       duration_ms = $4
		 WHERE id = $1`,
		id, httpStatus, errMsg, durationMs,
	)
	if err != nil {
		return fmt.Errorf("mark telegram failed: %w", err)
	}
	return nil
}

// ListTelegramMessages returns paginated history, newest first.
// `status` is optional — empty string means "all statuses".
// The 24h stats card is computed by GetTelegramStats, not by
// summing this list, so we don't pay the aggregation cost
// here.
func (s *PostgresStore) ListTelegramMessages(ctx context.Context, page, perPage int, status string) ([]TelegramMessageRow, int, error) {
	if page < 1 {
		page = 1
	}
	if perPage < 1 || perPage > 100 {
		perPage = 20
	}
	offset := (page - 1) * perPage

	// We do total + page in two queries rather than a window
	// function because the per_page cap is 100 and the dataset
	// is small; the planner-friendly two-query form is easier
	// to read and the latency difference at 100 rows is nil.
	var total int
	var countErr error
	if status == "" {
		countErr = s.pool.QueryRow(ctx, `SELECT COUNT(*) FROM telegram_messages`).Scan(&total)
	} else {
		countErr = s.pool.QueryRow(ctx, `SELECT COUNT(*) FROM telegram_messages WHERE status = $1`, status).Scan(&total)
	}
	if countErr != nil {
		return nil, 0, fmt.Errorf("count telegram messages: %w", countErr)
	}

	var rows pgx.Rows
	var err error
	if status == "" {
		rows, err = s.pool.Query(ctx, `
			SELECT id, order_id, chat_id, text, parse_mode, status,
			       http_status, COALESCE(error, ''), telegram_message_id,
			       duration_ms, sent_at::text
			  FROM telegram_messages
			  ORDER BY id DESC
			  LIMIT $1 OFFSET $2`,
			perPage, offset,
		)
	} else {
		rows, err = s.pool.Query(ctx, `
			SELECT id, order_id, chat_id, text, parse_mode, status,
			       http_status, COALESCE(error, ''), telegram_message_id,
			       duration_ms, sent_at::text
			  FROM telegram_messages
			 WHERE status = $1
			  ORDER BY id DESC
			  LIMIT $2 OFFSET $3`,
			status, perPage, offset,
		)
	}
	if err != nil {
		return nil, 0, fmt.Errorf("list telegram messages: %w", err)
	}
	defer rows.Close()

	var out []TelegramMessageRow
	for rows.Next() {
		var r TelegramMessageRow
		if err := rows.Scan(
			&r.ID, &r.OrderID, &r.ChatID, &r.Text, &r.ParseMode, &r.Status,
			&r.HTTPStatus, &r.Error, &r.TelegramMessageID,
			&r.DurationMs, &r.SentAt,
		); err != nil {
			return nil, 0, fmt.Errorf("scan telegram message: %w", err)
		}
		out = append(out, r)
	}
	return out, total, rows.Err()
}

// DeleteTelegramMessagesOlderThan removes rows whose sent_at
// is older than the given number of days. Returns the count
// of deleted rows. Called periodically by the notifier's
// background purge goroutine (default: every 24h, 90-day
// retention) to limit PII accumulation in the debug log.
// S1 (data-minimization, CLAUDE.md priority #2).
func (s *PostgresStore) DeleteTelegramMessagesOlderThan(ctx context.Context, retentionDays int) (int64, error) {
	tag, err := s.pool.Exec(ctx,
		`DELETE FROM telegram_messages WHERE sent_at < NOW() - MAKE_INTERVAL(days => $1)`, retentionDays)
	if err != nil {
		return 0, fmt.Errorf("purge telegram messages: %w", err)
	}
	return tag.RowsAffected(), nil
}

// GetTelegramStats is the Status-card payload. The 24h window
// is the only window the admin UI asks for, so we don't build
// a more flexible "last N hours" aggregator.
func (s *PostgresStore) GetTelegramStats(ctx context.Context) (TelegramStats, error) {
	var st TelegramStats
	row := s.pool.QueryRow(ctx, `
		SELECT
		    COUNT(*) FILTER (WHERE sent_at > now() - interval '24 hours'),
		    COUNT(*) FILTER (WHERE sent_at > now() - interval '24 hours' AND status = 'sent'),
		    COUNT(*) FILTER (WHERE sent_at > now() - interval '24 hours' AND status = 'failed')
		FROM telegram_messages
	`)
	if err := row.Scan(&st.Last24hTotal, &st.Last24hSent, &st.Last24hFailed); err != nil {
		return st, fmt.Errorf("telegram stats 24h: %w", err)
	}
	// Last send — the newest row's sent_at + its error, if any.
	// A sent_at without an error means the last send succeeded;
	// an error means the last send failed. If we have a recent
	// sent_at but no error, "everything is fine" and the UI
	// shows a green check.
	var lastAt *string
	var lastErr *string
	row = s.pool.QueryRow(ctx, `
		SELECT sent_at::text, error
		  FROM telegram_messages
		 ORDER BY id DESC
		 LIMIT 1`)
	if err := row.Scan(&lastAt, &lastErr); err != nil {
		if !errors.Is(err, pgx.ErrNoRows) {
			return st, fmt.Errorf("telegram stats last send: %w", err)
		}
	}
	if lastAt != nil {
		st.LastSendAt = *lastAt
	}
	if lastErr != nil {
		st.LastSendError = *lastErr
	}
	return st, nil
}
