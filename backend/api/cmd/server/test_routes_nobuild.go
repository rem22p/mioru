//go:build !e2e
// +build !e2e

// Stub used when building WITHOUT the `e2e` tag (i.e. the default
// production binary). The real implementation lives in
// test_routes_e2e.go behind the `e2e` build tag. This no-op keeps
// main.go source-clean so a single source file works under both
// build configurations.

package main

import (
	"net/http"

	"mioru/internal/config"
	"mioru/internal/store"
)

func registerTestRoutes(
	_ *http.ServeMux,
	_ *store.PostgresStore,
	_ config.Config,
	_ func(string) string,
) {
	// no-op: production binary has no /api/_test/* routes.
}
