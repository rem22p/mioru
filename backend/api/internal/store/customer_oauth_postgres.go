package store

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"mioru/internal/model"
)

// ErrOAuthAlreadyLinked is returned by CreateCustomerWithOAuth when the
// OAuth provider+ID pair is already associated with another customer.
var ErrOAuthAlreadyLinked = errors.New("telegram account already linked to another user")

// ── Customer OAuth operations on PostgreSQL ──

// GetCustomerByOAuth looks up a customer by OAuth provider and provider-side
// user ID. Returns both the customer and the oauth record; both are nil when
// no link exists (not found, not an error).
func (s *PostgresStore) GetCustomerByOAuth(ctx context.Context, provider, oauthID string) (*model.Customer, *model.CustomerOAuth, error) {
	c := &model.Customer{}
	oa := &model.CustomerOAuth{}
	var createdAt, updatedAt, oaCreatedAt string

	err := s.pool.QueryRow(ctx, `
		SELECT
			c.id, COALESCE(c.email, '') as email, COALESCE(c.hashed_password, '') as hashed_password,
			COALESCE(c.first_name, '') as first_name, COALESCE(c.last_name, '') as last_name,
			COALESCE(c.phone, '') as phone, COALESCE(c.avatar_color, '') as avatar_color,
			COALESCE(c.created_at::text, '') as created_at, COALESCE(c.updated_at::text, '') as updated_at,
			oa.id as oauth_id_pk, oa.customer_id, oa.provider, oa.oauth_id,
			COALESCE(oa.profile_data::text, '{}') as profile_data,
			COALESCE(oa.created_at::text, '') as oauth_created_at
		FROM customer_oauth oa
		JOIN customers c ON c.id = oa.customer_id
		WHERE oa.provider = $1 AND oa.oauth_id = $2`,
		provider, oauthID,
	).Scan(
		&c.ID, &c.Email, &c.HashedPW, &c.FirstName, &c.LastName,
		&c.Phone, &c.AvatarColor, &createdAt, &updatedAt,
		&oa.ID, &oa.CustomerID, &oa.Provider, &oa.OAuthID,
		&oa.ProfileData, &oaCreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil, nil
		}
		return nil, nil, fmt.Errorf("get customer by oauth: %w", err)
	}
	c.CreatedAt = createdAt
	c.UpdatedAt = updatedAt
	oa.CreatedAt = oaCreatedAt
	return c, oa, nil
}

// CreateCustomerWithOAuth inserts a new customer and an OAuth link in a single
// transaction. The customer row may have empty email and password (OAuth-only).
// password_changed_at is set to NOW() so the session-revocation check in the
// auth middleware does not trip on a NULL timestamp.
func (s *PostgresStore) CreateCustomerWithOAuth(ctx context.Context, c model.Customer, oa model.CustomerOAuth) error {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	var customerID int64
	err = tx.QueryRow(ctx, `
		INSERT INTO customers (email, hashed_password, first_name, last_name, phone, avatar_color, password_changed_at)
		VALUES (NULLIF($1, ''), NULLIF($2, ''), $3, $4, $5, $6, NOW())
		RETURNING id`,
		c.Email, c.HashedPW, c.FirstName, c.LastName, c.Phone, c.AvatarColor,
	).Scan(&customerID)
	if err != nil {
		return fmt.Errorf("insert customer: %w", err)
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO customer_oauth (customer_id, provider, oauth_id, profile_data)
		VALUES ($1, $2, $3, $4::jsonb)`,
		customerID, oa.Provider, oa.OAuthID, oa.ProfileData,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return ErrOAuthAlreadyLinked
		}
		return fmt.Errorf("insert customer_oauth: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit: %w", err)
	}
	return nil
}

// LinkOAuth attaches an OAuth provider to an existing customer. The insert is
// idempotent: ON CONFLICT DO NOTHING makes repeated calls harmless.
func (s *PostgresStore) LinkOAuth(ctx context.Context, customerID int64, oa model.CustomerOAuth) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO customer_oauth (customer_id, provider, oauth_id, profile_data)
		VALUES ($1, $2, $3, $4::jsonb)
		ON CONFLICT (provider, oauth_id) DO NOTHING`,
		customerID, oa.Provider, oa.OAuthID, oa.ProfileData,
	)
	if err != nil {
		return fmt.Errorf("link oauth: %w", err)
	}
	return nil
}
