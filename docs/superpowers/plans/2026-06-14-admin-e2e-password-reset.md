# Admin E2E: forgot/reset-password — Implementation Plan

> **For agentic workers:** Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Functional Playwright E2E for the admin forgot/reset-password flow, including a dev-only test endpoint that issues a fresh raw reset token (gated by env + build tag + constant-time secret compare — per the #49 hardening pattern).

**Architecture:**
1. **`POST /api/_test/create-reset-token`** (test-only, `//go:build e2e`, gated on `!IsProduction() && E2E_RESET_KEY != ""`) — issues a fresh reset token for a given username, persists only its SHA-256 hash (per `TestResetTokenHashedAtRest` security contract), returns the **raw** token once. Authenticates via `X-E2E-Reset-Key` header (constant-time compare to `E2E_RESET_KEY` env var). **No raw token in logs, no raw column in DB, no enumeration** — the dev endpoint *generates* the token rather than reading it back.
2. **`apps/admin/e2e/password-reset.spec.ts`** (authenticated project) — drives forgot → fetch token via test endpoint → reset UI → assert login with new password works.

**Why "create" not "read":** the production store stores only the SHA-256 hash of the reset token (per the security contract pinned by `TestResetTokenHashedAtRest` and CLAUDE.md "не логировать plaintext"). We can't reverse the hash. The test endpoint must therefore *issue a fresh* token rather than read the existing one — the same way the production forgot-password flow issues a new token, hashes it, and (in dev) returns it to the caller. The shape of the response is `{ token: "<raw>" }`; the lifetime and hash semantics match the regular flow.

**Tech Stack:** Playwright (from #42), Go stdlib `net/http` for the test endpoint (mirroring `test_reset.go` in #49), TypeScript.

**Spec:** `docs/superpowers/specs/2026-06-14-admin-e2e-password-reset.md`
**Issue:** #41
**Dependencies:** #42 (admin E2E foundation) + #49 (test endpoint hardening pattern).

---

## File Structure

| File | Responsibility |
|---|---|
| `backend/api/internal/handler/test_reset_token.go` (create, `//go:build e2e`) | Handler for `POST /api/_test/create-reset-token` |
| `backend/api/cmd/server/test_routes_e2e.go` (modify) | Register the new route when `E2E_RESET_KEY` set + dev mode |
| `backend/api/internal/handler/test_reset_token_test.go` (create, `//go:build e2e`) | Unit tests (mirroring `test_reset_test.go` from #49) |
| `apps/admin/src/pages/ForgotPassword.tsx` (modify) | `data-testid` hooks: `forgot-email`, `forgot-submit`, `forgot-error`, `forgot-success` |
| `apps/admin/src/pages/ResetPassword.tsx` (modify) | `data-testid` hooks: `reset-password`, `reset-password-confirm`, `reset-submit`, `reset-error`, `reset-success` |
| `apps/admin/e2e/password-reset.spec.ts` (create) | Full forgot → reset → login flow |

---

## Task 1: Add test-only create-reset-token endpoint

**Files:**
- `backend/api/internal/handler/test_reset_token.go` (new, `//go:build e2e`)
- `backend/api/internal/handler/test_reset_token_test.go` (new, `//go:build e2e`)
- `backend/api/cmd/server/test_routes_e2e.go` (modify — add registration)

### Security model (mirrors #49 `ResetAdminForTestHandler`)

```go
//go:build e2e

// TestCreateResetTokenHandler issues a fresh password-reset token for the
// given username and returns the raw token in the response. The handler
// uses the regular store.CreateResetToken path, so the SHA-256 hash lands
// in password_reset_tokens exactly as the production forgot-password
// flow would do. The raw token is the canonical response field — it is
// NOT written to logs, NOT stored in a side column, NOT echoed to any
// other channel. The handler is reachable only on e2e-tagged builds
// (file build-tag), only when !cfg.IsProduction(), and only when the
// caller presents the right X-E2E-Reset-Key header (constant-time
// compared to E2E_RESET_KEY env). Three independent barriers: compile
// time, runtime gate, auth gate.
```

### Wiring

```go
// cmd/server/test_routes_e2e.go — add to the existing block:
if getenv("E2E_RESET_KEY") != "" {
    resetTokenH := handler.NewTestCreateResetTokenHandler(pgStore)
    mux.HandleFunc("POST /api/_test/create-reset-token", resetTokenH.ServeHTTP)
}
```

### Handler behaviour

- `POST /api/_test/create-reset-token` with body `{ "username": "admin" }`.
- Auth: `X-E2E-Reset-Key` header constant-time-compared to `E2E_RESET_KEY` env (fail-closed: empty server-side secret = 503 `TEST_RESET_DISABLED`).
- Looks up the user by username; 404 `NOT_FOUND` if absent.
- Calls `store.CreateResetToken(ctx, username, rawToken)` — **same path the production forgot-password handler uses** — so the token's lifetime (1h) and hash semantics match exactly.
- Returns `{ "token": "<raw>" }`. The raw token is generated via `crypto/rand` and is 32 bytes base64 (mirrors the production token format).
- Errors: 400 `VALIDATION_FAILED` (missing username), 403 `FORBIDDEN` (bad key), 404 `NOT_FOUND` (no such user), 500 `INTERNAL` (DB error — generic envelope, `slog` server-side, never leak `err.Error()` to client).

### Unit tests (in `test_reset_token_test.go`, `//go:build e2e`)

- `TestCreateResetTokenRejectsMissingKey` (403, store not invoked)
- `TestCreateResetTokenRejectsWrongKey` (403, store not invoked)
- `TestCreateResetTokenRequiresServerSideKey` (503, store not invoked)
- `TestCreateResetTokenHappyPath` (200, returns a non-empty token, store was called with the same token — the round-trip pin)
- `TestCreateResetTokenErrorPathDoesNotLeak` (500, body has no internal-error substring)
- `TestCreateResetTokenNotFound` (404 for unknown username, store not invoked)

Mirror `test_reset_test.go` from #49 (which has the same shape and is the reference for `subtle.ConstantTimeCompare` + `slog.Error` + `jsonerr.ErrorCode`).

- [ ] Run `go test -tags e2e ./internal/handler/ -run TestCreateResetToken -v` — all PASS.

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
- testid-only selectors (no i18n copy, no Tailwind classes)
- unique within their scope
- `npm run build` after each to catch typos

---

## Task 3: Write `apps/admin/e2e/password-reset.spec.ts`

**Files:** `apps/admin/e2e/password-reset.spec.ts`

Self-skip when the dev-only test endpoint is not available (production binary, or no `E2E_RESET_KEY` set). Forgot-password always returns 200 without enumerating users; reset-password takes the raw token from the test endpoint.

Test cases (TDD):

```ts
// 1. Self-skip: POST /api/_test/create-reset-token must exist for the
//    spec to be useful. On production builds, the build-tag is excluded
//    and the route is 404 — self-skip cleanly. On dev builds without
//    E2E_RESET_KEY, the route returns 503 — also self-skip.
test.beforeAll(async ({ request }) => {
  const probe = await request.post(`${API}/api/_test/create-reset-token`, {
    headers: { "X-E2E-Reset-Key": process.env.E2E_RESET_KEY ?? "" },
    data: { username: "admin" },
    failOnStatusCode: false,
  });
  if (probe.status() === 404 || probe.status() === 503) {
    test.skip(true, "test endpoint not available (production build or E2E_RESET_KEY unset)");
  }
  if (probe.status() === 403) {
    test.skip(true, "X-E2E-Reset-Key mismatch — set E2E_RESET_KEY in test env to match backend");
  }
});

// 2. Forgot-password MUST return the same response for an existing user
//    and a non-existing user — privacy / no-enumeration contract.
test('forgot-password returns 200 without enumerating users', async ({ request }) => {
  const real = await request.post(`${API}/api/auth/forgot-password`, {
    data: { email: process.env.E2E_ADMIN_EMAIL },
  });
  const fake = await request.post(`${API}/api/auth/forgot-password`, {
    data: { email: `definitely-not-a-user-${Date.now()}@nowhere.example` },
  });
  expect(real.status()).toBe(200);
  expect(fake.status()).toBe(200);
  expect(await real.text()).toBe(await fake.text());
});

// 3. Full happy path: forgot → fetch raw token via test endpoint →
//    reset UI → login with new password. Restoration of the old
//    password happens via a second call to the test endpoint at the
//    end (we use ChangePassword indirectly: ask the backend to issue
//    a reset token, then have a separate step that uses it). For
//    simplicity and to keep this test self-contained, the spec
//    resets the password to a known value at the END using the
//    /api/_test/reset-admin endpoint from #49 (which the dev backend
//    also exposes). The test is therefore safe to re-run.
test('admin can reset password with a valid token and login with the new one', async ({ page, request }) => {
  const email = process.env.E2E_ADMIN_EMAIL!;
  const originalPassword = process.env.E2E_ADMIN_PASSWORD!;

  // 1. Trigger forgot-password (this also issues a real reset token
  //    through the production path, but we don't read that one — we
  //    use the test endpoint to issue our own so the spec doesn't
  //    depend on log-parsing or hash-reversal).
  await request.post(`${API}/api/auth/forgot-password`, { data: { email } });

  // 2. Issue a fresh test token for the admin username.
  const tokenResp = await request.post(`${API}/api/_test/create-reset-token`, {
    headers: { "X-E2E-Reset-Key": process.env.E2E_RESET_KEY ?? "" },
    data: { username: process.env.E2E_ADMIN_USERNAME },
  });
  expect(tokenResp.status()).toBe(200);
  const { token } = await tokenResp.json();
  expect(typeof token).toBe('string');
  expect(token.length).toBeGreaterThan(0);

  // 3. Visit the reset page in the browser.
  await page.goto(`/reset/${token}`);
  const newPassword = `reset-${Date.now()}-xY7`;
  await page.getByTestId('reset-password').fill(newPassword);
  await page.getByTestId('reset-password-confirm').fill(newPassword);
  const submitResp = page.waitForResponse(
    (r) => r.url().endsWith('/api/auth/reset-password') && r.request().method() === 'POST',
  );
  await page.getByTestId('reset-submit').click();
  expect((await submitResp).status()).toBe(200);
  await expect(page.getByTestId('reset-success')).toBeVisible();

  // 4. Login with the new password — must succeed.
  await page.goto('/login');
  await page.getByTestId('login-username').fill(process.env.E2E_ADMIN_USERNAME);
  await page.getByTestId('login-password').fill(newPassword);
  const loginResp = page.waitForResponse(
    (r) => r.url().endsWith('/api/auth/login') && r.request().method() === 'POST',
  );
  await page.getByTestId('login-submit').click();
  expect((await loginResp).status()).toBe(200);

  // 5. Restore the original password for the next test run.
  //    The dev backend exposes POST /api/_test/reset-admin (from #49) for
  //    exactly this — re-issuing the admin password + resetting
  //    password_changed_at to one hour in the past.
  const restore = await request.post(`${API}/api/_test/reset-admin`, {
    headers: { "X-E2E-Reset-Key": process.env.E2E_RESET_KEY ?? "" },
    data: {
      username: process.env.E2E_ADMIN_USERNAME,
      hashed_password: process.env.E2E_ADMIN_BCRYPT_HASH,
    },
  });
  expect(restore.status()).toBe(200);
});

// 4. Invalid token — UI error, no password change.
test('reset with an invalid token shows a UI error', async ({ page }) => {
  await page.goto('/reset/invalid-token-here');
  await page.getByTestId('reset-password').fill('some-password-12345');
  await page.getByTestId('reset-password-confirm').fill('some-password-12345');
  const submitResp = page.waitForResponse(
    (r) => r.url().endsWith('/api/auth/reset-password') && r.request().method() === 'POST',
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
node_modules/.bin/tsc -b   # type-check

# Backend must be on :8000 with APP_ENV=development AND E2E_RESET_KEY set.
cd ../..
go build -tags e2e -o /tmp/mioru ./backend/api/cmd/server
DATABASE_URL=... APP_ENV=development E2E_RESET_KEY=ci-... \
  SECRET_KEY=ci-... BOOTSTRAP_ADMIN_PASSWORD=Admin12345! /tmp/mioru &

cd apps/admin
E2E_RESET_KEY=ci-... E2E_ADMIN_PASSWORD=Admin12345! \
  E2E_ADMIN_USERNAME=admin E2E_ADMIN_EMAIL=admin@mioru.store \
  E2E_ADMIN_BCRYPT_HASH='$2a$12$...' \
  npx playwright test e2e/password-reset.spec.ts --reporter=list
```

- [ ] All tests green.
- [ ] Production check: with `APP_ENV=production`, the test endpoint returns 404 and the spec **self-skips** (not fails).
- [ ] Lint 0 errors, tsc 0 NEW errors.
- [ ] No `waitForTimeout`.

---

## Definition of Done

- [ ] `apps/admin/e2e/password-reset.spec.ts` зелёный.
- [ ] Test endpoint `POST /api/_test/create-reset-token` зарегистрирован **только** в non-production + `e2e` build + правильный `X-E2E-Reset-Key`.
- [ ] 6 unit-теста (handler, `//go:build e2e`) зелёные.
- [ ] testid'ы проставлены.
- [ ] PR ревьюнут и смержен.
- [ ] Issue #41 закрыт.

---

## Out of scope (deferred)

- Token expiry E2E (covered by integration tests in #35).
- Multi-channel email delivery.
- Rate-limit on forgot/reset (covered by integration tests).
- Admin self-reset (per #40 spec, password-change flow lives in `security.spec.ts`).
- Reading raw tokens from logs (rejected — see Security model above).
- Storing raw tokens in a side column (rejected — same reason).
