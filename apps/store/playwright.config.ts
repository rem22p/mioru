import { defineConfig } from '@playwright/test';
import { fileURLToPath } from 'node:url';

// Hides the animated/WebGL <canvas> surfaces so screenshot baselines are
// deterministic across runs and environments. See e2e/screenshot.css.
const SCREENSHOT_CSS = fileURLToPath(
  new URL('./e2e/screenshot.css', import.meta.url),
);

export default defineConfig({
  testDir: './e2e',
  fullyParallel: true,
  retries: process.env.CI ? 1 : 0,
  // #78 F9: keep the HTML report so a red visual job leaves actual/expected/
  // diff artifacts instead of forcing log-only debugging.
  reporter: [['list'], ['html', { open: 'never' }]],
  use: {
    baseURL: 'http://localhost:5173',
    trace: 'on-first-retry',
  },
  expect: {
    // Stabilise visual regression: hide animated canvases and tolerate a tiny
    // fraction of anti-aliasing jitter. Baselines are generated in the same
    // Playwright container image CI runs (mcr…:v1.60.0-jammy), so real diffs
    // are ~0; the ratio only absorbs sub-pixel font-rasterisation noise.
    toHaveScreenshot: {
      stylePath: SCREENSHOT_CSS,
      maxDiffPixelRatio: 0.01,
    },
  },
  webServer: {
    command: 'npm run dev',
    url: 'http://localhost:5173',
    reuseExistingServer: !process.env.CI,
  },
});
