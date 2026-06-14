import { defineConfig } from "@playwright/test";
import { fileURLToPath } from "node:url";

const AUTH_FILE = fileURLToPath(new URL("./e2e/.auth/admin.json", import.meta.url));

export default defineConfig({
  testDir: "./e2e",
  // Serial: the specs mutate shared backend state (products, users) and the
  // authenticated session is a single shared login. One worker avoids
  // cross-test interference and contention on the single dev server.
  fullyParallel: false,
  workers: 1,
  // One retry absorbs latency spikes from the cold dev server / backend on a
  // live-stack run; a deterministic logic failure still fails on every retry.
  retries: 1,
  // Generous default — assertions wait on a real React app talking to a real
  // API, not a mocked one.
  expect: { timeout: 15000 },
  use: {
    baseURL: "http://localhost:5174",
    trace: "on-first-retry",
  },
  // Pick up any e2e/*.spec.ts by default; the auth flows are excluded here
  // and routed to their own projects below. New authenticated specs
  // (orders, profile, settings, …) get picked up automatically — no
  // per-file edit needed. security.spec.ts is excluded here because it
  // mutates the shared admin password and runs in its own dedicated project
  // (see `security` below).
  testMatch: /.*\.spec\.ts$/,
  testIgnore: /auth\.(spec|setup)\.ts|security\.spec\.ts/,
  projects: [
    // Logs in once and saves the session.
    {
      name: "setup",
      testMatch: /auth\.setup\.ts/,
      // The top-level testIgnore would otherwise drop auth.setup.ts.
      testIgnore: [],
    },
    // Auth flows need a clean, logged-out context — they test login itself.
    { name: "logged-out", testMatch: /auth\.spec\.ts/, testIgnore: [] },
    // Everything else reuses the shared session (no per-spec login → no
    // auth rate-limit pressure). Inherits testMatch + testIgnore from the
    // top level, so any e2e/*.spec.ts outside auth is automatically picked
    // up.
    {
      name: "authenticated",
      dependencies: ["setup"],
      use: { storageState: AUTH_FILE },
    },
    // Security-critical specs: isolated because they MUTATE the shared
    // admin password, which corrupts the iat < password_changed_at check
    // for any subsequent login() in the same DB. Runs against the same
    // /api/_test/reset-admin endpoint (dev-mode only) to drop the admin
    // back to a known bcrypt hash before each test. Excluded from
    // `authenticated` via the top-level testIgnore; this project enables
    // it explicitly.
    {
      name: "security",
      testMatch: /security\.spec\.ts/,
      testIgnore: [],
      // Depend on the regular setup so the same shared session cookie
      // is in place when this project starts. The spec's beforeAll then
      // calls /api/_test/reset-admin to put the admin password in a
      // known bcrypt hash, so the shared session is still valid for
      // the subsequent login().
      dependencies: ["setup"],
      use: { storageState: AUTH_FILE },
    },
  ],
  webServer: {
    command: "npm run dev",
    url: "http://localhost:5174",
    reuseExistingServer: !process.env.CI,
  },
});
