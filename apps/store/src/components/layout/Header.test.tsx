import { describe, it, expect, beforeEach, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { HelmetProvider } from "react-helmet-async";
import Header from "./Header";
import "@testing-library/jest-dom/vitest";

// Mock react-i18next
vi.mock("react-i18next", () => ({
  useTranslation: () => ({
    t: (key: string) => {
      const map: Record<string, string> = {
        "nav.inStock": "КАТАЛОГ",
        "nav.customOrder": "ИНДИВИДУАЛЬНЫЙ ЗАКАЗ",
        "nav.avatar": "АВАТАР",
        "nav.cart": "КОРЗИНА",
        "nav.favorites": "ИЗБРАННОЕ",
        "nav.profile": "Профиль",
        "theme.toggle": "Переключить тему",
        "common.close": "Закрыть",
      };
      return map[key] || key;
    },
    i18n: {
      language: "ru",
      changeLanguage: () => {},
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
        <Header
          theme={theme}
          toggleTheme={() => {}}
          changeLanguage={() => {}}
        />
      </MemoryRouter>
    </HelmetProvider>,
  );
}

describe("Header — layout & rendering", () => {
  it("renders logo", () => {
    renderHeader();
    const logo = screen.getByAltText("MIORU");
    expect(logo).toBeInTheDocument();
    expect(logo.tagName).toBe("IMG");
  });

  it("renders all nav links", () => {
    renderHeader();
    expect(screen.getByText("КАТАЛОГ")).toBeInTheDocument();
    expect(screen.getByText("ИНДИВИДУАЛЬНЫЙ ЗАКАЗ")).toBeInTheDocument();
    expect(screen.getByText("АВАТАР")).toBeInTheDocument();
  });

  it("renders theme toggle button", () => {
    renderHeader();
    expect(screen.getByLabelText("Переключить тему")).toBeInTheDocument();
  });

  it("renders language switcher", () => {
    renderHeader();
    expect(screen.getByLabelText("Change language")).toBeInTheDocument();
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

  it("nav links have correct hrefs", () => {
    renderHeader();
    const links = screen.getAllByRole("link");
    const hrefs = links.map((l) => l.getAttribute("href"));

    expect(hrefs).toContain("/catalog");
    expect(hrefs).toContain("/custom-order");
    expect(hrefs).toContain("/avatar");
    expect(hrefs).toContain("/cart");
    expect(hrefs).toContain("/favorites");
    expect(hrefs).toContain("/profile");
  });

  it("does not show mobile menu overlay by default", () => {
    renderHeader();
    const catalogLinks = screen.getAllByText("КАТАЛОГ");
    // Only one set of nav links (desktop) when menu is closed
    expect(catalogLinks.length).toBe(1);
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
