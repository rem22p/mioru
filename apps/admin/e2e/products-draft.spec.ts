import { test, expect } from "@playwright/test";
import { backendUp } from "./helpers";

// Close-guard + draft behaviour against the real stack. Runs in the shared
// authenticated project; skips when the backend is down. Draft state lives in
// localStorage, so each test clears the product-draft slots before and after
// to avoid leaking a restore prompt into sibling specs (e.g. products CRUD).
const clearDrafts = () =>
  Object.keys(localStorage)
    .filter((k) => k.startsWith("mioru-admin-product-draft:"))
    .forEach((k) => localStorage.removeItem(k));

test.describe("admin product form — draft & close guard", () => {
  test.beforeEach(async ({ request, page }) => {
    test.skip(!(await backendUp(request)), "backend on :8000 unreachable");
    await page.goto("/products");
    await page.evaluate(clearDrafts);
    await expect(page.getByTestId("product-add")).toBeVisible();
  });

  test.afterEach(async ({ page }) => {
    await page.evaluate(clearDrafts).catch(() => {});
  });

  test("editing then closing prompts to confirm; cancel keeps the form, confirm discards", async ({
    page,
  }) => {
    await page.getByTestId("product-add").click();
    await page.getByTestId("pf-name").fill("Draft Guard Probe");

    // X while dirty → confirm dialog; cancel keeps the form (and the input).
    await page.getByTestId("pf-close-x").click();
    await expect(page.getByTestId("pf-close-dialog")).toBeVisible();
    await page.getByTestId("pf-close-cancel").click();
    await expect(page.getByTestId("pf-name")).toHaveValue("Draft Guard Probe");

    // X again → confirm closes the form.
    await page.getByTestId("pf-close-x").click();
    await page.getByTestId("pf-close-confirm").click();
    await expect(page.getByTestId("pf-name")).toHaveCount(0);
  });

  test("closing an untouched form does not prompt", async ({ page }) => {
    await page.getByTestId("product-add").click();
    await expect(page.getByTestId("pf-name")).toBeVisible();

    await page.getByTestId("pf-close-x").click();
    await expect(page.getByTestId("pf-close-dialog")).toHaveCount(0);
    await expect(page.getByTestId("pf-name")).toHaveCount(0);
  });
});
