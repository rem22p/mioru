import { test, expect } from "@playwright/test";
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

test("filtering by status issues a pending request and re-renders the list", async ({ page }) => {
  await page.goto("/orders");
  // Let the initial GET resolve before we apply a filter.
  await expect(page.getByTestId("orders-list").or(page.getByTestId("orders-empty"))).toBeVisible();

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

  // After the filter resolves the UI settles into one of two states. We
  // assert with a retrying matcher (toBeVisible polls), so the assertion
  // succeeds once React has re-rendered — no race against the network
  // callback. We deliberately do NOT compare the total text across the
  // filter change: if the DB has zero pending orders, total can stay
  // unchanged while the empty state is shown — both outcomes are valid.
  await expect(
    page.getByTestId("orders-list").or(page.getByTestId("orders-empty")),
  ).toBeVisible();
});

test("invalid status change surfaces a backend error in the UI banner", async ({ page }) => {
  // Stub both the list (so we always have exactly one row) and the PATCH
  // (so it always returns 400 VALIDATION_FAILED). The stub is independent
  // of what's in the DB, so the assertion runs on every run — no
  // test.skip hiding a regression.
  await page.route("**/api/admin/orders*", (route) => {
    const url = new URL(route.request().url());
    // Only stub the listing endpoint; let other admin/orders/* requests
    // through (none expected in this test).
    if (
      url.pathname === "/api/admin/orders" &&
      route.request().method() === "GET"
    ) {
      return route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify({
          orders: [
            {
              id: 4242,
              type: "cart",
              status: "pending",
              customer_id: 1,
              customer_email: "stub@example.com",
              customer_first_name: "Stub",
              total_minor: 1000,
              currency: "MDL",
              city: "Chișinău",
              delivery_method: "address",
              payment_method: "cash",
              comment: "",
              created_at: new Date().toISOString(),
              photos: [],
              items: [],
            },
          ],
          total: 1,
        }),
      });
    }
    return route.continue();
  });
  await page.route("**/api/admin/orders/*/status", (route) =>
    route.fulfill({
      status: 400,
      contentType: "application/json",
      body: JSON.stringify({ code: "VALIDATION_FAILED", error: "invalid status value" }),
    }),
  );

  await page.goto("/orders");
  const row = page.getByTestId("orders-row-4242");
  await expect(row).toBeVisible();
  await expect(row.getByTestId("orders-row-4242-status-select")).toBeVisible();

  await row.getByTestId("orders-row-4242-status-select").selectOption("processing");

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
