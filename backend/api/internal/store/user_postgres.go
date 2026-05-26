package store

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"mioru/internal/model"
)

// hashResetToken returns the hex SHA-256 of a raw password-reset token. Only the
// hash is persisted, so a leaked database or backup cannot be used to reset
// passwords — the raw token exists only in the email sent to the user. SHA-256
// (not bcrypt) is the right tool here: the token is 256 bits of CSPRNG output,
// so it is infeasible to brute-force and needs no salt or key-stretching.
func hashResetToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

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

// UpdatePassword updates the hashed password for a user identified by username
// and stamps password_changed_at = NOW() in the same statement. The timestamp is
// the session-revocation epoch: AuthMW rejects any token issued before it, so
// changing a password atomically logs out all of that user's other sessions.
func (s *PostgresStore) UpdatePassword(ctx context.Context, username, hashedPW string) error {
	tag, err := s.pool.Exec(ctx,
		`UPDATE users SET hashed_password = $1, password_changed_at = NOW() WHERE username = $2`,
		hashedPW, username)
	if err != nil {
		return fmt.Errorf("update password: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("user not found: %s", username)
	}
	return nil
}

// UserPasswordChangedAt returns the user's password_changed_at (the session
// epoch). ok is false when no such user exists, so the auth middleware can reject
// tokens for deleted accounts.
func (s *PostgresStore) UserPasswordChangedAt(ctx context.Context, username string) (time.Time, bool, error) {
	var changedAt time.Time
	err := s.pool.QueryRow(ctx,
		`SELECT password_changed_at FROM users WHERE username = $1`, username,
	).Scan(&changedAt)
	if err != nil {
		if strings.Contains(err.Error(), "no rows") {
			return time.Time{}, false, nil
		}
		return time.Time{}, false, fmt.Errorf("user password_changed_at: %w", err)
	}
	return changedAt, true, nil
}

// ── Password reset tokens ──

// CreateResetToken stores a one-time password-reset token for username with a
// one-hour TTL. Expired tokens are opportunistically purged on each write.
func (s *PostgresStore) CreateResetToken(ctx context.Context, username, token string) error {
	if _, err := s.pool.Exec(ctx, `DELETE FROM password_reset_tokens WHERE expires_at < NOW()`); err != nil {
		return fmt.Errorf("purge expired reset tokens: %w", err)
	}
	if _, err := s.pool.Exec(ctx, `
		INSERT INTO password_reset_tokens (token, username, expires_at)
		VALUES ($1, $2, $3)`,
		hashResetToken(token), username, time.Now().Add(time.Hour),
	); err != nil {
		return fmt.Errorf("create reset token: %w", err)
	}
	return nil
}

// ConsumeResetToken atomically deletes the token and returns its username. It
// fails if the token is unknown or expired (one-time use).
func (s *PostgresStore) ConsumeResetToken(ctx context.Context, token string) (string, error) {
	var username string
	err := s.pool.QueryRow(ctx, `
		DELETE FROM password_reset_tokens
		WHERE token = $1 AND expires_at > NOW()
		RETURNING username`, hashResetToken(token),
	).Scan(&username)
	if err != nil {
		if strings.Contains(err.Error(), "no rows") {
			return "", fmt.Errorf("invalid or expired token")
		}
		return "", fmt.Errorf("consume reset token: %w", err)
	}
	return username, nil
}
