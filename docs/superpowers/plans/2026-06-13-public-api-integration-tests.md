# Public API Integration Tests — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a handler-level integration test suite that drives the real handlers through the real middleware chain against a real throwaway Postgres, plus a public-API inventory document — closing the gap left by fake-store unit tests.

**Architecture:** A new `internal/storetest` package exposes `Fresh(t)` (real `*store.PostgresStore` + reset), backed by a new exported `(*PostgresStore).ResetTestData`. A black-box `package handler_test` harness mints real JWT cookies and wraps each handler in the same middleware stack `main.go` uses, driving requests via `httptest.NewRecorder`. Tests are split by domain (storefront / orders / admin). Real bugs found are filed as GitHub issues.

**Tech Stack:** Go 1.25 stdlib `testing`, `net/http/httptest`, `pgxpool`, `mioru/internal/{store,storetest,handler,middleware,cookieauth,auth,model,email}`.

**Spec:** `docs/superpowers/specs/2026-06-13-public-api-integration-tests-design.md`

**Preconditions for every test run:**
```bash
docker compose -f backend/api/docker-compose.test.yml up -d
export TEST_DATABASE_URL='postgres://mioru:mioru@127.0.0.1:55433/mioru_test?sslmode=disable'
```
All commands below run from `backend/api/`.

---

## File Structure

| File | Responsibility |
|---|---|
| `internal/store/reset_testdata.go` (create) | Exported `ResetTestData` — canonical TRUNCATE (single source of truth) |
| `internal/store/harness_test.go` (modify) | `resetTables` delegates to `s.ResetTestData` |
| `internal/storetest/storetest.go` (create) | `Fresh(t)` for external-package tests |
| `docs/api/public-api-inventory.md` (create) | Route inventory + test-case matrix |
| `internal/handler/integration_harness_test.go` (create) | `package handler_test`: sessions, middleware wrappers, `do()` helper, fixtures |
| `internal/handler/integration_storefront_test.go` (create) | Public catalog + customer auth/profile + cart/favorites |
| `internal/handler/integration_orders_test.go` (create) | CreateOrder end-to-end, idempotency, oversell, ListOrders, admin status |
| `internal/handler/integration_admin_test.go` (create) | Admin product CRUD + auth/CSRF gates |

---

## Task 1: Exported `ResetTestData` + harness delegation

**Files:**
- Create: `internal/store/reset_testdata.go`
- Modify: `internal/store/harness_test.go` (the `resetTables` function)

- [ ] **Step 1: Create the exported reset method**

Create `internal/store/reset_testdata.go`:

```go
package store

import "context"

// ResetTestData truncates every data table so a test starts from a known,
// empty state, while keeping the seeded category tree (categories is not
// truncated). RESTART IDENTITY resets SERIAL counters; CASCADE covers FK
// dependents.
//
// TEST-ONLY: this is destructive and must only be run against a disposable
// database (TEST_DATABASE_URL). It is exported (not a _test helper) so that
// both the in-package white-box store tests and the external internal/storetest
// package can share one canonical table list — Go's import-cycle rule forbids
// the in-package store tests from importing storetest, which imports store.
func (s *PostgresStore) ResetTestData(ctx context.Context) error {
	_, err := s.pool.Exec(ctx,
		`TRUNCATE products, product_sizes, size_chart_rows, product_images,
		         users, customers, customer_oauth, password_reset_tokens, orders,
		         order_items, order_idempotency,
		         customer_cart, customer_favorites
		 RESTART IDENTITY CASCADE`)
	return err
}
```

- [ ] **Step 2: Delegate `resetTables` to the new method**

In `internal/store/harness_test.go`, replace the body of `resetTables` so the SQL lives in one place:

```go
func resetTables(t *testing.T, s *PostgresStore) {
	t.Helper()
	if err := s.ResetTestData(context.Background()); err != nil {
		t.Fatalf("reset tables: %v", err)
	}
}
```

(Keep the existing imports; `context` is already imported.)

- [ ] **Step 3: Verify the store package still builds and passes**

Run: `go test ./internal/store/... -race -count=1`
Expected: PASS (all existing store tests green — the refactor is behaviour-preserving). If `TEST_DATABASE_URL` is unset they SKIP; ensure it is set per preconditions.

- [ ] **Step 4: Commit**

```bash
git add internal/store/reset_testdata.go internal/store/harness_test.go
git commit -m "test(store): extract canonical ResetTestData for reuse

One source of truth for the test-data TRUNCATE list. resetTables now
delegates to it so the upcoming internal/storetest package can share it
without an import cycle.

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Task 2: `internal/storetest.Fresh`

**Files:**
- Create: `internal/storetest/storetest.go`

- [ ] **Step 1: Write the package**

Create `internal/storetest/storetest.go`:

```go
// Package storetest provides a shared test harness for code that needs a real
// PostgreSQL-backed store (e.g. handler integration tests). It lives in a
// non-_test package so it is importable from any test package; store's own
// in-package white-box tests cannot use it (import cycle) and keep their own
// testStore helper, which shares the same canonical reset via ResetTestData.
package storetest

import (
	"context"
	"os"
	"testing"

	"mioru/internal/store"
)

// Fresh connects to TEST_DATABASE_URL, runs migrations (schema + seeded
// category tree), truncates all data tables, and returns a clean store.
// It calls t.Skip when TEST_DATABASE_URL is unset, so suites are a no-op
// without a dedicated test database.
//
// TEST_DATABASE_URL must point at a disposable database — never production:
// Fresh truncates user data on every call.
func Fresh(t testing.TB) *store.PostgresStore {
	t.Helper()

	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL not set — skipping PostgreSQL integration tests")
	}

	s, err := store.NewPostgresStore(context.Background(), url)
	if err != nil {
		t.Fatalf("connect test store: %v", err)
	}
	t.Cleanup(s.Close)

	if err := s.ResetTestData(context.Background()); err != nil {
		t.Fatalf("reset test data: %v", err)
	}
	return s
}
```

- [ ] **Step 2: Verify it builds**

Run: `go build ./internal/storetest/...`
Expected: no output (builds clean).

- [ ] **Step 3: Commit**

```bash
git add internal/storetest/storetest.go
git commit -m "test(storetest): add Fresh helper for real-Postgres integration tests

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Task 3: Public API inventory document

**Files:**
- Create: `docs/api/public-api-inventory.md`

