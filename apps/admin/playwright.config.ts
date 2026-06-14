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
  projects: [
    // Logs in once and saves the session.
    { name: "setup", testMatch: /auth\.setup\.ts/ },
    // Auth flows need a clean, logged-out context — they test login itself.
    { name: "logged-out", testMatch: /auth\.spec\.ts/ },
    // Everything else reuses the shared session (no per-spec login → no
    // auth rate-limit pressure).
    {
      name: "authenticated",
      testMatch: /(products|products-upload|users)\.spec\.ts/,
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
