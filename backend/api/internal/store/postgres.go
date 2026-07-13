package store

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"log"
	"os"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/tern/v2/migrate"

	"mioru/internal/auth"
)

// migrationsFS holds the versioned SQL migrations, embedded so the binary is
// self-contained (no migrations directory needs shipping alongside it).
//
//go:embed migrations/*.sql
var migrationsFS embed.FS

// PostgresStore provides PostgreSQL-based persistence for users, products, and categories.
type PostgresStore struct {
	pool  *pgxpool.Pool
	clock func() time.Time
}

// NewPostgresStore creates a connection pool and runs migrations.
// The clock defaults to time.Now; inject a fixed clock in tests.
func NewPostgresStore(ctx context.Context, databaseURL string) (*PostgresStore, error) {
	pool, err := pgxpool.NewWithConfig(ctx, func() *pgxpool.Config {
		c, err := pgxpool.ParseConfig(databaseURL)
		if err != nil {
			// Fallback: ParseConfig shouldn't fail for a validated URL.
			panic("pgxpool parse config: " + err.Error())
		}
		c.AfterConnect = func(ctx context.Context, conn *pgx.Conn) error {
			// Lower trigram threshold so the % operator catches
			// typos like "crhome" → "Chrome" (sim=0.27).
			_, err := conn.Exec(ctx, "SET pg_trgm.similarity_threshold = '0.2'")
			return err
		}
		return c
	}())
	if err != nil {
		return nil, fmt.Errorf("pgxpool connect: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("pgxpool ping: %w", err)
	}

	s := &PostgresStore{pool: pool, clock: time.Now}
	if err := s.migrate(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("migrate: %w", err)
	}

	return s, nil
}

// Close closes the connection pool.
func (s *PostgresStore) Close() {
	s.pool.Close()
}

func (s *PostgresStore) migrate(ctx context.Context) error {
	if err := s.runMigrations(ctx); err != nil {
		return err
	}

	// Seed the first admin from BOOTSTRAP_ADMIN_* env. Registration is
	// invite-only (admins create admins), so this resolves the chicken-and-egg
	// for the very first admin on a clean database. It depends on runtime env
	// and bcrypt, so it stays a Go step rather than a versioned SQL migration.
	if err := s.seedAdmin(ctx); err != nil {
		return fmt.Errorf("seed admin: %w", err)
	}

	return nil
}

// runMigrations applies every embedded versioned migration up to the latest via
// tern. The applied version is tracked in public.schema_version (created on
// first run) and each migration runs in its own transaction, so a restart or a
// deploy onto an already-migrated database is a safe no-op. tern operates on a
// single *pgx.Conn, so we borrow one from the pool for the duration.
func (s *PostgresStore) runMigrations(ctx context.Context) error {
	conn, err := s.pool.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("acquire migration conn: %w", err)
	}
	defer conn.Release()

	m, err := migrate.NewMigrator(ctx, conn.Conn(), "public.schema_version")
	if err != nil {
		return fmt.Errorf("new migrator: %w", err)
	}

	sub, err := fs.Sub(migrationsFS, "migrations")
	if err != nil {
		return fmt.Errorf("sub migrations fs: %w", err)
	}
	if err := m.LoadMigrations(sub); err != nil {
		return fmt.Errorf("load migrations: %w", err)
	}
	if err := m.Migrate(ctx); err != nil {
		return fmt.Errorf("apply migrations: %w", err)
	}

	return nil
}

// seedAdmin inserts the bootstrap admin defined by BOOTSTRAP_ADMIN_USERNAME,
// BOOTSTRAP_ADMIN_EMAIL and BOOTSTRAP_ADMIN_PASSWORD. It is a no-op when any of
// those vars is unset, and idempotent via ON CONFLICT DO NOTHING so repeated
// boots never duplicate or overwrite an existing admin.
func (s *PostgresStore) seedAdmin(ctx context.Context) error {
	username := os.Getenv("BOOTSTRAP_ADMIN_USERNAME")
	emailAddr := os.Getenv("BOOTSTRAP_ADMIN_EMAIL")
	password := os.Getenv("BOOTSTRAP_ADMIN_PASSWORD")
	if username == "" || emailAddr == "" || password == "" {
		return nil
	}

	hash, err := auth.HashPassword(password)
	if err != nil {
		return fmt.Errorf("hash bootstrap admin password: %w", err)
	}

	tag, err := s.pool.Exec(ctx, `
		INSERT INTO users (username, email, hashed_password, display_name, avatar_color, role)
		VALUES ($1, $2, $3, $1, '#44944A', 'super_admin')
		ON CONFLICT DO NOTHING`,
		username, strings.ToLower(emailAddr), hash,
	)
	if err != nil {
		return fmt.Errorf("insert bootstrap admin: %w", err)
	}
	if tag.RowsAffected() > 0 {
		log.Printf("Seeded bootstrap admin %q from BOOTSTRAP_ADMIN_* env", username)
	}
	return nil
}
