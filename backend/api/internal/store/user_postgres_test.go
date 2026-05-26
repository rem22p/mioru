package store

import (
	"context"
	"testing"
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