This is the coverage checklist. Build it by reading `backend/api/cmd/server/main.go` route registrations (lines 97–195) — do not invent routes.

- [ ] **Step 1: Write the inventory**

Create `docs/api/public-api-inventory.md` with this exact content:

````markdown
# Public API inventory

Source of truth: `backend/api/cmd/server/main.go`. Columns: auth (cookie required),
CSRF (mutation gate), RL (rate-limited), success code, key error codes, and the
integration test that covers it (or "store-level" / "fake-unit" where covered
elsewhere).

## Health
| Method+Path | Auth | CSRF | RL | Success | Errors | Integration test |
|---|---|---|---|---|---|---|
| GET /api/health | — | — | — | 200 `{status:ok}` | — | (trivial, skip) |

## Store — public catalog (no auth)
| Method+Path | Auth | CSRF | RL | Success | Errors | Integration test |
|---|---|---|---|---|---|---|
| GET /api/products | — | — | — | 200 `{items,total,page,per_page}` | — | storefront: list+paginate |
| GET /api/products/facets | — | — | — | 200 `{brands,colors,sizes}` | — | storefront: facets |
| GET /api/products/{slug} | — | — | — | 200 product | 404 NOT_FOUND | storefront: get + 404 |
| GET /api/categories | — | — | — | 200 tree | — | storefront: categories |

## Store — customer auth & profile
| Method+Path | Auth | CSRF | RL | Success | Errors | Integration test |
|---|---|---|---|---|---|---|
| POST /api/store/auth/register | — | — | ✓ | 200 customer | 400 VALIDATION_FAILED, 409 CONFLICT | storefront: register happy |
| POST /api/store/auth/login | — | — | ✓ | 200 customer | 401 AUTH_INVALID | storefront: login happy + bad creds |
| POST /api/store/auth/telegram | — | — | ✓ | 200 customer | 401 AUTH_INVALID | (telegram sig — store-level) |
| POST /api/store/auth/logout | ✓ | ✓ | — | 204/200 | 403 CSRF_INVALID | storefront: logout + CSRF gate |
| GET /api/store/customers/me | ✓ | — | — | 200 customer | 401 AUTH_REQUIRED | storefront: me + 401 |
| PUT /api/store/customers/me | ✓ | ✓ | — | 200 customer | 401/403 | storefront: profile update |
| PUT /api/store/customers/me/password | ✓ | ✓ | — | 200 | 400/401/403 | (covered by fake-unit; gate only) |
| POST /api/store/customers/me/set-password | ✓ | ✓ | — | 200 | 400/401/403 | (gate only) |
| POST /api/store/customers/me/oauth | ✓ | ✓ | — | 200 | 401/403/409 | (oauth verify — store-level) |
| GET /api/store/customers/me/orders | ✓ | — | — | 200 `{items,total,...}` | 401 | orders: ListOrders paginate+isolation |
| POST /api/store/orders | ✓ | ✓ | — | 201 order | 400 VALIDATION_FAILED, 409 IDEMPOTENCY_REPLAY, 409 INSUFFICIENT_STOCK | orders: full suite |
| POST /api/store/orders/upload-photo | ✓ | ✓ | — | 200 `{url}` | 400 | (multipart — out of base scope) |
| GET /api/store/customers/me/cart | ✓ | — | — | 200 cart | 401 | storefront: cart round-trip |
| PUT /api/store/customers/me/cart | ✓ | ✓ | — | 200 | 401/403 | storefront: cart round-trip |
| GET /api/store/customers/me/favorites | ✓ | — | — | 200 | 401 | storefront: favorites round-trip |
| PUT /api/store/customers/me/favorites | ✓ | ✓ | — | 200 | 401/403 | storefront: favorites round-trip |

## Admin — auth & profile
| Method+Path | Auth | CSRF | RL | Success | Errors | Integration test |
|---|---|---|---|---|---|---|
| POST /api/auth/register | ✓ super_admin | ✓ | — | 200 | 401/403 FORBIDDEN | admin: super-admin gate |
| POST /api/auth/login | — | — | ✓ | 200 user | 401 AUTH_INVALID | admin: login happy + bad creds |
| POST /api/auth/forgot-password | — | — | ✓ | 200 | — | (email — out of base scope) |
| POST /api/auth/reset-password | — | — | ✓ | 200 | 400 | (store-level) |
| POST /api/auth/logout | ✓ | ✓ | — | 204/200 | 403 CSRF_INVALID | admin: logout gate |
| GET /api/users/me | ✓ | — | — | 200 user | 401 AUTH_REQUIRED | admin: me + 401 |
| PUT /api/users/me/profile | ✓ | ✓ | — | 200 | 401/403 | (gate only) |
| PUT /api/users/me/password | ✓ | ✓ | — | 200 | 400/401/403 | (gate only) |

## Admin — resources (admin role, DB-checked)
| Method+Path | Auth | CSRF | RL | Success | Errors | Integration test |
|---|---|---|---|---|---|---|
| GET /api/admin/users | ✓ super_admin | — | — | 200 | 401/403 FORBIDDEN | admin: super-admin gate |
| DELETE /api/admin/users/{username} | ✓ super_admin | ✓ | — | 200 | 401/403 | admin: super-admin gate |
| GET /api/admin/categories | ✓ admin | — | — | 200 tree | 401/403 | admin: 403 customer-token |
| GET /api/admin/products | ✓ admin | — | — | 200 list | 401/403 | admin: CRUD + gates |
| POST /api/admin/products | ✓ admin | ✓ | — | 201/200 | 400/401/403 | admin: create (status canon) |
| GET /api/admin/products/{slug} | ✓ admin | — | — | 200 | 404 | admin: CRUD |
| PUT /api/admin/products/{slug} | ✓ admin | ✓ | — | 200 | 400/401/403/404 | admin: update |
| DELETE /api/admin/products/{slug} | ✓ admin | ✓ | — | 200 | 401/403/404 | admin: delete |
| GET /api/admin/orders | ✓ admin | — | — | 200 | 401/403 | orders: admin list |
| PATCH /api/admin/orders/{id}/status | ✓ admin | ✓ | — | 200 | 400/401/403/404 | orders: admin status update |
| POST /api/admin/upload | ✓ admin | ✓ | — | 200 `{url}` | 400 | (multipart — out of base scope) |

