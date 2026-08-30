import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, fireEvent, act } from "@testing-library/react";
import { MemoryRouter, Routes, Route } from "react-router-dom";
import { HelmetProvider } from "@dr.pogodin/react-helmet";
import CatalogPage from "./CatalogPage";
import { useCatalogStore } from "@/stores/catalogStore";
import { useFavoritesStore } from "@/stores/favoritesStore";
import { useCurrencyStore } from "@/stores/currencyStore";
import type { Category } from "@/types";
import "@testing-library/jest-dom/vitest";

// ── Mocks (non-store infra only) ──────────────────────────────────────────

vi.mock("framer-motion", () => ({
  motion: new Proxy(
    {},
    {
      get: (_t, tag: string) =>
        ({
          div: (p: React.HTMLAttributes<HTMLDivElement>) => <div {...p} />,
        })[tag] || ((p: React.HTMLAttributes<HTMLElement>) => <div {...p} />),
    },
  ),
  AnimatePresence: ({ children }: { children: React.ReactNode }) => <>{children}</>,
}));

vi.mock("react-i18next", () => ({
  useTranslation: () => ({ t: (k: string) => k }),
}));

vi.mock("@/stores/catalogStore", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@/stores/catalogStore")>();
  return { ...actual };
});

const categories: Category[] = [
  {
    id: 11, parent_id: null, name: "Обувь", slug: "shoes", criteria: [],
    sort_order: 2, products_count: 3,
    children: [
      { id: 12, parent_id: 11, name: "Кроссовки", slug: "sneakers", criteria: [], sort_order: 1, products_count: 2 },
      { id: 13, parent_id: 11, name: "Тапки", slug: "slides", criteria: [], sort_order: 2, products_count: 1 },
    ],
  },
];

function setupStore() {
  const fetchProducts = vi.fn().mockResolvedValue({});
  const fetchFacets = vi.fn().mockResolvedValue({});
  const fetchCategories = vi.fn();
  useCatalogStore.setState({
    items: [],
    total: 0,
    loading: false,
    error: null,
    categories,
    facets: { brands: [], colors: [], sizes: [] },
    fetchProducts,
    fetchFacets,
    fetchCategories,
  } as unknown as Partial<ReturnType<typeof useCatalogStore.getState>>);
  useFavoritesStore.setState({ items: [], toggleFavorite: vi.fn() });
  useCurrencyStore.setState({ currency: "MDL" } as never);
  return { fetchProducts, fetchFacets };
}

function renderPage(initialEntry = "/catalog") {
  return render(
    <HelmetProvider>
      <MemoryRouter initialEntries={[initialEntry]}>
        <Routes>
          <Route path="/catalog/:categorySlug?" element={<CatalogPage />} />
        </Routes>
      </MemoryRouter>
    </HelmetProvider>,
  );
}

describe("CatalogPage — KAN-55 filter panel", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("navigates by slug, not id, when the parent select changes (F1 regression)", async () => {
    const { fetchProducts } = setupStore();
    renderPage();
    fireEvent.click(screen.getByTestId("catalog-filter-button"));

    const parentSelect = screen.getByTestId("catalog-category-parent-select") as HTMLSelectElement;
    expect(parentSelect.options).toHaveLength(2); // «Все категории» + Обувь
    fireEvent.change(parentSelect, { target: { value: "shoes" } });

    // The fetch must be scoped to the slug's descendants (11,12,13).
    const calls = vi.mocked(fetchProducts).mock.calls;
    const last = calls[calls.length - 1]?.[0] as { category_id?: string[] } | undefined;
    expect(last?.category_id).toEqual(["11", "12", "13"]);
  });

  it("falls back to «all» availability for an unknown URL status", () => {
    const { fetchProducts } = setupStore();
    renderPage("/catalog?status=weird");

    const calls = vi.mocked(fetchProducts).mock.calls;
    const last = calls[calls.length - 1]?.[0] as { status?: string } | undefined;
    expect(last?.status).toBeUndefined();
  });

  it("resets page to 1 when a filter is toggled", () => {
    const { fetchProducts } = setupStore();
    renderPage();
    fireEvent.click(screen.getByTestId("catalog-filter-button"));

    // Toggle availability in the panel; the fetch must carry page=1.
    fireEvent.click(screen.getByTestId("catalog-availability-in_stock"));
    const calls = vi.mocked(fetchProducts).mock.calls;
    const last = calls[calls.length - 1]?.[0] as { page?: string } | undefined;
    expect(last?.page).toBe("1");
  });

  it("debounces search: nothing before 300ms, URL-driven fetch after", () => {
    vi.useFakeTimers();
    try {
      const { fetchProducts } = setupStore();
      renderPage();
      const input = screen.getByTestId("catalog-search-input") as HTMLInputElement;
      fireEvent.change(input, { target: { value: "ab" } });

      const callsBefore = vi.mocked(fetchProducts).mock.calls.length;
      act(() => {
        vi.advanceTimersByTime(299);
      });
      expect(vi.mocked(fetchProducts).mock.calls.length).toBe(callsBefore);

      act(() => {
        vi.advanceTimersByTime(1);
      });
      const calls = vi.mocked(fetchProducts).mock.calls;
      const last = calls[calls.length - 1]?.[0] as { search?: string } | undefined;
      expect(last?.search).toBe("ab");
    } finally {
      vi.useRealTimers();
    }
  });

  it("drops a non-positive price instead of advertising a dead filter", () => {
    const { fetchProducts } = setupStore();
    renderPage();
    fireEvent.click(screen.getByTestId("catalog-filter-button"));

    const priceInput = document.querySelector('input[placeholder="catalog.panel.priceFrom"]') as HTMLInputElement;
    fireEvent.change(priceInput, { target: { value: "0" } });
    fireEvent.blur(priceInput);

    const calls = vi.mocked(fetchProducts).mock.calls;
    const last = calls[calls.length - 1]?.[0] as { price_min?: string } | undefined;
    expect(last?.price_min).toBeUndefined();
  });
});
