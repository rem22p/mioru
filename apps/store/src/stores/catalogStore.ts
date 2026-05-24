import { create } from "zustand";
import type { Product, Category } from "@/types";
import { fetchStoreProducts, fetchStoreCategories } from "@/lib/api";

interface CatalogStore {
  products: Product[];
  categories: Category[];
  total: number;
  loading: boolean;
  error: string | null;
  fetchProducts: (params?: Record<string, string>) => Promise<void>;
  fetchCategories: () => Promise<void>;
}

export const useCatalogStore = create<CatalogStore>((set) => ({
  products: [],
  categories: [],
  total: 0,
  loading: false,
  error: null,

  fetchProducts: async (params = {}) => {
    set({ loading: true, error: null });
    try {
      const data = await fetchStoreProducts(params);
      set({ products: data.products, total: data.total, loading: false });
    } catch (err) {
      set({
        error: err instanceof Error ? err.message : "Failed to fetch products",
        loading: false,
      });
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
