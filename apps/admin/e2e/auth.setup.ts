import { test as setup } from "@playwright/test";
import fs from "node:fs";
import path from "node:path";
import { backendUp, login, AUTH_FILE } from "./helpers";

// Logs in ONCE and persists the session (cookies, incl. the HttpOnly auth
// token) so the authenticated specs reuse it instead of each logging in.
// The backend rate-limits /api/auth/login to 10/min per IP — one login per
// run keeps the whole suite comfortably under that ceiling.
setup("authenticate", async ({ page, request }) => {
  fs.mkdirSync(path.dirname(AUTH_FILE), { recursive: true });

  // Backend run manually; if it's down, write an empty state so the dependent
  // project can still load (its specs self-skip via their own backendUp gate).
  if (!(await backendUp(request))) {
    fs.writeFileSync(AUTH_FILE, JSON.stringify({ cookies: [], origins: [] }));
    return;
  }

  await login(page);
  await page.context().storageState({ path: AUTH_FILE });
});
