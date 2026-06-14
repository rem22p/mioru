import { test, expect, type APIRequestContext, type Page } from "@playwright/test";
import { API, backendUp } from "./helpers";

// Per #42 convention: the backend is run manually in this project, so we
// self-skip the whole suite when it is unreachable rather than hard-fail.
let backendReachable = false;
test.beforeAll(async ({ request }) => {
  backendReachable = await backendUp(request);
});
test.beforeEach(() => {
  test.skip(!backendReachable, `backend not reachable at ${API} (start it on :8000)`);
});

test("orders workspace renders its structural controls", async ({ page }) => {
  await page.goto("/orders");

  await expect(page.getByTestId("orders-page")).toBeVisible();
  await expect(page.getByTestId("orders-refresh")).toBeVisible();
  await expect(page.getByTestId("orders-filter-status")).toBeVisible();
  await expect(page.getByTestId("orders-filter-type")).toBeVisible();
  await expect(page.getByTestId("orders-total")).toBeVisible();

  // Exactly one of {list, empty} must be present after the load resolves.
  const list = page.getByTestId("orders-list");
  const empty = page.getByTestId("orders-empty");
  await expect(list.or(empty)).toBeVisible();
});

test("filtering by status issues a request and re-renders the total", async ({ page }) => {
  await page.goto("/orders");
  // Let the initial GET resolve before we apply a filter.
  await expect(page.getByTestId("orders-list").or(page.getByTestId("orders-empty"))).toBeVisible();

  const totalLocator = page.getByTestId("orders-total");
  const totalBefore = (await totalLocator.textContent()) ?? "";

  const resp = page.waitForResponse((r) => {
    const u = new URL(r.url());
    return (
      u.pathname === "/api/admin/orders" &&
      r.request().method() === "GET" &&
      u.searchParams.get("status") === "pending"
    );
  });
  await page.getByTestId("orders-filter-status").selectOption("pending");
  await resp;

  // The total text should change OR the empty state should appear (if the
  // pending-filtered list happens to be empty). Both are valid outcomes.
  const totalAfter = (await totalLocator.textContent()) ?? "";
  if (totalAfter === totalBefore) {
    await expect(page.getByTestId("orders-empty")).toBeVisible();
  } else {
    expect(totalAfter).not.toBe(totalBefore);
  }
});

test("invalid status change surfaces a backend error in the UI banner", async ({ page, request }) => {
  // Stub the PATCH to return 400 — avoids depending on which orders are
  // currently in the DB, and pins the UI's "VALIDATION_FAILED" handling
  // (per the same envelope contract enforced by integration tests in #35).
  await page.route("**/api/admin/orders/*/status", (route) =>
    route.fulfill({
      status: 400,
      contentType: "application/json",
      body: JSON.stringify({ code: "VALIDATION_FAILED", error: "invalid status value" }),
    }),
  );

  // We need at least one order row to click the select. If the empty state
  // is what renders, skip cleanly — the test then serves as a "the route
  // is wired but unused" probe.
  await page.goto("/orders");
  const list = page.getByTestId("orders-list");
  const firstRow = list.locator("[data-testid^='orders-row-']").first();
  await expect(list.or(page.getByTestId("orders-empty"))).toBeVisible();
  test.skip((await firstRow.count()) === 0, "no orders in DB to exercise the status PATCH");

  const statusSelect = firstRow.locator("[data-testid$='-status-select']");
  await statusSelect.selectOption("processing");

  await expect(page.getByTestId("orders-error")).toBeVisible();
  await expect(page.getByTestId("orders-error")).toContainText("invalid status value");
});

test("admin can update order status when the backend accepts it", async ({ page }) => {
  await page.goto("/orders");
  const list = page.getByTestId("orders-list");
  const firstRow = list.locator("[data-testid^='orders-row-']").first();
  await expect(list.or(page.getByTestId("orders-empty"))).toBeVisible();
  test.skip((await firstRow.count()) === 0, "no orders in DB to exercise the status PATCH");

  // Find the order ID from the testid (orders-row-{id}).
  const testid = await firstRow.getAttribute("data-testid");
  const orderId = testid?.replace("orders-row-", "");
  expect(orderId).toBeTruthy();

  const statusSelect = page.getByTestId(`orders-row-${orderId}-status-select`);
  await expect(statusSelect).toBeVisible();

  // The order's initial status is whatever the backend reports; pick a
  // different one (any value in the OPTIONS list).
  const current = await statusSelect.inputValue();
  const target = current === "processing" ? "shipped" : "processing";

  const patchResp = page.waitForResponse(
    (r) =>
      new RegExp(`/api/admin/orders/${orderId}/status$`).test(r.url()) &&
      r.request().method() === "PATCH",
  );
  await statusSelect.selectOption(target);
  await patchResp;

  // The select's value should reflect the new status.
  await expect(statusSelect).toHaveValue(target);
});
