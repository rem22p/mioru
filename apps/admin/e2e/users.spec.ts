import { test, expect } from "@playwright/test";
import { backendUp } from "./helpers";

// Super-admin user management. The bootstrap admin is seeded as super_admin,
// so the /users workspace is reachable and POST/DELETE on /api/admin/users
// are authorized. Each run creates a uniquely-named user and cleans it up.
// The session is established once by auth.setup.ts (storageState).
test.describe("admin user management (super-admin)", () => {
  test.beforeEach(async ({ request }) => {
    test.skip(!(await backendUp(request)), "backend on :8000 unreachable");
  });

  test("create → delete an admin user", async ({ page }) => {
    await page.goto("/users");
    // Visible add button proves the super_admin gate let us in.
    await expect(page.getByTestId("user-add")).toBeVisible();

    const stamp = Date.now();
    const username = `e2e_user_${stamp}`;
    const email = `e2e+${stamp}@example.com`;

    // ── Create ──
    await page.getByTestId("user-add").click();
    await page.getByTestId("uf-first-name").fill("E2E");
    await page.getByTestId("uf-last-name").fill("Tester");
    await page.getByTestId("uf-username").fill(username);
    await page.getByTestId("uf-email").fill(email);
    await page.getByTestId("uf-password").fill("Sup3rSecret!");

    const created = page.waitForResponse(
      (r) =>
        r.url().includes("/api/auth/register") &&
        r.request().method() === "POST",
    );
    await page.getByTestId("uf-submit").click();
    expect((await created).status()).toBe(201);

    const row = page.locator(
      `[data-testid="user-row"][data-username="${username}"]`,
    );
    await expect(row).toBeVisible();

    // ── Delete ──
    await row.getByTestId("user-delete").click();
    const deleted = page.waitForResponse(
      (r) =>
        r.url().includes(`/api/admin/users/${username}`) &&
        r.request().method() === "DELETE",
    );
    await page.getByTestId("user-delete-confirm").click();
    expect((await deleted).status()).toBe(204);

    await expect(row).toHaveCount(0);
  });

  test("create form rejects a too-short password", async ({ page }) => {
    await page.goto("/users");
    await page.getByTestId("user-add").click();
    await page.getByTestId("uf-first-name").fill("E2E");
    await page.getByTestId("uf-last-name").fill("Tester");
    await page.getByTestId("uf-username").fill(`e2e_short_${Date.now()}`);
    await page.getByTestId("uf-email").fill(`e2e+short${Date.now()}@example.com`);
    await page.getByTestId("uf-password").fill("short");
    await page.getByTestId("uf-submit").click();
    // Client-side guard (≥8 chars) blocks the request.
    await expect(page.getByText("Пароль минимум 8 символов")).toBeVisible();
  });
});
