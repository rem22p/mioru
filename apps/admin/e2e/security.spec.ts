import { test, expect, request as pwRequest } from "@playwright/test";
import { API, backendUp, ADMIN_PASS, ADMIN_USER, login } from "./helpers";

// SECURITY-CRITICAL TEST — isolated in its own spec because it MUTATES the
// shared admin user's password. Including it in the regular authenticated
// suite corrupts the next login's iat vs password_changed_at check, causing
// cascading login() failures downstream (see PR #49 review for the failure
// pattern this fix addresses).
//
// Isolated via:
//  1. Dedicated Playwright project `security` (testMatch in
//     playwright.config.ts) — the main `authenticated` project ignores this
//     file via testIgnore.
//  2. beforeAll below hits POST /api/_test/reset-admin (dev-mode only) to
//     UPSERT the bootstrap admin with a fresh bcrypt hash for ADMIN_PASS
//     (= "Admin12345!"), with password_changed_at pinned to one hour in the
//     past so the next login's iat > changed_at is guaranteed. Idempotent
//     across CI re-runs.
//  3. If the backend is built with APP_ENV=production, the reset endpoint
//     returns 404 and the test skips cleanly — defense in depth.
//
// The bcrypt hash below was generated with `go run /tmp/genhash.go 'Admin12345!'`
// (cost 12). It must match what BOOTSTRAP_ADMIN_PASSWORD would produce; the
// test endpoint round-trips through the same bcrypt verifier, so any drift
// would surface as a 401 on the first login().

const ADMIN_BCRYPT_HASH =
  "$2a$12$VGxg/AC6FYML.JmVFvE7W.agxpL1jQzJZGAitUvoLk/4xMorUJ3wS";

let backendReachable = false;
test.beforeAll(async () => {
  // Per #42 convention: the backend is run manually in this project, so we
  // self-skip the whole suite when it is unreachable rather than hard-fail.
  const ctx = await pwRequest.newContext();
  try {
    backendReachable = await backendUp(ctx);
  } finally {
    await ctx.dispose();
  }
  test.skip(!backendReachable, `backend not reachable at ${API} (start it on :8000)`);

  // Reset the admin user so this spec always starts from a known state. The
  // truncation also clears password_changed_at's "now" offset that other
  // tests may have left behind, so login() won't reject tokens with
  // iat < password_changed_at on the very first call.
  const pgCtx = await pwRequest.newContext();
  try {
    const res = await pgCtx.post(`${API}/api/_test/reset-admin`, {
      headers: { "x-test-reset-key": "playwright-security" },
      data: {
        username: ADMIN_USER,
        hashed_password: ADMIN_BCRYPT_HASH,
      },
    });
    // If the test-reset endpoint doesn't exist, the security project relies
    // on the backend's bootstrap admin being correct. That's a CI config
    // concern (BOOTSTRAP_ADMIN_PASSWORD) — we don't fail the suite here.
    if (!res.ok()) {
      test.skip(
        true,
        `admin reset endpoint returned ${res.status()} — security test requires a dedicated reset path; run in CI with --project=security AND backend with admin-reset enabled.`,
      );
    }
  } finally {
    await pgCtx.dispose();
  }
});

test("changing password invalidates the old session (security-critical)", async ({ page, request }) => {
  // SECURITY: per CLAUDE.md (security-by-default) and the integration tests
  // in #34/#35, the auth middleware rejects tokens with iat < password_changed_at.
  // This E2E pins the contract end-to-end: after PUT /api/users/me/password
  // succeeds, the old cookie no longer authenticates.
  await login(page);
  await page.goto("/profile");

  // Capture the auth cookie while it is still valid. The admin session
  // cookie is `auth_token` (see internal/cookieauth/cookieauth.go:17) — F3.
  const cookies = await page.context().cookies();
  const authCookie = cookies.find((c) => c.name === "auth_token");
  expect(authCookie, "admin auth cookie (auth_token) should be set after login()").toBeTruthy();

  // Open the change-password panel and submit valid new credentials.
  await page.getByTestId("profile-change-password-open").click();
  const newPassword = `NewPass-${Date.now()}-xY7`;

  await page.getByTestId("profile-password-current").fill(ADMIN_PASS);
  await page.getByTestId("profile-password-new").fill(newPassword);
  await page.getByTestId("profile-password-confirm").fill(newPassword);

  const putResp = page.waitForResponse(
    (r) => /\/api\/users\/me\/password$/.test(r.url()) && r.request().method() === "PUT",
  );
  await page.getByTestId("profile-password-submit").click();
  expect((await putResp).status()).toBe(200);

  // Try the OLD cookie against a protected endpoint — must be 401 AUTH_INVALID.
  // The protected endpoint is /api/users/me, not /api/auth/me (F2).
  const oldResp = await request.get(`${API}/api/users/me`, {
    headers: { Cookie: `${authCookie!.name}=${authCookie!.value}` },
  });
  expect(oldResp.status()).toBe(401);
  const body = await oldResp.json();
  expect(body.code).toBe("AUTH_INVALID");

  // Restore the bootstrap password so subsequent test runs still work.
  // Login with the new password, then PUT it back to ADMIN_PASS.
  await page.goto("/login");
  await page.getByTestId("login-username").fill(ADMIN_USER);
  await page.getByTestId("login-password").fill(newPassword);
  await page.getByTestId("login-submit").click();
  await expect(page.getByTestId("product-add")).toBeVisible();

  await page.goto("/profile");
  await page.getByTestId("profile-change-password-open").click();
  await page.getByTestId("profile-password-current").fill(newPassword);
  await page.getByTestId("profile-password-new").fill(ADMIN_PASS);
  await page.getByTestId("profile-password-confirm").fill(ADMIN_PASS);

  const restoreResp = page.waitForResponse(
    (r) => /\/api\/users\/me\/password$/.test(r.url()) && r.request().method() === "PUT",
  );
  await page.getByTestId("profile-password-submit").click();
  expect((await restoreResp).status()).toBe(200);
});
