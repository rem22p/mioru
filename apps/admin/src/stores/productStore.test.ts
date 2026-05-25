import { describe, it, expect, beforeEach, vi } from 'vitest';

vi.mock('@/lib/api', () => ({
  fetchProducts: vi.fn(),
  fetchCategories: vi.fn(),
}));

import { useProductStore } from './productStore';
import { fetchCategories as apiFetchCategories } from '@/lib/api';
import type { Category } from '@/types';

const mockFetchCategories = vi.mocked(apiFetchCategories);

describe('productStore.fetchCategories', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    useProductStore.setState({ categories: [], categoriesLoading: false });
  });

  it('populates categories on success', async () => {
    const cats: Category[] = [
      { id: 1, parent_id: null, name: 'Одежда', slug: 'clothing', criteria: [] },
      { id: 2, parent_id: 1, name: 'Футболки', slug: 'tshirts', criteria: [] },
    ];
    mockFetchCategories.mockResolvedValueOnce(cats);

    await useProductStore.getState().fetchCategories();

    const state = useProductStore.getState();
    expect(state.categories).toEqual(cats);
    expect(state.categoriesLoading).toBe(false);
  });

  it('keeps categories empty and logs on failure (no silent swallow)', async () => {
    const spy = vi.spyOn(console, 'error').mockImplementation(() => {});
    mockFetchCategories.mockRejectedValueOnce(new Error('network down'));

    await useProductStore.getState().fetchCategories();

    const state = useProductStore.getState();
    expect(state.categories).toEqual([]);
    expect(state.categoriesLoading).toBe(false);
    expect(spy).toHaveBeenCalled();
    spy.mockRestore();
  });
});
