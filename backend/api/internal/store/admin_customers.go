package store

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
)

// AdminCustomerRow is the projection used by the admin "Customers"
// workspace list. It deliberately drops hashed_password — the
// password hash must never leave the server (the bcrypt is the only
// thing that proves ownership of the account, so leaking it
// effectively compromises the customer's session).
//
// The order_aggregates sub-select keeps the query to a single
// round-trip even when the result set spans thousands of rows.
// Counting + SUM happen once per customer, not per order, because
// we GROUP BY customer_id before joining back to customers.
type AdminCustomerRow struct {
	ID              int64  `json:"id"`
	Email           string `json:"email"`
	FirstName       string `json:"first_name"`
	LastName        string `json:"last_name"`
	Phone           string `json:"phone"`
	AvatarColor     string `json:"avatar_color"`
	CreatedAt       string `json:"created_at"`
	OrdersCount     int    `json:"orders_count"`
	TotalSpentMinor int64  `json:"total_spent_minor"`
	LastOrderAt     string `json:"last_order_at"` // empty string if no orders

	// Telegram link, if the customer has linked a Telegram account
	// via /api/store/customers/me/oauth. Username may be empty
	// even when Linked=true (the user might not have a public
	// @handle) — chat_id is always present in that case.
	TelegramLinked   bool   `json:"telegram_linked"`
	TelegramUsername string `json:"telegram_username"`
	TelegramChatID   string `json:"telegram_chat_id"`
}

