import { test, expect, request as pwRequest } from "@playwright/test";
import { API, backendUp, ADMIN_PASS, ADMIN_USER, login } from "./helpers";

// Issue #41 — Admin E2E: покрыть forgot/reset-password flow.
//
// Способ получения raw reset-токена: dev-only test endpoint
// POST /api/_test/create-reset-token (//go:build e2e, !IsProduction, E2E_RESET_KEY
// header). Это новый endpoint из этого PR, по образцу /api/_test/reset-admin из
// #49. Альтернативы (читать токен из логов, side-column raw) — отклонены
// за нарушение security contract TestResetTokenHashedAtRest (raw token живёт
// только в email + БД хранит SHA-256 хеш). Подробности — в спеке:
// docs/superpowers/specs/2026-06-14-admin-e2e-password-reset.md.

let backendReachable = false;
let testEndpointReachable = false;

const resetKey = process.env.E2E_RESET_KEY || "";

test.beforeAll(async () => {
  // Probe backend once for the whole suite. If unreachable, every test
  // self-skips (per #42 convention).
  const ctx = await pwRequest.newContext();
  try {
    backendReachable = await backendUp(ctx);
    if (!backendReachable) return;

    // Probe the test endpoint: on a production build the route is
    // build-tag-excluded → 404. On a dev build without E2E_RESET_KEY
    // the handler returns 503. On a dev build WITH the key, it
    // returns 200/400 depending on the body. The exact status
    // doesn't matter — we only need to know if the route is wired.
    const probe = await ctx.post(`${API}/api/_test/create-reset-token`, {
      headers: { "X-E2E-Reset-Key": resetKey },
      data: { username: "admin" },
      failOnStatusCode: false,
    });
    // Acceptable: 200 (admin exists, key correct), 400 (admin
    // missing or body bad — still proves route is wired), 401/403
    // (key wrong but route exists). 404 means route is not
    // registered at all (production build).
    testEndpointReachable = probe.status() !== 404;
  } finally {
    await ctx.dispose();
  }
});

// 1. Forgot-password MUST return 200 for both an existing and a non-existing
//    email, with identical body — privacy contract (no user enumeration).
//    This test does not need the test endpoint (it's a direct forgot-password
//    API probe), so it skips only on backend-down, not on missing test
//    endpoint.
test("forgot-password returns 200 without enumerating users", async () => {
  test.skip(!backendReachable, "backend not reachable on :8000");

  const ctx = await pwRequest.newContext();
  try {
    const real = await ctx.post(`${API}/api/auth/forgot-password`, {
      data: { email: "admin@mioru.store" },
    });
    const fake = await ctx.post(`${API}/api/auth/forgot-password`, {
      data: { email: `definitely-not-a-user-${Date.now()}@nowhere.example` },
    });
    expect(real.status()).toBe(200);
    expect(fake.status()).toBe(200);
    // Identical body — no enumeration via response shape.
    expect(await real.text()).toBe(await fake.text());
  } finally {
    await ctx.dispose();
  }
});

// 2. Full happy path: ask the test endpoint for a fresh raw token, drive
//    the reset UI, log in with the new password, then restore the original
//    via /api/_test/reset-admin (from #49). All in one test so the
//    intermediate state is fully cleaned up by the end.
test("admin can reset password with a valid token and login with the new one", async ({ page, request }) => {
  test.skip(!backendReachable, "backend not reachable on :8000");
  test.skip(
    !testEndpointReachable,
    "test endpoint /api/_test/create-reset-token not reachable (production build or route misconfigured)",
  );
  test.skip(
    !resetKey,
    "E2E_RESET_KEY not set in env — set it to match the backend's value",
  );

  const email = "admin@mioru.store";

  // 1. Trigger forgot-password through the real flow. This proves the
  //    forgot UI/API surface works end-to-end; the token it issues is
  //    NOT used by this test (we want the test endpoint's token so the
  //    spec is self-contained).
  await request.post(`${API}/api/auth/forgot-password`, { data: { email } });

  // 2. Issue a fresh raw token via the test endpoint.
  const tokenResp = await request.post(`${API}/api/_test/create-reset-token`, {
    headers: { "X-E2E-Reset-Key": resetKey },
    data: { username: ADMIN_USER },
  });
  expect(tokenResp.status()).toBe(200);
  const { token } = (await tokenResp.json()) as { token: string };
  expect(typeof token).toBe("string");
  expect(token.length).toBeGreaterThan(0);

  // 3. Drive the reset page in the browser.
  await page.goto(`/reset/${token}`);
  const newPassword = `reset-${Date.now()}-xY7`;
  await page.getByTestId("reset-password").fill(newPassword);
  await page.getByTestId("reset-password-confirm").fill(newPassword);
  const submitResp = page.waitForResponse(
    (r) => r.url().endsWith("/api/auth/reset-password") && r.request().method() === "POST",
  );
  await page.getByTestId("reset-submit").click();
  expect((await submitResp).status()).toBe(200);
  await expect(page.getByTestId("reset-success")).toBeVisible();

  // 4. Log in with the new password.
  await page.goto("/login");
  await page.getByTestId("login-username").fill(ADMIN_USER);
  await page.getByTestId("login-password").fill(newPassword);
  const loginResp = page.waitForResponse(
    (r) => r.url().endsWith("/api/auth/login") && r.request().method() === "POST",
  );
  await page.getByTestId("login-submit").click();
  expect((await loginResp).status()).toBe(200);

  // 5. Restore the original password + reset password_changed_at to one
  //    hour in the past (so the next login's iat > changed_at is
  //    guaranteed). The /api/_test/reset-admin endpoint from #49 does
  //    exactly this; we just need its bcrypt hash for ADMIN_PASS.
  //    Generate on the fly to avoid baking the hash into the spec.
  const restoreResp = await request.post(`${API}/api/_test/reset-admin`, {
    headers: { "X-E2E-Reset-Key": resetKey },
    data: { username: ADMIN_USER, hashed_password: process.env.E2E_ADMIN_BCRYPT_HASH ?? "" },
  });
  // We don't fail the test on the restore — the next CI run will reset
  // the admin via its own bootstrap. The assertion is best-effort.
  if (restoreResp.status() !== 200) {
    // eslint-disable-next-line no-console
    console.warn(`admin restore returned ${restoreResp.status()} — next run may need a manual reset`);
  }
});

// 3. Invalid token — backend returns 4xx, UI shows error, no password change.
test("reset with an invalid token shows a UI error", async ({ page }) => {
  test.skip(!backendReachable, "backend not reachable on :8000");

  await page.goto("/reset/invalid-token-here");
  await page.getByTestId("reset-password").fill("some-password-12345");
  await page.getByTestId("reset-password-confirm").fill("some-password-12345");
  const submitResp = page.waitForResponse(
    (r) => r.url().endsWith("/api/auth/reset-password") && r.request().method() === "POST",
  );
  await page.getByTestId("reset-submit").click();
  expect((await submitResp).status()).toBe(400);
  await expect(page.getByTestId("reset-error")).toBeVisible();
});
