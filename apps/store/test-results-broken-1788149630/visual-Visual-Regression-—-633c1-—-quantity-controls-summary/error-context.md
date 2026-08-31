# Instructions

- Following Playwright test failed.
- Explain why, be concise, respect Playwright best practices.
- Provide a snippet of code with the fix, if possible.

# Test info

- Name: visual.spec.ts >> Visual Regression — Desktop (1280×800) >> Cart with items — quantity controls, summary
- Location: e2e/visual.spec.ts:39:3

# Error details

```
Test timeout of 30000ms exceeded.
```

```
Error: locator.click: Test timeout of 30000ms exceeded.
Call log:
  - waiting for locator('[data-testid="product-size"][data-size="42"]')

```

# Page snapshot

```yaml
- generic [ref=e3]:
  - banner [ref=e4]:
    - generic [ref=e5]:
      - link "MIORU" [ref=e6] [cursor=pointer]:
        - /url: /
        - img "MIORU" [ref=e7]
      - navigation [ref=e8]:
        - link "CATALOG" [ref=e9] [cursor=pointer]:
          - /url: /catalog
          - text: CATALOG
        - link "CUSTOM ORDER" [ref=e10] [cursor=pointer]:
          - /url: /custom-order
          - text: CUSTOM ORDER
        - link "FAVORITES" [ref=e11] [cursor=pointer]:
          - /url: /favorites
          - text: FAVORITES
      - generic [ref=e12]:
        - link "CART" [ref=e13] [cursor=pointer]:
          - /url: /cart
          - img [ref=e14]
        - generic [ref=e17]:
          - group "Language" [ref=e18]:
            - button "RU" [ref=e19]
            - button "RO" [ref=e20]
            - button "EN" [pressed] [ref=e21]
          - group "Currency" [ref=e22]:
            - button "RUB" [pressed] [ref=e23]
            - button "MDL" [ref=e24]
        - button "Toggle theme" [ref=e25]:
          - img [ref=e26]
        - link [ref=e32] [cursor=pointer]:
          - /url: /profile
          - img [ref=e33]
  - main [ref=e36]:
    - generic [ref=e38]:
      - generic [ref=e39]: ⚠️
      - heading "Error" [level=2] [ref=e40]
      - paragraph [ref=e41]: "[GET /api/products/midnight-runner] Network error"
      - link "Back to catalog" [ref=e42] [cursor=pointer]:
        - /url: /catalog
        - img [ref=e43]
        - text: Back to catalog
  - contentinfo [ref=e45]:
    - generic [ref=e46]:
      - generic [ref=e47]:
        - generic [ref=e48]:
          - link "MIORU" [ref=e49] [cursor=pointer]:
            - /url: /
            - img "MIORU" [ref=e50]
          - paragraph [ref=e51]: CATALOG OF ITEMS IN STOCK AND AVAILABLE FOR PRE-ORDER
        - generic [ref=e52]:
          - heading "Navigation" [level=3] [ref=e53]
          - list [ref=e54]:
            - listitem [ref=e55]:
              - link "CATALOG" [ref=e56] [cursor=pointer]:
                - /url: /catalog
            - listitem [ref=e57]:
              - link "CUSTOM ORDER" [ref=e58] [cursor=pointer]:
                - /url: /custom-order
            - listitem [ref=e59]:
              - link "CART" [ref=e60] [cursor=pointer]:
                - /url: /cart
        - generic [ref=e61]:
          - heading "Contacts" [level=3] [ref=e62]
          - list [ref=e63]:
            - listitem [ref=e64]:
              - 'link "Telegram: @miorumanager" [ref=e65] [cursor=pointer]':
                - /url: https://t.me/miorumanager
            - listitem [ref=e66]:
              - 'link "Instagram: @mioru.store" [ref=e67] [cursor=pointer]':
                - /url: https://instagram.com/mioru.store
            - listitem [ref=e68]: support@mioru.store
      - generic [ref=e69]:
        - paragraph [ref=e70]: © 2025 MIORU
        - paragraph [ref=e71]: All rights reserved
```

# Test source