// ListCustomers returns a paginated list of customers with rolled-up
// order stats and the Telegram link, if any.
//
// search is a free-text filter that matches email/first_name/last_name
// **by substring** case-insensitively. Empty string means "no filter".
// The search term is bounded (maxSearchLen) and wildcard-wrapped so a
// manager searching for "an" finds "Anna", "Ivan", "Dan".
//
// Sort is fixed to id DESC (newest customers first) — the
// storefront doesn't expose a sort parameter for this endpoint
// because the only sensible order is "newest first", and adding a
// sort picker would just be UI tax. If a manager asks for "oldest
// first" or "highest spender", add a sort param then.
//
// page is clamped to maxPage (defence-in-depth) to prevent an
// attacker from sending page=99999999 and forcing a seq-scan
// OFFSET into nothing.
func (s *PostgresStore) ListCustomers(ctx context.Context, search string, page, perPage int) ([]AdminCustomerRow, int, error) {
	const maxSearchLen = 200
	const maxPage = 1000

	if len(search) > maxSearchLen {
		search = search[:maxSearchLen]
	}
	if page < 1 {
		page = 1
	}
	if page > maxPage {
		page = maxPage
	}
	if perPage < 1 {
		perPage = 20
	}
	if perPage > 100 {
		perPage = 100
	}

	// Build a case-insensitive substring pattern for the WHERE
	// clause. An empty search string short-circuits via $1 = ''
	// without touching the LIKE branches; a non-empty search is
	// lower-cased and wrapped in % wildcards so a manager
	// searching for "анн" finds "Анна".
	searchPattern := ""
	if search != "" {
		searchPattern = "%" + strings.ToLower(search) + "%"
	}

	// Count first so the handler can build the pagination footer
	// without a second round trip to compute total.
	var total int
	err := s.pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM customers c
		WHERE $1 = ''
		   OR LOWER(c.email) LIKE $1
		   OR LOWER(c.first_name) LIKE $1
		   OR LOWER(c.last_name) LIKE $1`,
		searchPattern,
	).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("count customers: %w", err)
	}

	offset := (page - 1) * perPage

	rows, err := s.pool.Query(ctx, `
		SELECT c.id, c.email, c.first_name, c.last_name, c.phone, c.avatar_color,
		       c.created_at::text,
		       COALESCE(ord.orders_count, 0) AS orders_count,
		       COALESCE(ord.total_spent_minor, 0) AS total_spent_minor,
		       COALESCE(ord.last_order_at::text, '') AS last_order_at,
		       (tg.oauth_id IS NOT NULL) AS telegram_linked,
		       COALESCE(tg.telegram_username, '') AS telegram_username,
		       COALESCE(tg.oauth_id, '') AS telegram_chat_id
		FROM customers c
		LEFT JOIN (
			SELECT customer_id,
			       COUNT(*) AS orders_count,
			       SUM(total_minor) AS total_spent_minor,
			       MAX(created_at) AS last_order_at
			FROM orders
			GROUP BY customer_id
		) ord ON ord.customer_id = c.id
		LEFT JOIN (
			SELECT customer_id, oauth_id,
			       profile_data->>'username' AS telegram_username
			FROM customer_oauth
			WHERE provider = 'telegram'
		) tg ON tg.customer_id = c.id
		WHERE $1 = ''
		   OR LOWER(c.email) LIKE $1
		   OR LOWER(c.first_name) LIKE $1
		   OR LOWER(c.last_name) LIKE $1
		ORDER BY c.id DESC
		LIMIT $2 OFFSET $3`,
		searchPattern, perPage, offset,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("list customers: %w", err)
	}
	defer rows.Close()

	var out []AdminCustomerRow
	for rows.Next() {
		var r AdminCustomerRow
		if err := rows.Scan(
			&r.ID, &r.Email, &r.FirstName, &r.LastName, &r.Phone, &r.AvatarColor,
			&r.CreatedAt,
			&r.OrdersCount, &r.TotalSpentMinor, &r.LastOrderAt,
			&r.TelegramLinked, &r.TelegramUsername, &r.TelegramChatID,
		); err != nil {
			return nil, 0, fmt.Errorf("scan customer: %w", err)
		}
		out = append(out, r)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate customers: %w", err)
	}
	return out, total, nil
}

// AdminCustomerDetail is the projection used by the admin
// "Customer detail" page. Same security boundary as AdminCustomerRow
// (no HashedPW) plus the timestamps needed to render the profile
// header, the first/last order dates for the stats card, and the
// full list of orders.
type AdminCustomerDetail struct {
	ID                int64  `json:"id"`
	Email             string `json:"email"`
	FirstName         string `json:"first_name"`
	LastName          string `json:"last_name"`
	Phone             string `json:"phone"`
	AvatarColor       string `json:"avatar_color"`
	CreatedAt         string `json:"created_at"`
	UpdatedAt         string `json:"updated_at"`
	PasswordChangedAt string `json:"-"`
	OrdersCount       int    `json:"orders_count"`
	TotalSpentMinor   int64  `json:"total_spent_minor"`
	FirstOrderAt      string `json:"first_order_at"`
	LastOrderAt       string `json:"last_order_at"`
	TelegramLinked    bool   `json:"telegram_linked"`
	TelegramUsername  string `json:"telegram_username"`
	TelegramChatID    string `json:"telegram_chat_id"`
	Orders            []AdminOrderSummary `json:"orders"`
}

// AdminOrderSummary is a per-order row included in the customer
// detail response. We keep it lean — phone, items, photos are
// already available in the dedicated /api/admin/orders endpoint
// and the manager can click through to the full order card.
type AdminOrderSummary struct {
	ID         int64  `json:"id"`
	Type       string `json:"type"`
	TotalMinor int64  `json:"total_minor"`
	Status     string `json:"status"`
	CreatedAt  string `json:"created_at"`
	ItemsCount int    `json:"items_count"`
}

// ⚠️ LinkCustomerTelegramForTest wires a customer_oauth row for
// integration tests ONLY. It bypasses the Telegram bot signature
// verification, CSRF token, and rate limiting that the production
// route (/api/store/customers/me/oauth) enforces. CALLING THIS
// FROM A PRODUCTION REQUEST PATH WOULD LET ANY AUTHENTICATED USER
// CLAIM ANY TELEGRAM IDENTITY — IT IS A SECURITY FOOT-GUN.
//
// The method is kept exported (not unexported) because the
// integration test lives in package handler_test and needs access
// through *store.PostgresStore. The "ForTest" suffix is the
// contract that this is test-only.
//
// Re-review PR #52 finding MEDIUM #4 — originally considered
// moving behind //go:build e2e, but the CI runs go test ./...
// without -tags e2e and the handler_test caller has no build tag.
func (s *PostgresStore) LinkCustomerTelegramForTest(ctx context.Context, customerID int64, chatID, username string) error {
	profileData := []byte(`{"username": "` + username + `"}`)
	_, err := s.pool.Exec(ctx, `
		INSERT INTO customer_oauth (customer_id, provider, oauth_id, profile_data)
		VALUES ($1, 'telegram', $2, $3)`,
		customerID, chatID, profileData)
	return err
}

// GetCustomerFullDetail returns the customer profile, rolled-up
// stats, Telegram link, and the customer's order history (most
// recent first). The orders list is intentionally *not* paginated
// here — a typical mioru customer has <20 orders over the lifetime
// of the shop, and showing them all on one page is what a manager
// wants when investigating "did this customer ever complain about
// late shipping?". If a single customer ever accumulates hundreds
// of orders we can add pagination then.
func (s *PostgresStore) GetCustomerFullDetail(ctx context.Context, id int64) (*AdminCustomerDetail, error) {
	d := &AdminCustomerDetail{}
	err := s.pool.QueryRow(ctx, `
		SELECT c.id, c.email, c.first_name, c.last_name, c.phone,
		       COALESCE(c.avatar_color, '') as avatar_color,
		       c.created_at::text, c.updated_at::text, c.password_changed_at::text,
		       (SELECT COUNT(*) FROM orders WHERE customer_id = c.id),
		       (SELECT COALESCE(SUM(total_minor), 0) FROM orders WHERE customer_id = c.id),
		       COALESCE((SELECT MIN(created_at)::text FROM orders WHERE customer_id = c.id), ''),
		       COALESCE((SELECT MAX(created_at)::text FROM orders WHERE customer_id = c.id), ''),
		       COALESCE((SELECT (oauth_id IS NOT NULL) FROM customer_oauth
		                 WHERE customer_id = c.id AND provider = 'telegram' LIMIT 1), FALSE),
		       COALESCE((SELECT profile_data->>'username' FROM customer_oauth
		                 WHERE customer_id = c.id AND provider = 'telegram' LIMIT 1), ''),
		       COALESCE((SELECT oauth_id FROM customer_oauth
		                 WHERE customer_id = c.id AND provider = 'telegram' LIMIT 1), '')
		FROM customers c
		WHERE c.id = $1`, id,
	).Scan(
		&d.ID, &d.Email, &d.FirstName, &d.LastName, &d.Phone, &d.AvatarColor,
		&d.CreatedAt, &d.UpdatedAt, &d.PasswordChangedAt,
		&d.OrdersCount, &d.TotalSpentMinor, &d.FirstOrderAt, &d.LastOrderAt,
		&d.TelegramLinked, &d.TelegramUsername, &d.TelegramChatID,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("get customer detail: %w", err)
	}

	// Fetch the order list separately. Doing this in a second
	// query keeps the main profile query simple and matches the
	// shape of the customer /api/store/customers/me/orders endpoint
	// (just without pagination — see the type comment above).
	//
	// The items_count is computed in a single LEFT JOIN sub-select
	// (not a per-row correlated subquery) to avoid the N+1 that
	// CLAUDE.md forbids. A typical mioru customer has <20 orders
	// so the join is cheap, but a wholesale buyer with hundreds of
	// orders would have triggered O(n) round-trips in the original
	// `(SELECT COUNT(*) FROM order_items WHERE order_id = o.id)`
	// pattern — re-review finding MEDIUM #3.
	rows, err := s.pool.Query(ctx, `
		SELECT o.id, o.type, o.total_minor, o.status, o.created_at::text,
		       COALESCE(ic.cnt, 0)
		FROM orders o
		LEFT JOIN (
			SELECT order_id, COUNT(*) AS cnt
			FROM order_items
			GROUP BY order_id
		) ic ON ic.order_id = o.id
		WHERE o.customer_id = $1
		ORDER BY o.created_at DESC, o.id DESC`, id,
	)
	if err != nil {
		return nil, fmt.Errorf("list customer orders: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var o AdminOrderSummary
		if err := rows.Scan(&o.ID, &o.Type, &o.TotalMinor, &o.Status, &o.CreatedAt, &o.ItemsCount); err != nil {
			return nil, fmt.Errorf("scan order summary: %w", err)
		}
		d.Orders = append(d.Orders, o)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate orders: %w", err)
	}
	return d, nil
}
