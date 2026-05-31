# CLAUDE.md

Single source of truth for agents working in this repo. `AGENTS.md` is just a pointer here.

## Priorities (resolve trade-offs top-down)

1. **Reliability of financial operations** — orders, payments, stock, XP/VIP accrual must never silently lose or double-count. Correctness and atomicity over convenience.
2. **Safety of user data** — personal data and credentials. Security-by-default, least privilege, nothing sensitive in logs or responses.
3. **Speed / performance** — pursue only after 1 and 2 hold.

## Development principles (enforced)

- **TDD** — failing test → green → refactor. New logic ships with tests; verify a module's tests before wiring it into the rest. **Tests are never deferred** — if the harness it needs is missing, building that harness *is* step one of the work, not a separate "later" task.
- **YAGNI** — build what the current task needs; no speculative abstractions.
- **Don't reinvent the wheel** — for *solved, non-domain* concerns (migrations, JWT, password hashing, SQL driver, validation, parsing) adopt a mature library that fits the stack (e.g. pgx-native > database/sql shim). Custom code only for MIORU-unique logic.
- **DRY** — one source of truth per fact (config, types, the category tree). Reference, don't duplicate.
- **OWASP / security-by-default** — validate all input, parameterize all SQL, assemble JSON only via `encoding/json` (never build it by string concatenation — that re-invents escaping and invites injection), never reflect internal errors to clients, enforce authz on every privileged route.
- **Input validation** — type, presence, bounds on every incoming argument. Strings always have a max length (and min/format where it matters); never persist or process an unbounded user-supplied string. Fail early with a generic 4xx; never trust the client.
- **Repo hygiene** — never commit local databases (`*.db`, `*.db-shm`, `*.db-wal` — and in this Postgres-only repo a stray `*.db` is itself a red flag: there is no SQLite driver), user uploads (`uploads/`), `.env*` files (except `.env.example`), or build artifacts. Keep `.gitignore` current; a generated/binary artifact that reaches the tree is a review bug, and a pushed one that may hold real data is scrubbed from history.
- **Modular decomposition** — one concern per package (`auth`, `store`, `handler`, `middleware`, `config`, `model`), one responsibility per file/handler.
- **Architectural integrity** — handler → store → DB; handlers never touch the pool directly. No import cycles. Flag a boundary-crossing change before making it.
- **No N+1; EXPLAIN ANALYZE hot SQL** — never run DB queries in a loop over rows; batch via `IN` / `= ANY($n::…[])` / `pgx.CollectRows`. Any new query that can touch >100 rows is `EXPLAIN ANALYZE`'d before merge. Hot endpoints ship a regression test pinning query count (see `TestListProductsAttachesRelatedData`, the 5-query batch reference).
- **Conventional Commits** — `feat(scope): …`, `fix(scope): …`, `docs:`, `refactor:`, `test:`, `chore:`. Subject ≤72 chars, imperative (`add`, not `added`). Body explains *why*, not *what*. AI-assisted commits carry a `Co-Authored-By:` trailer.

## Workflow rule (hard)

**Agent workflow (every change):**
1. Create a feature branch (`fix/<slug>`, `feat/<slug>`, `chore/<slug>` — kebab-case, ≤50 chars)
2. Present a plan → user approves
3. Make the change → user checks
4. User says commit → agent commits
5. Push the branch → create a PR → user reviews and merges

No commits without explicit user approval; no code changes before presenting a plan.

- **Agent never pushes to `main`.** Every agent change reaches main via a user-approved PR.
- **Human pushes to main are unrestricted**, but changes touching security, auth, or OAuth paths should go through a PR when possible.
- **Agent may push the feature branch and create a PR but never merges** — merge is the user's gate.
- **PR should reference an existing issue** (e.g. `Closes #N`).
- **Branch is deleted after merge** (agent or user cleans up).
- Remote: `git@github.com:rem22p/mioru.git` (SSH) for the user. The agent uses HTTPS with a temporary GitHub PAT (`repo` scope) to push feature branches and create PRs.

## What this is

MIORU — virtual clothing try-on with a 3D avatar. Monorepo:

