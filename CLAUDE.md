# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository. It is the **single source of truth** for agents — `AGENTS.md` now just points here.

## Priorities (in order)

When trade-offs collide, resolve them top-down:

1. **Reliability of financial operations** — orders, payments, stock, XP/VIP accrual must never silently lose or double-count. Prefer correctness and atomicity over convenience.
2. **Safety of user data** — personal data and credentials. Security-by-default, least privilege, nothing sensitive in logs or responses.
3. **Speed / performance** — pursue only after 1 and 2 hold.

## Development principles (enforced)

- **TDD** — write a failing test, make it pass, refactor. New logic ships with tests; verify a module's tests before wiring it into the rest. (The Go side has no tests yet — start the habit.)
- **YAGNI** — build what the current task needs, not speculative abstractions.
- **Don't reinvent the wheel** — for a *solved, non-domain* concern (schema migrations, JWT, password hashing, SQL driver, validation, parsing), adopt a mature, well-maintained library that fits the stack instead of hand-rolling one. Prefer the option that adds least to the stack and integrates cleanly (e.g. a pgx-native tool over one needing a `database/sql` shim). This is not a license to pile on dependencies — weigh each against the minimalist baseline (stdlib `net/http`, pgx-only, no framework) — but a proven solution beats bespoke for anything that isn't MIORU's own logic. Write custom code for what's genuinely unique to MIORU.
- **DRY** — one source of truth per fact (config, types, the category tree). Reference, don't duplicate.
- **OWASP / security-by-default** — validate all input, parameterize all SQL, never reflect internal errors to clients, enforce authz on every privileged route. See "Security posture".
- **Input validation** — validate every incoming argument before use: type, presence, and bounds. For strings, always enforce a maximum length (and minimum/format where it matters) — never persist or process unbounded user-supplied strings. Fail early with a generic 4xx; never rely on client-side validation as the enforcement point.
- **Modular decomposition** — keep concerns in their package (`auth`, `store`, `handler`, `middleware`, `config`, `model`); one responsibility per file/handler.
- **Architectural integrity** — respect layer boundaries (handler → store → DB; handlers never touch the pool directly). No import cycles. When a change crosses a boundary, flag it before doing it.

## Workflow rule (hard)

Every change follows: **plan → user approves → make change → user checks → user says commit → agent commits.**
No commits without explicit approval. No code changes before presenting a plan. Remote is HTTPS (`https://github.com/rem22p/mioru`) with a stored token, so the agent *can* push non-interactively (`GIT_TERMINAL_PROMPT=0 git push origin main`), but does NOT push by default — same approval gate as commit. Push only when the user explicitly says so, or under an explicit batch waiver (e.g. an autonomous run the user pre-authorized for a specific scope).

## What this is

MIORU — virtual clothing try-on with a 3D avatar. A monorepo with two React SPAs and one Go API:

```
apps/store/    Storefront SPA (React 19 + Vite), port 5173 — public, served as static/CDN
apps/admin/    Admin panel SPA (React 19 + Vite), port 5174 — product management, auth
backend/api/   Go API (net/http + PostgreSQL + HttpOnly-cookie JWT + CSRF), port 8000
scripts/       VPS provisioning + DB seeding
```

`packages/shared/` is referenced in old docs but does not exist yet. Historical note: the project once used Redis and a notes/WebSocket feature — **both removed**; PostgreSQL is now the only datastore.

## Commands

Frontends (`apps/store/` and `apps/admin/` are independent npm projects — run from each dir):

```bash
npm install
npm run dev            # store→5173, admin→5174 (both bind 0.0.0.0)
npm run build          # tsc -b && vite build → dist/
npm run lint           # eslint
npm test               # vitest (watch)
npx vitest run path/to/file.test.tsx   # single unit test file
```

Store also has Playwright E2E (screenshot tests):

```bash
cd apps/store
npm run test:e2e
npx playwright test e2e/some.spec.ts   # single E2E file
```

Backend (Go 1.25, requires a running **PostgreSQL**):

```bash
cd backend/api
go run ./cmd/server      # builds binary mioru/server (gitignored)
go build ./...
```

Run Go tests (see "Testing standard" for the test-DB convention):

```bash
go test ./... -race
```

