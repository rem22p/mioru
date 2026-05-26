import { create } from "zustand";
import type { Product, Category } from "@/types";
import {
  fetchStoreProducts,
  fetchStoreProductFacets,
  fetchStoreCategories,
  type CatalogQuery,
  type ProductFacets,
} from "@/lib/api";

interface CatalogStore {
  products: Product[];
  categories: Category[];
  facets: ProductFacets;
  total: number;
  page: number;
  perPage: number;
  loading: boolean;
  error: string | null;
  fetchProducts: (query?: CatalogQuery) => Promise<void>;
  fetchFacets: (query?: CatalogQuery) => Promise<void>;
  fetchCategories: () => Promise<void>;
}

const emptyFacets: ProductFacets = { brands: [], colors: [], sizes: [] };

export const useCatalogStore = create<CatalogStore>((set) => ({
  products: [],
  categories: [],
  facets: emptyFacets,
  total: 0,
  page: 1,
  perPage: 20,
  loading: false,
  error: null,

  fetchProducts: async (query = {}) => {
    set({ loading: true, error: null });
    try {
      const data = await fetchStoreProducts(query);
      set({
        products: data.products,
        total: data.total,
        page: data.page,
        perPage: data.per_page,
        loading: false,
      });
    } catch (err) {
      set({
        error: err instanceof Error ? err.message : "Failed to fetch products",
        loading: false,
      });
    }
  },

  fetchFacets: async (query = {}) => {
    try {
      const data = await fetchStoreProductFacets(query);
      set({ facets: data });
    } catch {
      // Facets are best-effort: a failure here must not block the catalog.
      set({ facets: emptyFacets });
    }
  },

  fetchCategories: async () => {
    set((state) => ({
      error: null,
      loading: state.categories.length === 0,
    }));
    try {
      const data = await fetchStoreCategories();
      set({ categories: data, loading: false });
    } catch (err) {
      set({
        error:
          err instanceof Error ? err.message : "Failed to fetch categories",
        loading: false,
      });
    }
  },
}));
