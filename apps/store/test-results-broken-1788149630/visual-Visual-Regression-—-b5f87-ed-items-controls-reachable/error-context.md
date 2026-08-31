# Instructions

- Following Playwright test failed.
- Explain why, be concise, respect Playwright best practices.
- Provide a snippet of code with the fix, if possible.

# Test info

- Name: visual.spec.ts >> Visual Regression — Mobile (375×812) >> Cart — stacked items, controls reachable
- Location: e2e/visual.spec.ts:135:3

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
      - generic [ref=e8]:
        - link "CART" [ref=e9] [cursor=pointer]:
          - /url: /cart
          - img [ref=e10]
        - button "Toggle theme" [ref=e13]:
          - img [ref=e14]
        - link [ref=e20] [cursor=pointer]:
          - /url: /profile
          - img [ref=e21]
        - button "Menu" [ref=e24]:
          - img [ref=e25]
  - main [ref=e26]:
    - generic [ref=e28]:
      - generic [ref=e29]: ⚠️
      - heading "Error" [level=2] [ref=e30]
      - paragraph [ref=e31]: "[GET /api/products/midnight-runner] Network error"
      - link "Back to catalog" [ref=e32] [cursor=pointer]:
        - /url: /catalog
        - img [ref=e33]
        - text: Back to catalog
  - contentinfo [ref=e35]:
    - generic [ref=e36]:
      - generic [ref=e37]:
        - generic [ref=e38]:
          - link "MIORU" [ref=e39] [cursor=pointer]:
            - /url: /
            - img "MIORU" [ref=e40]
          - paragraph [ref=e41]: CATALOG OF ITEMS IN STOCK AND AVAILABLE FOR PRE-ORDER
        - generic [ref=e42]:
          - heading "Navigation" [level=3] [ref=e43]
          - list [ref=e44]:
            - listitem [ref=e45]:
              - link "CATALOG" [ref=e46] [cursor=pointer]:
                - /url: /catalog
            - listitem [ref=e47]:
              - link "CUSTOM ORDER" [ref=e48] [cursor=pointer]:
                - /url: /custom-order
            - listitem [ref=e49]:
              - link "CART" [ref=e50] [cursor=pointer]:
                - /url: /cart
        - generic [ref=e51]:
          - heading "Contacts" [level=3] [ref=e52]
          - list [ref=e53]:
            - listitem [ref=e54]:
              - 'link "Telegram: @miorumanager" [ref=e55] [cursor=pointer]':
                - /url: https://t.me/miorumanager
            - listitem [ref=e56]:
              - 'link "Instagram: @mioru.store" [ref=e57] [cursor=pointer]':
                - /url: https://instagram.com/mioru.store
            - listitem [ref=e58]: support@mioru.store
      - generic [ref=e59]:
        - paragraph [ref=e60]: © 2025 MIORU
        - paragraph [ref=e61]: All rights reserved
