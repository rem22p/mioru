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
No commits without explicit approval. No code changes before presenting a plan. The agent can commit/branch but **cannot push** (SSH passphrase); the user pushes. Remote: `git@github.com:rem22p/mioru.git`.

## What this is

MIORU — virtual clothing try-on with a 3D avatar. A monorepo with two React SPAs and one Go API:

```
apps/store/    Storefront SPA (React 19 + Vite), port 5173 — public, served as static/CDN
apps/admin/    Admin panel SPA (React 19 + Vite), port 5174 — product management, auth
backend/api/   Go API (net/http + PostgreSQL + JWT), port 8000
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
- **Auth:** JWT (`internal/auth`), HS256 with a single `SECRET_KEY`. Two audiences via the `typ` claim: `TokenTypeUser` (admin/staff) and `TokenTypeCustomer` (storefront). Enforced by `middleware.AuthMW` / `middleware.CustomerAuthMW`; `middleware.RequireAdmin` does a DB-backed role check. Public store endpoints use a plain `cors()` wrapper.
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
- **Store public (no auth):** `GET /api/products` (query: `category_id, search, brand, sort, page, per_page`), `GET /api/products/{slug}`, `GET /api/categories` (returns the category **tree**, not flat).
- **Store customers (JWT typ=customer):** `POST /api/store/auth/{register,login}`, `/api/store/customers/me*`.
- **Admin (JWT typ=user):** `POST /api/auth/{register,login,forgot-password,reset-password}` (**register is invite-only** — admin-only), `/api/users/me*`, `/api/admin/products*`, `/api/admin/upload`, `/api/admin/categories`.

### Backend env vars
All loaded from `.env` via `godotenv.Load()` in `cmd/server/main.go`. They are read in three places, not one — don't assume `config.go` owns them all:

- **`config.Load()` (`internal/config/config.go`):** `APP_ENV` (default `development`), `SECRET_KEY`, `DATABASE_URL`, `PORT` (8000), `UPLOAD_DIR`. `SECRET_KEY` handling is environment-aware (`resolveSecretKey`): in **production** (`APP_ENV=production`/`prod`) a missing or <32-char key is **fatal** (never silently replaced — tokens survive restarts and a forgotten key can't ship); outside production a missing key is replaced by a random one (≥32 chars) with a warning. Token expiry is hardcoded to 1440 min. (`setup-vps.sh` writes `APP_ENV=production` into the VPS `.env`.)
- **`main.go`:** `CORS_ORIGINS` (comma-separated allowlist; when set it **replaces** the built-in defaults), `TRUST_PROXY` (bool, default `false`; when `true` the rate limiter trusts `X-Real-IP`/last `X-Forwarded-For` hop for client identity — enable **only** behind a trusted proxy, else the headers are spoofable and the per-IP limit is bypassable; `setup-vps.sh` sets it `true`).
- **`seedAdmin()` (`internal/store/postgres.go`):** `BOOTSTRAP_ADMIN_USERNAME` / `BOOTSTRAP_ADMIN_EMAIL` / `BOOTSTRAP_ADMIN_PASSWORD` (first-admin seed).
- **`email.NewService()` (`internal/email/email.go`):** `APP_BASE_URL` (base for password-reset links — the **admin** app hosts `/reset/{token}`, so it defaults to `http://localhost:5174`; set it to the admin domain in prod), `EMAIL_SENDER` (From address, default `onboarding@resend.dev`), `RESEND_API_KEY` (Resend API key; **when empty the reset email is only logged, not sent**).

### Security posture
- Admin registration is **invite-only** (an existing admin creates admins); `RequireAdmin` re-checks the role from the DB on every privileged route.
- JWT `typ` separation prevents a customer token from reaching admin routes; HS256 is pinned via `jwt.WithValidMethods`.
- bcrypt cost 12; login runs a constant-time dummy hash on a missing user (timing guard).
- CORS is an explicit allowlist (`CORS_ORIGINS`); credentialed responses reflect only allowlisted origins (`Vary: Origin`).
- Per-IP rate limiting on login / register / forgot / reset.
- Upload validation: extension check **plus** `http.DetectContentType` MIME sniff (SVG rejected); `/uploads` served with `nosniff` and a `default-src 'none'; sandbox` CSP.
- 500-level errors log internally and return a generic message — no `err.Error()` leakage to clients.
- Security headers + HSTS on all responses.

## Frontend architecture (both apps share conventions)

