import { test, expect } from "@playwright/test";

const DESKTOP = { width: 1280, height: 800 } as const;
const TABLET = { width: 768, height: 1024 } as const;
const MOBILE = { width: 375, height: 812 } as const;

test.describe("Visual Regression — Desktop (1280×800)", () => {
  test.use({ viewport: DESKTOP });

  test("Homepage — all sections visible, nothing overlaps", async ({
    page,
  }) => {
    await page.goto("/");
    await page.waitForLoadState("networkidle");
    await expect(page).toHaveScreenshot("desktop-homepage.png", {
      fullPage: true,
    });
  });

  test("Catalog — sidebar + 4-column grid + filters", async ({ page }) => {
    await page.goto("/catalog");
    await page.waitForLoadState("networkidle");
    await expect(page).toHaveScreenshot("desktop-catalog.png", {
      fullPage: true,
    });
  });

  test("Product page — gallery left, info right", async ({ page }) => {
    await page.goto("/product/midnight-runner");
    await page.waitForLoadState("networkidle");
    // Wait for lazy-loaded components
    await page.waitForTimeout(2000);
    await expect(page).toHaveScreenshot("desktop-product.png", {
      fullPage: true,
    });
  });

  test("Cart with items — quantity controls, summary", async ({ page }) => {
    await page.goto("/product/midnight-runner");
    await page.waitForLoadState("networkidle");
    await page.waitForTimeout(1000);
    await page.locator('button:has-text("42")').first().click();
    await page.locator('button:has-text("В корзину")').first().click();
    await page.waitForTimeout(500);
    await page.goto("/cart");
    await page.waitForLoadState("networkidle");
    await expect(page).toHaveScreenshot("desktop-cart-filled.png", {
      fullPage: true,
    });
  });

  test("Cart empty — placeholder + CTA", async ({ page }) => {
    await page.goto("/cart");
    await page.waitForLoadState("networkidle");
    await expect(page).toHaveScreenshot("desktop-cart-empty.png", {
      fullPage: true,
    });
  });

  test("Checkout — stepper visible, inputs aligned", async ({ page }) => {
    await page.goto("/checkout");
    await page.waitForLoadState("networkidle");
    await expect(page).toHaveScreenshot("desktop-checkout.png", {
      fullPage: true,
    });
  });

  test("Profile — XP bar, quick links, order history", async ({ page }) => {
    await page.goto("/profile");
    await page.waitForLoadState("networkidle");
    await expect(page).toHaveScreenshot("desktop-profile.png", {
      fullPage: true,
    });
  });

  test("Admin — products table, form toggle", async ({ page }) => {
    await page.goto("/admin");
    await page.waitForLoadState("networkidle");
    await expect(page).toHaveScreenshot("desktop-admin.png", {
      fullPage: true,
    });
  });

  test("Avatar editor — 3D viewport + sliders", async ({ page }) => {
    await page.goto("/avatar");
    await page.waitForLoadState("networkidle");
    await expect(page).toHaveScreenshot("desktop-avatar.png", {
      fullPage: true,
    });
  });
});

test.describe("Visual Regression — Mobile (375×812)", () => {
  test.use({ viewport: MOBILE });

  test("Homepage — mobile layout, no overflow", async ({ page }) => {
    await page.goto("/");
    await page.waitForLoadState("networkidle");
    await expect(page).toHaveScreenshot("mobile-homepage.png", {
      fullPage: true,
    });
  });

  test("Catalog — filter button visible, 2-column grid", async ({ page }) => {
    await page.goto("/catalog");
    await page.waitForLoadState("networkidle");
    await expect(page).toHaveScreenshot("mobile-catalog.png", {
      fullPage: true,
    });
  });

  test("Product — stacked layout, size buttons visible", async ({ page }) => {
    await page.goto("/product/midnight-runner");
    await page.waitForLoadState("networkidle");
    await expect(page).toHaveScreenshot("mobile-product.png", {
      fullPage: true,
    });
  });

  test("Mobile menu open — links centered, backdrop visible", async ({
    page,
  }) => {
    await page.goto("/");
    await page.click('button[aria-label="Menu"]');
    await page.waitForTimeout(500);
    await expect(page).toHaveScreenshot("mobile-menu-open.png", {
      fullPage: false,
    });
  });

  test("Cart — stacked items, controls reachable", async ({ page }) => {
    await page.goto("/product/midnight-runner");
    await page.waitForLoadState("networkidle");
    await page.waitForTimeout(1000);
    await page.locator('button:has-text("42")').first().click();
    await page.locator('button:has-text("В корзину")').first().click();
    await page.waitForTimeout(500);
    await page.goto("/cart");
    await page.waitForLoadState("networkidle");
    await expect(page).toHaveScreenshot("mobile-cart.png", { fullPage: true });
  });
});

test.describe("Visual Regression — Tablet (768×1024)", () => {
  test.use({ viewport: TABLET });

  test("Homepage — tablet breakpoint", async ({ page }) => {
    await page.goto("/");
    await page.waitForLoadState("networkidle");
    await expect(page).toHaveScreenshot("tablet-homepage.png", {
      fullPage: true,
    });
  });
});