Seed test data (generates SQL on stdout, pipe into psql):

```bash
bash scripts/seed-300.sh | psql "$DATABASE_URL"   # 300 random products
```

## Backend architecture

- **Routing:** Go 1.25 stdlib `http.ServeMux` with method+pattern routes (`"GET /api/products/{slug}"`). All routes registered in `cmd/server/main.go`. No web framework.
- **Single datastore — PostgreSQL** (`internal/store/postgres.go`, `user_postgres.go`, `customer_postgres.go`) via `pgxpool`: users, customers, products, categories, sizes, size charts, images, and password-reset tokens.
- **Schema = versioned migrations** via **tern** (`github.com/jackc/tern/v2/migrate`). SQL files live in `internal/store/migrations/` (`NNN_*.sql`), are `//go:embed`-ed into the binary, and `runMigrations()` (called from `PostgresStore.migrate()`) applies them up to the latest on startup, tracking the applied version in `public.schema_version`. `001_baseline.sql` is the original schema as `CREATE TABLE IF NOT EXISTS` (so an existing DB adopts it as a no-op); `002_seed_categories.sql` seeds the category tree. New schema/seed changes ship as new numbered files, never edits to applied ones.
- **First admin:** seeded inside `migrate()` (`seedAdmin`) from `BOOTSTRAP_ADMIN_*` env — idempotent `INSERT ... ON CONFLICT DO NOTHING`. Registration is invite-only, so this resolves the first-admin chicken-and-egg.
- **Auth:** **HttpOnly-cookie JWT**, **no Bearer fallback**. JWT (`internal/auth`) is HS256 with a single `SECRET_KEY`; two audiences via the `typ` claim — `TokenTypeUser` (admin/staff) and `TokenTypeCustomer` (storefront). Cookies are minted via `internal/cookieauth`: admin → `auth_token` (HttpOnly) + `csrf_token` (JS-readable); storefront → `store_auth` (HttpOnly) + `store_csrf` (JS-readable). The audience pair is separate by design so the two apps can coexist on the same eTLD+1 without clobbering each other. `Secure` is gated on `cfg.IsProduction()` so dev over plain HTTP still works. `middleware.AuthMW` / `middleware.CustomerAuthMW` read the token ONLY from the cookie (Authorization header is ignored — tested as a regression guard); `middleware.RequireAdmin` re-checks role from the DB on every privileged route. State-changing requests under either AuthMW must echo the matching CSRF cookie back in `X-CSRF-Token` — `middleware.CSRF` does a constant-time double-submit compare and 403s on miss/mismatch. Login/forgot/reset are unauthenticated and bootstrap the session itself, so they sit outside the CSRF gate (rate limiting is the abuse guard there). `POST /api/auth/logout` + `POST /api/store/auth/logout` clear both cookies (mounted under CSRF so a foreign origin can't force-log-out an authenticated session). Public store endpoints use a plain `cors()` wrapper.
- **Rate limiting:** in-process fixed-window limiter (`middleware/memstore.go`, `MemoryRateLimiter`) on auth routes — per client IP.
- **Handlers** (`internal/handler/`): `handler.go` (admin auth + profile), `customer.go` (storefront customer auth + profile), `store.go` (public storefront read), `product.go` (admin product CRUD + image upload).
- **Uploads** saved to `UPLOAD_DIR` (default `uploads/`), served at `GET /uploads/` (hardened: nosniff + locked-down CSP).

### Go naming conventions (enforced)

Canonical Go style — Effective Go, *Go Code Review Comments*, and the Google Go Style Guide. This is the standard regardless of any current drift in the code: bring the code into line, not the reverse.

- **MixedCaps, never underscores** — identifiers are `MixedCaps` (exported) / `mixedCaps` (unexported). No `snake_case`, no `SCREAMING_SNAKE` — even for constants (`MaxTokenLen`, not `MAX_TOKEN_LEN`). Underscores appear only in *file* names (`user_postgres.go`, `*_test.go`).
- **Initialisms keep uniform case** — `ID`, `URL`, `HTTP`, `API`, `JSON`, `JWT`: write `userID`, `ServeHTTP`, `JWTConfig` — never `userId`, `ServeHttp`, `JwtConfig`.
- **Packages** — short, lowercase, single word, no plurals/underscores. The name is a namespace prefix: avoid stutter (`store.New`, not `store.NewStore`) and grab-bag names (`util`, `common`, `helpers`, `base`). Drop redundant context inside a package (`user.Count`, not `user.UserCount`).
- **Getters/setters** — no `Get` prefix: `o.Owner()`, not `o.GetOwner()`; mutator is `o.SetOwner(v)`.
- **Interfaces** — single-method interfaces are named method + `-er` (`Reader`, `Validator`); define them where consumed, not where implemented; keep them small.
- **Constructors** — `New` when a package builds one main type (`store.New`), `NewT` when several (`NewClient`); return the concrete `*T`.
- **Receivers** — short (1–2 letters), identical on every method of a type, reflecting the type (`c *Client`); never `this`/`self`/`me`. Be consistent about pointer vs value.
- **Errors** — sentinel vars `ErrNotFound` / `errFoo`; error *types* `FooError`. Error strings are lowercase, no trailing punctuation, so they compose (`"invalid token"`). Wrap with `fmt.Errorf("load user: %w", err)`.
- **Scope-proportional length** — terse names in tight scopes (`i`, `r`, `buf`, `ctx`), descriptive names at package scope. `context.Context` is always the first parameter, named `ctx`.

### Concurrency, timeouts & transactions (enforced)

Serves priority #1 (reliability of financial operations) — money / stock / XP must never partially apply, double-count, or hang.

**Transactions**
- Any multi-statement write that must be all-or-nothing runs in one `pgx` transaction (`pool.BeginTx` → `defer tx.Rollback(ctx)` → explicit `tx.Commit(ctx)`). Mandatory for orders, payments, stock changes, and XP/VIP accrual.
- Mutating a shared counter is atomic: a conditional `UPDATE … SET stock = stock - $1 WHERE stock >= $1` (check `RowsAffected`) or `SELECT … FOR UPDATE` inside the tx — never read-then-write across two round-trips.
- Keep transactions short; no external/network calls while one is open.

**Timeouts**
- Every DB / HTTP / external call takes a `context.Context` with a deadline derived from the request context — never `context.Background()` on a request path.
- `http.Server` sets `ReadTimeout` / `WriteTimeout` / `IdleTimeout` (Slowloris guard).

**Retries**
- Retry only idempotent operations and only on transient errors (serialization failure `40001`, deadlock `40P01`, transient network): bounded attempts + exponential backoff with jitter.
- Never auto-retry a non-idempotent financial write — guard it with an idempotency key so a client retry can't double-charge or double-decrement stock.

**Asynchronous work**
- Background goroutines must be bounded and leak-free, recover from panics, and use a long-lived context (not the request's — that's cancelled when the handler returns).
- Shared state touched by goroutines is synchronised (mutex / channels); keep `go test -race` and `go vet` clean.
- Background jobs updating shared state use compare-and-swap (write only if the current value still matches) to avoid stale overwrites.

### API contract (how the frontends talk to it)

All authenticated calls are **cookie-based** — the SPA must send `credentials: "include"` (so the browser attaches `auth_token` / `store_auth`) and, for any state-changing method (`POST`/`PUT`/`PATCH`/`DELETE`), copy the matching CSRF cookie value into the `X-CSRF-Token` header. There is no `Authorization: Bearer …` path and no token in the JSON response body — login/register return only the user/customer profile. The server's `Access-Control-Allow-Headers` advertises `Content-Type, X-CSRF-Token`.

- **Store public (no auth):** `GET /api/products` (query: `category_id, search, brand, sort, page, per_page`), `GET /api/products/{slug}`, `GET /api/categories` (returns the category **tree**, not flat).
- **Store customers (cookie `store_auth`, CSRF `store_csrf` ↔ `X-CSRF-Token`):** `POST /api/store/auth/{register,login}` (no CSRF — bootstraps the session, rate-limited), `POST /api/store/auth/logout` (CSRF required), `/api/store/customers/me*` (CSRF required on mutations).
- **Admin (cookie `auth_token`, CSRF `csrf_token` ↔ `X-CSRF-Token`):** `POST /api/auth/{register,login,forgot-password,reset-password}` (register is **invite-only**, admin-only; login/forgot/reset bootstrap and so sit outside CSRF — rate limiting is the guard), `POST /api/auth/logout` (CSRF required), `/api/users/me*`, `/api/admin/products*`, `/api/admin/upload`, `/api/admin/categories` (all admin routes CSRF-required on mutations).

### List endpoints & pagination (enforced)

Any list that can grow past ~50 rows paginates on the server. No "load N + client-side slice" — that pattern doesn't scale and silently hides rows once the dataset outgrows N. The storefront catalog is the worked example (`GET /api/products` + `GET /api/products/facets`); same contract applies to every future list (orders, reviews, customers, …).

- **Request:** `page` (1-based, default 1) and `per_page` (positive int, default 20). `per_page` is **clamped to an upper bound in the handler** (`maxPerPage = 100`) before reaching the store — a client cannot ask for a million rows. The store layer clamps again as defence in depth.
- **Response:** `{ items, total, page, per_page }`. `total` is the row count under the active filter (without pagination), not the global table size. The client computes `totalPages = ceil(total / per_page)`.
- **Filters and sort live on the server.** If you slice the result client-side after the server already paginated, the page shows fewer rows than expected and `total` becomes inconsistent with what's visible. Everything that participates in filtering or sorting (price, brand, color, size, search) goes in query params and into the SQL `WHERE`. Multi-value filters use `?key=A&key=B` or `?key=A,B` and resolve via `= ANY($n::text[])` (Postgres-native, no IN-list assembly).
- **Facets get their own endpoint.** Chip-style filter UIs (brand/color/size) cannot derive their options from the current page — only part of the dataset is visible. Add `GET /<resource>/facets` that takes the same filter params and returns distinct values; the facets endpoint **drops the selection of the facet itself** so picking one brand doesn't hide every other brand from the UI. See `internal/store/product_postgres.go::ListProductFacets` for the pattern.
- **Resetting `page` on filter change.** When any filter or sort changes on the client, reset `page` to 1 — otherwise you can land on a page that no longer exists (e.g. you were on page 5, then narrowed the filter so the result set has only 2 pages).
- **Tests:** the store-layer pagination/filter test (e.g. `TestListProductsPagination`, `TestListProductsFilters`) ships with the feature, and the handler-layer parser test asserts `per_page` capping and multi-value parsing.

### Backend env vars
All loaded from `.env` via `godotenv.Load()` in `cmd/server/main.go`. They are read in three places, not one — don't assume `config.go` owns them all:

- **`config.Load()` (`internal/config/config.go`):** `APP_ENV` (default `development`), `SECRET_KEY`, `DATABASE_URL`, `PORT` (8000), `UPLOAD_DIR`. `SECRET_KEY` handling is environment-aware (`resolveSecretKey`): in **production** (`APP_ENV=production`/`prod`) a missing or <32-char key is **fatal** (never silently replaced — tokens survive restarts and a forgotten key can't ship); outside production a missing key is replaced by a random one (≥32 chars) with a warning. Token expiry is hardcoded to 1440 min. (`setup-vps.sh` writes `APP_ENV=production` into the VPS `.env`.)
- **`main.go`:** `CORS_ORIGINS` (comma-separated allowlist; when set it **replaces** the built-in defaults), `TRUST_PROXY` (bool, default `false`; when `true` the rate limiter trusts `X-Real-IP`/last `X-Forwarded-For` hop for client identity — enable **only** behind a trusted proxy, else the headers are spoofable and the per-IP limit is bypassable; `setup-vps.sh` sets it `true`).
- **`seedAdmin()` (`internal/store/postgres.go`):** `BOOTSTRAP_ADMIN_USERNAME` / `BOOTSTRAP_ADMIN_EMAIL` / `BOOTSTRAP_ADMIN_PASSWORD` (first-admin seed).
- **`email.NewService()` (`internal/email/email.go`):** `APP_BASE_URL` (base for password-reset links — the **admin** app hosts `/reset/{token}`, so it defaults to `http://localhost:5174`; set it to the admin domain in prod), `EMAIL_SENDER` (From address, default `onboarding@resend.dev`), `RESEND_API_KEY` (Resend API key; **when empty the reset email is only logged, not sent**).

### Security posture
- Admin registration is **invite-only** (an existing admin creates admins); `RequireAdmin` re-checks the role from the DB on every privileged route.
- **Auth lives in HttpOnly cookies — never in `localStorage` and never in the JSON body.** XSS in either SPA cannot exfiltrate the session token (HttpOnly hides it from JS). The CSRF cookie *is* readable so the SPA can echo it in `X-CSRF-Token`; that's safe because a foreign origin cannot read it (Same-Origin Policy) — the double-submit-cookie pattern.
- **CSRF gate on every authenticated mutation** (`middleware.CSRF`) — constant-time compare (`crypto/subtle.ConstantTimeCompare`) between the readable CSRF cookie and the `X-CSRF-Token` request header; missing or mismatching → 403, no logging of the cookie or header values. Logout is gated too, so a foreign origin can't force-log-out an authenticated session.
- **`SameSite=Lax`** on both cookie pairs limits drive-by CSRF to top-level navigations; `Secure` is on in production (gated via `cfg.IsProduction()`), off in dev so plain-HTTP localhost works. Separate cookie names per audience (`auth_token`/`csrf_token` vs `store_auth`/`store_csrf`) so admin and storefront can coexist on the same eTLD+1 without overwriting each other.
- JWT `typ` separation prevents a customer token from reaching admin routes; HS256 is pinned via `jwt.WithValidMethods`. Tokens carry an `iat`; `AuthMW`/`CustomerAuthMW` reject any token issued before the account's `password_changed_at`, so changing or resetting a password atomically invalidates all of that account's prior sessions (one DB lookup per authenticated request).
- bcrypt cost 12; login runs a constant-time dummy hash on a missing user (timing guard); password-reset tokens are stored only as a SHA-256 hash (raw token exists only in the email).
- JSON request bodies are capped at 1 MiB (`http.MaxBytesReader`); admin login bounds username ≤ 100 / password ≤ 72 before any store lookup.
- CORS is an explicit allowlist (`CORS_ORIGINS`); credentialed responses reflect only allowlisted origins (`Vary: Origin`). `Allow-Headers` exposes only `Content-Type, X-CSRF-Token` — `Authorization` is not advertised because Bearer auth is intentionally not supported.
- Per-IP rate limiting on login / register / forgot / reset (the unauthenticated routes that sit outside the CSRF gate).
- Upload validation: extension check **plus** `http.DetectContentType` MIME sniff (SVG rejected); `/uploads` served with `nosniff` and a `default-src 'none'; sandbox` CSP.
- 500-level errors log internally and return a generic message — no `err.Error()` leakage to clients.
- Security headers + HSTS on all responses.

## Frontend architecture (both apps share conventions)

- **Stack:** React 19, Vite, TypeScript, Tailwind **v4** (`@tailwindcss/vite`, no config file — theme in `index.css`), shadcn/ui-style components built on Radix in `src/components/ui/`.
- **Path alias:** `@/` → `src/` (used everywhere; configured in vite + tsconfig).
- **State:** Zustand stores in `src/stores/` (one store per domain: `cartStore`, `authStore`, `catalogStore`, `avatarStore`, etc.). Async fetch logic lives inside the store actions.
- **API layer:** all HTTP in `src/lib/api.ts`. Base URL from `VITE_API_URL` (store defaults to `https://api.mioru.store`, admin defaults to `""`). **Auth is cookie-based**: every call uses `credentials: "include"` so the browser attaches the HttpOnly session cookie; on mutating methods (`POST`/`PUT`/`PATCH`/`DELETE`) the helper reads the readable CSRF cookie (`csrf_token` for admin, `store_csrf` for store) and copies it into `X-CSRF-Token`. **No token is read from or written to `localStorage` — that pattern is deliberately removed (XSS exfil risk).** `getImageUrl()` prefixes relative upload paths.
- **i18n:** i18next, three locales RU/EN/RO in `src/i18n/locales/`.
- **Theme:** dark/light toggled by adding/removing the `light` class on `<html>`; colors are CSS variables `--color-*` in `index.css` — **CSS variables are the source of truth**, not the `COLORS` object in `src/lib/constants.ts` (that object is legacy/stale).
- **Tests:** colocated `*.test.ts(x)` next to source — framework and conventions in **Testing standard**.

### Store app specifics
- Routing in `App.tsx` — all pages `lazy()`-loaded, `react-router-dom` v6. Pages in `src/routes/`.
- **3D avatar** (`src/avatar/AvatarManager.ts`): loads a GLB via `GLTFLoader`/`DRACOLoader`; if loading fails, builds a **procedural body mesh** parameterized by gender/fat/muscle. Models in `public/models/`. Rendered through `@react-three/fiber` + `drei`.
- Catalog currently loads up to 100 products and paginates client-side.

### Admin app specifics
- `App.tsx` routes only auth pages; everything else (`/*`) goes to `AdminLayout`, which performs the auth check (no duplicate auth logic in routes). `authStore.isAuthenticated` is **tri-state** (`null | true | false`): the layout renders a neutral loader while it's `null` (probe in flight) and only redirects to `/login` when it resolves to `false` — there's no synchronous "do we have a token" check anymore (the cookie is HttpOnly and the SPA can't see it), so the answer comes from `GET /api/users/me` succeeding or returning 401.
- UI organized as **workspaces** (`src/workspaces/`, declared in `src/lib/constants.ts WORKSPACES`); only `products` is active, the rest are placeholders.
- The category tree is **owned by the backend**: the admin fetches it from `/api/admin/categories` into `productStore.categories` (no hardcoded copy in the frontend). The single source is the seed migration `internal/store/migrations/002_seed_categories.sql`, pinned by `TestSeededCategoryTree` in the `store` package.

## Testing standard

- **Frontend:** Vitest + Testing Library (unit/component), colocated `*.test.tsx`, jsdom setup in `src/test/setup.ts`; store also has Playwright E2E (`apps/store/e2e/*.spec.ts`).
- **Go:** stdlib `testing`, **table-driven** tests, colocated `*_test.go` in-package (white-box); use a black-box `_test` package only for public-API surface tests. Assertions are plain `if got != want { t.Errorf(...) }`; `testify/require` is allowed but not required. Run `go test ./... -race`. (The auth package is the worked example — see `internal/auth/auth_test.go`.)
- **DB-touching logic:** test the `store` layer against a real PostgreSQL, not mocks — money/stock correctness depends on real SQL semantics (constraints, `FOR UPDATE`, serialization). Use a **dedicated throwaway test database via `TEST_DATABASE_URL`** (dev machine or CI) — **never the production VPS database, never reuse `DATABASE_URL`**. Locally we ship `backend/api/docker-compose.test.yml` (image `postgres:16-alpine`, container `mioru-postgres-test`) which exposes the test DB on `127.0.0.1:55433`, creds `mioru/mioru`, database `mioru_test`. Bring it up once with `docker compose -f backend/api/docker-compose.test.yml up -d`, then export `TEST_DATABASE_URL='postgres://mioru:mioru@127.0.0.1:55433/mioru_test?sslmode=disable'` and run `go test ./... -race`. The harness (`internal/store/harness_test.go`) connects via `NewPostgresStore` (which runs migrations + seeds the category tree), then `TRUNCATE ... RESTART IDENTITY CASCADE` resets data tables before each test (the seeded categories survive); store tests `t.Skip()` when `TEST_DATABASE_URL` is unset. Get a clean store with `testStore(t)`.
- **Priority #1 paths** (orders, payments, stock, XP/VIP) ship with tests covering the atomic/transactional behaviour before merge — TDD is non-negotiable here.
- **Tests are never deferred.** A fix ships with its test in the same change. If the harness it needs is missing (e.g. no PostgreSQL setup for `store` tests), building that harness is the *first step of the work*, not a separate "later" task — "there's no harness yet" is the reason to build it now, not to skip the test. (Deferring is how the Go side reached zero coverage.)

## Deployment

`scripts/setup-vps.sh` provisions an Ubuntu 22.04 VPS: Go 1.25, PostgreSQL 16, Nginx, 2GB swap, a dedicated system user `mioru`, and a hardened systemd service. Secrets (DB password, `SECRET_KEY`) are generated on first run and preserved in `/opt/mioru/.env`. The store is shipped as static `dist/`; the backend runs as a separate service. No Redis.
