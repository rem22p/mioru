package store

import (
	"context"
	"io/fs"
	"os"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// embeddedMigrationSeqs reads the embedded migrations directory and returns the
// numeric prefix of every *.sql file, failing the test on any malformed name.
func embeddedMigrationSeqs(t *testing.T) []int {
	t.Helper()

	sub, err := fs.Sub(migrationsFS, "migrations")
	if err != nil {
		t.Fatalf("sub migrations fs: %v", err)
	}
	entries, err := fs.ReadDir(sub, ".")
	if err != nil {
		t.Fatalf("read migrations dir: %v", err)
	}

	var seqs []int
	for _, e := range entries {
		name := e.Name()
		if !strings.HasSuffix(name, ".sql") {
			t.Errorf("unexpected non-sql file in migrations: %q", name)
			continue
		}
		idx := strings.IndexByte(name, '_')
		if idx <= 0 {
			t.Errorf("migration %q must be named NNN_name.sql", name)
			continue
		}
		n, err := strconv.Atoi(name[:idx])
		if err != nil {
			t.Errorf("migration %q has non-numeric prefix: %v", name, err)
			continue
		}
		seqs = append(seqs, n)
	}
	return seqs
}

// TestEmbeddedMigrationsAreSequential is a pure unit test (no database): it
// guards that migration files are numbered 1..N with no gaps or duplicates,
// which is exactly what tern requires to apply them in order.
func TestEmbeddedMigrationsAreSequential(t *testing.T) {
	seqs := embeddedMigrationSeqs(t)
	if len(seqs) == 0 {
		t.Fatal("no migrations embedded")
	}

	sort.Ints(seqs)
	for i, n := range seqs {
		if n != i+1 {
			t.Fatalf("migrations must be gapless and 1-based; got sequence %v", seqs)
		}
	}
}

// currentSchemaVersion reads the version tern records in public.schema_version.
func currentSchemaVersion(t *testing.T, s *PostgresStore) int32 {
	t.Helper()
	var v int32
	if err := s.pool.QueryRow(context.Background(),
		`SELECT version FROM public.schema_version`).Scan(&v); err != nil {
		t.Fatalf("read schema_version: %v", err)
	}
	return v
}

// TestMigrationsVersionAfterMigrate verifies that connecting (which runs
// migrations) leaves the database recorded at the latest version — i.e. every
// embedded migration was applied.
func TestMigrationsVersionAfterMigrate(t *testing.T) {
	s := testStore(t) // skips when TEST_DATABASE_URL is unset
	got := currentSchemaVersion(t, s)
	want := int32(len(embeddedMigrationSeqs(t)))
	if got != want {
		t.Errorf("schema_version = %d, want %d (number of migrations)", got, want)
	}
}

// TestMigrationsIdempotentRerun proves that re-running migrations against an
// already-migrated database (every restart and every redeploy) is a safe no-op:
// the version is unchanged and no error is returned.
func TestMigrationsIdempotentRerun(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL not set — skipping PostgreSQL store tests")
	}
	ctx := context.Background()

	s1, err := NewPostgresStore(ctx, url)
	if err != nil {
		t.Fatalf("first connect: %v", err)
	}
	v1 := currentSchemaVersion(t, s1)
	s1.Close()

	s2, err := NewPostgresStore(ctx, url) // re-applies migrations on a migrated DB
	if err != nil {
		t.Fatalf("rerun migrations: %v", err)
	}
	defer s2.Close()
	v2 := currentSchemaVersion(t, s2)

	if v1 != v2 {
		t.Errorf("schema version changed on rerun: %d -> %d", v1, v2)
	}
}
