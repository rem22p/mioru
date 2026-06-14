// Package storetest provides a shared test harness for code that needs a real
// PostgreSQL-backed store (e.g. handler integration tests). It lives in a
// non-_test package so it is importable from any test package; store's own
// in-package white-box tests cannot use it (import cycle) and keep their own
// testStore helper, which shares the same canonical reset via ResetTestData.
package storetest

import (
	"context"
	"testing"

	"mioru/internal/store"
	"mioru/internal/testdb"
)

// Fresh connects to a database dedicated to this test process (derived from
// TEST_DATABASE_URL by testdb, so packages running concurrently under
// `go test ./...` do not share tables), runs migrations (schema + seeded
// category tree), truncates all data tables, and returns a clean store.
// It calls t.Skip when TEST_DATABASE_URL is unset, so suites are a no-op
// without a dedicated test database.
//
// TEST_DATABASE_URL must point at a disposable database — never production:
// testdb creates/drops sibling databases and Fresh truncates user data.
func Fresh(t testing.TB) *store.PostgresStore {
	t.Helper()

	ctx := context.Background()
	url, err := testdb.URL(ctx)
	if err != nil {
		t.Fatalf("provision test db: %v", err)
	}
	if url == "" {
		t.Skip("TEST_DATABASE_URL not set — skipping PostgreSQL integration tests")
	}

	s, err := store.NewPostgresStore(ctx, url)
	if err != nil {
		t.Fatalf("connect test store: %v", err)
	}
	t.Cleanup(s.Close)

	if err := s.ResetTestData(ctx); err != nil {
		t.Fatalf("reset test data: %v", err)
	}
	return s
}
