package store

import (
	"context"
	"testing"
	"time"

	"mioru/internal/model"
)

// TestResetAdminForTestSeedsNewUser verifies the happy path: a brand-new
// username is created with the supplied bcrypt hash, email, role, and
// password_changed_at. This is what apps/admin/e2e/security.spec.ts relies
// on when seeding a known admin state before the security-critical test.
func TestResetAdminForTestSeedsNewUser(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	const username = "reset-seed"
	pastTime := time.Now().Add(-2 * time.Hour).UTC().Truncate(time.Second)

	if err := s.ResetAdminForTest(ctx, model.User{
		Username:    username,
		Email:       username + "@mioru.store",
		HashedPW:    "hashed-from-test",
		DisplayName: "Reset Seed",
		AvatarColor: "#44944A",
		Role:        "super_admin",
	}, pastTime); err != nil {
		t.Fatalf("ResetAdminForTest: %v", err)
	}

	// Re-read the user via the canonical GetUser path and verify every
	// field round-trips.
	u, err := s.GetUser(ctx, username)
	if err != nil {
		t.Fatalf("GetUser after seed: %v", err)
	}
	if u == nil {
		t.Fatal("GetUser returned nil for seeded user")
	}
	if u.Email != username+"@mioru.store" {
		t.Errorf("email: got %q, want %q", u.Email, username+"@mioru.store")
	}
	if u.HashedPW != "hashed-from-test" {
		t.Errorf("hashed_password: got %q, want %q", u.HashedPW, "hashed-from-test")
	}
	if u.Role != "super_admin" {
		t.Errorf("role: got %q, want %q", u.Role, "super_admin")
	}
	if u.DisplayName != "Reset Seed" {
		t.Errorf("display_name: got %q, want %q", u.DisplayName, "Reset Seed")
	}

	got, ok, err := s.UserPasswordChangedAt(ctx, username)
	if err != nil {
		t.Fatalf("UserPasswordChangedAt: %v", err)
	}
	if !ok {
		t.Fatal("expected user to exist after seed")
	}
	if !got.Equal(pastTime) {
		t.Errorf("password_changed_at: got %v, want %v", got, pastTime)
	}
}

// TestResetAdminForTestIsIdempotent verifies that calling the helper twice
// on the same username updates (not duplicates) the row. This is what makes
// the helper safe to call from CI re-runs.
func TestResetAdminForTestIsIdempotent(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	const username = "reset-idempotent"
	t1 := time.Now().Add(-3 * time.Hour).UTC().Truncate(time.Second)
	t2 := time.Now().Add(-1 * time.Hour).UTC().Truncate(time.Second)

	if err := s.ResetAdminForTest(ctx, model.User{
		Username: username, Email: "first@x", HashedPW: "h1", Role: "admin",
	}, t1); err != nil {
		t.Fatalf("first seed: %v", err)
	}
	if err := s.ResetAdminForTest(ctx, model.User{
		Username: username, Email: "second@x", HashedPW: "h2", Role: "super_admin",
	}, t2); err != nil {
		t.Fatalf("second seed: %v", err)
	}

	// Single row, latest values.
	var count int
	if err := s.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM users WHERE username = $1`, username,
	).Scan(&count); err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected exactly 1 row, got %d", count)
	}

	got, ok, err := s.UserPasswordChangedAt(ctx, username)
	if err != nil {
		t.Fatalf("UserPasswordChangedAt: %v", err)
	}
	if !ok {
		t.Fatal("expected user to exist")
	}
	if !got.Equal(t2) {
		t.Errorf("password_changed_at: got %v, want %v (later value should win)", got, t2)
	}

	u, err := s.GetUser(ctx, username)
	if err != nil {
		t.Fatalf("GetUser: %v", err)
	}
	if u.HashedPW != "h2" {
		t.Errorf("hashed_password: got %q, want %q (later value should win)", u.HashedPW, "h2")
	}
	if u.Role != "super_admin" {
		t.Errorf("role: got %q, want %q (later value should win)", u.Role, "super_admin")
	}
}
