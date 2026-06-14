import { expect, type APIRequestContext, type Page } from "@playwright/test";
import { fileURLToPath } from "node:url";

// Where auth.setup.ts persists the logged-in session for reuse by the
// authenticated project. Gitignored (see .gitignore).
export const AUTH_FILE = fileURLToPath(
  new URL("./.auth/admin.json", import.meta.url),
);

// The admin SPA talks to the backend through the Vite dev proxy (same-origin
// /api → :8000), so the browser needs no API URL. Playwright's own request
// context is NOT proxied, so the health probe hits the backend directly.
export const API = process.env.E2E_API_URL || "http://localhost:8000";

// Bootstrap super-admin credentials. The backend seeds this account from
// BOOTSTRAP_ADMIN_* on startup; the E2E runner must use the same values.
// Override via env when the local backend was seeded differently.
export const ADMIN_USER = process.env.E2E_ADMIN_USERNAME || "admin";
export const ADMIN_PASS = process.env.E2E_ADMIN_PASSWORD || "Admin12345!";

// A stable seeded leaf category ("Футболки / поло", parent "Одежда").
// See internal/store/migrations/002_seed_categories.sql.
export const SEEDED_CATEGORY_ID = "2";

// backendUp probes a public endpoint. Returns false when the backend is not
// reachable so specs can self-skip instead of failing — the backend is run
// manually in this project, not by the Playwright webServer.
export async function backendUp(request: APIRequestContext): Promise<boolean> {
  // Probe a couple of times: with several workers starting at once the first
  // request can lose the race against a cold connection and time out, which
  // would wrongly skip a write-path spec.
  for (let attempt = 0; attempt < 3; attempt++) {
    try {
      const res = await request.get(`${API}/api/categories`, {
        timeout: 10000,
      });
      if (res.ok()) return true;
    } catch {
      // retry
    }
  }
  return false;
}

// login drives the real login form and waits until the authenticated shell
// (the products workspace) is rendered. Uses the bootstrap super-admin, which
// covers both product CRUD and user management.
export async function login(
  page: Page,
  username = ADMIN_USER,
  password = ADMIN_PASS,
): Promise<void> {
  await page.goto("/login");
  await page.getByTestId("login-username").fill(username);
  await page.getByTestId("login-password").fill(password);
  await page.getByTestId("login-submit").click();
  // nav("/") → redirect to /products; the add button only renders inside the
  // authenticated layout, so its visibility proves the session is live.
  await expect(page.getByTestId("product-add")).toBeVisible();
}
