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

  const newName = `Hermes ${Date.now()}`;
  await page.getByTestId("profile-display-name").fill(newName);
  // #58a6ff is in AVATAR_COLORS (see src/lib/constants.ts); click the
  // corresponding swatch so the PUT body contains a valid avatar_color.
  await page.getByTestId("profile-avatar-color-58a6ff").click();

  const putResp = page.waitForResponse(
    (r) => /\/api\/users\/me\/profile$/.test(r.url()) && r.request().method() === "PUT",
  );
  // Profile.tsx calls await fetchUser() right after updateUser() — wait for
  // that post-save GET (not the mount-time one, which AdminLayout triggers
  // in its useEffect). Polling for a second GET /api/users/me after the PUT
  // is robust against that.
  const meResp = page.waitForResponse(
    (r) => /\/api\/users\/me$/.test(r.url()) && r.request().method() === "GET",
    { timeout: 10000 },
  );
  await page.getByTestId("profile-save").click();
  expect((await putResp).status()).toBe(200);
  // The post-save GET may not be the *first* GET /api/users/me after
  // waitForResponse was registered (AdminLayout's mount-time fetch has
  // already resolved by then). The robust signal is: PUT 200, then reload
  // the page and verify the new display_name persists.
  await meResp.catch(() => {
    // mount-time fetch already consumed; reload will force a new one.
  });

  // Reload to see the persisted profile changes from the DB. The success
  // alert has a 4s auto-dismiss and is hard to assert race-free; the actual
  // proof of save is that the new name round-trips through the backend.
  await page.reload();
  await expect(page.getByTestId("profile-display-name")).toHaveValue(newName);
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
  // The protected profile endpoint is /api/users/me, not /api/auth/me (F2).
  const resp = await request.get(`${API}/api/users/me`);
  expect(resp.status()).toBe(200);
});
