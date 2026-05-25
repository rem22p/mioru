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
- **DRY** — one source of truth per fact (config, types, the category tree). Reference, don't duplicate.
- **OWASP / security-by-default** — validate all input, parameterize all SQL, never reflect internal errors to clients, enforce authz on every privileged route. See "Security posture".
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

Backend (Go 1.22, requires a running **PostgreSQL**):

```bash
cd backend/api
go run ./cmd/server      # builds binary mioru/server (gitignored)
go build ./...
```
There are currently no Go tests (no `*_test.go` files).

Seed test data (generates SQL on stdout, pipe into psql):

```bash
bash scripts/seed-300.sh | psql "$DATABASE_URL"   # 300 random products
```

## Backend architecture

- **Routing:** Go 1.22 stdlib `http.ServeMux` with method+pattern routes (`"GET /api/products/{slug}"`). All routes registered in `cmd/server/main.go`. No web framework.
- **Single datastore — PostgreSQL** (`internal/store/postgres.go`, `user_postgres.go`, `customer_postgres.go`) via `pgxpool`: users, customers, products, categories, sizes, size charts, images, and password-reset tokens.
- **Schema = inline migrations** in `PostgresStore.migrate()` (`postgres.go`), run on startup via `CREATE TABLE IF NOT EXISTS`. There is no separate migration tool.
- **First admin:** seeded inside `migrate()` (`seedAdmin`) from `BOOTSTRAP_ADMIN_*` env — idempotent `INSERT ... ON CONFLICT DO NOTHING`. Registration is invite-only, so this resolves the first-admin chicken-and-egg.
- **Auth:** JWT (`internal/auth`), HS256 with a single `SECRET_KEY`. Two audiences via the `typ` claim: `TokenTypeUser` (admin/staff) and `TokenTypeCustomer` (storefront). Enforced by `middleware.AuthMW` / `middleware.CustomerAuthMW`; `middleware.RequireAdmin` does a DB-backed role check. Public store endpoints use a plain `cors()` wrapper.
- **Rate limiting:** in-process fixed-window limiter (`middleware/memstore.go`, `MemoryRateLimiter`) on auth routes — per client IP.
- **Handlers** (`internal/handler/`): `handler.go` (admin auth + profile), `customer.go` (storefront customer auth + profile), `store.go` (public storefront read), `product.go` (admin product CRUD + image upload).
- **Uploads** saved to `UPLOAD_DIR` (default `uploads/`), served at `GET /uploads/` (hardened: nosniff + locked-down CSP).

### API contract (how the frontends talk to it)
- **Store public (no auth):** `GET /api/products` (query: `category_id, search, brand, sort, page, per_page`), `GET /api/products/{slug}`, `GET /api/categories` (returns the category **tree**, not flat).
- **Store customers (JWT typ=customer):** `POST /api/store/auth/{register,login}`, `/api/store/customers/me*`.
- **Admin (JWT typ=user):** `POST /api/auth/{register,login,forgot-password,reset-password}` (**register is invite-only** — admin-only), `/api/users/me*`, `/api/admin/products*`, `/api/admin/upload`, `/api/admin/categories`.

### Backend env vars (`internal/config/config.go`)
`SECRET_KEY` (≥32 chars, else generated + warns), `DATABASE_URL`, `PORT` (8000), `UPLOAD_DIR`, `CORS_ORIGINS` (comma-separated allowlist; when set it **replaces** the built-in defaults), `BOOTSTRAP_ADMIN_USERNAME` / `BOOTSTRAP_ADMIN_EMAIL` / `BOOTSTRAP_ADMIN_PASSWORD` (first-admin seed). Loaded from `.env` via godotenv. Token expiry hardcoded to 1440 min.

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
- **Tests:** colocated `*.test.ts(x)` next to source, Vitest + Testing Library, jsdom, setup in `src/test/setup.ts`.

### Store app specifics
- Routing in `App.tsx` — all pages `lazy()`-loaded, `react-router-dom` v6. Pages in `src/routes/`.
- **3D avatar** (`src/avatar/AvatarManager.ts`): loads a GLB via `GLTFLoader`/`DRACOLoader`; if loading fails, builds a **procedural body mesh** parameterized by gender/fat/muscle. Models in `public/models/`. Rendered through `@react-three/fiber` + `drei`.
- Catalog currently loads up to 100 products and paginates client-side.

### Admin app specifics
- `App.tsx` routes only auth pages; everything else (`/*`) goes to `AdminLayout`, which performs the auth check (no duplicate auth logic in routes).
- UI organized as **workspaces** (`src/workspaces/`, declared in `src/lib/constants.ts WORKSPACES`); only `products` is active, the rest are placeholders.
- The category tree is **hardcoded** in `apps/admin/src/lib/constants.ts CATEGORIES` and must stay in sync with what's seeded into the DB (the inline categories in `postgres.go migrate()`).

## Deployment

`scripts/setup-vps.sh` provisions an Ubuntu 22.04 VPS: Go 1.22, PostgreSQL 16, Nginx, 2GB swap, a dedicated system user `mioru`, and a hardened systemd service. Secrets (DB password, `SECRET_KEY`) are generated on first run and preserved in `/opt/mioru/.env`. The store is shipped as static `dist/`; the backend runs as a separate service. No Redis.
