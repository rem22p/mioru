# AGENTS.md — Mioru

## Repository layout (monorepo)

```
apps/store/       — E-commerce store (React 19 + Vite + shadcn/ui)
backend/api/      — Go API (REST + WebSocket + JWT + Redis)
packages/shared/  — Shared types/utilities (future)
docs/             — Documentation
```

Old projects (`mioru-admin/`, `mioru-store/`, `mioru-site/`) are kept for reference but excluded from git.

## Workflow rules

**Every change follows this sequence:**
1. Agent analyzes the problem and presents a plan
2. User approves the plan
3. Agent makes changes
4. User checks the result
5. User gives command to commit
6. Agent commits with description of what was done

**No commits without user approval. No changes without a plan first.**

## Store frontend

- **Entrypoint:** `apps/store/src/main.tsx`
- **Dev:** `cd apps/store && npm run dev` → port `5173`
- **Build:** `npm run build` → `dist/`
- **Tests:** `npm test` (Vitest), `npm run test:e2e` (Playwright)
- **Colors:** Accent `#44944A`, theme-aware via CSS variables `--color-*`
- **i18n:** 3 languages (RU, EN, RO), keys in `src/i18n/locales/`
- **Theme:** Dark/light via class on `<html>`, CSS variables in `index.css`
- **3D Avatar:** GLB loader with procedural fallback in `src/avatar/AvatarManager.ts`
- **State:** Zustand stores in `src/stores/`

## Backend

- **Entrypoint:** `backend/api/cmd/server/main.go`
- **Hard dependency:** Redis
- **Port:** `8000` (default)

## Git

- Remote: `git@github.com:rem22p/mioru.git`
- User pushes; agent cannot push (SSH passphrase)
- Agent can commit, create branches, etc.
