// Package testdb derives a PostgreSQL database dedicated to the current test
// process out of TEST_DATABASE_URL, so that test binaries running concurrently
// under `go test ./...` never share tables. Without this, two DB-touching
// packages (e.g. internal/store and internal/handler) point at the same
// database and TRUNCATE each other's rows mid-test, producing flaky failures.
//
// It depends only on pgx and the standard library — no internal packages — so
// it is importable from both internal/storetest and store's own white-box
// tests without an import cycle.
//
// TEST_DATABASE_URL must point at a disposable test database — never the
// production DATABASE_URL: this package CREATEs and DROPs sibling databases.
package testdb

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	"github.com/jackc/pgx/v5"
)

// maxIdentLen is PostgreSQL's identifier length limit; longer names are
// silently truncated by the server, which would risk two processes colliding
// on the same truncated database name. We truncate deliberately instead.
const maxIdentLen = 63

var (
	once   sync.Once
	isoURL string
	isoErr error
)

// nonIdent matches every run of characters that may not appear unquoted in a
// database identifier; we collapse each run to a single underscore.
var nonIdent = regexp.MustCompile(`[^a-z0-9]+`)

// URL returns a connection URL to a database dedicated to this test process,
// provisioned (dropped if a stale copy lingers, then created) on first call.
// All later calls in the same process return the same URL. It returns "" when
// TEST_DATABASE_URL is unset, so callers can t.Skip as before.
func URL(ctx context.Context) (string, error) {
	once.Do(func() { isoURL, isoErr = provision(ctx) })
	return isoURL, isoErr
}

// provision parses TEST_DATABASE_URL, derives a per-process database name, and
// (re)creates that database via an admin connection to the base database.
func provision(ctx context.Context) (string, error) {
	base := os.Getenv("TEST_DATABASE_URL")
	if base == "" {
		return "", nil
	}

	u, err := url.Parse(base)
	if err != nil {
		return "", fmt.Errorf("parse TEST_DATABASE_URL: %w", err)
	}
	baseDB := strings.TrimPrefix(u.Path, "/")
	if baseDB == "" {
		return "", fmt.Errorf("TEST_DATABASE_URL has no database name in its path")
	}

	isoDB := isolatedName(baseDB)
	if isoDB == baseDB {
		// Defensive: never reprovision (drop!) the base database itself.
		return "", fmt.Errorf("derived test database name %q collides with the base database", isoDB)
	}

	// Connect to the base database purely to issue DDL against the sibling
	// isolated database (CREATE/DROP DATABASE cannot run while connected to the
	// target). The admin connection is short-lived.
	admin, err := pgx.Connect(ctx, base)
	if err != nil {
		return "", fmt.Errorf("connect admin db: %w", err)
	}
	defer admin.Close(ctx)

	// WITH (FORCE) terminates any leftover connections (PostgreSQL 13+); the
	// throwaway test cluster runs 16. Identifiers are validated by isolatedName
	// to the [a-z0-9_] set, so the quoted interpolation cannot inject.
	if _, err := admin.Exec(ctx, fmt.Sprintf(`DROP DATABASE IF EXISTS %q WITH (FORCE)`, isoDB)); err != nil {
		return "", fmt.Errorf("drop stale test db %q: %w", isoDB, err)
	}
	if _, err := admin.Exec(ctx, fmt.Sprintf(`CREATE DATABASE %q`, isoDB)); err != nil {
		return "", fmt.Errorf("create test db %q: %w", isoDB, err)
	}

	u.Path = "/" + isoDB
	return u.String(), nil
}

// isolatedName builds a per-process database name: the base name plus a tag
// derived from the test binary (os.Args[0]), e.g. "mioru_test" + "handler" →
// "mioru_test_handler". The tag is stable per package, so reruns reuse (and
// thus drop+recreate) the same database rather than leaking new ones.
func isolatedName(baseDB string) string {
	tag := processTag()
	name := baseDB
	if tag != "" {
		name = baseDB + "_" + tag
	}
	if len(name) > maxIdentLen {
		name = name[:maxIdentLen]
	}
	return name
}

// processTag sanitizes the test binary name into a database-identifier-safe
// token. Go names a package's test binary "<pkg>.test", so this yields the
// package name (e.g. "handler", "store").
func processTag() string {
	bin := filepath.Base(os.Args[0])
	bin = strings.TrimSuffix(bin, ".test")
	bin = strings.ToLower(bin)
	bin = nonIdent.ReplaceAllString(bin, "_")
	return strings.Trim(bin, "_")
}
