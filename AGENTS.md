# AGENTS.md — Mioru

## Repository layout

- `mioru-admin/` — admin panel
  - `backend/` — **Go** service (legacy `.venv/` and `requirements.txt` are artifacts; no Python source exists)
  - `frontend/` — React 19 + Vite
  - `lending - coming soon/` — separate Vite/React 18 landing page (space in directory name)
- `mioru-store/` — Next.js 16 + React 19 e-commerce store (static export to `dist/`)
- `mioru-site/` — placeholder (`index.html`)

No workspace manager. Each package has its own `node_modules` and lockfile; run `npm install` independently.

## Backend (Go)

- **Entrypoint:** `mioru-admin/backend/cmd/server/main.go`
- **Build:** `cd mioru-admin/backend && go build -o mioru ./cmd/server`
- **Run:** `./mioru` (default port `8000`, overridable via `PORT`)
- **Hard dependency:** Redis must be running before startup (`REDIS_URL` or `REDIS_ADDR`).
- **Env:** `godotenv.Load()` reads `mioru-admin/backend/.env`. The real `.env` is committed and contains secrets; `.env.example` is incomplete.
- **CORS whitelist** is hardcoded in `main.go` (includes `localhost:5173`, `localhost:8080`, `admin.mioru.store`). Edit `main.go` to add origins.
- **Security headers** are manually injected in `main.go`.
- **Committed binaries:** `backend/mioru` and `backend/server` are checked in. Rebuild after Go changes; do not edit binaries directly.

## Frontend (admin)

- **Entrypoint:** `mioru-admin/frontend/src/main.tsx`
- **Dev:** `cd mioru-admin/frontend && npm run dev` → port `8080`, host `0.0.0.0`
- **Proxy:** `/api` → `http://localhost:8000`, `/ws` → `ws://localhost:8000` (configured in `vite.config.ts`)
- **Build:** `npm run build` → `dist/`
- **Production API URL:** `VITE_API_URL` in `.env.production` (created at deploy time, not committed)
- **No tests, lint, or formatter** configured.

## Local development startup order

1. Ensure Redis is running (`redis-cli ping` → `PONG`)
2. Start backend: `cd mioru-admin/backend && ./mioru` (rebuild first if Go sources changed)
3. Start frontend: `cd mioru-admin/frontend && npm run dev`

## mioru-store (E-commerce)

- **Entrypoint:** `mioru-store/src/app/page.tsx`
- **Stack:** Next.js 16, React 19, Tailwind CSS v4 (`@tailwindcss/postcss`), App Router, static export
- **Dev:** `cd mioru-store && npm run dev`
- **Build:** `npm run build` → static export to `dist/`
- **Tests:** `npm run test` (Vitest), `npm run test:e2e` (Playwright)
- **Has its own `AGENTS.md`** with a breaking-change warning for Next.js. Read it before modifying store code.

## Notable quirks

- The `lending - coming soon/` directory name contains spaces. Quote it in shell commands.
- No automated tests in `mioru-admin`.
- The `render.yaml` in `backend/` defines the Render.com deploy config.
- Deployment docs are in `mioru-admin/DEPLOY-GUIDE.md` (Russian).
