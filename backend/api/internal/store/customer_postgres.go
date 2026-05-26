package store

import (
	"context"
	"fmt"
	"strings"
	"time"

	"mioru/internal/model"
)

// ── Customer operations on PostgreSQL ──

// CreateCustomer inserts a new customer into PostgreSQL.
func (s *PostgresStore) CreateCustomer(ctx context.Context, c model.Customer) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO customers (email, hashed_password, first_name, last_name, phone, avatar_color)
		VALUES ($1, $2, $3, $4, $5, $6)`,
		c.Email, c.HashedPW, c.FirstName, c.LastName, c.Phone, c.AvatarColor,
	)
	if err != nil {
		errStr := err.Error()
		if strings.Contains(errStr, "unique") || strings.Contains(errStr, "duplicate") {
			return fmt.Errorf("email already registered")
		}
		return fmt.Errorf("create customer: %w", err)
	}
	return nil
}

// GetCustomer retrieves a customer by ID.
func (s *PostgresStore) GetCustomer(ctx context.Context, id int64) (*model.Customer, error) {
	c := &model.Customer{}
	var createdAt, updatedAt string
	err := s.pool.QueryRow(ctx, `
		SELECT id, email, hashed_password, first_name, last_name,
			COALESCE(phone, '') as phone,
			COALESCE(avatar_color, '') as avatar_color,
			COALESCE(created_at::text, '') as created_at,
			COALESCE(updated_at::text, '') as updated_at
		FROM customers WHERE id = $1`, id,
	).Scan(&c.ID, &c.Email, &c.HashedPW, &c.FirstName, &c.LastName,
		&c.Phone, &c.AvatarColor, &createdAt, &updatedAt)
	if err != nil {
		if strings.Contains(err.Error(), "no rows") {
			return nil, nil
		}
		return nil, fmt.Errorf("get customer: %w", err)
	}
	c.CreatedAt = createdAt
	c.UpdatedAt = updatedAt
	return c, nil
}

// GetCustomerByEmail retrieves a customer by email.
func (s *PostgresStore) GetCustomerByEmail(ctx context.Context, email string) (*model.Customer, error) {
	c := &model.Customer{}
	var createdAt, updatedAt string
	err := s.pool.QueryRow(ctx, `
		SELECT id, email, hashed_password, first_name, last_name,
			COALESCE(phone, '') as phone,
			COALESCE(avatar_color, '') as avatar_color,
			COALESCE(created_at::text, '') as created_at,
			COALESCE(updated_at::text, '') as updated_at
		FROM customers WHERE email = $1`, email,
	).Scan(&c.ID, &c.Email, &c.HashedPW, &c.FirstName, &c.LastName,
		&c.Phone, &c.AvatarColor, &createdAt, &updatedAt)
	if err != nil {
		if strings.Contains(err.Error(), "no rows") {
			return nil, nil
		}
		return nil, fmt.Errorf("get customer by email: %w", err)
	}
	c.CreatedAt = createdAt
	c.UpdatedAt = updatedAt
	return c, nil
}

// UpdateCustomer updates profile fields for a customer.
func (s *PostgresStore) UpdateCustomer(ctx context.Context, id int64, updates map[string]string) error {
	setClauses := []string{}
	args := []interface{}{}
	argIdx := 1

	for k, v := range updates {
		switch k {
		case "first_name", "last_name", "phone", "avatar_color":
			setClauses = append(setClauses, fmt.Sprintf("%s = $%d", k, argIdx))
			args = append(args, v)
			argIdx++
		}
	}

	if len(setClauses) == 0 {
		return fmt.Errorf("no valid fields to update")
	}

	args = append(args, id)
	query := fmt.Sprintf("UPDATE customers SET %s, updated_at = NOW() WHERE id = $%d",
		strings.Join(setClauses, ", "), argIdx)
	tag, err := s.pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("update customer: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("customer not found: %d", id)
	}
	return nil
}

// UpdateCustomerPassword updates the hashed password for a customer and stamps
// password_changed_at = NOW() in the same statement, the session-revocation epoch
// that invalidates every token issued before the change (see UpdatePassword).
func (s *PostgresStore) UpdateCustomerPassword(ctx context.Context, id int64, hashedPW string) error {
	tag, err := s.pool.Exec(ctx,
		`UPDATE customers SET hashed_password = $1, password_changed_at = NOW(), updated_at = NOW() WHERE id = $2`,
		hashedPW, id)
	if err != nil {
		return fmt.Errorf("update customer password: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("customer not found: %d", id)
	}
	return nil
}

// CustomerPasswordChangedAt returns the customer's password_changed_at (the
// session epoch). ok is false when no such customer exists.
func (s *PostgresStore) CustomerPasswordChangedAt(ctx context.Context, id int64) (time.Time, bool, error) {
	var changedAt time.Time
	err := s.pool.QueryRow(ctx,
		`SELECT password_changed_at FROM customers WHERE id = $1`, id,
	).Scan(&changedAt)
	if err != nil {
		if strings.Contains(err.Error(), "no rows") {
			return time.Time{}, false, nil
		}
		return time.Time{}, false, fmt.Errorf("customer password_changed_at: %w", err)
	}
	return changedAt, true, nil
}
