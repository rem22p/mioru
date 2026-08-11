import { createHmac, createHash } from "node:crypto";
import { test, expect, type APIRequestContext } from "@playwright/test";

/**
 * Functional storefront E2E: register → catalog → product → cart → checkout → order.
 *
 * Unlike visual.spec.ts (layout/a11y snapshots only) this exercises the real
 * priority-#1 journey end-to-end against a live backend, asserting behaviour and
 * the real `POST /api/store/orders`. It targets stable `data-testid` hooks, never
 * i18n copy or Tailwind classes (both churn — see CLAUDE.md "Stable selectors").
 *
 * Backend is assumed to run manually on :8000 (override via E2E_API_URL). When it
 * is unreachable the whole suite skips rather than failing — there is no docker
 * harness wired into playwright.config yet.
 */

const API = process.env.E2E_API_URL || "http://localhost:8000";

// Bot token for signing the Telegram widget payload in the E2E.  The
// backend under test verifies the signature with its own TELEGRAM_BOT_TOKEN
// (same test stand), so the two must match.  When unset the ordering part
// of the flow is skipped (the widget cannot be driven from Playwright —
// it is an external iframe on oauth.telegram.org).
const TG_BOT_TOKEN = process.env.E2E_TELEGRAM_BOT_TOKEN || "";

// Matches the App-startup `GET /api/store/customers/me` (not the /orders or
// /cart siblings) so we can wait for auth to be restored after a full reload.
const ME_RE = /\/api\/store\/customers\/me(\?|$)/;

let backendUp = false;

async function findInStockSlug(request: APIRequestContext): Promise<string | null> {
  const res = await request.get(`${API}/api/products?per_page=100`);
  if (!res.ok()) return null;
  const body = (await res.json()) as {
    products?: Array<{
      slug: string;
      in_stock: boolean;
      stock_quantity: number;
      sizes: string[] | null;
    }>;
  };
  const p = (body.products ?? []).find(
    (x) => x.in_stock && x.stock_quantity > 0 && (x.sizes?.length ?? 0) > 0,
  );
  return p?.slug ?? null;
}

// signTelegram mirrors the Login Widget's data-check-string exactly:
// sorted keys, "key=value" joined by '\n', HMAC-SHA256 with
// secret = SHA256(botToken), hex-encoded.  Empty optional fields are
// excluded per the Telegram spec.  Replicated here so the E2E can bind a
// Telegram identity through the real /me/oauth endpoint and pass
// auth.VerifyTelegramAuth for real.
function signTelegram(botToken: string, d: { id: number; first_name: string; username?: string; auth_date: number }): string {
  const pairs: Record<string, string> = {
    auth_date: String(d.auth_date),
    first_name: d.first_name,
    id: String(d.id),
  };
  if (d.username) pairs.username = d.username;
  const keys = Object.keys(pairs).sort();
  const dataCheck = keys.map((k) => `${k}=${pairs[k]}`).join("\n");
  const secret = createHash("sha256").update(botToken).digest();
  return createHmac("sha256", secret).update(dataCheck).digest("hex");
}

test.beforeAll(async ({ request }) => {
  try {
    const r = await request.get(`${API}/api/health`);
    backendUp = r.ok();
  } catch {
    backendUp = false;
  }
});

test.beforeEach(() => {
  test.skip(!backendUp, `backend not reachable at ${API} (start it on :8000)`);
});

