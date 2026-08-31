import { describe, it, expect, beforeEach, vi } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { HelmetProvider } from "@dr.pogodin/react-helmet";
import Header from "./Header";
import "@testing-library/jest-dom/vitest";

const changeLanguageMock = vi.fn();

// Mock react-i18next
vi.mock("react-i18next", () => ({
  useTranslation: () => ({
    t: (key: string) => {
      const map: Record<string, string> = {
        "nav.inStock": "КАТАЛОГ",
        "nav.customOrder": "ИНДИВИДУАЛЬНЫЙ ЗАКАЗ",
        "nav.cart": "КОРЗИНА",
        "nav.favorites": "ИЗБРАННОЕ",
        "nav.profile": "Профиль",
        "nav.language": "Язык",
        "nav.currency": "Валюта",
        "theme.toggle": "Переключить тему",
        "common.close": "Закрыть",
      };
      return map[key] || key;
    },
    i18n: {
      language: "ru",
      resolvedLanguage: "ru",
      changeLanguage: changeLanguageMock,
    },
  }),
}));

// Mock Zustand cart store
vi.mock("@/stores/cartStore", () => ({
  useCartStore: vi.fn(
    (
      selector?: (state: {
        items: unknown[];
        totalItems: () => number;
      }) => unknown,
    ) => {
      const state = { items: [], totalItems: () => 0 };
      return selector ? selector(state) : state;
    },
  ),
}));

function renderHeader(theme: "dark" | "light" = "dark") {
  return render(
    <HelmetProvider>
      <MemoryRouter>
        <Header theme={theme} toggleTheme={() => {}} />
      </MemoryRouter>
    </HelmetProvider>,
  );
}

beforeEach(() => {
  changeLanguageMock.mockClear();
  localStorage.clear();
});

describe("Header — layout & rendering", () => {
  it("renders logo", () => {
    renderHeader();
    const logo = screen.getByAltText("MIORU");
    expect(logo).toBeInTheDocument();
    expect(logo.tagName).toBe("IMG");
  });

  it("renders nav links — Каталог, Индивидуальный заказ, Избранное (no АВАТАР)", () => {
    renderHeader();
    expect(screen.getByText("КАТАЛОГ")).toBeInTheDocument();
    expect(screen.getByText("ИНДИВИДУАЛЬНЫЙ ЗАКАЗ")).toBeInTheDocument();
    expect(screen.getByText("ИЗБРАННОЕ")).toBeInTheDocument();
    expect(screen.queryByText("АВАТАР")).not.toBeInTheDocument();
  });

  it("has no heart-icon favorites link (KAN-56: favorites is a text nav link)", () => {
    renderHeader();
    // Favorites is now a text link, not an icon-only link — there is no
    // separate icon link labeled "ИЗБРАННОЕ" beyond the nav text link.
    const favLinks = screen.getAllByRole("link").filter(
      (l) => l.getAttribute("href") === "/favorites",
    );
    expect(favLinks.length).toBe(1);
  });

  it("has no globe language dropdown (KAN-56)", () => {
    renderHeader();
    expect(screen.queryByLabelText("Change language")).not.toBeInTheDocument();
  });

  it("renders theme toggle button", () => {
    renderHeader();
    expect(screen.getByLabelText("Переключить тему")).toBeInTheDocument();
  });

  it("renders profile link", () => {
    renderHeader();
    const links = screen.getAllByRole("link");
    const hasProfileLink = links.some(
      (link) => link.getAttribute("href") === "/profile",
    );
    expect(hasProfileLink).toBe(true);
  });

  it("renders mobile menu toggle", () => {
    renderHeader();
    const toggle = screen.getByLabelText("Menu");
    expect(toggle).toBeInTheDocument();
  });

  it("nav links have correct hrefs (no /avatar)", () => {
    renderHeader();
    const links = screen.getAllByRole("link");
    const hrefs = links.map((l) => l.getAttribute("href"));

    expect(hrefs).toContain("/catalog");
    expect(hrefs).toContain("/custom-order");
    expect(hrefs).toContain("/favorites");
    expect(hrefs).toContain("/cart");
    expect(hrefs).toContain("/profile");
    expect(hrefs).not.toContain("/avatar");
  });

  it("cart link is visible on mobile — not display-none (KAN-56)", () => {
    renderHeader();
    const cartLink = screen.getByLabelText("КОРЗИНА");
    expect(cartLink.className).not.toContain("hidden");
  });

  it("does not show mobile menu overlay by default", () => {
    renderHeader();
    expect(screen.queryByTestId("mobile-menu")).not.toBeInTheDocument();
    // Only one set of nav links (desktop) when menu is closed
    const catalogLinks = screen.getAllByText("КАТАЛОГ");
    expect(catalogLinks.length).toBe(1);
  });
});

describe("Header — preference toggles (KAN-56)", () => {
  it("renders language toggle with RU/RO/EN options", () => {
    renderHeader();
    expect(screen.getByTestId("language-toggle-ru")).toBeInTheDocument();
    expect(screen.getByTestId("language-toggle-ro")).toBeInTheDocument();
    expect(screen.getByTestId("language-toggle-en")).toBeInTheDocument();
  });

  it("marks the current language as pressed", () => {
    renderHeader();
    expect(screen.getByTestId("language-toggle-ru")).toHaveAttribute(
      "aria-pressed",
      "true",
    );
    expect(screen.getByTestId("language-toggle-en")).toHaveAttribute(
      "aria-pressed",
      "false",
    );
  });

  it("clicking EN switches language via i18n.changeLanguage", () => {
    renderHeader();
    fireEvent.click(screen.getByTestId("language-toggle-en"));
    expect(changeLanguageMock).toHaveBeenCalledWith("en");
  });

  it("renders currency toggle with RUB/MDL and persists choice", () => {
    renderHeader();
    const mdl = screen.getByTestId("currency-toggle-mdl");
    expect(screen.getByTestId("currency-toggle-pmr")).toBeInTheDocument();
    fireEvent.click(mdl);
    expect(localStorage.getItem("mioru-currency")).toBe("MDL");
  });

  it("mobile menu overlay contains both toggles at the bottom", () => {
    renderHeader();
    fireEvent.click(screen.getByLabelText("Menu"));
    const menu = screen.getByTestId("mobile-menu");
    // Desktop toggles are CSS-hidden but in the DOM; the menu adds its own pair
    expect(
      menu.querySelectorAll('[data-testid="language-toggle"]').length,
    ).toBe(1);
    expect(
      menu.querySelectorAll('[data-testid="currency-toggle"]').length,
    ).toBe(1);
    // Menu shows mobile-only links (KAN-56: cart is a header icon, not a menu item)
    expect(menu.textContent).toContain("ИЗБРАННОЕ");
    expect(menu.textContent).not.toContain("КОРЗИНА");
    expect(menu.textContent).not.toContain("АВАТАР");
  });
});

describe("Header — CSS class assertions", () => {
  it("header has fixed positioning and correct z-index", () => {
    renderHeader();
    const header = document.querySelector("header");
    expect(header?.className).toContain("fixed");
    expect(header?.className).toContain("z-50");
  });

  it("nav is hidden on mobile (md:block present for absolute centering)", () => {
    renderHeader();
    const nav = document.querySelector("header nav");
    // Nav should exist and have hidden class for mobile
    expect(nav).toBeTruthy();
    expect(nav?.className).toContain("hidden");
  });
});
