package store

import (
	"database/sql"
	"fmt"
	"strings"

	"mioru/internal/model"
)

// ── User operations on SQLite ──

// CreateUser inserts a new user into SQLite.
func (s *SQLiteStore) CreateUser(u model.User) error {
	_, err := s.db.Exec(`
		INSERT INTO users (username, email, hashed_password, first_name, last_name, display_name, avatar_color, role)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		u.Username, u.Email, u.HashedPW, u.FirstName, u.LastName, u.DisplayName, u.AvatarColor, u.Role,
	)
	if err != nil {
		errStr := err.Error()
		if strings.Contains(errStr, "UNIQUE") {
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
func (s *SQLiteStore) GetUser(username string) (*model.User, error) {
	u := &model.User{}
	err := s.db.QueryRow(`
		SELECT id, username, email, hashed_password, first_name, last_name, display_name, avatar_color, role, created_at
		FROM users WHERE username = ?`, username,
	).Scan(&u.ID, &u.Username, &u.Email, &u.HashedPW, &u.FirstName, &u.LastName, &u.DisplayName, &u.AvatarColor, &u.Role, &u.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get user: %w", err)
	}
	return u, nil
}

// GetUserByEmail retrieves a user by email.
func (s *SQLiteStore) GetUserByEmail(email string) (*model.User, error) {
	u := &model.User{}
	err := s.db.QueryRow(`
		SELECT id, username, email, hashed_password, first_name, last_name, display_name, avatar_color, role, created_at
		FROM users WHERE email = ?`, email,
	).Scan(&u.ID, &u.Username, &u.Email, &u.HashedPW, &u.FirstName, &u.LastName, &u.DisplayName, &u.AvatarColor, &u.Role, &u.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get user by email: %w", err)
	}
	return u, nil
}

// UpdateUser updates profile fields for a user identified by username.
// The updates map may contain: display_name, avatar_color, first_name, last_name.
func (s *SQLiteStore) UpdateUser(username string, updates map[string]string) error {
	setClauses := []string{}
	args := []interface{}{}

	for k, v := range updates {
		switch k {
		case "display_name":
			setClauses = append(setClauses, "display_name = ?")
			args = append(args, v)
		case "avatar_color":
			setClauses = append(setClauses, "avatar_color = ?")
			args = append(args, v)
		case "first_name":
			setClauses = append(setClauses, "first_name = ?")
			args = append(args, v)
		case "last_name":
			setClauses = append(setClauses, "last_name = ?")
			args = append(args, v)
		}
	}

	if len(setClauses) == 0 {
		return fmt.Errorf("no valid fields to update")
	}

	args = append(args, username)
	query := "UPDATE users SET " + strings.Join(setClauses, ", ") + " WHERE username = ?"
	res, err := s.db.Exec(query, args...)
	if err != nil {
		return fmt.Errorf("update user: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("user not found: %s", username)
	}
	return nil
}

// UpdatePassword updates the hashed password for a user identified by username.
func (s *SQLiteStore) UpdatePassword(username, hashedPW string) error {
	res, err := s.db.Exec(`UPDATE users SET hashed_password = ? WHERE username = ?`, hashedPW, username)
	if err != nil {
		return fmt.Errorf("update password: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("user not found: %s", username)
	}
	return nil
}

// IsUsersTableEmpty returns true if the users table has no rows (used for auto-migration).
func (s *SQLiteStore) IsUsersTableEmpty() (bool, error) {
	var count int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM users`).Scan(&count); err != nil {
		return false, fmt.Errorf("count users: %w", err)
	}
	return count == 0, nil
}