test("customer registers, adds a product to cart, and places an order", async ({
  page,
  request,
}) => {
  const slug = await findInStockSlug(request);
  test.skip(!slug, "no in-stock product with sizes available to order");

  // ── Register a fresh customer ──────────────────────────────────────────
  await page.goto("/profile");
  const email = `e2e+${Date.now()}@example.com`;

  await page.getByTestId("auth-tab-register").click();
  await page.getByTestId("auth-reg-first-name").fill("E2E");
  await page.getByTestId("auth-reg-email").fill(email);
  await page.getByTestId("auth-reg-phone").fill("+37377123456");
  await page.getByTestId("auth-reg-password").fill("password123");

  const registerResp = page.waitForResponse(
    (r) => r.url().includes("/api/store/auth/register") && r.request().method() === "POST",
  );
  await page.getByTestId("auth-reg-submit").click();
  expect((await registerResp).status()).toBe(201);
  await expect(page.getByTestId("auth-error")).toHaveCount(0);

  // ── Browse the catalog ─────────────────────────────────────────────────
  await page.goto("/catalog");
  await expect(page.getByTestId("catalog-product-card").first()).toBeVisible();

  // ── Open a known in-stock product and wait for auth to be restored ──────
  const meRestored = page.waitForResponse((r) => ME_RE.test(r.url()) && r.ok());
  await page.goto(`/product/${slug}`);
  await meRestored; // auth now true in-memory → all later nav stays client-side

  // ── Pick a size and add to cart ────────────────────────────────────────
  await page.getByTestId("product-size").first().click();
  await page.getByTestId("product-add-to-cart").click();

  // Cart now has the item → the page swaps the CTA to a "go to cart" link.
  await page.getByTestId("product-go-to-cart").click();

  // ── Cart → checkout (client-side, auth still in memory) ─────────────────
  await expect(page.getByTestId("cart-row")).toHaveCount(1);
  await page.getByTestId("cart-checkout").click();

  // ── Bind a Telegram identity so the order gate passes ───────────────────
  // The widget is an external iframe (oauth.telegram.org) — not drivable
  // from Playwright.  Instead we sign a payload with the same bot token
  // the backend verifies against and hit the real /me/oauth endpoint.
  // Without a token the ordering part of this journey is skipped.
  test.skip(!TG_BOT_TOKEN, "E2E_TELEGRAM_BOT_TOKEN unset — cannot bind Telegram for the order gate");
  const tgAuthDate = Math.floor(Date.now() / 1000);
  const tgID = 420000000 + Math.floor(Math.random() * 1000000);
  const tgUsername = `e2e_tg_${Date.now() % 100000}`;
  const tgHash = signTelegram(TG_BOT_TOKEN, {
    id: tgID,
    first_name: "E2E",
    username: tgUsername,
    auth_date: tgAuthDate,
  });
  // page.request shares the browser context, so store_auth + store_csrf
  // cookies from the registration are attached.  The CSRF header must
  // echo the readable store_csrf cookie value (double-submit pattern).
  const cookies = await page.context().cookies();
  const csrf = cookies.find((c) => c.name === "store_csrf")?.value ?? "";
  const linkResp = await page.request.post(`${API}/api/store/customers/me/oauth`, {
    headers: { "Content-Type": "application/json", "X-CSRF-Token": csrf },
    data: {
      provider: "telegram",
      id: tgID,
      first_name: "E2E",
      username: tgUsername,
      auth_date: tgAuthDate,
      hash: tgHash,
      profile_data: JSON.stringify({ username: tgUsername, first_name: "E2E" }),
    },
  });
  expect(linkResp.status()).toBe(200);
  // Re-enter from the cart, not `reload()`: reloading /checkout bounces to
  // /profile because `isAuthenticated` starts false and the page guard fires
  // before GET /me resolves. The link's href is the rendered proof auth is back.
  await page.goto("/cart");
  await expect(page.getByTestId("cart-checkout")).toHaveAttribute("href", "/checkout");
  await page.getByTestId("cart-checkout").click();

  // ── Checkout step 1: city + delivery + phone ─────────────────────
  // Tiraspol allows the free "personal" pickup method.
  // E2E reviewer finding #1: `canProceed()` for step 1 now
  // requires `phone` (was added in this PR — every checkout
  // needs the manager to be able to call the buyer). The
  // E2E must fill it before clicking "next" or the click
  // is a no-op and the test deadlocks on the next step.
  await page.getByTestId("checkout-phone").fill("+37377711234");
  await page.getByTestId("checkout-city").fill("Тирасполь");
  await page.keyboard.press("Escape"); // close the autocomplete dropdown overlay
  await page.getByTestId("checkout-delivery-personal").check();
  await page.getByTestId("checkout-next").click();
  // ── Checkout step 2: payment (card is the default) ─────────────────────
  await page.getByTestId("checkout-next").click();

  // ── Checkout step 3: confirm → real POST /api/store/orders ─────────────
  // The gate banner lives next to the confirm button: absent here means the
  // binding really propagated to the SPA, not just to the database.
  await expect(page.getByTestId("checkout-telegram-required")).toHaveCount(0);
  const orderResp = page.waitForResponse(
    (r) => r.url().includes("/api/store/orders") && r.request().method() === "POST",
  );
  await page.getByTestId("checkout-confirm").click();
  expect((await orderResp).status()).toBe(201);
  await expect(page.getByTestId("checkout-success")).toBeVisible();

  // ── The order shows up in the customer's order history ──────────────────
  const ordersResp = page.waitForResponse((r) =>
    /\/api\/store\/customers\/me\/orders/.test(r.url()),
  );
  await page.goto("/profile");
  await ordersResp;
  await expect(
    page.getByTestId("orders-list").getByTestId("order-row").first(),
  ).toBeVisible();
});
