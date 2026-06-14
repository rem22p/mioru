package store

import (
	"context"
	"testing"

	"mioru/internal/testdb"
)

// testStore connects to a database dedicated to this test process (derived from
// TEST_DATABASE_URL by testdb, so this package and internal/handler do not
// share tables when run concurrently), runs migrations (schema + seeded
// category tree), and returns a store with a clean slate. It calls t.Skip when
// TEST_DATABASE_URL is unset, so the suite is a no-op without a dedicated test
// database.
//
// TEST_DATABASE_URL must point at a disposable test database — never the
// production DATABASE_URL: resetTables truncates user data on every call.
func testStore(t *testing.T) *PostgresStore {
	t.Helper()

	s, err := NewPostgresStore(context.Background(), testStoreURL(t))
	if err != nil {
		t.Fatalf("connect test store: %v", err)
	}
	t.Cleanup(s.Close)

	resetTables(t, s)
	return s
}

// testStoreURL returns the per-process isolated database URL, or skips the test
// when TEST_DATABASE_URL is unset. Tests that connect directly (e.g. migration
// rerun checks) call this so they target the same isolated database.
func testStoreURL(t *testing.T) string {
	t.Helper()

	url, err := testdb.URL(context.Background())
	if err != nil {
		t.Fatalf("provision test db: %v", err)
	}
	if url == "" {
		t.Skip("TEST_DATABASE_URL not set — skipping PostgreSQL store tests")
	}
	return url
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