- **Stack:** React 19, Vite, TypeScript, Tailwind **v4** (`@tailwindcss/vite`, no config file — theme in `index.css`), shadcn/ui-style components built on Radix in `src/components/ui/`.
- **Path alias:** `@/` → `src/` (used everywhere; configured in vite + tsconfig).
- **State:** Zustand stores in `src/stores/` (one store per domain: `cartStore`, `authStore`, `catalogStore`, `avatarStore`, etc.). Async fetch logic lives inside the store actions.
- **API layer:** all HTTP in `src/lib/api.ts`. Base URL from `VITE_API_URL` (store defaults to `https://api.mioru.store`, admin defaults to `""`). JWT read from `localStorage.getItem("token")`. `getImageUrl()` prefixes relative upload paths.
- **i18n:** i18next, three locales RU/EN/RO in `src/i18n/locales/`.
- **Theme:** dark/light toggled by adding/removing the `light` class on `<html>`; colors are CSS variables `--color-*` in `index.css` — **CSS variables are the source of truth**, not the `COLORS` object in `src/lib/constants.ts` (that object is legacy/stale).
- **Tests:** colocated `*.test.ts(x)` next to source — framework and conventions in **Testing standard**.

### Store app specifics
- Routing in `App.tsx` — all pages `lazy()`-loaded, `react-router-dom` v6. Pages in `src/routes/`.
- **3D avatar** (`src/avatar/AvatarManager.ts`): loads a GLB via `GLTFLoader`/`DRACOLoader`; if loading fails, builds a **procedural body mesh** parameterized by gender/fat/muscle. Models in `public/models/`. Rendered through `@react-three/fiber` + `drei`.
- Catalog currently loads up to 100 products and paginates client-side.

### Admin app specifics
- `App.tsx` routes only auth pages; everything else (`/*`) goes to `AdminLayout`, which performs the auth check (no duplicate auth logic in routes).
- UI organized as **workspaces** (`src/workspaces/`, declared in `src/lib/constants.ts WORKSPACES`); only `products` is active, the rest are placeholders.
- The category tree is **owned by the backend**: the admin fetches it from `/api/admin/categories` into `productStore.categories` (no hardcoded copy in the frontend). The single source is the seed migration `internal/store/migrations/002_seed_categories.sql`, pinned by `TestSeededCategoryTree` in the `store` package.

## Testing standard

- **Frontend:** Vitest + Testing Library (unit/component), colocated `*.test.tsx`, jsdom setup in `src/test/setup.ts`; store also has Playwright E2E (`apps/store/e2e/*.spec.ts`).
- **Go:** stdlib `testing`, **table-driven** tests, colocated `*_test.go` in-package (white-box); use a black-box `_test` package only for public-API surface tests. Assertions are plain `if got != want { t.Errorf(...) }`; `testify/require` is allowed but not required. Run `go test ./... -race`. (The auth package is the worked example — see `internal/auth/auth_test.go`.)
- **DB-touching logic:** test the `store` layer against a real PostgreSQL, not mocks — money/stock correctness depends on real SQL semantics (constraints, `FOR UPDATE`, serialization). Use a **dedicated throwaway test database via `TEST_DATABASE_URL`** (dev machine or CI) — **never the production VPS database, never reuse `DATABASE_URL`**. The harness (`internal/store/harness_test.go`) connects via `NewPostgresStore` (which runs migrations + seeds the category tree), then `TRUNCATE ... RESTART IDENTITY CASCADE` resets data tables before each test (the seeded categories survive); store tests `t.Skip()` when `TEST_DATABASE_URL` is unset. Get a clean store with `testStore(t)`.
- **Priority #1 paths** (orders, payments, stock, XP/VIP) ship with tests covering the atomic/transactional behaviour before merge — TDD is non-negotiable here.
- **Tests are never deferred.** A fix ships with its test in the same change. If the harness it needs is missing (e.g. no PostgreSQL setup for `store` tests), building that harness is the *first step of the work*, not a separate "later" task — "there's no harness yet" is the reason to build it now, not to skip the test. (Deferring is how the Go side reached zero coverage.)

## Deployment

`scripts/setup-vps.sh` provisions an Ubuntu 22.04 VPS: Go 1.25, PostgreSQL 16, Nginx, 2GB swap, a dedicated system user `mioru`, and a hardened systemd service. Secrets (DB password, `SECRET_KEY`) are generated on first run and preserved in `/opt/mioru/.env`. The store is shipped as static `dist/`; the backend runs as a separate service. No Redis.
