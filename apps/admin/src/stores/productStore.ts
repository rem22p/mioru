import { create } from 'zustand';
import type { Product, Category, ProductFilter } from '@/types';
import { fetchProducts as apiFetchProducts, fetchCategories as apiFetchCategories } from '@/lib/api';

interface ProductStore {
  products: Product[];
  total: number;
  loading: boolean;
  error: string | null;
  filter: ProductFilter;
  categories: Category[];
  categoriesLoading: boolean;

  setFilter: (f: Partial<ProductFilter>) => void;
  resetFilter: () => void;
  fetchProducts: () => Promise<void>;
  fetchCategories: () => Promise<void>;
}

const defaultFilter: ProductFilter = {
  search: '',
  category_id: '',
  brand: '',
  sort: 'newest',
  page: 1,
  limit: 20,
};

export const useProductStore = create<ProductStore>((set, get) => ({
  products: [],
  total: 0,
  loading: false,
  error: null,
  filter: { ...defaultFilter },
  categories: [],
  categoriesLoading: false,

  setFilter: (f) => {
    set((state) => ({
      filter: { ...state.filter, ...f, page: f.page ?? 1 },
    }));
  },

  resetFilter: () => {
    set({ filter: { ...defaultFilter } });
  },

  fetchProducts: async () => {
    const { filter } = get();
    set({ loading: true, error: null });
    try {
      const data = await apiFetchProducts(filter);
      set({ products: data.products, total: data.total, loading: false });
    } catch (err) {
      const msg = err instanceof Error ? err.message : 'Failed to fetch products';
      set({ error: msg, loading: false });
    }
  },

  fetchCategories: async () => {
    set({ categoriesLoading: true });
    try {
      const categories = await apiFetchCategories();
      set({ categories, categoriesLoading: false });
    } catch {
      set({ categoriesLoading: false });
    }
  },
}));
