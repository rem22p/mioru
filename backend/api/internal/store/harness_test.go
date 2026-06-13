package store

import (
	"context"
	"os"
	"testing"
)

// testStore connects to the throwaway test database named by TEST_DATABASE_URL,
// runs migrations (schema + seeded category tree), and returns a store with a
// clean slate. It calls t.Skip when TEST_DATABASE_URL is unset, so the suite is
// a no-op without a dedicated test database.
//
// TEST_DATABASE_URL must point at a disposable test database — never the
// production DATABASE_URL: resetTables truncates user data on every call.
func testStore(t *testing.T) *PostgresStore {
	t.Helper()

	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL not set — skipping PostgreSQL store tests")
	}

	s, err := NewPostgresStore(context.Background(), url)
	if err != nil {
		t.Fatalf("connect test store: %v", err)
	}
	t.Cleanup(s.Close)

	resetTables(t, s)
	return s
}

// resetTables delegates to ResetTestData so the SQL table list lives in one
// place and can be shared with the external internal/storetest package without
// an import cycle.
func resetTables(t *testing.T, s *PostgresStore) {
	t.Helper()
	if err := s.ResetTestData(context.Background()); err != nil {
		t.Fatalf("reset tables: %v", err)
	}
}