```
apps/store/    Storefront SPA (React 19 + Vite), port 5173 — public
apps/admin/    Admin panel SPA  (React 19 + Vite), port 5174 — auth-gated
backend/api/   Go API (net/http + pgxpool + HttpOnly-cookie JWT + CSRF), port 8000
scripts/       VPS provisioning + DB seeding
```

Historical note: Redis and a notes/WebSocket feature were removed — PostgreSQL is now the only datastore. `packages/shared/` is referenced in stale docs but does not exist.

## Commands

Frontends (`apps/store/`, `apps/admin/` — independent npm projects, run from each dir):

```bash
npm install
npm run dev            # store→5173, admin→5174 (bind 0.0.0.0)
npm run build          # tsc -b && vite build → dist/
npm run lint
npm test               # vitest (watch)
npx vitest run path/to/file.test.tsx
```

Store has Playwright E2E too:

```bash
cd apps/store && npm run test:e2e
npx playwright test e2e/some.spec.ts
```

Backend (Go 1.25, needs a running PostgreSQL):

```bash
cd backend/api
go run ./cmd/server
go build ./...
go test ./... -race    # see "Testing standard" for TEST_DATABASE_URL
```

Seed (writes SQL to stdout, pipe into psql):

```bash
bash scripts/seed-300.sh | psql "$DATABASE_URL"
```

## Backend architecture

- **Routing:** Go 1.25 stdlib `http.ServeMux`, method+pattern routes (`"GET /api/products/{slug}"`). Registered in `cmd/server/main.go`. No web framework.
- **Single datastore — PostgreSQL** (`internal/store/`) via `pgxpool`: users, customers, products, categories, sizes, size charts, images, password-reset tokens.
- **Schema = versioned migrations** via **tern** (`github.com/jackc/tern/v2/migrate`). `internal/store/migrations/NNN_*.sql`, `//go:embed`-ed; `runMigrations()` (called from `PostgresStore.migrate()`) applies up to head on startup, tracking the version in `public.schema_version`. `001_baseline.sql` is the original schema as `CREATE TABLE IF NOT EXISTS` (existing DBs adopt it as a no-op); `002_seed_categories.sql` seeds the category tree. New schema/seed changes ship as new numbered files — never edits to applied ones. Migrations **never hardcode environment- or person-specific values** (usernames, hosts, keys): a literal like `WHERE username = 'rem22p'` is a silent no-op on any other deployment and bakes one operator into shared schema. Role/first-admin assignment is driven by `seedAdmin` / `BOOTSTRAP_ADMIN_*` or a stable criterion (e.g. the oldest admin), idempotently — one source of truth, not a migration and a seeder both claiming it.
- **First admin:** seeded inside `migrate()` (`seedAdmin`) from `BOOTSTRAP_ADMIN_*` env, idempotent `INSERT ... ON CONFLICT DO NOTHING`. Resolves the invite-only chicken-and-egg.
- **Auth:** HttpOnly-cookie JWT, **no Bearer fallback**. JWT (`internal/auth`) is HS256 with one `SECRET_KEY`; two audiences via `typ` — `TokenTypeUser` (admin/staff) and `TokenTypeCustomer` (storefront). Cookies minted in `internal/cookieauth`: admin `auth_token` (HttpOnly) + `csrf_token` (JS-readable); storefront `store_auth` + `store_csrf`. Separate names so admin and storefront can coexist on one eTLD+1. `Secure` is gated on `cfg.IsProduction()` so dev HTTP works. `middleware.AuthMW` / `CustomerAuthMW` read the token **only** from the cookie (Authorization header is ignored — tested as a regression guard); `middleware.RequireAdmin` re-checks role from the DB on every privileged route. State-changing requests under either AuthMW must echo the matching CSRF cookie in `X-CSRF-Token` — `middleware.CSRF` constant-time compares and 403s on miss/mismatch. Login/forgot/reset bootstrap the session itself, so they sit outside CSRF (rate limiting is the abuse guard). `POST /api/auth/logout` + `POST /api/store/auth/logout` clear both cookies and are gated by CSRF (no force-logout from a foreign origin).
- **Rate limiting:** in-process fixed-window limiter (`middleware/memstore.go::MemoryRateLimiter`) on auth routes, per client IP.
- **Handlers** (`internal/handler/`): `handler.go` (admin auth + profile), `customer.go` (storefront customer auth + profile), `store.go` (public storefront read), `product.go` (admin product CRUD + image upload).
- **Uploads:** saved to `UPLOAD_DIR` (default `uploads/`), served at `GET /uploads/` with nosniff + locked-down CSP.

