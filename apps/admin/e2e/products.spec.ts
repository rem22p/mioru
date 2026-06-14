import { test, expect } from "@playwright/test";
import { backendUp, SEEDED_CATEGORY_ID } from "./helpers";

// Full product CRUD against the real backend + real middleware chain.
// Each run uses a unique slug so reruns never collide on the same DB.
// The session is established once by auth.setup.ts (storageState).
test.describe("admin product CRUD", () => {
  test.beforeEach(async ({ request, page }) => {
    test.skip(!(await backendUp(request)), "backend on :8000 unreachable");
    await page.goto("/products");
    await expect(page.getByTestId("product-add")).toBeVisible();
  });

  test("create → edit → delete a product", async ({ page }) => {
    const stamp = Date.now();
    const slug = `e2e-tshirt-${stamp}`;
    const name = `E2E Tshirt ${stamp}`;

    // ── Create ──
    await page.getByTestId("product-add").click();
    await page.getByTestId("pf-name").fill(name);
    await page.getByTestId("pf-slug").fill(slug);
    await page.getByTestId("pf-category").selectOption(SEEDED_CATEGORY_ID);
    await page.getByTestId("pf-price").fill("499");

    const created = page.waitForResponse(
      (r) =>
        r.url().includes("/api/admin/products") &&
        r.request().method() === "POST",
    );
    await page.getByTestId("pf-submit").click();
    expect((await created).status()).toBe(201);

    // Find the new row via the search filter (robust against pagination).
    await page.getByTestId("product-search").fill(name);
    const row = page.locator(`[data-testid="product-row"][data-slug="${slug}"]`);
    await expect(row).toBeVisible();

    // ── Edit ── clicking the card opens the edit form.
    await row.click();
    await expect(page.getByTestId("pf-name")).toHaveValue(name);
    await page.getByTestId("pf-price").fill("799");
    const updated = page.waitForResponse(
      (r) =>
        r.url().includes(`/api/admin/products/${slug}`) &&
        r.request().method() === "PUT",
    );
    await page.getByTestId("pf-submit").click();
    expect((await updated).status()).toBe(200);

    // ── Delete ── hover the card to reveal the overlay action.
    await page.getByTestId("product-search").fill(name);
    await expect(row).toBeVisible();
    await row.hover();
    await row.getByTestId("product-delete").click();
    const deleted = page.waitForResponse(
      (r) =>
        r.url().includes(`/api/admin/products/${slug}`) &&
        r.request().method() === "DELETE",
    );
    await page.getByTestId("product-delete-confirm").click();
    expect((await deleted).status()).toBe(200);

    await expect(row).toHaveCount(0);
  });

  test("create form blocks an empty required field", async ({ page }) => {
    await page.getByTestId("product-add").click();
    // Submit with no name/category/price → client-side validation error,
    // no network write.
    await page.getByTestId("pf-submit").click();
    await expect(page.getByTestId("pf-error")).toBeVisible();
  });
});
