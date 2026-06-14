//go:build e2e
// +build e2e

package main

import (
	"net/http"
	"os"

	"mioru/internal/config"
	"mioru/internal/handler"
	"mioru/internal/store"
)

// registerTestRoutes wires the dev-only POST /api/_test/reset-admin endpoint
// that lets apps/admin/e2e/security.spec.ts drop the admin user back to a
// known bcrypt hash + password_changed_at before running security-critical
// specs that mutate the password.
//
// Gated on:
//   1. !cfg.IsProduction() — fail-safe if APP_ENV is misconfigured
//   2. E2E_RESET_KEY env var is set — without it the handler itself
//      returns 503 with a generic envelope, so registering the route
//      without a server-side secret serves no purpose.
//
// This file is gated on the `e2e` build tag. Production binaries
// (`go build ./cmd/server`) do not include it, so the route does not
// exist in production regardless of env. E2E and CI build with
// `go build -tags e2e`.
func registerTestRoutes(
	mux *http.ServeMux,
	pgStore *store.PostgresStore,
	cfg config.Config,
	getenv func(string) string,
) {
	if cfg.IsProduction() {
		return
	}
	if getenv("E2E_RESET_KEY") == "" {
		return
	}
	testAdminH := handler.NewTestResetAdminHandler(pgStore)
	mux.HandleFunc("POST /api/_test/reset-admin", testAdminH.ServeHTTP)

	// Reference os so this file is not flagged as unused when the
	// registration block above is conditionally skipped.
	_ = os.Getenv
}
