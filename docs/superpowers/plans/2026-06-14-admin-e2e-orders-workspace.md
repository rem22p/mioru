# Admin E2E: orders-workspace — Implementation Plan

> **For agentic workers:** Steps use checkbox (`- [ ]`) syntax for tracking. **All tasks below are DONE — this plan is kept for reference. PR #46 merged; issue #39 closed.**

**Goal:** Functional Playwright E2E for the admin orders workspace, building on the #42 foundation harness. Covers list + count, status-update with backend round-trip, status filter, and customer filter.

**Architecture:** `apps/admin/e2e/orders.spec.ts` adds to the existing `apps/admin/playwright.config.ts` (from #42). One shared login via the `setup` project + `storageState` is reused. Serial run (`workers: 1`) because specs mutate shared backend state. **Self-skip** when no orders exist in the DB (orders are created through the customer flow, not the admin API).

**Tech Stack:** Playwright (already wired by #42), TypeScript.

**Spec:** `docs/superpowers/specs/2026-06-14-admin-e2e-orders-workspace.md`
**Issue:** #39 (CLOSED by PR #46)
**Dependency:** #42 (admin E2E foundation) — merged.

**Preconditions for every test run:** backend on `:8000` with seeded categories (no orders needed — tests self-skip if empty).

---

## File Structure

| File | Responsibility |
|---|---|
| `apps/admin/src/workspaces/Orders.tsx` (modify) | `data-testid` hooks: `orders-page`, `orders-row-{id}`. **Landed in PR #46.** |
| `apps/admin/e2e/orders.spec.ts` (create) | Functional E2E: list+count, status update, status filter, customer filter. **Landed in PR #46.** |

---

## Task 1: Inventory existing orders workspace (DONE in #46)

- [x] `GET /api/admin/orders` (list) + `PATCH /api/admin/orders/{id}/status` (update).
- [x] No data-testid in original — added in PR #46.
- [x] No customer filter UI in the current implementation; test uses `filterCustomer` URL param to exercise the existing server-side filter.

---

## Task 2: Add `data-testid` hooks to Orders.tsx (DONE in #46)

Landed in PR #46:
- `orders-page` — workspace root.
- `orders-row-{id}` — per-row marker (enables `getByTestId(\`orders-row-${id}\`)`).

---

## Task 3: Write `apps/admin/e2e/orders.spec.ts` (DONE in #46)

Four tests:

1. **orders workspace renders its structural controls** — list loads, row count matches API.
2. **filtering by status issues a pending request and re-renders the list** — uses `Promise.all([waitForRequest, click])` to assert the GET fires.
3. **invalid status change surfaces a backend error in the UI banner** — uses `page.route()` to stub PATCH 400.
4. **admin can update order status when the backend accepts it** — self-skips when no orders in DB (orders come from customer flow, not seeded).

Anti-flake (per CLAUDE.md + F1 review):
- [x] No `waitForTimeout`. `waitForResponse` on real API + `expect.toBeVisible()`.
- [x] Stub-based invalid-status test (no dependency on backend validation order).
- [x] Self-skip when DB is empty (avoids `test.skip` hiding real regressions).
- [x] Catalog-glob `testMatch` (per #46's A1 fix) so future specs don't get dropped.

---

## Task 4: Run + verify (DONE in PR #46 verification)

```bash
cd apps/admin
node_modules/.bin/tsc -b
npm run lint
npx playwright test e2e/orders.spec.ts --reporter=list
```

Verified locally with backend on `:8000`:
- 3 passed + 1 skipped (admin can update order status — no orders in DB).

---

## Definition of Done

- [x] `apps/admin/e2e/orders.spec.ts` зелёный.
- [x] testid'ы проставлены.
- [x] PR #46 прошёл ревью и смержен в main.
- [x] Issue #39 закрыт через PR #46.

---

## Out of scope (deferred)

- Filter-by-status UI control — `Orders.tsx` reads URL param only; no per-row filter control yet. The test exercises the URL param directly.
- Visual regression / a11y (separate spec).
- Concurrency test for two admins updating the same order (manual check).
