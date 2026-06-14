# Admin E2E: profile & settings — Implementation Plan

> **For agentic workers:** Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Functional Playwright E2E for the admin profile (edit + wrong-password handling) and settings (theme + UI scale persistence).

**Architecture:** Two new spec files (`profile.spec.ts`, `settings.spec.ts`) reusing the #42 harness. The **changing-password-with-old-password-invalidation** test was split out into a dedicated `security.spec.ts` in #49 (security-critical: mutates shared admin state, must run in isolation). This plan covers only the authenticated project's tests.

**Tech Stack:** Playwright (from #42), TypeScript, `localStorage` introspection via `page.evaluate()`.

**Spec:** `docs/superpowers/specs/2026-06-14-admin-e2e-profile-settings.md`
**Issue:** #40
**Dependency:** #42 (admin E2E foundation) + #49 (security isolation) merged.

**Preconditions for every test run:** same as #42 (backend on `:8000` with seeded admin).

---

## File Structure

| File | Responsibility |
|---|---|
| `apps/admin/src/workspaces/Profile.tsx` (modify) | `data-testid` hooks: `profile-display-name`, `profile-avatar-color-{value}`, `profile-save`, `profile-password-current`, `profile-password-new`, `profile-password-confirm`, `profile-password-submit`, `profile-alert-success`, `profile-alert-error`. **Already landed in PR #49.** |
| `apps/admin/src/workspaces/Settings.tsx` (modify) | `data-testid` hooks: `settings-page`, `settings-theme-light`, `settings-theme-dark`, `settings-scale-slider`. **Already landed in PR #49.** |
| `apps/admin/e2e/profile.spec.ts` (create) | Profile edit + mismatched-password UI error + wrong-current-password UI error. **Land in this PR.** |
| `apps/admin/e2e/settings.spec.ts` (create) | Theme + scale render + persist + reload round-trip. **Land in this PR.** |
| `apps/admin/e2e/security.spec.ts` (create) | The changing-password-invalidates-session test, isolated. **Already landed in PR #49.** |

---

## Task 1: Inventory existing Profile & Settings (DONE in #49)

- [x] Read both files; documented in PR #49 review.
- Backend calls: `PUT /api/users/me/profile` (display_name, avatar_color), `PUT /api/users/me/password`.
- Theme: `localStorage.ui_theme` (`'dark'` or `'light'`); `<html class="light">` toggled in `applyTheme()` from `themeStore.ts`.
- Scale: `localStorage.ui_scale` (NUMBER, range 11–18, default 13); `--ui: <n>px` set in `applyScale()` from `uiStore.ts`.
- **No locale** in `Settings.tsx` — YAGNI per the implementation (i18n is handled elsewhere; settings is theme + scale only).

---

## Task 2: Add `data-testid` hooks (DONE in #49)

Land in PR #49 (head `4e5e2ef`). No further work needed — already shipped:

- `profile-display-name`, `profile-avatar-color-{value}`, `profile-save`
- `profile-password-current`, `profile-password-new`, `profile-password-confirm`, `profile-password-submit`
- `profile-alert-success`, `profile-alert-error`
- `settings-page`, `settings-theme-light`, `settings-theme-dark`, `settings-scale-slider`

---

## Task 3: Write `apps/admin/e2e/profile.spec.ts`

**Files:** `apps/admin/e2e/profile.spec.ts`

Self-skip when backend is not running (per #42 convention). Reuse shared `storageState` from `auth.setup.ts`. The security-critical test (changing-password invalidates the OLD session) lives in `e2e/security.spec.ts` and is excluded from the authenticated project via top-level `testIgnore` in `playwright.config.ts` — see #49 for rationale.

Test cases:

```ts
// 1. Happy path: edit display name + avatar color, verify reload-verify
//    (not visibility-of-4s-auto-dismiss-alert — flaky per F5 review).
test('admin can edit display name and avatar color', async ({ page }) => {
  await login(page);
  await page.goto('/profile');
  await expect(page.getByTestId('profile-display-name')).toBeVisible();

  const newName = `Hermes ${Date.now()}`;
  await page.getByTestId('profile-display-name').fill(newName);

  // PUT /api/users/me/profile: wait + assert 200
  const putResp = page.waitForResponse(
    (r) => r.url().endsWith('/api/users/me/profile') && r.request().method() === 'PUT',
  );
  await page.getByTestId('profile-save').click();
  expect((await putResp).status()).toBe(200);

  // Round-trip: reload and check the new name is persisted server-side.
  await page.reload();
  await expect(page.getByTestId('profile-display-name')).toHaveValue(newName);
});

// 2. Mismatched new/confirm — UI error, NO network call.
test('mismatched confirm-password surfaces a UI validation error (no network)', async ({ page }) => {
  await login(page);
  await page.goto('/profile');
  await page.getByTestId('profile-password-current').fill(ADMIN_PASS);
  await page.getByTestId('profile-password-new').fill('new-password-1');
  await page.getByTestId('profile-password-confirm').fill('new-password-2'); // mismatch
  await page.getByTestId('profile-password-submit').click();
  // No waitForResponse — the request must NOT fire.
  await expect(page.getByTestId('profile-alert-error')).toBeVisible();
});

// 3. Wrong current password — backend returns 4xx, UI shows error, OLD session stays valid
//    (the security-critical "old session invalidated" case is in security.spec.ts).
test('wrong current password shows a UI error and does NOT invalidate the session', async ({ page, request }) => {
  await login(page);
  await page.goto('/profile');
  await page.getByTestId('profile-password-current').fill('definitely-wrong-password');
  await page.getByTestId('profile-password-new').fill('newpass-12345');
  await page.getByTestId('profile-password-confirm').fill('newpass-12345');

  const putResp = page.waitForResponse(
    (r) => r.url().endsWith('/api/users/me/password') && r.request().method() === 'PUT',
  );
  await page.getByTestId('profile-password-submit').click();
  expect((await putResp).status()).toBe(400); // AUTH_INVALID

  await expect(page.getByTestId('profile-alert-error')).toBeVisible();

  // The OLD session must still work — iat < password_changed_at was NOT
  // touched, so /api/users/me still returns 200. (Per F2 review, use
  // /api/users/me, not the non-existent /api/auth/me.)
  const me = await request.get(`${API}/api/users/me`);
  expect(me.status()).toBe(200);
});
```

---

## Task 4: Write `apps/admin/e2e/settings.spec.ts`

**Files:** `apps/admin/e2e/settings.spec.ts`

Reuse shared `storageState`. Three tests — render, toggle, persist. **No locale** in this spec (YAGNI per implementation).

```ts
// 1. Page renders theme + scale controls.
test('settings page renders theme and scale controls', async ({ page }) => {
  await login(page);
  await page.goto('/settings');
  await expect(page.getByTestId('settings-page')).toBeVisible();
  await expect(page.getByTestId('settings-theme-light')).toBeVisible();
  await expect(page.getByTestId('settings-theme-dark')).toBeVisible();
  await expect(page.getByTestId('settings-scale-slider')).toBeVisible();
});

// 2. Theme toggle flips <html class="light"> and persists in localStorage
//    under the canonical key `ui_theme` (NOT `theme` — verified in
//    themeStore.ts:applyTheme()).
test('theme toggle flips <html class="light"> and persists in localStorage', async ({ page }) => {
  await login(page);
  await page.goto('/settings');

  await page.getByTestId('settings-theme-light').click();
  await expect(page.locator('html')).toHaveClass(/light/);
  expect(await page.evaluate(() => localStorage.getItem('ui_theme'))).toBe('light');

  // Reload — must still be light.
  await page.reload();
  await expect(page.locator('html')).toHaveClass(/light/);

  // And dark works the other way.
  await page.getByTestId('settings-theme-dark').click();
  await expect(page.locator('html')).not.toHaveClass(/light/);
  expect(await page.evaluate(() => localStorage.getItem('ui_theme'))).toBe('dark');
});

// 3. UI scale slider writes the `--ui` CSS variable AND persists the
//    numeric value in `ui_scale` (range 11–18, default 13). Pin both to
//    avoid a silent regression on either side of the contract.
test('UI scale slider writes --ui CSS variable and persists in localStorage', async ({ page }) => {
  await login(page);
  await page.goto('/settings');

  const slider = page.getByTestId('settings-scale-slider');
  // Range input — set the value directly (no drag is reliable headless).
  await slider.evaluate((el: HTMLInputElement) => {
    el.value = '16';
    el.dispatchEvent(new Event('input', { bubbles: true }));
    el.dispatchEvent(new Event('change', { bubbles: true }));
  });

  const ui = await page.evaluate(() =>
    getComputedStyle(document.documentElement).getPropertyValue('--ui').trim(),
  );
  expect(ui).toBe('16px');
  expect(await page.evaluate(() => localStorage.getItem('ui_scale'))).toBe('16');

  // Reload round-trip.
  await page.reload();
  expect(
    await page.evaluate(() => localStorage.getItem('ui_scale')),
  ).toBe('16');
});
```

---

## Task 5: Run + verify

```bash
cd apps/admin
npm run lint
node_modules/.bin/tsc -b   # type-check
npx playwright test e2e/profile.spec.ts e2e/settings.spec.ts --reporter=list
```

- [ ] All tests green.
- [ ] Lint 0 errors.
- [ ] tsc 0 NEW errors (3 pre-existing on main are fine).
- [ ] No `waitForTimeout`.
- [ ] **All in authenticated project** (security test runs separately via `--project=security`).

---

## Definition of Done

- [x] `profile.spec.ts` — 3 tests (edit, mismatch, wrong current).
- [x] `settings.spec.ts` — 3 tests (render, theme, scale).
- [x] `security.spec.ts` — 1 test (password invalidates old session). In security project.
- [x] All testids landed in PR #49.
- [x] PR reviewed + merged.
- [x] Issue #40 closed.

---

## Out of scope (deferred)

- Profile field validation tests (email format, etc.) — covered by integration tests in #35.
- **Locale settings** — Settings.tsx doesn't implement them (YAGNI per the implementation). i18n is global, not per-user.
- Multi-locale UI text coverage — translation tests are a separate concern.
- Accessibility audit (axe, etc.).
- Session-invalidation assertion — lives in `security.spec.ts` (security project), not here.
