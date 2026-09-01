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

function setupStore(total = 0) {
  const fetchProducts = vi.fn().mockResolvedValue({});
  const fetchFacets = vi.fn().mockResolvedValue({});
  const fetchCategories = vi.fn();
  useCatalogStore.setState({
    items: [],
    total,
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
    const { fetchProducts } = setupStore(201);
    renderPage();
    // Move to page 3 first, then toggle a filter — the fetch must
    // carry page=1 again (without the reset it would carry page=3).
    const page3Btn = screen.getAllByRole("button").find((b) => b.textContent?.trim() === "3");
    expect(page3Btn).toBeDefined();
    fireEvent.click(page3Btn!);
    fireEvent.click(screen.getByTestId("catalog-filter-button"));
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

  it("drops a non-positive or fractional price instead of advertising a dead filter", () => {
    const { fetchProducts } = setupStore();
    renderPage();
    fireEvent.click(screen.getByTestId("catalog-filter-button"));

    // The input is keyed by priceMin, so it remounts after every blur —
    // re-query the node instead of caching it.
    const priceInput = () =>
      document.querySelector('input[placeholder="catalog.panel.priceFrom"]') as HTMLInputElement;
    // A legal integer lands in the URL and drives the fetch.
    fireEvent.change(priceInput(), { target: { value: "100" } });
    fireEvent.blur(priceInput());
    let calls = vi.mocked(fetchProducts).mock.calls;
    let last = calls[calls.length - 1]?.[0] as { price_min?: string } | undefined;
    expect(last?.price_min).toBe("100");

    // Zero and fractions are silently ignored by the backend — the
    // URL must not advertise them.
    fireEvent.change(priceInput(), { target: { value: "0" } });
    fireEvent.blur(priceInput());
    calls = vi.mocked(fetchProducts).mock.calls;
    last = calls[calls.length - 1]?.[0] as { price_min?: string } | undefined;
    expect(last?.price_min).toBeUndefined();

    fireEvent.change(priceInput(), { target: { value: "49.5" } });
    fireEvent.blur(priceInput());
    calls = vi.mocked(fetchProducts).mock.calls;
    last = calls[calls.length - 1]?.[0] as { price_min?: string } | undefined;
    expect(last?.price_min).toBeUndefined();
  });
});

describe("CatalogPage — brand facet cap (R5)", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("shows 25 brands plus «show all», then expands to the full list", () => {
    const brands = Array.from({ length: 26 }, (_, i) => `Brand${i}`);
    setupStore();
    useCatalogStore.setState({
      facets: { brands, colors: [], sizes: [] },
    } as never);
    renderPage();
    fireEvent.click(screen.getByTestId("catalog-filter-button"));

    expect(screen.getAllByText(/^Brand\d+$/)).toHaveLength(25);
    expect(screen.getByTestId("catalog-brands-show-all")).toBeInTheDocument();

    fireEvent.click(screen.getByTestId("catalog-brands-show-all"));
    expect(screen.getAllByText(/^Brand\d+$/)).toHaveLength(26);
    expect(screen.queryByTestId("catalog-brands-show-all")).toBeNull();
  });
});

describe("CatalogPage — brand selection cap (S1)", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("refuses to select more than 20 brands", () => {
    const brands = Array.from({ length: 25 }, (_, i) => `Brand${i}`);
    const { fetchProducts } = setupStore();
    useCatalogStore.setState({
      facets: { brands, colors: [], sizes: [] },
    } as never);
    renderPage();
    fireEvent.click(screen.getByTestId("catalog-filter-button"));

    // Select 20 brands — allowed (25 fit the default view, no show-all).
    // Re-query after every click: each toggle re-renders the panel and
    // invalidates the previous node handles.
    for (let i = 0; i < 20; i++) {
      fireEvent.click(screen.getAllByText(/^Brand\d+$/)[i]);
    }
    const calls = vi.mocked(fetchProducts).mock.calls;
    const last = calls[calls.length - 1]?.[0] as { brand?: string[] } | undefined;
    expect(last?.brand).toHaveLength(20);

    // The 21st selection is refused — the URL filter stays at 20 values.
    fireEvent.click(screen.getAllByText(/^Brand\d+$/)[20]);
    const after = vi.mocked(fetchProducts).mock.calls;
    const last2 = after[after.length - 1]?.[0] as { brand?: string[] } | undefined;
    expect(last2?.brand).toHaveLength(20);
  });
});

describe("CatalogPage — color selection cap (F4 symmetry)", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("refuses to select more than 20 colors", () => {
    const colors = Array.from({ length: 21 }, (_, i) => `Color${i}`);
    const { fetchProducts } = setupStore();
    useCatalogStore.setState({
      facets: { brands: [], colors, sizes: [] },
    } as never);
    renderPage();
    fireEvent.click(screen.getByTestId("catalog-filter-button"));

    for (let i = 0; i < 20; i++) {
      fireEvent.click(screen.getAllByText(/^catalog\.colorFilter\.Color\d+$/)[i]);
    }
    const calls = vi.mocked(fetchProducts).mock.calls;
    const last = calls[calls.length - 1]?.[0] as { color?: string[] } | undefined;
    expect(last?.color).toHaveLength(20);

    fireEvent.click(screen.getAllByText(/^catalog\.colorFilter\.Color\d+$/)[20]);
    const after = vi.mocked(fetchProducts).mock.calls;
    const last2 = after[after.length - 1]?.[0] as { color?: string[] } | undefined;
    expect(last2?.color).toHaveLength(20);
  });
});

describe("CatalogPage — size selection cap (F4 symmetry)", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("refuses to select more than 20 sizes", () => {
    // Size chips render the raw label (no i18n prefix, unlike colors) and are
    // sorted by localeCompare here — none of these parse as a number, so the
    // chip order is stable across the re-render each toggle triggers.
    const sizes = Array.from({ length: 21 }, (_, i) => `SZ${i}`);
    const { fetchProducts } = setupStore();
    useCatalogStore.setState({
      facets: { brands: [], colors: [], sizes },
    } as never);
    renderPage();
    fireEvent.click(screen.getByTestId("catalog-filter-button"));

    for (let i = 0; i < 20; i++) {
      fireEvent.click(screen.getAllByText(/^SZ\d+$/)[i]);
    }
    const calls = vi.mocked(fetchProducts).mock.calls;
    const last = calls[calls.length - 1]?.[0] as { size?: string[] } | undefined;
    expect(last?.size).toHaveLength(20);

    fireEvent.click(screen.getAllByText(/^SZ\d+$/)[20]);
    const after = vi.mocked(fetchProducts).mock.calls;
    const last2 = after[after.length - 1]?.[0] as { size?: string[] } | undefined;
    expect(last2?.size).toHaveLength(20);
  });
});
