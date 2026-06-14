# Admin E2E: forgot/reset-password — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Functional Playwright E2E for the admin forgot/reset-password flow, plus a test-only endpoint to obtain the raw reset token (gated by `APP_ENV != production`).

**Architecture:** Two new pieces:
1. **`POST /api/test/reset-token`** (test-only, env-gated) — returns the raw reset token for a given email. The endpoint MUST be a no-op (404) when `APP_ENV == production`.
2. **`apps/admin/e2e/password-reset.spec.ts`** — drives forgot → email log → fetch token via test endpoint → reset UI → assert login with new password works.

**Tech Stack:** Playwright (from #42), Go stdlib `net/http` for the test endpoint, TypeScript.

**Spec:** `docs/superpowers/specs/2026-06-14-admin-e2e-password-reset.md`
**Issue:** #41
**Dependency:** #42 must be merged first (foundation harness + storageState pattern).

**Preconditions for every test run:** `backend on :8000` with `APP_ENV=development` (otherwise the test endpoint returns 404 and the spec skips).

---

## File Structure

| File | Responsibility |
|---|---|
| `backend/api/cmd/server/main.go` (modify) | Register the `/api/test/reset-token` route ONLY when `APP_ENV != production`. Wire it before the 404 fallback. |
| `backend/api/internal/handler/test_only.go` (create) | Handler: `POST /api/test/reset-token` with body `{ email }`. Returns `{ token: "<raw>" }` if a recent reset token exists for the email, else 404. |
| `apps/admin/src/pages/ForgotPassword.tsx` (modify) | `data-testid` hooks: `forgot-email`, `forgot-submit`, `forgot-error`, `forgot-success`. |
| `apps/admin/src/pages/ResetPassword.tsx` (modify) | `data-testid` hooks: `reset-password`, `reset-password-confirm`, `reset-submit`, `reset-error`, `reset-success`. |
| `apps/admin/e2e/password-reset.spec.ts` (create) | Full forgot → reset flow. |

---

## Task 1: Add test-only reset-token endpoint

**Files:** `backend/api/internal/handler/test_only.go`, `backend/api/cmd/server/main.go`

**Security:** This endpoint MUST be **inert** in production. Guard at the wiring level:

```go
// cmd/server/main.go
if cfg.AppEnv != "production" {
    mux.HandleFunc("POST /api/test/reset-token", handler.NewTestOnlyHandler(st).GetResetToken)
}
```

Handler behaviour:
- `POST /api/test/reset-token` with JSON body `{ "email": "..." }`.
- Look up the most recent `password_reset_tokens` row for that email.
- If found **and** still within expiry window: return `{ "token": "<raw-token>" }` where `<raw-token>` is reconstructed from the SHA-256 hash... wait, **we can't reverse SHA-256**.

**Re-think:** the store stores only the SHA-256 hash. To return a raw token, the spec must change the **data model** OR the test must trigger a fresh token creation.

### Two valid approaches:

#### Approach A (recommended): raw token logged + test endpoint reads logs

- The store writes the raw token to a structured log line at creation time: `RESET_TOKEN_EMAIL=foo@bar TOKEN=raw-token-here`.
- The test endpoint reads the most recent line for the given email from a per-process ring buffer (or a file).
- Production keeps the **log write** (it already exists for dev), but the **endpoint** is gated.

This preserves the security model (raw token only in logs, not in DB) and is test-friendly.

#### Approach B: store raw token alongside hash

- Add a `raw_token` column to `password_reset_tokens`, populated **only when** a special env var is set (e.g. `TEST_KEEP_RAW_TOKENS=1`).
- The test endpoint reads the raw column.
- Production never sets `TEST_KEEP_RAW_TOKENS`, so the column stays NULL.

Less elegant but simpler to implement.

**For this plan, take Approach A.** The store already logs the token; the test endpoint reads from a ring buffer that the test hooks into.

- [ ] Read `backend/api/internal/email` and `backend/api/internal/store/password_reset*.go` to understand the current log-write path.
- [ ] Decide: log ring buffer in store, OR separate `internal/testonly` package.
- [ ] Implement: `(*PostgresStore).GetLatestResetToken(ctx, email) (string, error)` that reads from the ring buffer.
- [ ] Wire `POST /api/test/reset-token` to call it.

**Verification (TDD):**
- [ ] When `APP_ENV=production`, `POST /api/test/reset-token` returns 404 (and route not registered).
- [ ] When `APP_ENV=development`, after a forgot-password request, the test endpoint returns the raw token.
- [ ] When `APP_ENV=development` and no recent token, returns 404.

Add a small unit test for the route gating in `internal/handler/test_only_test.go` (mirror the #31-style JSON envelope pin).

---

## Task 2: Add `data-testid` hooks to ForgotPassword.tsx and ResetPassword.tsx

**Files:** `apps/admin/src/pages/ForgotPassword.tsx`, `ResetPassword.tsx`

### ForgotPassword.tsx
- `forgot-email` (input)
- `forgot-submit` (button)
- `forgot-error`, `forgot-success` (banners)

### ResetPassword.tsx (route `/reset/:token`)
- `reset-password` (new password input)
- `reset-password-confirm` (confirm input)
- `reset-submit` (button)
- `reset-error`, `reset-success` (banners)

Rules (per CLAUDE.md):
- [ ] testid-only selectors.
- [ ] No i18n copy duplication.

Run `npm run build` after each.

---

## Task 3: Write `apps/admin/e2e/password-reset.spec.ts`

**Files:** `apps/admin/e2e/password-reset.spec.ts`

Test cases (TDD):

```ts
// Self-skip if test endpoint is not available (production env, etc.)
test.beforeAll(async ({ request }) => {
  // POST /api/test/reset-token with empty body should 404 in production,
  // or 400 in dev. We just check that the route exists.
  // (route registered → 405 Method Not Allowed, route missing → 404)
});

test('forgot-password returns 200 without enumerating users', async ({ request }) => {
  const respA = await request.post('/api/auth/forgot-password', { data: { email: 'real@user.com' } });
  const respB = await request.post('/api/auth/forgot-password', { data: { email: 'definitely-not-a-user@nowhere.com' } });
  expect(respA.status()).toBe(200);
  expect(respB.status()).toBe(200);
  const bodyA = await respA.text();
  const bodyB = await respB.text();
  expect(bodyA).toBe(bodyB); // same body for both — no enumeration
});

test('admin can reset password with a valid token and login with the new one', async ({ page, request }) => {
  // 1. Pick a target admin email (or seed one via helpers).
  const email = process.env.E2E_ADMIN_EMAIL!;
  const oldPassword = process.env.E2E_ADMIN_PASSWORD!;

  // 2. Trigger forgot.
  await request.post('/api/auth/forgot-password', { data: { email } });

  // 3. Fetch the raw token via the test endpoint.
  const tokenResp = await request.post('/api/test/reset-token', { data: { email } });
  expect(tokenResp.status()).toBe(200);
  const { token } = await tokenResp.json();

  // 4. Visit the reset page in the browser.
  await page.goto(`/reset/${token}`);
  const newPassword = 'reset-' + Date.now();
  await page.getByTestId('reset-password').fill(newPassword);
  await page.getByTestId('reset-password-confirm').fill(newPassword);
  const submitResp = page.waitForResponse((r) =>
    r.url().includes('/api/auth/reset-password') && r.request().method() === 'POST'
  );
  await page.getByTestId('reset-submit').click();
  expect((await submitResp).status()).toBe(200);
  await expect(page.getByTestId('reset-success')).toBeVisible();

  // 5. Login with the new password — must succeed.
  await page.goto('/login');
  await page.getByTestId('login-email').fill(email);
  await page.getByTestId('login-password').fill(newPassword);
  const loginResp = page.waitForResponse((r) =>
    r.url().includes('/api/auth/login') && r.request().method() === 'POST'
  );
  await page.getByTestId('login-submit').click();
  expect((await loginResp).status()).toBe(200);

  // 6. Restore the old password for the next test run.
  // (Requires a helper to set password directly, or trigger another reset.)
});

test('reset with an invalid token shows a UI error', async ({ page }) => {
  await page.goto('/reset/invalid-token-here');
  await page.getByTestId('reset-password').fill('some-password');
  await page.getByTestId('reset-password-confirm').fill('some-password');
  const submitResp = page.waitForResponse((r) =>
    r.url().includes('/api/auth/reset-password') && r.request().method() === 'POST'
  );
  await page.getByTestId('reset-submit').click();
  expect((await submitResp).status()).toBe(400);
  await expect(page.getByTestId('reset-error')).toBeVisible();
});
```

---

## Task 4: Run + verify

```bash
cd apps/admin
npm run lint
npm run build
npx vitest run

# Backend must be on :8000 with APP_ENV=development
cd ../..
DATABASE_URL=... APP_ENV=development go run ./cmd/server &
cd apps/admin
npx playwright test e2e/password-reset.spec.ts
```

- [ ] All tests green.
- [ ] Production check: with `APP_ENV=production`, the test endpoint returns 404 and the spec **self-skips** (not fails).
- [ ] Lint 0 errors, build green, vitest green.

---

## Definition of Done

- [ ] `apps/admin/e2e/password-reset.spec.ts` зелёный.
- [ ] Test endpoint `POST /api/test/reset-token` зарегистрирован **только** в non-production.
- [ ] testid'ы проставлены.
- [ ] PR ревьюнут и смержен.
- [ ] Issue #41 закрыт.

---

## Out of scope (deferred)

- Token expiry E2E (covered by integration tests in #35).
- Multi-channel email delivery.
- Rate-limit on forgot/reset (covered by integration tests).
- Self-reset для admin (per #40 spec, password-change flow).
