import { describe, it, expect, beforeEach, vi } from "vitest";
import { useCatalogStore } from "./catalogStore";
import type { Product } from "@/types";

const sampleProduct: Product = {
  id: 1,
  slug: "demo",
  category_id: 2,
  category_name: "Tees",
  brand: "ACME",
  name: "Demo",
  price: 100,
  color: "red",
  model: "",
  material: "",
  care: [],
  description: "",
  xp_reward: 0,
  in_stock: true,
  status: "in_stock",
  stock_quantity: 1,
  created_by: "",
  created_at: "",
  updated_at: "",
  sizes: [{label: "S", stock_quantity: 5}, {label: "M", stock_quantity: 3}],
  size_chart: [],
  images: [],
};

const fetchMock = vi.fn();
vi.stubGlobal("fetch", fetchMock);

function jsonResponse(body: unknown, ok = true) {
  return {
    ok,
    json: () => Promise.resolve(body),
  } as unknown as Response;
}

beforeEach(() => {
  fetchMock.mockReset();
  useCatalogStore.setState({
    products: [],
    categories: [],
    facets: { brands: [], colors: [], sizes: [] },
    total: 0,
    page: 1,
    perPage: 20,
    loading: false,
    error: null,
  });
});

describe("catalogStore", () => {
  it("fetchProducts builds URLSearchParams with multi-value brand/color/size", async () => {
    fetchMock.mockResolvedValueOnce(
      jsonResponse({
        products: [sampleProduct],
        total: 42,
        page: 2,
        per_page: 20,
      }),
    );

    await useCatalogStore.getState().fetchProducts({
      category_id: ["2", "3"],
      brand: ["Nike", "ACME"],
      color: ["red"],
      size: ["S", "M"],
      price_min: "100",
      sort: "-price",
      page: "2",
      per_page: "20",
    });

    expect(fetchMock).toHaveBeenCalledOnce();
    const [url] = fetchMock.mock.calls[0]!;
    expect(typeof url).toBe("string");
    // Parse query so order doesn't matter, just presence + multi-key correctness.
    const qs = new URL(url as string, "http://x").searchParams;
    expect(qs.getAll("category_id")).toEqual(["2", "3"]);
    expect(qs.getAll("brand")).toEqual(["Nike", "ACME"]);
    expect(qs.getAll("color")).toEqual(["red"]);
    expect(qs.getAll("size")).toEqual(["S", "M"]);
    expect(qs.get("price_min")).toBe("100");
    expect(qs.get("sort")).toBe("-price");
    expect(qs.get("page")).toBe("2");
    expect(qs.get("per_page")).toBe("20");

    const state = useCatalogStore.getState();
    expect(state.products).toEqual([sampleProduct]);
    expect(state.total).toBe(42);
    expect(state.page).toBe(2);
    expect(state.perPage).toBe(20);
    expect(state.loading).toBe(false);
  });

  it("fetchFacets strips chip selections (brand/color/size) before hitting /facets", async () => {
    fetchMock.mockResolvedValueOnce(
      jsonResponse({
        brands: ["ACME"],
        colors: ["red", "blue"],
        sizes: ["S", "M"],
      }),
    );

    await useCatalogStore.getState().fetchFacets({
      category_id: "2",
      price_min: "100",
      brand: ["Nike"],
      color: ["red"],
      size: ["M"],
    });

    const [url] = fetchMock.mock.calls[0]!;
    const qs = new URL(url as string, "http://x").searchParams;
    expect(qs.get("category_id")).toBe("2");
    expect(qs.get("price_min")).toBe("100");
    // Chip selections must not narrow the facet scope.
    expect(qs.getAll("brand")).toEqual([]);
    expect(qs.getAll("color")).toEqual([]);
    expect(qs.getAll("size")).toEqual([]);

    expect(useCatalogStore.getState().facets).toEqual({
      brands: ["ACME"],
      colors: ["red", "blue"],
      sizes: ["S", "M"],
    });
  });

  it("fetchFacets swallows network errors so a failure can't block the catalog", async () => {
    fetchMock.mockRejectedValueOnce(new Error("boom"));
    await useCatalogStore.getState().fetchFacets({ category_id: "2" });
    expect(useCatalogStore.getState().facets).toEqual({
      brands: [],
      colors: [],
      sizes: [],
    });
  });

  it("fetchProducts records error message on failure", async () => {
    fetchMock.mockResolvedValueOnce(jsonResponse({ error: "boom" }, false));
    await useCatalogStore.getState().fetchProducts();
    const state = useCatalogStore.getState();
    // The api wrapper prefixes every thrown error with "[METHOD path]"
    // (PR #56 round 1). catalogStore stores err.message verbatim, so the
    // user-visible value — and the assertion — must include the prefix.
    expect(state.error).toBe("[GET /api/products] boom");
    expect(state.loading).toBe(false);
    expect(state.products).toEqual([]);
  });
});