## Known contract notes (candidates to verify / file as issues)
- `INSUFFICIENT_STOCK` is returned by `CreateOrder` but is NOT in the CLAUDE.md reserved code list. Confirm intended; file an issue if it should be added to the contract.
- Middleware auth/CSRF 401/403 responses (`CustomerAuthMW`, `AuthMW`) currently use `http.Error` (`text/plain`, no machine `code`) per `internal/middleware/customer_auth.go`. CLAUDE.md requires the JSON envelope with `code` from middleware too. The gate tests below assert the status; if they reveal a `text/plain`/no-`code` body, file an issue (do NOT fix in this branch).
````

- [ ] **Step 2: Commit**

```bash
git add ../../docs/api/public-api-inventory.md
git commit -m "docs(api): add public API inventory + test-case matrix

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

(Path note: commands run from `backend/api/`; the doc is at repo `docs/api/`. Use the repo-root-relative path when committing, e.g. `cd /repo/root && git add docs/api/public-api-inventory.md`.)

---

## Task 4: Integration harness

**Files:**
- Create: `internal/handler/integration_harness_test.go`

This file holds NO `Test*` functions — only shared helpers used by Tasks 5–7. It is `package handler_test` (black-box).

- [ ] **Step 1: Write the harness**

Create `internal/handler/integration_harness_test.go`:

```go
package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"mioru/internal/auth"
	"mioru/internal/cookieauth"
	"mioru/internal/email"
	"mioru/internal/handler"
	"mioru/internal/middleware"
	"mioru/internal/model"
	"mioru/internal/store"
	"mioru/internal/storetest"
)

// testSecret is the HS256 secret used to mint tokens in integration tests.
const testSecret = "integration-test-secret-key-32chars-min"

const tokenExpiryMin = 1440

// env bundles the real store and the handlers under test, all sharing one
// throwaway database.
type env struct {
	st        *store.PostgresStore
	customerH *handler.CustomerHandler
	authH     *handler.AuthHandler
	productH  *handler.ProductHandler
	storeH    *handler.StoreHandler
	adminOrdH *handler.AdminOrderHandler
}

// newEnv builds an env on a fresh test database. Handlers are constructed with
// the same constructors main.go uses, with secure=false (dev), nil Telegram
// notifier (no network), and an in-memory email service.
func newEnv(t *testing.T) *env {
	t.Helper()
	st := storetest.Fresh(t)
	return &env{
		st:        st,
		customerH: handler.NewCustomerHandler(st, testSecret, tokenExpiryMin, false, "", "", t.TempDir(), nil),
		authH:     handler.NewAuthHandler(st, email.NewService(), testSecret, tokenExpiryMin, false, ""),
		productH:  handler.NewProductHandler(st, t.TempDir()),
		storeH:    handler.NewStoreHandler(st),
		adminOrdH: handler.NewAdminOrderHandler(st),
	}
}

// getRole resolves a user's role from the DB — mirrors main.go's getRole.
func (e *env) getRole(ctx context.Context, username string) (string, error) {
	u, err := e.st.GetUser(ctx, username)
	if err != nil {
		return "", err
	}
	if u == nil {
		return "", nil
	}
	return u.Role, nil
}

// --- middleware wrappers (mirror main.go's composition) ---

func (e *env) wrapCustomer(h http.HandlerFunc) http.Handler {
	mw := middleware.CustomerAuthMW(testSecret, e.st.CustomerPasswordChangedAt)
	csrf := middleware.CSRF(cookieauth.StoreCSRFCookie)
	return mw(csrf(h))
}

func (e *env) wrapAdmin(h http.HandlerFunc) http.Handler {
	mw := middleware.AuthMW(testSecret, e.st.UserPasswordChangedAt)
	csrf := middleware.CSRF(cookieauth.AdminCSRFCookie)
	return mw(middleware.RequireAdmin(e.getRole)(csrf(h)))
}

func (e *env) wrapSuperAdmin(h http.HandlerFunc) http.Handler {
	mw := middleware.AuthMW(testSecret, e.st.UserPasswordChangedAt)
	csrf := middleware.CSRF(cookieauth.AdminCSRFCookie)
	return mw(middleware.RequireSuperAdmin(e.getRole)(csrf(h)))
}

// --- session fixtures ---

type session struct {
	authCookie *http.Cookie
	csrfValue  string // value echoed in X-CSRF-Token and the csrf cookie
}

// customerSession creates a customer row and mints a real customer JWT cookie.
func (e *env) customerSession(t *testing.T, email string) (*session, int64) {
	t.Helper()
	ctx := context.Background()
	if err := e.st.CreateCustomer(ctx, model.Customer{Email: email, HashedPW: "x", FirstName: "T"}); err != nil {
		t.Fatalf("CreateCustomer: %v", err)
	}
	c, err := e.st.GetCustomerByEmail(ctx, email)
	if err != nil || c == nil {
		t.Fatalf("GetCustomerByEmail: %v / %v", c, err)
	}
	tok, err := auth.CreateToken(strconv.FormatInt(c.ID, 10), auth.TokenTypeCustomer, testSecret, tokenExpiryMin)
	if err != nil {
		t.Fatalf("CreateToken: %v", err)
	}
	return &session{
		authCookie: &http.Cookie{Name: cookieauth.StoreAuthCookie, Value: tok},
		csrfValue:  "csrf-customer-token",
	}, c.ID
}

// userSession creates an admin/staff user row and mints a real user JWT cookie.
func (e *env) userSession(t *testing.T, username, role string) *session {
	t.Helper()
	ctx := context.Background()
	if err := e.st.CreateUser(ctx, model.User{Username: username, Email: username + "@ex.com", HashedPW: "x", Role: role}); err != nil {
		t.Fatalf("CreateUser: %v", err)
	}
	tok, err := auth.CreateToken(username, auth.TokenTypeUser, testSecret, tokenExpiryMin)
	if err != nil {
		t.Fatalf("CreateToken: %v", err)
	}
	return &session{
		authCookie: &http.Cookie{Name: cookieauth.AdminAuthCookie, Value: tok},
		csrfValue:  "csrf-admin-token",
	}
}

// --- request driver ---

type reqOpts struct {
	sess           *session
	csrfCookieName string            // StoreCSRFCookie/AdminCSRFCookie — sends the CSRF cookie AND a matching X-CSRF-Token header (a valid mutation)
	badCSRF        bool              // with csrfCookieName set: sends the cookie but a wrong X-CSRF-Token header (mismatch → expect 403)
	idempotencyKey string
	body           any
	pathValues     map[string]string // applied via req.SetPathValue (for {slug}/{id} routes)
}

