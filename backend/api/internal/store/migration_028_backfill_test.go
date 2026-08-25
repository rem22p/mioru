package store

import (
	"context"
	"fmt"
	"io/fs"
	"net/url"
	"os"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/tern/v2/migrate"
)

// legacyBrandDB provisions a database of its own (the per-process one from
// testdb is already migrated to head, and 028 cannot be replayed there) and
// returns a connection sitting just below the brands migration.
//
// The backfill is the one step of 028 that never runs in any other test: the
// harness applies 001..head to an empty database, so no legacy row exists when
// the UPDATE executes. It is also effectively one-way for parsing purposes
// (B2: the legacy column is dropped by a separate migration in the next
// release, not by 028), so its parsing is verified here or nowhere.
func legacyBrandDB(t *testing.T) (*pgx.Conn, *migrate.Migrator) {
	t.Helper()

	base := os.Getenv("TEST_DATABASE_URL")
	if base == "" {
		t.Skip("TEST_DATABASE_URL not set — skipping PostgreSQL store tests")
	}
	u, err := url.Parse(base)
	if err != nil {
		t.Fatalf("parse TEST_DATABASE_URL: %v", err)
	}
	name := strings.TrimPrefix(u.Path, "/") + "_mig028"
	if len(name) > 63 {
		name = name[:63]
	}

	ctx := context.Background()
	admin, err := pgx.Connect(ctx, base)
	if err != nil {
		t.Fatalf("connect admin db: %v", err)
	}
	defer admin.Close(ctx)
	if _, err := admin.Exec(ctx, fmt.Sprintf(`DROP DATABASE IF EXISTS %q WITH (FORCE)`, name)); err != nil {
		t.Fatalf("drop stale %q: %v", name, err)
	}
	if _, err := admin.Exec(ctx, fmt.Sprintf(`CREATE DATABASE %q`, name)); err != nil {
		t.Fatalf("create %q: %v", name, err)
	}

	u.Path = "/" + name
	conn, err := pgx.Connect(ctx, u.String())
	if err != nil {
		t.Fatalf("connect %q: %v", name, err)
	}
	t.Cleanup(func() { conn.Close(ctx) })

	m, err := migrate.NewMigrator(ctx, conn, "public.schema_version")
	if err != nil {
		t.Fatalf("new migrator: %v", err)
	}
	sub, err := fs.Sub(migrationsFS, "migrations")
	if err != nil {
		t.Fatalf("sub migrations fs: %v", err)
	}
	if err := m.LoadMigrations(sub); err != nil {
		t.Fatalf("load migrations: %v", err)
	}
	if err := m.MigrateTo(ctx, 27); err != nil {
		t.Fatalf("migrate to 27: %v", err)
	}
	return conn, m
}

// TestMigration028BacksfillsLegacyBrands pins how the split reads the
// legacy free-text brand column: the separator is " x " however it was padded,
// values are trimmed, and nothing empty survives into the array.
func TestMigration028BacksfillsLegacyBrands(t *testing.T) {
	conn, m := legacyBrandDB(t)
	ctx := context.Background()

	cases := []struct {
		name   string
		legacy string
		want   []string
	}{
		{"single brand", "Nike", []string{"Nike"}},
		{"collaboration", "Bape x Mastermind", []string{"Bape", "Mastermind"}},
		{"padded separator", "Bape  x  Mastermind", []string{"Bape", "Mastermind"}},
		{"padded value", "  Nike  ", []string{"Nike"}},
		{"dangling separator", "Bape x ", []string{"Bape"}},
		{"leading separator", " x Mastermind", []string{"Mastermind"}},
		{"empty brand", "", nil},
		{"x inside a word", "Xerox Company", []string{"Xerox Company"}},
		// Prod preflight (B1): the real legacy collaboration separator is the
		// semicolon — prod carries "Bape; Mastermind".
		{"semicolon separator", "Bape; Mastermind", []string{"Bape", "Mastermind"}},
		{"semicolon without spaces", "Bape;Mastermind", []string{"Bape", "Mastermind"}},
	}

	for i, c := range cases {
		if _, err := conn.Exec(ctx,
			`INSERT INTO products (slug, name, brand, category_id)
			 VALUES ($1, $2, $3, (SELECT id FROM categories ORDER BY id LIMIT 1))`,
			fmt.Sprintf("legacy-%d", i), c.name, c.legacy); err != nil {
			t.Fatalf("insert %q: %v", c.name, err)
		}
	}

	if err := m.MigrateTo(ctx, 28); err != nil {
		t.Fatalf("migrate to 28: %v", err)
	}

	for i, c := range cases {
		var got []string
		if err := conn.QueryRow(ctx,
			`SELECT brands FROM products WHERE slug = $1`,
			fmt.Sprintf("legacy-%d", i)).Scan(&got); err != nil {
			t.Fatalf("read brands for %q: %v", c.name, err)
		}
		if len(got) != len(c.want) {
			t.Errorf("%s (%q): brands = %q, want %q", c.name, c.legacy, got, c.want)
			continue
		}
		for j := range got {
			if got[j] != c.want[j] {
				t.Errorf("%s (%q): brands = %q, want %q", c.name, c.legacy, got, c.want)
				break
			}
		}
	}
}