```

# Test source

```ts
  40  |     await page.goto("/product/midnight-runner");
  41  |     await page.waitForLoadState("networkidle");
  42  |     await page.waitForTimeout(1000);
  43  |     // Stable selectors (CLAUDE.md): pick size 42 by data-size, then the
  44  |     // add-to-cart button by testid — not i18n copy / arbitrary text.
  45  |     await page.locator('[data-testid="product-size"][data-size="42"]').click();
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
> 140 |     await page.locator('[data-testid="product-size"][data-size="42"]').click();
      |                                                                        ^ Error: locator.click: Test timeout of 30000ms exceeded.
  141 |     await page.getByTestId("product-add-to-cart").click();
  142 |     await page.waitForTimeout(500);
  143 |     await page.goto("/cart");
  144 |     await page.waitForLoadState("networkidle");
  145 |     await expect(page).toHaveScreenshot("mobile-cart.png", { fullPage: true });
  146 |   });
  147 | });
  148 | 
  149 | test.describe("Visual Regression — Tablet (768×1024)", () => {
  150 |   test.use({ viewport: TABLET });
  151 | 
  152 |   test("Homepage — tablet breakpoint", async ({ page }) => {
  153 |     await page.goto("/");
  154 |     await page.waitForLoadState("networkidle");
  155 |     await expect(page).toHaveScreenshot("tablet-homepage.png", {
  156 |       fullPage: true,
  157 |       maxDiffPixelRatio: 0.03,
  158 |     });
  159 |   });
  160 | });
  161 | 
  162 | test.describe("Header — visual states", () => {
  163 |   test("Scrolled header — background + blur + border", async ({ page }) => {
  164 |     await page.setViewportSize(DESKTOP);
  165 |     await page.goto("/");
  166 |     await page.evaluate(() => window.scrollTo(0, 200));
  167 |     await page.waitForTimeout(600);
  168 |     const header = page.locator("header");
  169 |     await expect(header).toHaveScreenshot("header-scrolled.png");
  170 |   });
  171 | 
  172 |   test("Transparent header at top", async ({ page }) => {
  173 |     await page.setViewportSize(DESKTOP);
  174 |     await page.goto("/");
  175 |     await page.waitForTimeout(300);
  176 |     const header = page.locator("header");
  177 |     await expect(header).toHaveScreenshot("header-top.png");
  178 |   });
  179 | });
  180 | 
  181 | // The no-overlap margin is narrowest at the smallest lg width in the longest
  182 | // locale (RU: 29px at 1024) — pinning it only at 1280/en leaves 210px of slack,
  183 | // so that assertion cannot fail. Logo width and cluster clipping cover the
  184 | // 768-1023 band, where the burger replaces the nav.
  185 | for (const [locale, lng] of [
  186 |   ["ru-RU", "ru"],
  187 |   ["ro-RO", "ro"],
  188 |   ["en-US", "en"],
  189 | ] as const) {
  190 |   test.describe(`Header layout invariants - ${lng}`, () => {
  191 |     test.use({ locale });
  192 | 
  193 |     test("logo, right cluster and nav across breakpoints", async ({ page }) => {
  194 |       for (const width of [768, 900, 1023, 1024, 1280]) {
  195 |         await page.setViewportSize({ width, height: 900 });
  196 |         await page.goto("/");
  197 |         await page.waitForLoadState("networkidle");
  198 | 
  199 |         const at = `${lng} @ ${width}px`;
  200 | 
  201 |         const logo = await page
  202 |           .locator('header img[alt="MIORU"]')
  203 |           .boundingBox();
  204 |         expect(logo, `logo missing - ${at}`).toBeTruthy();
  205 |         expect(logo!.width, `logo squashed - ${at}`).toBe(40);
  206 | 
  207 |         const cluster = await page
  208 |           .locator("header > div > div:last-child")
  209 |           .boundingBox();
  210 |         expect(cluster, `right cluster missing - ${at}`).toBeTruthy();
  211 |         expect(
  212 |           cluster!.x + cluster!.width,
  213 |           `right cluster clipped - ${at}`,
  214 |         ).toBeLessThanOrEqual(width - 24);
  215 | 
  216 |         const nav = page.locator("header nav");
  217 |         const burger = page.locator('header button[aria-label="Menu"]');
  218 |         const navVisible = await nav.isVisible();
  219 |         expect(navVisible, `nav visibility - ${at}`).toBe(width >= 1024);
  220 |         expect(await burger.isVisible(), `burger visibility - ${at}`).toBe(
  221 |           width < 1024,
  222 |         );
  223 | 
  224 |         if (navVisible) {
  225 |           const navBox = await nav.boundingBox();
  226 |           const cartBox = await page
  227 |             .locator('header a[href="/cart"]')
  228 |             .first()
  229 |             .boundingBox();
  230 |           expect(navBox, `nav box missing - ${at}`).toBeTruthy();
  231 |           expect(cartBox, `cart box missing - ${at}`).toBeTruthy();
  232 |           expect(
  233 |             navBox!.x + navBox!.width,
  234 |             `nav overlaps the right cluster - ${at}`,
  235 |           ).toBeLessThanOrEqual(cartBox!.x + 1);
  236 |         }
  237 |       }
  238 |     });
  239 |   });
  240 | }
```