// do builds a request, applies cookies/headers per opts, runs it through h via
// a recorder, and returns the recorded result.
func (e *env) do(t *testing.T, h http.Handler, method, target string, o reqOpts) *httptest.ResponseRecorder {
	t.Helper()
	var bodyReader *bytes.Reader
	if o.body != nil {
		b, err := json.Marshal(o.body)
		if err != nil {
			t.Fatalf("marshal body: %v", err)
		}
		bodyReader = bytes.NewReader(b)
	} else {
		bodyReader = bytes.NewReader(nil)
	}
	req := httptest.NewRequest(method, target, bodyReader)
	if o.body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if o.sess != nil {
		req.AddCookie(o.sess.authCookie)
		if o.csrfCookieName != "" {
			req.AddCookie(&http.Cookie{Name: o.csrfCookieName, Value: o.sess.csrfValue})
			if o.badCSRF {
				req.Header.Set("X-CSRF-Token", "wrong-value")
			} else {
				req.Header.Set("X-CSRF-Token", o.sess.csrfValue)
			}
		}
	}
	if o.idempotencyKey != "" {
		req.Header.Set("Idempotency-Key", o.idempotencyKey)
	}
	for k, v := range o.pathValues {
		req.SetPathValue(k, v)
	}
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	return rr
}

// decode unmarshals the recorder body into v.
func decode(t *testing.T, rr *httptest.ResponseRecorder, v any) {
	t.Helper()
	if err := json.Unmarshal(rr.Body.Bytes(), v); err != nil {
		t.Fatalf("decode body %q: %v", rr.Body.String(), err)
	}
}

