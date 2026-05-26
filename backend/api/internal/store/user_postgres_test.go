package store

import (
	"context"
	"testing"
	"time"

	"mioru/internal/model"
)

// TestResetTokenHashedAtRest verifies that a reset token is stored only as its
// SHA-256 hash, that the raw token still consumes it, and that consumption is
// one-time.
func TestResetTokenHashedAtRest(t *testing.T) {
	s := testStore(t) // skips when TEST_DATABASE_URL is unset
	ctx := context.Background()

	const username = "alice"
	// 64 hex chars, shaped like the real 32-byte token the handler generates.
	const raw = "deadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef"

	if err := s.CreateResetToken(ctx, username, raw); err != nil {
		t.Fatalf("CreateResetToken: %v", err)
	}

	var stored string
	if err := s.pool.QueryRow(ctx,
		`SELECT token FROM password_reset_tokens WHERE username = $1`, username,
	).Scan(&stored); err != nil {
		t.Fatalf("read stored token: %v", err)
	}
	if stored == raw {
		t.Fatal("raw reset token stored in plaintext")
	}
	if stored != hashResetToken(raw) {
		t.Errorf("stored token = %q, want sha256 hash of raw", stored)
	}

	// The raw token still works (the store hashes it for the lookup).
	got, err := s.ConsumeResetToken(ctx, raw)
	if err != nil {
		t.Fatalf("ConsumeResetToken: %v", err)
	}
	if got != username {
		t.Errorf("username = %q, want %q", got, username)
	}

	// One-time use: a second consume must fail.
	if _, err := s.ConsumeResetToken(ctx, raw); err == nil {
		t.Error("expected error on second consume (one-time use)")
	}
}

// TestConsumeResetTokenRejectsUnknown verifies an unknown token is rejected.
func TestConsumeResetTokenRejectsUnknown(t *testing.T) {
	s := testStore(t)
	if _, err := s.ConsumeResetToken(context.Background(), "no-such-token"); err == nil {
		t.Error("expected error for unknown token")
	}
}

// TestUpdatePasswordBumpsEpoch verifies that changing a user's password advances
// password_changed_at — the session-revocation epoch the auth middleware uses to
// reject tokens minted before the change.
func TestUpdatePasswordBumpsEpoch(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	const username = "epoch-user"
	if err := s.CreateUser(ctx, model.User{
		Username: username, Email: "epoch@example.com", HashedPW: "h1", Role: "admin",
	}); err != nil {
		t.Fatalf("CreateUser: %v", err)
	}

	e1, ok, err := s.UserPasswordChangedAt(ctx, username)
	if err != nil {
		t.Fatalf("UserPasswordChangedAt: %v", err)
	}
	if !ok {
		t.Fatal("expected user to exist")
	}

	time.Sleep(10 * time.Millisecond) // ensure NOW() advances measurably
	if err := s.UpdatePassword(ctx, username, "h2"); err != nil {
		t.Fatalf("UpdatePassword: %v", err)
	}

	e2, ok, err := s.UserPasswordChangedAt(ctx, username)
	if err != nil {
		t.Fatalf("UserPasswordChangedAt after update: %v", err)
	}
	if !ok {
		t.Fatal("expected user to still exist")
	}
	if !e2.After(e1) {
		t.Errorf("password_changed_at did not advance: before=%v after=%v", e1, e2)
	}
}

// TestUserPasswordChangedAtUnknown verifies the epoch lookup reports a missing
// user via ok=false (not an error), so the middleware rejects its tokens.
func TestUserPasswordChangedAtUnknown(t *testing.T) {
	s := testStore(t)
	_, ok, err := s.UserPasswordChangedAt(context.Background(), "ghost")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ok {
		t.Error("expected ok=false for a non-existent user")
	}
}