### Go naming conventions (enforced)

Effective Go + Go Code Review Comments + Google Go Style Guide. The standard regardless of any current drift — bring the code into line, not the reverse.

- **MixedCaps, never underscores** in identifiers; `MaxTokenLen` (not `MAX_TOKEN_LEN`). Underscores only in *file* names.
- **Initialisms keep uniform case** — `userID`, `ServeHTTP`, `JWTConfig` — never `userId`, `ServeHttp`, `JwtConfig`.
- **Packages** — short, lowercase, single word, no plurals/underscores. The package name is a namespace prefix: avoid stutter (`store.New`, not `store.NewStore`); avoid grab-bag names (`util`, `common`, `helpers`).
- **Getters/setters** — no `Get` prefix; mutator is `SetOwner(v)`.
- **Interfaces** — single-method = method + `-er` (`Reader`, `Validator`). Define them where consumed, not where implemented. Keep them small.
- **Constructors** — `New` for the one main type, `NewT` when several (`NewClient`). Return concrete `*T`.
- **Receivers** — short (1–2 letters), identical on every method of a type. Never `this`/`self`/`me`.
- **Scope-proportional length** — terse in tight scopes (`i`, `r`, `buf`, `ctx`), descriptive at package scope. `context.Context` is always first, named `ctx`.

### Error handling (enforced)

