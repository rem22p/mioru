# Admin E2E: profile & settings — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Functional Playwright E2E for the admin profile (edit + password change with session invalidation) and settings (theme/scale/locale persistence).

**Architecture:** Two new spec files (`profile.spec.ts`, `settings.spec.ts`) reusing the #42 harness. The **password-change** spec is the security-critical one: it asserts that the old token gets `401 AUTH_INVALID` after the change, validating the `iat < password_changed_at` middleware contract end-to-end.

**Tech Stack:** Playwright (from #42), TypeScript, `localStorage` introspection via `page.evaluate()`.

**Spec:** `docs/superpowers/specs/2026-06-14-admin-e2e-profile-settings.md`
**Issue:** #40
**Dependency:** #42 must be merged first (foundation harness + storageState pattern).

**Preconditions for every test run:** same as #42 (`backend on :8000` or `docker compose`).

---

## File Structure

| File | Responsibility |
|---|---|
| `apps/admin/src/workspaces/Profile.tsx` (modify) | `data-testid` hooks: `profile-first-name`, `profile-last-name`, `profile-email`, `profile-save`, `profile-password-current`, `profile-password-new`, `profile-password-confirm`, `profile-password-submit`, `profile-error`, `profile-success`. |
| `apps/admin/src/workspaces/Settings.tsx` (modify) | `data-testid` hooks: `settings-theme-{light,dark}`, `settings-scale-{compact,comfortable,spacious}` (or slider testid), `settings-locale-{ru,en,ro}`, `settings-save`. |
| `apps/admin/e2e/profile.spec.ts` (create) | Profile edit + password change + session invalidation. |
| `apps/admin/e2e/settings.spec.ts` (create) | Theme/scale/locale toggle + `localStorage` persist. |
| `apps/admin/e2e/helpers.ts` (modify, optional) | Add `changePasswordViaAPI()` (uses raw auth cookie + a new password). |

---

## Task 1: Inventory existing Profile & Settings

**Files:** `apps/admin/src/workspaces/Profile.tsx`, `Settings.tsx`

- [ ] Read both files; document:
  - Backend calls: `PUT /api/users/me/profile`, `PUT /api/users/me/password`, etc.
  - How theme/scale/locale are persisted (localStorage keys).
  - Current selectors.
- [ ] Read `apps/admin/e2e/helpers.ts` for the #42 inventory (don't duplicate).

---

## Task 2: Add `data-testid` hooks

Required testids (mirroring the patterns from PR #37 / #42):

### Profile.tsx
- `profile-first-name`, `profile-last-name`, `profile-email` (inputs)
- `profile-save` (save button)
- `profile-password-current`, `profile-password-new`, `profile-password-confirm`
- `profile-password-submit` (change password button)
- `profile-error`, `profile-success` (banners)

### Settings.tsx
- `settings-theme-light`, `settings-theme-dark`
- `settings-scale-{...}` (one testid per option)
- `settings-locale-ru`, `settings-locale-en`, `settings-locale-ro`
- `settings-save` if there's a save button (else auto-save on change)

Rules (per CLAUDE.md):
- [ ] testid-only selectors
- [ ] unique within their scope

Run `npm run build` to catch typos.

---

## Task 3: Write `apps/admin/e2e/profile.spec.ts`

**Files:** `apps/admin/e2e/profile.spec.ts`

Test cases (TDD):

```ts
test('admin can edit profile (first/last name + email)', async ({ page, request }) => {
  // Reuse setup storageState for login
  await page.goto('/profile');
  await page.getByTestId('profile-first-name').fill('HermesNew');
  await page.getByTestId('profile-last-name').fill('AgentNew');
  await page.getByTestId('profile-email').fill(`hermes+${Date.now()}@example.com`);
  const saveResp = page.waitForResponse((r) =>
    r.url().includes('/api/users/me/profile') && r.request().method() === 'PUT'
  );
  await page.getByTestId('profile-save').click();
  expect((await saveResp).status()).toBe(200);
  await expect(page.getByTestId('profile-success')).toBeVisible();
});
```

```ts
test('changing password invalidates the old session (security-critical)', async ({ page, request }) => {
  // 1. Reuse setup login; we're already authenticated.
  // 2. Change password to a new value.
  await page.goto('/profile');
  const newPassword = 'newpass-' + Date.now();
  await page.getByTestId('profile-password-current').fill(process.env.E2E_ADMIN_PASSWORD!);
  await page.getByTestId('profile-password-new').fill(newPassword);
  await page.getByTestId('profile-password-confirm').fill(newPassword);
  const pwResp = page.waitForResponse((r) =>
    r.url().includes('/api/users/me/password') && r.request().method() === 'PUT'
  );
  await page.getByTestId('profile-password-submit').click();
  expect((await pwResp).status()).toBe(200);

  // 3. The page reloads and the OLD token is now rejected.
  // Use a raw API request with the OLD cookie (capture before the change).
  const cookies = await page.context().cookies();
  const adminCookie = cookies.find((c) => c.name === '<admin-auth-cookie-name>')!;
  const oldResp = await request.get('/api/admin/users', {
    headers: { Cookie: `${adminCookie.name}=${adminCookie.value}` },
  });
  expect(oldResp.status()).toBe(401);
  // Body shape: { error: "session revoked", code: "AUTH_INVALID" }
  const body = await oldResp.json();
  expect(body.code).toBe('AUTH_INVALID');
});
```

If step 3 returns 200 instead of 401 — that's a **HIGH-severity security regression**. File as a new issue, `t.Skip` the test (per #32 / #31 convention: pin the contract).

```ts
test('mismatched new-password confirmation shows a UI error', async ({ page }) => {
  await page.goto('/profile');
  await page.getByTestId('profile-password-current').fill(process.env.E2E_ADMIN_PASSWORD!);
  await page.getByTestId('profile-password-new').fill('new-password-1');
  await page.getByTestId('profile-password-confirm').fill('new-password-2'); // mismatch
  await page.getByTestId('profile-password-submit').click();
  await expect(page.getByTestId('profile-error')).toBeVisible();
});
```

---

## Task 4: Write `apps/admin/e2e/settings.spec.ts`

**Files:** `apps/admin/e2e/settings.spec.ts`

Test cases:

```ts
test('theme toggle flips <html class="light"> and persists in localStorage', async ({ page }) => {
  await page.goto('/settings');
  await page.getByTestId('settings-theme-dark').click();
  await expect(page.locator('html')).not.toHaveClass(/light/);
  const theme = await page.evaluate(() => localStorage.getItem('theme'));
  expect(theme).toBe('dark');

  // Reload — must still be dark
  await page.reload();
  await expect(page.locator('html')).not.toHaveClass(/light/);
});

test('UI scale change writes --ui CSS variable and persists', async ({ page }) => {
  await page.goto('/settings');
  await page.getByTestId('settings-scale-spacious').click();
  const ui = await page.evaluate(() => getComputedStyle(document.documentElement).getPropertyValue('--ui'));
  expect(ui.trim()).not.toBe('');
  const persisted = await page.evaluate(() => localStorage.getItem('ui-scale'));
  expect(persisted).toBe('spacious');
});

test('locale change updates <html lang> and persists', async ({ page }) => {
  await page.goto('/settings');
  await page.getByTestId('settings-locale-en').click();
  await expect(page.locator('html')).toHaveAttribute('lang', 'en');
  const persisted = await page.evaluate(() => localStorage.getItem('locale'));
  expect(persisted).toBe('en');
});
```

Use the actual `localStorage` keys from the implementation (Task 1). If they're not yet stable, pin them in the spec via a helper.

---

## Task 5: Run + verify

```bash
cd apps/admin
npm run lint
npm run build
npx vitest run
npx playwright test e2e/profile.spec.ts e2e/settings.spec.ts
```

- [ ] All tests green.
- [ ] Lint 0 errors.
- [ ] Build green.
- [ ] No `waitForTimeout`.
- [ ] Session-invalidation test: if it fails, file as issue and `t.Skip` per #31/#32 convention.

---

## Definition of Done

- [ ] `profile.spec.ts` зелёный (включая session-invalidation test).
- [ ] `settings.spec.ts` зелёный.
- [ ] testid'ы проставлены.
- [ ] PR ревьюнут и смержен.
- [ ] Issue #40 закрыт.

---

## Out of scope (deferred)

- Profile field validation tests (email format, etc.) — covered by integration tests in #35.
- Multi-locale UI text coverage — translation tests are a separate concern.
- Accessibility audit (axe, etc.).
