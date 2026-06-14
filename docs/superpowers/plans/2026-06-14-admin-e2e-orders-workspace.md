# Admin E2E: orders-workspace — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Functional Playwright E2E for the admin orders workspace, building on the #42 foundation harness. Covers list + pagination, status update (valid + invalid), and reserves `data-testid` hooks for filters not yet implemented.

**Architecture:** New `apps/admin/e2e/orders.spec.ts` adds to the existing `apps/admin/playwright.config.ts` (from #42). One shared login via the `setup` project + `storageState` is reused (no new auth in this spec). Serial run (`workers: 1`) because specs mutate shared backend state. Each test cleans up after itself (the order it created) via the API, not the UI.

**Tech Stack:** Playwright (already wired by #42), TypeScript, React Testing Library not required (E2E only).

**Spec:** `docs/superpowers/specs/2026-06-14-admin-e2e-orders-workspace.md`
**Issue:** #39
**Dependency:** #42 must be merged first (foundation harness + storageState pattern).

**Preconditions for every test run:**
```bash
# Backend on :8000 (locally or docker)
docker compose -f backend/api/docker-compose.test.yml up -d
# Or manual:
export DATABASE_URL=postgres://mioru:mioru@127.0.0.1:55433/mioru_test?sslmode=disable
cd backend/api && go run ./cmd/server
# Then from another shell:
cd apps/admin && npm run e2e
```

---

## File Structure

| File | Responsibility |
|---|---|
| `apps/admin/src/workspaces/Orders.tsx` (modify) | Add `data-testid` hooks: `orders-row`, `orders-status-select`, `orders-pagination-{next,prev}`, `orders-list`. No behaviour change. |
| `apps/admin/src/components/orders/*` (create/modify) | If status-update is a sub-component, ensure it forwards a `data-testid` prop. |
| `apps/admin/e2e/orders.spec.ts` (create) | Functional E2E: list + status update + invalid status error. |
| `apps/admin/e2e/helpers.ts` (modify, optional) | Add a `seedOrderViaAPI()` helper that creates an order via the customer API + login (uses the existing customer flow, not the admin backend) so the spec has a real order to manipulate. |

---

## Task 1: Inventory existing orders workspace

**Files:** `apps/admin/src/workspaces/Orders.tsx`

- [ ] Read `apps/admin/src/workspaces/Orders.tsx` and document:
  - Which backend calls it makes (`GET /api/admin/orders`, `PATCH /api/admin/orders/{id}/status`).
  - Whether status-update is in the same component or a child.
  - Current selectors (data-testid present? i18n copy? Tailwind classes?).
  - Whether filters/pagination are already implemented (per #39: «если есть»).
- [ ] Read `apps/admin/e2e/helpers.ts` (from #42) to see what's available.

---

## Task 2: Add `data-testid` hooks to Orders.tsx

**Files:** `apps/admin/src/workspaces/Orders.tsx`, optional sub-components

Required testids (per #39 + convention from PR #37/42):
- `orders-list` — the root table/list element.
- `orders-row` — each row (one per order in the list).
- `orders-row-{id}` — optional, for `getByTestId('orders-row').filter({ hasText: '#123' })`.
- `orders-status-select` (or `orders-row-{id}-status-select`) — the status update control.
- `orders-pagination-next`, `orders-pagination-prev` — only if pagination exists in UI.
- `orders-error` — error banner (e.g. «Invalid status value»).

Rules (per CLAUDE.md «Stable selectors»):
- [ ] Testids only — never i18n copy or Tailwind classes.
- [ ] No duplication: each testid is unique within its scope.

Run `npm run build` after each component change to catch typos.

---

## Task 3: Add `seedOrderViaAPI()` helper to helpers.ts (optional)

**Files:** `apps/admin/e2e/helpers.ts`

If the admin has no API to create orders (it doesn't, by design — orders come from the customer flow), we need to seed via the customer path:

```ts
// Pseudocode
export async function seedOrderViaAPI(request: APIRequestContext): Promise<{ id: number; status: string }> {
  // 1. Register a fresh customer (or use seeded one)
  // 2. Browse to /catalog, get a known in-stock product slug (similar to #37 helper)
  // 3. POST /api/store/orders (with Idempotency-Key + CSRF)
  // 4. Return { id, status: 'new' }
}
```

If this duplicates code from `apps/store/e2e/user-flow.spec.ts` (from #37), factor the shared bits into `apps/admin/e2e/helpers.ts` and consider moving to a workspace-level `e2e-utils` package later (out of scope here).

- [ ] Decide: import from store E2E helpers (if possible) OR duplicate-and-extend in admin helpers.
- [ ] If `seedOrderViaAPI` exists, write it.

---

## Task 4: Write `apps/admin/e2e/orders.spec.ts`

**Files:** `apps/admin/e2e/orders.spec.ts`

The spec must:
1. Reuse the #42 `setup` project for login (or import `storageState` directly).
2. Self-skip if backend is down (per #42 convention).
3. Cover the matrix below.

**Test cases (TDD):**

```ts
test('admin can list orders and the seed order appears', async ({ page, request }) => {
  const seeded = await seedOrderViaAPI(request);
  await page.goto('/orders'); // or /admin/orders
  await expect(page.getByTestId('orders-row').first()).toBeVisible();
  await expect(page.getByTestId(`orders-row-${seeded.id}`)).toBeVisible();
});

test('admin can update order status (new → processing)', async ({ page, request }) => {
  const seeded = await seedOrderViaAPI(request);
  await page.goto('/orders');
  const row = page.getByTestId(`orders-row-${seeded.id}`);
  await row.getByTestId('orders-status-select').selectOption('processing');
  // assert the API call returned 200 and the UI re-rendered
  await expect(row).toContainText('processing');
});

test('admin sees an error on invalid status', async ({ page, request }) => {
  const seeded = await seedOrderViaAPI(request);
  // Stub the PATCH response to return 400 (avoids the per-type validation
  // order on the backend; we just want to verify the UI surfaces it).
  await page.route('**/api/admin/orders/*/status', (route) =>
    route.fulfill({ status: 400, body: JSON.stringify({ code: 'VALIDATION_FAILED', error: 'invalid status' }) })
  );
  await page.goto('/orders');
  const row = page.getByTestId(`orders-row-${seeded.id}`);
  await row.getByTestId('orders-status-select').selectOption('bogus');
  await expect(page.getByTestId('orders-error')).toBeVisible();
});
```

Anti-flake (per CLAUDE.md):
- [ ] No `waitForTimeout`. Use `waitForResponse` on the real API + `expect.toBeVisible()`.
- [ ] Unique order ID per run (timestamp-based slug, like #37).
- [ ] Cleanup the seeded order via API after the test (`DELETE` if supported, else leave for the per-process DB reset).

---

## Task 5: Run + verify

```bash
cd apps/admin
# 1. Lint
npm run lint
# 2. Build
npm run build
# 3. Unit tests still pass
npx vitest run
# 4. E2E (requires backend on :8000)
npx playwright test e2e/orders.spec.ts
```

- [ ] All 3+ tests green.
- [ ] Lint 0 errors.
- [ ] Build green.
- [ ] `vitest` unit tests still green (sanity: no component breakage from testid additions).
- [ ] No `waitForTimeout` introduced.

---

## Definition of Done

- [ ] `apps/admin/e2e/orders.spec.ts` зелёный.
- [ ] testid'ы проставлены и не дублируют i18n-копию.
- [ ] PR с этим планом прошёл ревью и смержен в main.
- [ ] Issue #39 закрыт через PR.

---

## Out of scope (deferred)

- Filter-by-status, filter-by-date UI (per #39: «если есть» — реализация отложена).
- Visual regression / a11y (separate spec).
- Concurrency test for two admins updating the same order (manual check).