- Sentinels: `ErrNotFound` / `errFoo`; error *types*: `FooError`. Strings lowercase, no trailing punctuation, so they compose: `"invalid token"`.
- Always wrap with `%w`: `fmt.Errorf("load user: %w", err)`. Don't lose origin.
- Compare sentinels only via `errors.Is`; type-assert via `errors.As`. Never `err.Error() == "…"`.
- `defer` that can return a meaningful error (`tx.Rollback`, `f.Close` on a *written* file, `rows.Close` doesn't matter as much) is not silent — log it or assign to a named return. No bare `_ = x.Close()` without a comment explaining why.
- Public handlers never leak `err.Error()` to clients; map to the error envelope (see API contract).

### Time = UTC, no `time.Now()` in business logic (enforced)

- DB: all timestamp columns are `TIMESTAMPTZ`, stored UTC.
- Go: interpret `time.Time` as UTC end-to-end. Never `time.Local`. JSON serialization is RFC 3339 with the `Z` suffix.
- Business logic takes the clock via a `Clock` interface (or a `now func() time.Time` arg) — never calls `time.Now()` directly. Tests inject a fixed clock; production injects `time.Now`. The only exceptions are edge glue (`cmd/server/main.go`, middleware logging) where the value is observational, not part of a business decision.

### Logging (enforced)

Priority #1 is unworkable without per-request context in the logs — debugging a payment incident from `log.Printf("foo: %v", err)` is impossible.

- `log/slog` (Go 1.21+ stdlib) is the only logger. JSON handler when `APP_ENV=production`, text handler otherwise.
- One log line per request, written by a logging middleware near the top of the chain: `time, level, request_id, method, path, status, latency_ms, user_id`. `request_id` is a UUID stashed in `context.Context` — every handler-level log builds on `slog.With(ctx, …)` and inherits it. Echo it back in `X-Request-ID` so client reports tie to server lines.
- Levels: `DEBUG` (dev noise), `INFO` (one-per-request + lifecycle), `WARN` (recovered anomaly), `ERROR` (operation failed, includes an `err` field). No fatal from a request path — only boot in `cmd/server/main.go` may `log.Fatal`.
- **Never log:** passwords, hash output, plaintext password-reset tokens, JWTs, cookie values, `Authorization` / `Cookie` / `X-CSRF-Token` headers, raw request/response bodies (PII risk), and — when payments land — full PAN/CVV. Log the fact and the *id*: `slog.Info("login failed", "user_id", id, "reason", "bad_password")`. The "no `err.Error()` to clients" rule does **not** mean don't log it — log internally with full context, return a generic envelope.

### Concurrency, timeouts & transactions (enforced)

Serves priority #1 — money / stock / XP must never partially apply, double-count, or hang.

**Transactions**
- Multi-statement writes that must be all-or-nothing run in one `pgx` transaction: `pool.BeginTx` → `defer tx.Rollback(ctx)` → explicit `tx.Commit(ctx)`. Mandatory for orders, payments, stock changes, XP/VIP accrual.
- Mutating a shared counter is atomic: conditional `UPDATE … SET stock = stock - $1 WHERE stock >= $1` (check `RowsAffected`) or `SELECT … FOR UPDATE` inside the tx — never read-then-write across two round-trips.
- Keep transactions short; no external/network calls while one is open.

**Timeouts & pool**
- Every DB / HTTP / external call takes a `context.Context` with a deadline derived from the request context — never `context.Background()` on a request path.
- `http.Server` sets `ReadTimeout` / `WriteTimeout` / `IdleTimeout` (Slowloris guard).
- `pgxpool` `MaxConns` is set explicitly (from env, sized to the deployment), not left at the default. Connections set `statement_timeout` server-side as defence in depth — a forgotten context deadline still can't hang the DB.

**Retries**
- Retry only idempotent operations and only on transient errors (serialization failure `40001`, deadlock `40P01`, transient network): bounded attempts + exponential backoff with jitter.
- Never auto-retry a non-idempotent financial write — protect it via the Idempotency protocol below.

**Idempotency (mandatory for finance-critical mutations)**
- Every POST/PUT that changes money / stock / XP accepts an `Idempotency-Key` header (client-generated UUID). Recommended for other mutations.
- Server persists `(key, user_id, request_hash, status, response_body, expires_at)` in a dedicated table, TTL 24–48h. `request_hash = sha256(method ‖ path ‖ canonicalized_body ‖ user_id)`.
- Replay with same key + same request_hash → return the stored status+body, do not re-execute. Same key + different request_hash → `409 CONFLICT` with code `IDEMPOTENCY_REPLAY`.
- Write the idempotency record inside the same transaction as the business effect — otherwise a crash between the two leaks the guarantee.

**Asynchronous work**
- Background goroutines are bounded, leak-free, recover from panics, and use a long-lived context (not the request's — that's cancelled when the handler returns).
- Shared state synchronised (mutex / channels); `go test -race` and `go vet` stay clean.
- Background jobs updating shared state use compare-and-swap (write only if the current value still matches) to avoid stale overwrites.

### API contract

All authenticated calls are **cookie-based**: the SPA sends `credentials: "include"` and, on `POST`/`PUT`/`PATCH`/`DELETE`, copies the matching CSRF cookie into `X-CSRF-Token`. There is no `Authorization: Bearer …` path and no token in the JSON body — login/register return only the user/customer profile. `Access-Control-Allow-Headers` advertises only `Content-Type, X-CSRF-Token`.

- **Error envelope** (all 4xx and 5xx): `{ "error": "<human>", "code": "<MACHINE_CODE>" }`. The SPA branches on `code`, not on the text. Reserved codes: `AUTH_REQUIRED`, `AUTH_INVALID`, `CSRF_INVALID`, `RATE_LIMITED`, `VALIDATION_FAILED`, `NOT_FOUND`, `CONFLICT`, `IDEMPOTENCY_REPLAY`, `PAYLOAD_TOO_LARGE`, `INTERNAL`. 5xx carries only `{"error":"Internal error","code":"INTERNAL"}` — never `err.Error()`. **Every** error response — including those emitted from middleware (authz/CSRF gates) — goes through the JSON envelope helper carrying a machine `code`; never `http.Error`, which sends `text/plain` with no `code` and breaks the SPA's `code`-based branching.
- **Store public (no auth):** `GET /api/products` (query: `category_id, brand, color, size, search, sort, price_min, price_max, page, per_page`; multi-value via repeated keys *or* CSV), `GET /api/products/facets` (same params, chip selections dropped), `GET /api/products/{slug}`, `GET /api/categories` (category **tree**, not flat).
- **Store customers (cookie `store_auth`, CSRF `store_csrf` ↔ `X-CSRF-Token`):** `POST /api/store/auth/{register,login}` (no CSRF — bootstraps the session, rate-limited), `POST /api/store/auth/logout` (CSRF), `/api/store/customers/me*` (CSRF on mutations).
- **Admin (cookie `auth_token`, CSRF `csrf_token` ↔ `X-CSRF-Token`):** `POST /api/auth/{register,login,forgot-password,reset-password}` (register is invite-only, admin-only; login/forgot/reset bootstrap → outside CSRF, rate-limited), `POST /api/auth/logout` (CSRF), `/api/users/me*`, `/api/admin/products*`, `/api/admin/upload`, `/api/admin/categories` (CSRF on mutations).

### List endpoints & pagination (enforced)

Any list that can grow past ~50 rows paginates on the server. No "load N + client-side slice" — that pattern silently hides rows once the dataset outgrows N. Storefront catalog is the worked example.

- **Request:** `page` (1-based, default 1), `per_page` (positive int, default 20). Handler clamps `per_page ≤ maxPerPage (=100)` *before* the store; the store layer clamps again (defence in depth).
- **Response:** `{ items, total, page, per_page }`. `total` is the row count under the active filter (not the global table). Client computes `totalPages = ceil(total / per_page)`.
- **Filters and sort live on the server.** Everything that participates in filtering/sorting (price, brand, color, size, search) goes in query params and into SQL `WHERE`. Multi-value uses `?key=A&key=B` or `?key=A,B`; resolve via `= ANY($n::text[])` (Postgres-native, no IN-list assembly).
- **Facets have their own endpoint.** Chip-style UIs cannot derive options from the current page. `GET /<resource>/facets` takes the same filter params and returns distinct values, but **drops the selection of the facet itself** — picking one brand must not hide every other brand from the UI. See `internal/store/product_postgres.go::ListProductFacets`.
- **Reset `page` on filter change** — otherwise the client lands on a page that no longer exists.
- **Tests** ship with the feature: store-layer pagination/filter (`TestListProductsPagination`, `TestListProductsFilters`), and a handler-layer parser test for `per_page` capping + multi-value parsing.

### Backend env vars

All loaded from `.env` via `godotenv.Load()` in `cmd/server/main.go`. Read in three places — not just `config.go`.

- **`config.Load()` (`internal/config/config.go`):** `APP_ENV` (default `development`), `SECRET_KEY`, `DATABASE_URL`, `PORT` (8000), `UPLOAD_DIR`. `SECRET_KEY` is environment-aware (`resolveSecretKey`): in production a missing or <32-char key is **fatal**; outside production a missing key is replaced by a random one (≥32 chars) with a warning. Token expiry hardcoded to 1440 min.
- **`main.go`:** `CORS_ORIGINS` (comma-separated allowlist; when set, replaces defaults), `TRUST_PROXY` (default `false`; `true` makes the rate limiter trust `X-Real-IP`/last `X-Forwarded-For` hop — enable **only** behind a trusted proxy or per-IP limiting is bypassable).
- **`seedAdmin()` (`internal/store/postgres.go`):** `BOOTSTRAP_ADMIN_USERNAME` / `BOOTSTRAP_ADMIN_EMAIL` / `BOOTSTRAP_ADMIN_PASSWORD`.
- **`email.NewService()` (`internal/email/email.go`):** `APP_BASE_URL` (admin app hosts `/reset/{token}`; default `http://localhost:5174`), `EMAIL_SENDER` (From, default `onboarding@resend.dev`), `RESEND_API_KEY` (when empty the reset email is only logged, not sent).

### Security posture

- **External identity is verified cryptographically before it is trusted.** Any route that authenticates or *links* by a third-party identifier (OAuth `oauth_id`, provider user id, webhook payload) must verify the provider-signed payload (`auth.VerifyTelegramAuth` and the like) before writing the binding — never trust a bare, client-supplied id. The trust boundary must be identical on every path to the same resource: if login verifies a signature, the link/bind sibling must verify it too, or an authenticated user can claim an identity they do not own and hijack the victim's later login. Each such endpoint ships a regression test that rejects an unsigned / non-owned identifier.
- Admin registration is invite-only; `RequireAdmin` re-checks role from the DB on every privileged route.
- Auth lives in HttpOnly cookies — never in `localStorage`, never in the JSON body. XSS in either SPA cannot exfiltrate the session token. The CSRF cookie *is* readable so the SPA can echo it in `X-CSRF-Token`; that's safe because a foreign origin cannot read it (SOP) — double-submit-cookie pattern.
- **CSRF gate on every authenticated mutation** (`middleware.CSRF`): constant-time compare (`crypto/subtle.ConstantTimeCompare`) of the readable CSRF cookie against `X-CSRF-Token`; missing/mismatch → 403, no logging of the values. Logout is gated too.
- **`SameSite=Lax`** on both cookie pairs; `Secure` on in production, off in dev. Separate cookie names per audience (admin vs storefront).
- JWT `typ` separation prevents a customer token from reaching admin routes; HS256 pinned via `jwt.WithValidMethods`. Tokens carry `iat`; `AuthMW`/`CustomerAuthMW` reject any token issued before the account's `password_changed_at`, so a password change/reset atomically invalidates all prior sessions (one DB lookup per authenticated request).
- bcrypt cost 12; login runs a constant-time dummy hash on missing user (timing guard); password-reset tokens are stored only as a SHA-256 hash (raw token exists only in the email).
- JSON bodies capped at 1 MiB (`http.MaxBytesReader`); admin login bounds username ≤ 100 / password ≤ 72 before any store lookup.
- CORS is an explicit allowlist (`CORS_ORIGINS`); credentialed responses reflect only allowlisted origins (`Vary: Origin`). `Allow-Headers` exposes only `Content-Type, X-CSRF-Token` — `Authorization` is not advertised because Bearer auth is intentionally absent.
- Per-IP rate limiting on login / register / forgot / reset (unauthenticated routes outside CSRF).
- Upload validation: extension + `http.DetectContentType` MIME sniff (SVG rejected); `/uploads` served with nosniff + `default-src 'none'; sandbox` CSP.
- 500-level errors log internally and return the generic envelope — no `err.Error()` to clients.
- Security headers + HSTS on all responses.

## Frontend architecture (both apps share conventions)

- **Stack:** React 19, Vite, TypeScript, Tailwind v4 (`@tailwindcss/vite`, no config file — theme in `index.css`), shadcn/ui-style components on Radix in `src/components/ui/`.
- **TypeScript strictness:** `strict: true` in both apps. ESLint `@typescript-eslint/no-explicit-any: error` — `any` is only allowed with an inline `// eslint-disable-next-line` plus justification. Prefer `unknown` + narrowing; for state machines use discriminated unions, not flag soup.
- **Path alias:** `@/` → `src/` (vite + tsconfig).
- **State:** Zustand stores in `src/stores/` (one store per domain — `cartStore`, `authStore`, `catalogStore`, `avatarStore`). Async fetch lives inside store actions.
- **API layer:** all HTTP in `src/lib/api.ts`. Base URL from `VITE_API_URL` (store defaults to `https://api.mioru.store`, admin to `""`). **Cookie-based auth**: every call sets `credentials: "include"`; on mutations the helper reads the readable CSRF cookie (`csrf_token` for admin, `store_csrf` for store) and copies it into `X-CSRF-Token`. No token in `localStorage` (deliberately removed — XSS exfil risk). `getImageUrl()` prefixes relative upload paths.
- **i18n:** i18next, RU/EN/RO in `src/i18n/locales/`. All user-facing strings via i18n keys — no hard-coded copy.
- **Theme:** dark/light toggled by `light` class on `<html>`; colors are CSS variables `--color-*` in `index.css` (the source of truth — the `COLORS` object in `src/lib/constants.ts` is legacy/stale).

### Store app specifics
- Routing in `App.tsx`, all pages `lazy()`-loaded, `react-router-dom` v6. Pages in `src/routes/`.
- **3D avatar** (`src/avatar/AvatarManager.ts`): loads a GLB via `GLTFLoader`/`DRACOLoader`; on failure builds a procedural body mesh parameterized by gender/fat/muscle. Models in `public/models/`. Rendered through `@react-three/fiber` + `drei`.
- Catalog uses server-side pagination + facets (`/api/products` + `/api/products/facets`).

### Admin app specifics
- `App.tsx` routes only auth pages; everything else (`/*`) goes through `AdminLayout`, which performs the auth check. `authStore.isAuthenticated` is **tri-state** (`null | true | false`): layout renders a neutral loader while `null` and only redirects to `/login` on `false` — the cookie is HttpOnly so the answer comes from `GET /api/users/me` succeeding or returning 401.
- UI organized as **workspaces** (`src/workspaces/`, declared in `src/lib/constants.ts WORKSPACES`); only `products` is active, the rest are placeholders.
- The category tree is owned by the backend: admin fetches `/api/admin/categories` into `productStore.categories`. The source is `internal/store/migrations/002_seed_categories.sql`, pinned by `TestSeededCategoryTree`.

## Testing standard

- **Frontend:** Vitest + Testing Library (unit/component), colocated `*.test.tsx`, jsdom setup in `src/test/setup.ts`. Store also has Playwright E2E (`apps/store/e2e/*.spec.ts`).
- **Go:** stdlib `testing`, table-driven, colocated `*_test.go` in-package (white-box); black-box `_test` package only for public-API surface tests. Assertions are plain `if got != want { t.Errorf(...) }`; `testify/require` allowed but not required. Run `go test ./... -race`. `internal/auth/auth_test.go` is the worked example.
- **DB-touching logic** is tested against a real PostgreSQL, not mocks — money/stock correctness depends on real SQL semantics (constraints, `FOR UPDATE`, serialization). Use a **dedicated throwaway** DB via `TEST_DATABASE_URL` — **never** the production VPS DB, **never** reuse `DATABASE_URL`. Locally: `backend/api/docker-compose.test.yml` (postgres:16-alpine, container `mioru-postgres-test`, `127.0.0.1:55433`, creds `mioru/mioru`, db `mioru_test`). `docker compose -f backend/api/docker-compose.test.yml up -d`, then `export TEST_DATABASE_URL='postgres://mioru:mioru@127.0.0.1:55433/mioru_test?sslmode=disable'` and `go test ./... -race`. Harness in `internal/store/harness_test.go`: `NewPostgresStore` runs migrations + seeds categories, then `TRUNCATE ... RESTART IDENTITY CASCADE` resets data tables before each test (seeded categories survive). Store tests `t.Skip()` when `TEST_DATABASE_URL` is unset. Get a clean store with `testStore(t)`.
- **Priority #1 paths** (orders, payments, stock, XP/VIP) ship with tests for atomic/transactional behaviour *before* merge — TDD is non-negotiable here.
- **No `time.Sleep` / blind `setTimeout` in tests** — flaky CI. Wait on an event (channel receive, polling on a real condition, `t.Eventually`-style, Playwright `expect…toHave…`/`waitFor`), or inject the clock and advance it. A deliberate <10ms OS tick in a test that already passes deterministically is the only exception — comment why.
- **Tests are never deferred** — a fix ships with its test. Missing harness ⇒ build the harness first, not later. (Deferring is how the Go side reached zero coverage.)

## Deployment

`scripts/setup-vps.sh` provisions Ubuntu 22.04: Go 1.25, PostgreSQL 16, Nginx, 2GB swap, system user `mioru`, hardened systemd service. Secrets (DB password, `SECRET_KEY`) generated on first run and preserved in `/opt/mioru/.env`. The store ships as static `dist/`; the backend runs as a separate service. No Redis.
