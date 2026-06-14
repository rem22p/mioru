# Internal / test-only API surface

This document covers routes that exist in **e2e builds only** (gated by
the `//go:build e2e` build tag in `backend/api/internal/handler/test_reset.go`
and the matching registration stub in
`backend/api/cmd/server/test_routes_e2e.go`). They are **not compiled into
production binaries** — the production build (`go build ./cmd/server`)
produces a binary that has zero references to these routes, file paths,
or symbols.

## Why they exist

`apps/admin/e2e/security.spec.ts` mutates the shared admin user's
password and `password_changed_at`. Both the auth middleware (rejects
tokens with `iat < password_changed_at`) and the bootstrap login used
by the regular `authenticated` Playwright project depend on those
fields being in a known state. Running the security spec in the same
project as the regular suite corrupts the next `login()` call and
cascades into 3–4 follow-up `product-add not visible` failures (see
PR #49 review history).

The reset endpoint below lets the security spec drop the admin back to
a known bcrypt hash before each test, in its own dedicated Playwright
project that runs *after* the regular suite finishes.

## Routes

### POST /api/_test/reset-admin

Reset the bootstrap admin user to a caller-supplied state.

| | |
|---|---|
| Auth | `X-E2E-Reset-Key` header must constant-time-equal the server-side `E2E_RESET_KEY` env var |
| CSRF | — (not a browser-initiated mutation; no session is established) |
| Rate-limit | — (test endpoint; not exposed in production) |
| Build | `e2e` tag required — `go build -tags e2e ./cmd/server` |
| Source | `backend/api/internal/handler/test_reset.go` |
| Registration | `backend/api/cmd/server/test_routes_e2e.go` (no-op stub `test_routes_nobuild.go` for production builds) |
| Tests | `backend/api/internal/handler/test_reset_test.go` (5 unit tests, all gated `e2e`) |
| E2E consumer | `apps/admin/e2e/security.spec.ts` (`beforeAll` calls this) |

#### Request

```http
POST /api/_test/reset-admin
X-E2E-Reset-Key: <must match server E2E_RESET_KEY>
Content-Type: application/json

{
  "username": "admin",
  "hashed_password": "$2a$12$...",   // bcrypt cost 12
  "email": "admin@mioru.store",       // optional, default username@mioru.store
  "display_name": "Admin",            // optional, default username
  "role": "super_admin",              // optional, default super_admin
  "password_changed_at": "2026-06-14T10:00:00Z"  // optional RFC3339, default = now-1h
}
```

#### Responses

| Status | Code | When |
|---|---|---|
| 200 | `{"ok":true}` | Success — admin upserted |
| 400 | `VALIDATION_FAILED` | Missing `username` or `hashed_password` |
| 403 | `FORBIDDEN` | Header missing or doesn't match `E2E_RESET_KEY` |
| 500 | `INTERNAL` | DB error (logged server-side via slog; never leaks detail) |
| 503 | `TEST_RESET_DISABLED` | Server has no `E2E_RESET_KEY` set (misconfiguration) |

#### Defense in depth

The route is unreachable in production in **three** independent ways:

1. **Build tag.** `test_reset.go` and `test_routes_e2e.go` have `//go:build e2e`. Production binaries do not compile these files. Verified:
   ```
   $ go build ./cmd/server -o /tmp/prod
   $ strings /tmp/prod | grep -E "_test/reset-admin|TestResetAdminHandler"
   (no output)
   $ go build -tags e2e ./cmd/server -o /tmp/e2e
   $ strings /tmp/e2e | grep -E "_test/reset-admin|TestResetAdminHandler"
   *handler.TestResetAdminHandler
   POST /api/_test/reset-admin
   X-E2E-Reset-Key
   ```

2. **Runtime gate.** Even on an e2e build, the route is only registered when `!cfg.IsProduction() && E2E_RESET_KEY != ""`.

3. **Auth gate.** Every request must pass constant-time compare against `E2E_RESET_KEY`. An empty server-side secret returns 503 (not 200, not 401) — fail-closed.

#### Verification commands

```bash
# 1. Production binary: route is absent
$ go build -o /tmp/prod ./backend/api/cmd/server
$ strings /tmp/prod | grep "_test/reset-admin"
(no output — excluded by build tag)

# 2. E2E binary: route is present
$ go build -tags e2e -o /tmp/e2e ./backend/api/cmd/server
$ strings /tmp/e2e | grep "_test/reset-admin"
POST /api/_test/reset-admin

# 3. Unit tests
$ go test -tags e2e ./internal/handler/ -run TestResetAdmin -v
=== RUN   TestResetAdminRejectsMissingKey      --- PASS
=== RUN   TestResetAdminRejectsWrongKey        --- PASS
=== RUN   TestResetAdminRequiresServerSideKey  --- PASS
=== RUN   TestResetAdminHappyPath              --- PASS
=== RUN   TestResetAdminErrorPathDoesNotLeak   --- PASS
PASS

# 4. E2E: full security project
$ E2E_RESET_KEY=ci-secret npm run --prefix apps/admin e2e:security
✓ [security] › e2e/security.spec.ts:83:1
   › changing password invalidates the old session (security-critical) (7.4s)
1 passed
```