// errEnvelope is the standard error response shape.
type errEnvelope struct {
	Error string `json:"error"`
	Code  string `json:"code"`
}
```

- [ ] **Step 2: Verify the harness compiles**

Run: `go vet ./internal/handler/...`
Expected: no errors. If `NewCustomerHandler` / `NewAuthHandler` / `NewProductHandler` / `NewStoreHandler` / `NewAdminOrderHandler` argument lists differ from the calls above, read each constructor signature in `internal/handler/*.go` and adjust the call — do not change the constructors.

> **Constructor signature check (verify before relying on the code above):** the `newEnv` calls mirror `cmd/server/main.go:56–61`. If main.go changed, copy the exact argument order from there.

- [ ] **Step 3: Add a smoke test to prove sessions work end-to-end**

Append to `integration_harness_test.go`:

```go
// TestHarnessCustomerSessionAuthenticates is a smoke test: a minted customer
// cookie must pass CustomerAuthMW and reach the handler (GET me → 200).
func TestHarnessCustomerSessionAuthenticates(t *testing.T) {
	e := newEnv(t)
	sess, _ := e.customerSession(t, "smoke@ex.com")
	rr := e.do(t, e.wrapCustomer(e.customerH.Me), http.MethodGet, "/api/store/customers/me", reqOpts{sess: sess})
	if rr.Code != http.StatusOK {
		t.Fatalf("authenticated GET me: want 200, got %d (body %q)", rr.Code, rr.Body.String())
	}
}
```

- [ ] **Step 4: Run the smoke test**

Run: `go test ./internal/handler/ -run TestHarnessCustomerSession -race -count=1 -v`
Expected: PASS. A 401 here means the minted token's `iat` is rejected by the password-changed-at epoch check — if so, inspect `CustomerPasswordChangedAt` for a freshly created customer; if the epoch is set to a time AFTER the token `iat`, that is a real finding → file an issue and note it before continuing.

- [ ] **Step 5: Commit**

```bash
git add internal/handler/integration_harness_test.go
git commit -m "test(handler): add real-Postgres integration harness

Black-box harness: real store via storetest.Fresh, real middleware chain
wrappers mirroring main.go, minted JWT cookies, and a request driver.

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Task 5: Storefront integration tests

**Files:**
- Create: `internal/handler/integration_storefront_test.go`

- [ ] **Step 1: Write a shared product fixture (top of file) + public catalog tests**

Create `internal/handler/integration_storefront_test.go`:

```go
package handler_test

import (
	"context"
	"net/http"
	"testing"

	"mioru/internal/model"
)

// seedProduct inserts an active product with stock and one size via the public
// store API (works from a black-box package — no pool access needed).
func seedProduct(t *testing.T, e *env, slug string, priceMDL, stock int) int64 {
	t.Helper()
	id, err := e.st.CreateProduct(context.Background(), model.Product{
		Slug: slug, CategoryID: 1, Brand: "TestBrand", Name: "Test " + slug,
		Price: priceMDL, Color: "red", Status: "active", InStock: true,
		StockQty: stock, CreatedBy: "test", Sizes: []string{"M", "L"},
	})
	if err != nil {
		t.Fatalf("seedProduct %s: %v", slug, err)
	}
	return id
}

func TestIntegrationListProducts(t *testing.T) {
	e := newEnv(t)
	seedProduct(t, e, "p-1", 500, 10)
	seedProduct(t, e, "p-2", 300, 10)

	rr := e.do(t, http.HandlerFunc(e.storeH.ListProducts), http.MethodGet, "/api/products?per_page=20", reqOpts{})
	if rr.Code != http.StatusOK {
		t.Fatalf("ListProducts: want 200, got %d (%s)", rr.Code, rr.Body.String())
	}
	var resp struct {
		Items   []model.Product `json:"items"`
		Total   int             `json:"total"`
		Page    int             `json:"page"`
		PerPage int             `json:"per_page"`
	}
	decode(t, rr, &resp)
	if resp.Total != 2 || len(resp.Items) != 2 {
		t.Errorf("want total=2 items=2, got total=%d items=%d", resp.Total, len(resp.Items))
	}
}

func TestIntegrationGetProductBySlug(t *testing.T) {
	e := newEnv(t)
	seedProduct(t, e, "shirt-1", 500, 10)

	rr := e.do(t, http.HandlerFunc(e.storeH.GetProduct), http.MethodGet, "/api/products/shirt-1",
		reqOpts{pathValues: map[string]string{"slug": "shirt-1"}})
	if rr.Code != http.StatusOK {
		t.Fatalf("GetProduct: want 200, got %d (%s)", rr.Code, rr.Body.String())
	}
	var p model.Product
	decode(t, rr, &p)
	if p.Slug != "shirt-1" {
		t.Errorf("slug = %q, want shirt-1", p.Slug)
	}
}

func TestIntegrationGetProductNotFound(t *testing.T) {
	e := newEnv(t)
	rr := e.do(t, http.HandlerFunc(e.storeH.GetProduct), http.MethodGet, "/api/products/does-not-exist",
		reqOpts{pathValues: map[string]string{"slug": "does-not-exist"}})
	if rr.Code != http.StatusNotFound {
		t.Errorf("missing product: want 404, got %d (%s)", rr.Code, rr.Body.String())
	}
}

func TestIntegrationListCategories(t *testing.T) {
	e := newEnv(t)
	rr := e.do(t, http.HandlerFunc(e.storeH.ListCategories), http.MethodGet, "/api/categories", reqOpts{})
	if rr.Code != http.StatusOK {
		t.Fatalf("ListCategories: want 200, got %d (%s)", rr.Code, rr.Body.String())
	}
	var cats []model.Category
	decode(t, rr, &cats)
	if len(cats) == 0 {
		t.Error("expected seeded category tree, got 0 categories")
	}
}
```

> **Note:** `storeH.GetProduct` reads `r.PathValue("slug")`. `httptest.NewRequest` does NOT populate path values from the URL automatically, so pass them explicitly via `reqOpts.pathValues` (the harness applies them with `req.SetPathValue`). Update the two slug-routed calls below to add `pathValues: map[string]string{"slug": "shirt-1"}` (and `"does-not-exist"`). The list/categories calls need no path values.

- [ ] **Step 2: Run catalog tests**

Run: `go test ./internal/handler/ -run TestIntegration -race -count=1 -v`
Expected: PASS. If slug-routed tests fail on path values, apply the `SetPathValue` fix from the note above.

- [ ] **Step 3: Add customer auth + cart/favorites round-trip tests**

Append to `integration_storefront_test.go`:

```go
func TestIntegrationCustomerMeRequiresAuth(t *testing.T) {
	e := newEnv(t)
	// No session cookie → CustomerAuthMW must reject.
	rr := e.do(t, e.wrapCustomer(e.customerH.Me), http.MethodGet, "/api/store/customers/me", reqOpts{})
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("unauthenticated me: want 401, got %d (%s)", rr.Code, rr.Body.String())
	}
}

func TestIntegrationCartRoundTrip(t *testing.T) {
	e := newEnv(t)
	sess, _ := e.customerSession(t, "cart@ex.com")

	// Save cart (mutation → needs CSRF cookie + header).
	saveBody := map[string]any{"items": []map[string]any{{"product_id": 1, "size_label": "M", "quantity": 2}}}
	rr := e.do(t, e.wrapCustomer(e.customerH.SaveCart), http.MethodPut, "/api/store/customers/me/cart",
		reqOpts{sess: sess, csrfCookieName: "store_csrf", body: saveBody})
	if rr.Code != http.StatusOK {
		t.Fatalf("SaveCart: want 200, got %d (%s)", rr.Code, rr.Body.String())
	}

	// Get cart back.
	rr2 := e.do(t, e.wrapCustomer(e.customerH.GetCart), http.MethodGet, "/api/store/customers/me/cart", reqOpts{sess: sess})
	if rr2.Code != http.StatusOK {
		t.Fatalf("GetCart: want 200, got %d (%s)", rr2.Code, rr2.Body.String())
	}
	// Assert the saved item survived the round-trip.
	if !contains(rr2.Body.String(), `"product_id":1`) {
		t.Errorf("cart round-trip lost the item: %s", rr2.Body.String())
	}
}

func TestIntegrationSaveCartCSRFGate(t *testing.T) {
	e := newEnv(t)
	sess, _ := e.customerSession(t, "csrf@ex.com")
	// Mutation with a wrong CSRF token → 403.
	rr := e.do(t, e.wrapCustomer(e.customerH.SaveCart), http.MethodPut, "/api/store/customers/me/cart",
		reqOpts{sess: sess, csrfCookieName: "store_csrf", badCSRF: true, body: map[string]any{"items": []any{}}})
	if rr.Code != http.StatusForbidden {
		t.Errorf("bad CSRF: want 403, got %d (%s)", rr.Code, rr.Body.String())
	}
}
```

Add this helper at the bottom of the file (or reuse `strings.Contains` directly — if so, import `strings` and drop this):

```go
func contains(haystack, needle string) bool { return bytes.Contains([]byte(haystack), []byte(needle)) }
```

(If you use `bytes.Contains`, add `"bytes"` to the imports; otherwise prefer `strings.Contains` inline.)

- [ ] **Step 4: Run storefront tests**

Run: `go test ./internal/handler/ -run TestIntegration -race -count=1 -v`
Expected: PASS. Investigate any failure per the bug-handling rule (Task 8). If the CSRF/auth gate returns `text/plain` without a `code`, that matches the inventory's known-note — file an issue, keep the status assertion.

- [ ] **Step 5: Commit**

```bash
git add internal/handler/integration_storefront_test.go
git commit -m "test(handler): storefront integration tests (catalog, auth, cart)

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Task 6: Orders integration tests (end-to-end financial invariants)

**Files:**
- Create: `internal/handler/integration_orders_test.go`

These exercise the REAL store — distinct from the existing fake-store branch-logic unit tests.

- [ ] **Step 1: Write the order body builder + happy path + stock decrement**

Create `internal/handler/integration_orders_test.go`:

```go
package handler_test

import (
	"context"
	"net/http"
	"testing"
)

// orderBody returns a valid cart-order request body for the given product/qty.
func orderBody(productID int64, qty int) map[string]any {
	return map[string]any{
		"type":            "cart",
		"city":            "Tiraspol",
		"delivery_method": "personal",
		"payment_method":  "cash",
		"items":           []map[string]any{{"product_id": productID, "size_label": "M", "quantity": qty}},
	}
}

func TestIntegrationCreateOrderHappyAndDecrementsStock(t *testing.T) {
	e := newEnv(t)
	sess, _ := e.customerSession(t, "order@ex.com")
	pid := seedProduct(t, e, "ord-1", 500, 10) // price 500 MDL, stock 10

	rr := e.do(t, e.wrapCustomer(e.customerH.CreateOrder), http.MethodPost, "/api/store/orders",
		reqOpts{sess: sess, csrfCookieName: "store_csrf", idempotencyKey: "key-happy-1", body: orderBody(pid, 3)})
	if rr.Code != http.StatusCreated {
		t.Fatalf("CreateOrder: want 201, got %d (%s)", rr.Code, rr.Body.String())
	}
	var order struct {
		ID         int64 `json:"id"`
		TotalMinor int64 `json:"total_minor"`
	}
	decode(t, rr, &order)
	if order.ID == 0 {
		t.Error("created order has no ID")
	}
	// Server-side price: 3 * 500 * 100 = 150000 minor.
	if order.TotalMinor != 150000 {
		t.Errorf("TotalMinor = %d, want 150000 (server-calculated)", order.TotalMinor)
	}
	// Stock decremented in the real DB: 10 - 3 = 7.
	p, err := e.st.GetProduct(context.Background(), "ord-1")
	if err != nil || p == nil {
		t.Fatalf("GetProduct: %v / %v", p, err)
	}
	if p.StockQty != 7 {
		t.Errorf("stock after order = %d, want 7", p.StockQty)
	}
}

func TestIntegrationCreateOrderMissingIdempotencyKey(t *testing.T) {
	e := newEnv(t)
	sess, _ := e.customerSession(t, "noidem@ex.com")
	pid := seedProduct(t, e, "ord-noidem", 500, 10)

	rr := e.do(t, e.wrapCustomer(e.customerH.CreateOrder), http.MethodPost, "/api/store/orders",
		reqOpts{sess: sess, csrfCookieName: "store_csrf", body: orderBody(pid, 1)})
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("missing Idempotency-Key: want 400, got %d (%s)", rr.Code, rr.Body.String())
	}
	var env errEnvelope
	decode(t, rr, &env)
	if env.Code != "VALIDATION_FAILED" {
		t.Errorf("code = %q, want VALIDATION_FAILED", env.Code)
	}
}
```

- [ ] **Step 2: Run them**

Run: `go test ./internal/handler/ -run TestIntegrationCreateOrder -race -count=1 -v`
Expected: PASS.

- [ ] **Step 3: Add idempotency replay (same key+body → same order, stock once)**

Append to `integration_orders_test.go`:

```go
func TestIntegrationCreateOrderIdempotentReplay(t *testing.T) {
	e := newEnv(t)
	sess, _ := e.customerSession(t, "replay@ex.com")
	pid := seedProduct(t, e, "ord-replay", 500, 10)
	body := orderBody(pid, 2)
	const key = "replay-key-1"

	first := e.do(t, e.wrapCustomer(e.customerH.CreateOrder), http.MethodPost, "/api/store/orders",
		reqOpts{sess: sess, csrfCookieName: "store_csrf", idempotencyKey: key, body: body})
	if first.Code != http.StatusCreated {
		t.Fatalf("first submit: want 201, got %d (%s)", first.Code, first.Body.String())
	}
	var o1 struct{ ID int64 `json:"id"` }
	decode(t, first, &o1)

	// Replay: same key + same body → same order, NOT a new one.
	second := e.do(t, e.wrapCustomer(e.customerH.CreateOrder), http.MethodPost, "/api/store/orders",
		reqOpts{sess: sess, csrfCookieName: "store_csrf", idempotencyKey: key, body: body})
	if second.Code != http.StatusCreated {
		t.Fatalf("replay: want 201, got %d (%s)", second.Code, second.Body.String())
	}
	var o2 struct{ ID int64 `json:"id"` }
	decode(t, second, &o2)
	if o1.ID != o2.ID {
		t.Errorf("replay created a new order: first=%d second=%d", o1.ID, o2.ID)
	}
	// Stock decremented exactly once: 10 - 2 = 8.
	p, _ := e.st.GetProduct(context.Background(), "ord-replay")
	if p.StockQty != 8 {
		t.Errorf("stock after replay = %d, want 8 (decremented once)", p.StockQty)
	}
}

func TestIntegrationCreateOrderIdempotencyConflict(t *testing.T) {
	e := newEnv(t)
	sess, _ := e.customerSession(t, "conflict@ex.com")
	pid := seedProduct(t, e, "ord-conflict", 500, 10)
	const key = "conflict-key-1"

	first := e.do(t, e.wrapCustomer(e.customerH.CreateOrder), http.MethodPost, "/api/store/orders",
		reqOpts{sess: sess, csrfCookieName: "store_csrf", idempotencyKey: key, body: orderBody(pid, 1)})
	if first.Code != http.StatusCreated {
		t.Fatalf("first submit: want 201, got %d (%s)", first.Code, first.Body.String())
	}
	// Same key, DIFFERENT body → 409 IDEMPOTENCY_REPLAY.
	second := e.do(t, e.wrapCustomer(e.customerH.CreateOrder), http.MethodPost, "/api/store/orders",
		reqOpts{sess: sess, csrfCookieName: "store_csrf", idempotencyKey: key, body: orderBody(pid, 5)})
	if second.Code != http.StatusConflict {
		t.Fatalf("idempotency conflict: want 409, got %d (%s)", second.Code, second.Body.String())
	}
	var env errEnvelope
	decode(t, second, &env)
	if env.Code != "IDEMPOTENCY_REPLAY" {
		t.Errorf("code = %q, want IDEMPOTENCY_REPLAY", env.Code)
	}
}

func TestIntegrationCreateOrderOversell(t *testing.T) {
	e := newEnv(t)
	sess, _ := e.customerSession(t, "oversell@ex.com")
	pid := seedProduct(t, e, "ord-oversell", 500, 2) // only 2 in stock

	rr := e.do(t, e.wrapCustomer(e.customerH.CreateOrder), http.MethodPost, "/api/store/orders",
		reqOpts{sess: sess, csrfCookieName: "store_csrf", idempotencyKey: "oversell-1", body: orderBody(pid, 5)})
	if rr.Code != http.StatusConflict {
		t.Fatalf("oversell: want 409, got %d (%s)", rr.Code, rr.Body.String())
	}
	var env errEnvelope
	decode(t, rr, &env)
	if env.Code != "INSUFFICIENT_STOCK" {
		t.Errorf("code = %q, want INSUFFICIENT_STOCK", env.Code)
	}
	// Stock must be untouched after a rejected oversell.
	p, _ := e.st.GetProduct(context.Background(), "ord-oversell")
	if p.StockQty != 2 {
		t.Errorf("stock after rejected oversell = %d, want 2 (unchanged)", p.StockQty)
	}
}
```

- [ ] **Step 4: Add ListOrders isolation + admin status update**

Append to `integration_orders_test.go`:

```go
func TestIntegrationListOrdersIsolation(t *testing.T) {
	e := newEnv(t)
	alice, aID := e.customerSession(t, "alice-ord@ex.com")
	_, _ = e.customerSession(t, "bob-ord@ex.com") // bob exists but places no orders
	pid := seedProduct(t, e, "ord-list", 500, 100)

	// Alice places 2 orders.
	for i, k := range []string{"a-1", "a-2"} {
		rr := e.do(t, e.wrapCustomer(e.customerH.CreateOrder), http.MethodPost, "/api/store/orders",
			reqOpts{sess: alice, csrfCookieName: "store_csrf", idempotencyKey: k, body: orderBody(pid, 1)})
		if rr.Code != http.StatusCreated {
			t.Fatalf("alice order %d: want 201, got %d (%s)", i, rr.Code, rr.Body.String())
		}
	}

	rr := e.do(t, e.wrapCustomer(e.customerH.ListOrders), http.MethodGet, "/api/store/customers/me/orders?per_page=20", reqOpts{sess: alice})
	if rr.Code != http.StatusOK {
		t.Fatalf("ListOrders: want 200, got %d (%s)", rr.Code, rr.Body.String())
	}
	var resp struct {
		Total int `json:"total"`
		Items []struct {
			CustomerID int64 `json:"customer_id"`
		} `json:"items"`
	}
	decode(t, rr, &resp)
	if resp.Total != 2 {
		t.Errorf("alice total = %d, want 2", resp.Total)
	}
	for _, o := range resp.Items {
		if o.CustomerID != aID {
			t.Errorf("ListOrders leaked another customer's order: got customer_id %d, want %d", o.CustomerID, aID)
		}
	}
}

func TestIntegrationAdminUpdateOrderStatus(t *testing.T) {
	e := newEnv(t)
	cust, _ := e.customerSession(t, "statuscust@ex.com")
	pid := seedProduct(t, e, "ord-status", 500, 10)
	createRR := e.do(t, e.wrapCustomer(e.customerH.CreateOrder), http.MethodPost, "/api/store/orders",
		reqOpts{sess: cust, csrfCookieName: "store_csrf", idempotencyKey: "status-1", body: orderBody(pid, 1)})
	if createRR.Code != http.StatusCreated {
		t.Fatalf("seed order: want 201, got %d (%s)", createRR.Code, createRR.Body.String())
	}
	var created struct{ ID int64 `json:"id"` }
	decode(t, createRR, &created)

	admin := e.userSession(t, "admin1", "admin")
	target := "/api/admin/orders/" + itoa(created.ID) + "/status"
	rr := e.do(t, e.wrapAdmin(e.adminOrdH.UpdateStatus), http.MethodPatch, target,
		reqOpts{sess: admin, csrfCookieName: "csrf_token",
			pathValues: map[string]string{"id": itoa(created.ID)}, body: map[string]any{"status": "processing"}})
	if rr.Code != http.StatusOK {
		t.Fatalf("UpdateStatus: want 200, got %d (%s)", rr.Code, rr.Body.String())
	}
}
```

> **Required helper for this task:** add a tiny `itoa` helper near the top of the orders test file (the admin status handler reads `r.PathValue("id")`, supplied via `reqOpts.pathValues` which the harness already applies):
> ```go
> import "strconv"
> func itoa(n int64) string { return strconv.FormatInt(n, 10) }
> ```

- [ ] **Step 5: Run the full orders suite**

Run: `go test ./internal/handler/ -run TestIntegration -race -count=1 -v`
Expected: PASS. Apply the bug-handling rule (Task 8) to any failure that reflects real product behaviour.

- [ ] **Step 6: Commit**

```bash
git add internal/handler/integration_orders_test.go internal/handler/integration_harness_test.go
git commit -m "test(handler): orders integration tests (end-to-end stock, idempotency)

Real-Postgres CreateOrder: server-side pricing, stock decrement, idempotent
replay returns the same order with stock decremented once, hash-mismatch
conflict, oversell, list isolation, admin status update.

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Task 7: Admin integration tests (CRUD + gates)

**Files:**
- Create: `internal/handler/integration_admin_test.go`

- [ ] **Step 1: Write product CRUD happy path + auth/role/CSRF gates**

Create `internal/handler/integration_admin_test.go`:

```go
package handler_test

import (
	"net/http"
	"testing"
)

func TestIntegrationAdminCreateAndGetProduct(t *testing.T) {
	e := newEnv(t)
	admin := e.userSession(t, "admincrud", "admin")

	create := map[string]any{
		"slug": "admin-prod-1", "category_id": 1, "brand": "B", "name": "N",
		"price": 1000, "color": "blue", "status": "active", "stock_quantity": 5,
		"sizes": []string{"M"},
	}
	rr := e.do(t, e.wrapAdmin(e.productH.Create), http.MethodPost, "/api/admin/products",
		reqOpts{sess: admin, csrfCookieName: "csrf_token", body: create})
	if rr.Code != http.StatusCreated && rr.Code != http.StatusOK {
		t.Fatalf("admin create product: want 201/200, got %d (%s)", rr.Code, rr.Body.String())
	}

	// Read it back via the admin Get handler.
	rr2 := e.do(t, e.wrapAdmin(e.productH.Get), http.MethodGet, "/api/admin/products/admin-prod-1",
		reqOpts{sess: admin, pathValues: map[string]string{"slug": "admin-prod-1"}})
	if rr2.Code != http.StatusOK {
		t.Fatalf("admin get product: want 200, got %d (%s)", rr2.Code, rr2.Body.String())
	}
}

func TestIntegrationAdminRouteRejectsNoSession(t *testing.T) {
	e := newEnv(t)
	rr := e.do(t, e.wrapAdmin(e.productH.List), http.MethodGet, "/api/admin/products", reqOpts{})
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("admin route, no session: want 401, got %d (%s)", rr.Code, rr.Body.String())
	}
}

func TestIntegrationAdminRouteRejectsCustomerToken(t *testing.T) {
	e := newEnv(t)
	// A customer session must not satisfy an admin route: the user JWT typ is
	// "customer", so AuthMW (which wants typ "user") rejects it as 401.
	cust, _ := e.customerSession(t, "sneaky@ex.com")
	// Reuse the customer auth cookie but on an admin-wrapped handler. The admin
	// wrapper reads the admin cookie name, so present it under that name to
	// prove the typ check (not just the cookie name) is what rejects.
	adminShaped := &session{
		authCookie: &http.Cookie{Name: "auth_token", Value: cust.authCookie.Value},
		csrfValue:  cust.csrfValue,
	}
	rr := e.do(t, e.wrapAdmin(e.productH.List), http.MethodGet, "/api/admin/products", reqOpts{sess: adminShaped})
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("admin route with customer-typ token: want 401, got %d (%s)", rr.Code, rr.Body.String())
	}
}

func TestIntegrationAdminCreateProductCSRFGate(t *testing.T) {
	e := newEnv(t)
	admin := e.userSession(t, "admincsrf", "admin")
	rr := e.do(t, e.wrapAdmin(e.productH.Create), http.MethodPost, "/api/admin/products",
		reqOpts{sess: admin, csrfCookieName: "csrf_token", badCSRF: true, body: map[string]any{"slug": "x"}})
	if rr.Code != http.StatusForbidden {
		t.Errorf("admin create, bad CSRF: want 403, got %d (%s)", rr.Code, rr.Body.String())
	}
}

func TestIntegrationNonSuperAdminCannotListUsers(t *testing.T) {
	e := newEnv(t)
	admin := e.userSession(t, "plainadmin", "admin") // admin, NOT super_admin
	rr := e.do(t, e.wrapSuperAdmin(e.authH.ListUsers), http.MethodGet, "/api/admin/users", reqOpts{sess: admin})
	if rr.Code != http.StatusForbidden {
		t.Errorf("admin (not super) listing users: want 403, got %d (%s)", rr.Code, rr.Body.String())
	}
}
```

> **Constructor/handler-name check:** verify `productH.Create`, `productH.Get`, `productH.List`, `authH.ListUsers`, `adminOrdH.UpdateStatus` are the real exported method names (they match `main.go` route registrations). If `Create` returns 200 (not 201), the assertion already accepts both.

- [ ] **Step 2: Run the admin suite**

Run: `go test ./internal/handler/ -run TestIntegration -race -count=1 -v`
Expected: PASS. Any gate that returns the wrong status or a `text/plain` envelope → Task 8.

- [ ] **Step 3: Commit**

```bash
git add internal/handler/integration_admin_test.go
git commit -m "test(handler): admin integration tests (product CRUD + auth/CSRF/role gates)

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Task 8: Full verification + file issues for real findings

**Files:** none (verification + issue triage)

- [ ] **Step 1: Run the whole backend suite with the race detector**

Run: `go test ./... -race -count=1`
Expected: PASS (or SKIP where `TEST_DATABASE_URL` is unset — but it is set per preconditions).

- [ ] **Step 2: vet + build**

Run: `go vet ./... && go build ./...`
Expected: clean.

- [ ] **Step 3: Triage every failure/finding**

For each test that revealed PRODUCT behaviour at odds with the contract (not a test bug):
- Decide severity. Security gate / financial invariant → keep the test asserting CORRECT behaviour, mark it `t.Skip("blocked by #N: <reason>")`, and file the issue. Cosmetic contract drift (e.g. middleware `http.Error` body without `code`) → assert the ACTUAL behaviour with a `// FIXME(#N)` comment, and file the issue.
- File with:

```bash
gh issue create --title "<route>: <expected> vs <actual>" \
  --body "Found by integration test \`<TestName>\` (internal/handler/<file>).

Expected (per CLAUDE.md API contract): <...>
Actual: <...>
Severity: <security|finance|cosmetic>
Repro: go test ./internal/handler/ -run <TestName>"
```
- Record the issue number in `docs/api/public-api-inventory.md` under "Known contract notes".

- [ ] **Step 4: Commit any inventory updates / skips**

```bash
git add -A
git commit -m "test(handler): annotate integration findings with filed issues

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

- [ ] **Step 5: Push the branch and open the PR (user merges)**

```bash
git push -u origin feat/api-integration-tests
gh pr create --base main --title "test: public API integration tests + inventory" \
  --body "Adds handler-level integration tests against a real throwaway Postgres,
driving real handlers through the real middleware chain. New: internal/storetest
harness, public API inventory, storefront/orders/admin suites. References any
issues filed for findings.

🤖 Generated with [Claude Code](https://claude.com/claude-code)"
```

(Per CLAUDE.md the agent pushes the branch and opens the PR but never merges — the user reviews and merges.)

---

## Self-Review

**Spec coverage:**
- Component 1 (`ResetTestData` + delegation) → Task 1. ✓
- Component 2 (`storetest.Fresh`) → Task 2. ✓
- Component 3 (harness + middleware wrappers + sessions + `do`) → Task 4. ✓
- Component 4 (inventory) → Task 3. ✓
- Test files: storefront → Task 5, orders → Task 6, admin → Task 7. ✓
- Financial invariants end-to-end (stock decrement, replay same-order, conflict, oversell, missing key) → Task 6. ✓
- Auth/CSRF/role gates → Tasks 5 (customer CSRF/401) + 7 (admin 401/403/CSRF/super-admin). ✓
- Bug → issue process → Task 8. ✓
- Error envelope `{error,code}` assertions → Tasks 6 (codes) + 7 (gates). ✓

**Placeholder scan:** No TBD/TODO. Two explicit "verify the signature against main.go" notes (constructors, handler method names) are deliberate guardrails, not placeholders — the code to run is fully written; the note says where to look if a signature drifted.

**Type consistency:** `session{authCookie, csrfValue}`, `reqOpts{sess, csrfCookieName, badCSRF, idempotencyKey, body, pathValues}`, `env{st, customerH, authH, productH, storeH, adminOrdH}` used consistently. `pathValues` is introduced in Task 6's harness-addition note and reused in Task 7 — both reference the same field. `seedProduct(t, e, slug, price, stock)` defined in Task 5, reused in Task 6. `orderBody`, `itoa`, `decode`, `errEnvelope`, `contains` defined once and reused.

**Known cross-task dependency:** `reqOpts.pathValues` (used by Tasks 5, 6, 7 for `{slug}`/`{id}` routes) is defined in the Task 4 harness from the start, so tasks can run in order with no retro-fit.
