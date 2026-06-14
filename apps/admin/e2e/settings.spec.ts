import { test, expect } from "@playwright/test";
import { API, backendUp, login } from "./helpers";

let backendReachable = false;
test.beforeAll(async ({ request }) => {
  backendReachable = await backendUp(request);
});
test.beforeEach(() => {
  test.skip(!backendReachable, `backend not reachable at ${API} (start it on :8000)`);
});

test("settings page renders theme and scale controls", async ({ page }) => {
  await login(page);
  await page.goto("/settings");

  await expect(page.getByTestId("settings-page")).toBeVisible();
  await expect(page.getByTestId("settings-theme-dark")).toBeVisible();
  await expect(page.getByTestId("settings-theme-light")).toBeVisible();
  await expect(page.getByTestId("settings-scale-slider")).toBeVisible();
  await expect(page.getByTestId("settings-scale-value")).toBeVisible();
});

test("theme toggle flips <html class=\"light\"> and persists in localStorage", async ({ page }) => {
  await login(page);
  await page.goto("/settings");

  // Start dark (bootstrap from the store).
  await page.getByTestId("settings-theme-light").click();
  await expect(page.locator("html")).toHaveClass(/light/);
  expect(await page.evaluate(() => localStorage.getItem("ui_theme"))).toBe("light");

  await page.getByTestId("settings-theme-dark").click();
  await expect(page.locator("html")).not.toHaveClass(/light/);
  expect(await page.evaluate(() => localStorage.getItem("ui_theme"))).toBe("dark");

  // Reload — must still be dark.
  await page.reload();
  await expect(page.locator("html")).not.toHaveClass(/light/);
});

test("UI scale slider writes --ui CSS variable and persists in localStorage", async ({ page }) => {
  await login(page);
  await page.goto("/settings");

  const slider = page.getByTestId("settings-scale-slider");
  await expect(slider).toBeVisible();

  // Move off the default (13) to a new value.
  await slider.fill("16");
  await slider.dispatchEvent("change");

  // The displayed value updates.
  await expect(page.getByTestId("settings-scale-value")).toContainText("16px");

  // The CSS variable was written.
  const ui = await page.evaluate(() =>
    getComputedStyle(document.documentElement).getPropertyValue("--ui").trim(),
  );
  expect(ui).toBe("16px");

  // And persisted.
  const persisted = await page.evaluate(() => localStorage.getItem("ui_scale"));
  expect(persisted).toBe("16");

  // Reload — must restore from localStorage.
  await page.reload();
  await expect(page.getByTestId("settings-scale-value")).toContainText("16px");
});