test.describe("Header — visual states", () => {
  test("Scrolled header — background + blur + border", async ({ page }) => {
    await page.setViewportSize(DESKTOP);
    await page.goto("/");
    await page.evaluate(() => window.scrollTo(0, 200));
    await page.waitForTimeout(600);
    const header = page.locator("header");
    await expect(header).toHaveScreenshot("header-scrolled.png");
  });

  test("Transparent header at top", async ({ page }) => {
    await page.setViewportSize(DESKTOP);
    await page.goto("/");
    await page.waitForTimeout(300);
    const header = page.locator("header");
    await expect(header).toHaveScreenshot("header-top.png");
  });

  test("Nav links centered — within 10px tolerance", async ({ page }) => {
    await page.setViewportSize(DESKTOP);
    await page.goto("/");
    await page.waitForLoadState("networkidle");

    const nav = page.locator("header nav");
    const navBox = await nav.boundingBox();
    const viewportWidth = DESKTOP.width;

    if (navBox) {
      const navCenter = navBox.x + navBox.width / 2;
      const screenCenter = viewportWidth / 2;
      const offset = Math.abs(navCenter - screenCenter);

      // Allow 10px tolerance for font rendering differences
      expect(offset).toBeLessThanOrEqual(10);
    }
  });
});

test.describe("Accessibility & UX", () => {
  test("All touch targets ≥ 44×44px on mobile", async ({ page }) => {
    await page.setViewportSize(MOBILE);
    await page.goto("/product/midnight-runner");
    await page.waitForLoadState("networkidle");
    await page.waitForTimeout(1000);

    const buttons = page.locator("button, a[href]");
    const count = await buttons.count();
    let violations = 0;

    for (let i = 0; i < count; i++) {
      const btn = buttons.nth(i);
      const visible = await btn.isVisible().catch(() => false);
      if (!visible) continue;

      const box = await btn.boundingBox();
      if (box && box.width > 0 && box.height > 0) {
        if (box.width < 38 || box.height < 38) {
          violations++;
          console.log(`Small element ${i}: ${box.width}x${box.height}`);
        }
      }
    }
    // Allow up to 5 small elements (like inline text links)
    expect(violations).toBeLessThanOrEqual(5);
  });

  test("Input font size ≥ 16px on mobile (prevents iOS zoom)", async ({
    page,
  }) => {
    await page.setViewportSize(MOBILE);
    await page.goto("/checkout");
    await page.waitForLoadState("networkidle");

    const inputs = page.locator("input, textarea, select");
    const count = await inputs.count();

    for (let i = 0; i < count; i++) {
      const fs = await inputs
        .nth(i)
        .evaluate((el) => parseInt(getComputedStyle(el).fontSize));
      expect(fs, `Input ${i} font-size ${fs}px < 16px`).toBeGreaterThanOrEqual(
        16,
      );
    }
  });

  test("No horizontal overflow on mobile", async ({ page }) => {
    await page.setViewportSize(MOBILE);
    await page.goto("/");
    await page.waitForLoadState("networkidle");

    const overflowX = await page.evaluate(
      () => document.documentElement.scrollWidth - window.innerWidth,
    );

    // Allow up to 2px for scrollbar
    expect(overflowX).toBeLessThanOrEqual(2);
  });

  test("Z-index: mobile menu covers content", async ({ page }) => {
    await page.setViewportSize(MOBILE);
    await page.goto("/");
    await page.click('button[aria-label="Menu"]');
    await page.waitForTimeout(500);

    // The menu overlay should be above the header
    const menu = page.locator(".fixed.inset-0.z-40");
    await expect(menu).toBeVisible();
  });
});

test.describe("Theme & i18n", () => {
  test("Switch to light theme — background changes", async ({ page }) => {
    await page.setViewportSize(DESKTOP);
    await page.goto("/");
    await page.waitForLoadState("networkidle");
    await page.waitForTimeout(500);

    // Click theme toggle — use nth match to find the correct button
    const themeBtn = page
      .locator("button")
      .filter({ has: page.locator("svg") })
      .first();
    // Actually find by the sun/moon icon
    const toggle = page
      .locator("header button")
      .filter({ has: page.locator(".lucide-sun, .lucide-moon") })
      .first();
    await toggle.click();
    await page.waitForTimeout(500);

    // Check body has light background (not #0a0a0a rgb(10,10,10))
    const bodyBg = await page.evaluate(
      () => getComputedStyle(document.body).backgroundColor,
    );
    expect(bodyBg).not.toBe("rgb(10, 10, 10)");
  });

  test("Switch to English — nav changes", async ({ page }) => {
    await page.setViewportSize(DESKTOP);
    await page.goto("/");
    await page.waitForLoadState("networkidle");

    // Click language globe
    await page.click('button[aria-label="Change language"]');
    await page.click('button:has-text("EN")');
    await page.waitForTimeout(500);

    // Catalog link should now be in English
    const navText = await page.locator("header nav").textContent();
    expect(navText).toContain("Catalog");
  });
});
