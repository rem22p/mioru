import { test, expect } from "@playwright/test";
import { API, backendUp, ADMIN_PASS, ADMIN_USER, login } from "./helpers";

// Per #42 convention: the backend is run manually in this project, so we
// self-skip the whole suite when it is unreachable rather than hard-fail.
let backendReachable = false;
test.beforeAll(async ({ request }) => {
  backendReachable = await backendUp(request);
});
test.beforeEach(() => {
  test.skip(!backendReachable, `backend not reachable at ${API} (start it on :8000)`);
});

test("admin can edit display name and avatar color", async ({ page }) => {
  await login(page);
  await page.goto("/profile");

  await expect(page.getByTestId("profile-display-name")).toBeVisible();
  await expect(page.getByTestId("profile-save")).toBeVisible();

  // Change the display name and a different avatar color.
  const newName = `Hermes ${Date.now()}`;
  await page.getByTestId("profile-display-name").fill(newName);
  // Pick any other color (default is #f85149).
  await page.getByTestId("profile-avatar-color-44944a").click();

  const putResp = page.waitForResponse(
    (r) => /\/api\/users\/me\/profile$/.test(r.url()) && r.request().method() === "PUT",
  );
  await page.getByTestId("profile-save").click();
  expect((await putResp).status()).toBe(200);

  // Success alert appears (auto-dismisses after 4s — capture it before it does).
  await expect(page.getByTestId("profile-alert-success")).toContainText("Профиль сохранён");
});

test("mismatched confirm-password surfaces a UI validation error (no network)", async ({ page }) => {
  await login(page);
  await page.goto("/profile");

  await page.getByTestId("profile-change-password-open").click();

  await page.getByTestId("profile-password-current").fill(ADMIN_PASS);
  await page.getByTestId("profile-password-new").fill("new-pass-1");
  await page.getByTestId("profile-password-confirm").fill("new-pass-2"); // mismatch

  await page.getByTestId("profile-password-submit").click();

  await expect(page.getByTestId("profile-alert-error")).toContainText("Пароли не совпадают");
});

test("wrong current password shows a UI error and does NOT invalidate the session", async ({ page, request }) => {
  await login(page);
  await page.goto("/profile");
  await page.getByTestId("profile-change-password-open").click();

  await page.getByTestId("profile-password-current").fill("definitely-wrong-current-password");
  await page.getByTestId("profile-password-new").fill("GoodPassword123");
  await page.getByTestId("profile-password-confirm").fill("GoodPassword123");

  const putResp = page.waitForResponse(
    (r) => /\/api\/users\/me\/password$/.test(r.url()) && r.request().method() === "PUT",
  );
  await page.getByTestId("profile-password-submit").click();
  expect((await putResp).status()).toBe(401);

  await expect(page.getByTestId("profile-alert-error")).toBeVisible();

  // The session must still work — the rest of the admin SPA is reachable.
  const resp = await request.get(`${API}/api/auth/me`);
  expect(resp.status()).toBe(200);
});

test("changing password invalidates the old session (security-critical)", async ({ page, request }) => {
  // SECURITY: per CLAUDE.md (security-by-default) and the integration tests
  // in #34/#35, the auth middleware rejects tokens with iat < password_changed_at.
  // This E2E pins the contract end-to-end: after PUT /api/users/me/password
  // succeeds, the old cookie no longer authenticates.
  await login(page);
  await page.goto("/profile");

  // Capture the auth cookie while it is still valid.
  const cookies = await page.context().cookies();
  const authCookie = cookies.find(
    (c) => c.name === "admin_token" || c.name === "session" || c.name.includes("admin"),
  );
  expect(authCookie, "admin auth cookie should be set after login()").toBeTruthy();

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
  const oldResp = await request.get(`${API}/api/auth/me`, {
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