```ts
  1   | import { test, expect } from "@playwright/test";
  2   | 
  3   | const DESKTOP = { width: 1280, height: 800 } as const;
  4   | const TABLET = { width: 768, height: 1024 } as const;
  5   | const MOBILE = { width: 375, height: 812 } as const;
  6   | 
  7   | test.describe("Visual Regression — Desktop (1280×800)", () => {
  8   |   test.use({ viewport: DESKTOP });
  9   | 
  10  |   test("Homepage — all sections visible, nothing overlaps", async ({
  11  |     page,
  12  |   }) => {
  13  |     await page.goto("/");
  14  |     await page.waitForLoadState("networkidle");
  15  |     await expect(page).toHaveScreenshot("desktop-homepage.png", {
  16  |       fullPage: true,
  17  |       maxDiffPixelRatio: 0.03,
  18  |     });
  19  |   });
  20  | 
  21  |   test("Catalog — sidebar + 4-column grid + filters", async ({ page }) => {
  22  |     await page.goto("/catalog");
  23  |     await page.waitForLoadState("networkidle");
  24  |     await expect(page).toHaveScreenshot("desktop-catalog.png", {
  25  |       fullPage: true,
  26  |     });
  27  |   });
  28  | 
  29  |   test("Product page — gallery left, info right", async ({ page }) => {
  30  |     await page.goto("/product/midnight-runner");
  31  |     await page.waitForLoadState("networkidle");
  32  |     // Wait for lazy-loaded components
  33  |     await page.waitForTimeout(2000);
  34  |     await expect(page).toHaveScreenshot("desktop-product.png", {
  35  |       fullPage: true,
  36  |     });
  37  |   });
  38  | 
  39  |   test("Cart with items — quantity controls, summary", async ({ page }) => {
  40  |     await page.goto("/product/midnight-runner");
  41  |     await page.waitForLoadState("networkidle");
  42  |     await page.waitForTimeout(1000);
  43  |     // Stable selectors (CLAUDE.md): pick size 42 by data-size, then the
  44  |     // add-to-cart button by testid — not i18n copy / arbitrary text.
> 45  |     await page.locator('[data-testid="product-size"][data-size="42"]').click();
      |                                                                        ^ Error: locator.click: Test timeout of 30000ms exceeded.
  46  |     await page.getByTestId("product-add-to-cart").click();
  47  |     await page.waitForTimeout(500);
  48  |     await page.goto("/cart");
  49  |     await page.waitForLoadState("networkidle");
  50  |     await expect(page).toHaveScreenshot("desktop-cart-filled.png", {
  51  |       fullPage: true,
  52  |     });
  53  |   });
  54  | 
  55  |   test("Cart empty — placeholder + CTA", async ({ page }) => {
  56  |     await page.goto("/cart");
  57  |     await page.waitForLoadState("networkidle");
  58  |     await expect(page).toHaveScreenshot("desktop-cart-empty.png", {
  59  |       fullPage: true,
  60  |     });
  61  |   });
  62  | 
  63  |   test("Checkout — stepper visible, inputs aligned", async ({ page }) => {
  64  |     await page.goto("/checkout");
  65  |     await page.waitForLoadState("networkidle");
  66  |     await expect(page).toHaveScreenshot("desktop-checkout.png", {
  67  |       fullPage: true,
  68  |     });
  69  |   });
  70  | 
  71  |   test("Profile — XP bar, quick links, order history", async ({ page }) => {
  72  |     await page.goto("/profile");
  73  |     await page.waitForLoadState("networkidle");
  74  |     await expect(page).toHaveScreenshot("desktop-profile.png", {
  75  |       fullPage: true,
  76  |     });
  77  |   });
  78  | 
  79  |   test("Admin — products table, form toggle", async ({ page }) => {
  80  |     await page.goto("/admin");
  81  |     await page.waitForLoadState("networkidle");
  82  |     await expect(page).toHaveScreenshot("desktop-admin.png", {
  83  |       fullPage: true,
  84  |     });
  85  |   });
  86  | 
  87  |   test("Avatar editor — 3D viewport + sliders", async ({ page }) => {
  88  |     await page.goto("/avatar");
  89  |     await page.waitForLoadState("networkidle");
  90  |     await expect(page).toHaveScreenshot("desktop-avatar.png", {
  91  |       fullPage: true,
  92  |     });
  93  |   });
  94  | });
  95  | 
  96  | test.describe("Visual Regression — Mobile (375×812)", () => {
  97  |   test.use({ viewport: MOBILE });
  98  | 
  99  |   test("Homepage — mobile layout, no overflow", async ({ page }) => {
  100 |     await page.goto("/");
  101 |     await page.waitForLoadState("networkidle");
  102 |     await expect(page).toHaveScreenshot("mobile-homepage.png", {
  103 |       fullPage: true,
  104 |       maxDiffPixelRatio: 0.03,
  105 |     });
  106 |   });
  107 | 
  108 |   test("Catalog — filter button visible, 2-column grid", async ({ page }) => {
  109 |     await page.goto("/catalog");
  110 |     await page.waitForLoadState("networkidle");
  111 |     await expect(page).toHaveScreenshot("mobile-catalog.png", {
  112 |       fullPage: true,
  113 |     });
  114 |   });
  115 | 
  116 |   test("Product — stacked layout, size buttons visible", async ({ page }) => {
  117 |     await page.goto("/product/midnight-runner");
  118 |     await page.waitForLoadState("networkidle");
  119 |     await expect(page).toHaveScreenshot("mobile-product.png", {
  120 |       fullPage: true,
  121 |     });
  122 |   });
  123 | 
  124 |   test("Mobile menu open — links centered, backdrop visible", async ({
  125 |     page,
  126 |   }) => {
  127 |     await page.goto("/");
  128 |     await page.click('button[aria-label="Menu"]');
  129 |     await page.waitForTimeout(500);
  130 |     await expect(page).toHaveScreenshot("mobile-menu-open.png", {
  131 |       fullPage: false,
  132 |     });
  133 |   });
  134 | 
  135 |   test("Cart — stacked items, controls reachable", async ({ page }) => {
  136 |     await page.goto("/product/midnight-runner");
  137 |     await page.waitForLoadState("networkidle");
  138 |     await page.waitForTimeout(1000);
  139 |     // Stable selectors (CLAUDE.md): size by data-size, add-to-cart by testid.
  140 |     await page.locator('[data-testid="product-size"][data-size="42"]').click();
  141 |     await page.getByTestId("product-add-to-cart").click();
  142 |     await page.waitForTimeout(500);
  143 |     await page.goto("/cart");
  144 |     await page.waitForLoadState("networkidle");
  145 |     await expect(page).toHaveScreenshot("mobile-cart.png", { fullPage: true });
```