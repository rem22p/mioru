import { test, expect } from "@playwright/test";
import { backendUp, login, ADMIN_USER } from "./helpers";

test.describe("admin auth", () => {
  test.beforeEach(async ({ request }) => {
    test.skip(!(await backendUp(request)), "backend on :8000 unreachable");
  });

  test("rejects invalid credentials and stays on /login", async ({ page }) => {
    await page.goto("/login");
    await page.getByTestId("login-username").fill("nope_" + Date.now());
    await page.getByTestId("login-password").fill("wrongpassword");
    await page.getByTestId("login-submit").click();

    await expect(page.getByTestId("login-error")).toBeVisible();
    await expect(page).toHaveURL(/\/login$/);
    // The authenticated shell must not have rendered.
    await expect(page.getByTestId("product-add")).toHaveCount(0);
  });

  test("logs in and lands on the products workspace", async ({ page }) => {
    await login(page);
    await expect(page).toHaveURL(/\/products$/);
    await expect(page.getByTestId("product-add")).toBeVisible();
  });

  test.describe("invalid", () => {
    test("blank password keeps the user on the form", async ({ page }) => {
      await page.goto("/login");
      await page.getByTestId("login-username").fill(ADMIN_USER);
      await page.getByTestId("login-submit").click();
      await expect(page.getByTestId("login-error")).toBeVisible();
    });
  });
});
