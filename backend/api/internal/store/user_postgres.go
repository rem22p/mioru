package store

import (
	"context"
	"fmt"
	"strings"

	"mioru/internal/model"
)

// ── User operations on PostgreSQL ──

// CreateUser inserts a new user into PostgreSQL.
func (s *PostgresStore) CreateUser(ctx context.Context, u model.User) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO users (username, email, hashed_password, first_name, last_name, display_name, avatar_color, role)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		u.Username, u.Email, u.HashedPW, u.FirstName, u.LastName, u.DisplayName, u.AvatarColor, u.Role,
	)
	if err != nil {
		errStr := err.Error()
		if strings.Contains(errStr, "unique") || strings.Contains(errStr, "duplicate") {
			if strings.Contains(errStr, "email") {
				return fmt.Errorf("email already registered")
			}
			return fmt.Errorf("username already exists")
		}
		return fmt.Errorf("create user: %w", err)
	}
	return nil
}

// GetUser retrieves a user by username.
func (s *PostgresStore) GetUser(ctx context.Context, username string) (*model.User, error) {
	u := &model.User{}
	var createdAt string
	err := s.pool.QueryRow(ctx, `
		SELECT id, username, email, hashed_password, first_name, last_name, display_name, avatar_color, role,
			COALESCE(created_at::text, '') as created_at
		FROM users WHERE username = $1`, username,
	).Scan(&u.ID, &u.Username, &u.Email, &u.HashedPW, &u.FirstName, &u.LastName, &u.DisplayName, &u.AvatarColor, &u.Role, &createdAt)
	if err != nil {
		if strings.Contains(err.Error(), "no rows") {
			return nil, nil
		}
		return nil, fmt.Errorf("get user: %w", err)
	}
	u.CreatedAt = createdAt
	return u, nil
}

// GetUserByEmail retrieves a user by email.
func (s *PostgresStore) GetUserByEmail(ctx context.Context, email string) (*model.User, error) {
	u := &model.User{}
	var createdAt string
	err := s.pool.QueryRow(ctx, `
		SELECT id, username, email, hashed_password, first_name, last_name, display_name, avatar_color, role,
			COALESCE(created_at::text, '') as created_at
		FROM users WHERE email = $1`, email,
	).Scan(&u.ID, &u.Username, &u.Email, &u.HashedPW, &u.FirstName, &u.LastName, &u.DisplayName, &u.AvatarColor, &u.Role, &createdAt)
	if err != nil {
		if strings.Contains(err.Error(), "no rows") {
			return nil, nil
		}
		return nil, fmt.Errorf("get user by email: %w", err)
	}
	u.CreatedAt = createdAt
	return u, nil
}

// UpdateUser updates profile fields for a user identified by username.
// The updates map may contain: display_name, avatar_color, first_name, last_name.
func (s *PostgresStore) UpdateUser(ctx context.Context, username string, updates map[string]string) error {
	setClauses := []string{}
	args := []interface{}{}
	argIdx := 1

	for k, v := range updates {
		switch k {
		case "display_name":
			setClauses = append(setClauses, fmt.Sprintf("display_name = $%d", argIdx))
			args = append(args, v)
			argIdx++
		case "avatar_color":
			setClauses = append(setClauses, fmt.Sprintf("avatar_color = $%d", argIdx))
			args = append(args, v)
			argIdx++
		case "first_name":
			setClauses = append(setClauses, fmt.Sprintf("first_name = $%d", argIdx))
			args = append(args, v)
			argIdx++
		case "last_name":
			setClauses = append(setClauses, fmt.Sprintf("last_name = $%d", argIdx))
			args = append(args, v)
			argIdx++
		}
	}

	if len(setClauses) == 0 {
		return fmt.Errorf("no valid fields to update")
	}

	args = append(args, username)
	query := fmt.Sprintf("UPDATE users SET %s WHERE username = $%d", strings.Join(setClauses, ", "), argIdx)
	tag, err := s.pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("update user: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("user not found: %s", username)
	}
	return nil
}

// UpdatePassword updates the hashed password for a user identified by username.
func (s *PostgresStore) UpdatePassword(ctx context.Context, username, hashedPW string) error {
	tag, err := s.pool.Exec(ctx, `UPDATE users SET hashed_password = $1 WHERE username = $2`, hashedPW, username)
	if err != nil {
		return fmt.Errorf("update password: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("user not found: %s", username)
	}
	return nil
}

// IsUsersTableEmpty returns true if the users table has no rows (used for auto-migration).
func (s *PostgresStore) IsUsersTableEmpty(ctx context.Context) (bool, error) {
	var count int
	if err := s.pool.QueryRow(ctx, `SELECT COUNT(*) FROM users`).Scan(&count); err != nil {
		return false, fmt.Errorf("count users: %w", err)
	}
	return count == 0, nil
}
