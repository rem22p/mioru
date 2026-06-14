import { test, expect } from "@playwright/test";
import { backendUp } from "./helpers";

// A 1×1 PNG built in-memory. The repo ignores *.png (upload-hygiene rule),
// so the fixture is generated here instead of committed as a binary.
const PIXEL_PNG = Buffer.from(
  "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mNk+M9QDwADhgGAWjR9awAAAABJRU5ErkJggg==",
  "base64",
);

// Exercises the real image-upload path: POST /api/admin/upload through the
// admin auth + CSRF middleware chain, then asserts the preview renders.
test.describe("admin product image upload", () => {
  test.beforeEach(async ({ request, page }) => {
    test.skip(!(await backendUp(request)), "backend on :8000 unreachable");
    await page.goto("/products");
    await expect(page.getByTestId("product-add")).toBeVisible();
  });

  test("uploads a PNG and shows it in the form preview", async ({ page }) => {
    await page.getByTestId("product-add").click();

    const uploaded = page.waitForResponse(
      (r) =>
        r.url().includes("/api/admin/upload") &&
        r.request().method() === "POST",
    );
    await page.getByTestId("pf-image-input").setInputFiles({
      name: "pixel.png",
      mimeType: "image/png",
      buffer: PIXEL_PNG,
    });
    expect((await uploaded).status()).toBe(200);

    // The uploaded image renders as the only <img> inside the form grid.
    await expect(page.locator("form img")).toHaveCount(1);
  });
});
