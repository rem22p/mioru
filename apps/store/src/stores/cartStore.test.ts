import { describe, it, expect, beforeEach, vi } from 'vitest';
import { useCartStore, loadCartFromServer } from '@/stores/cartStore';
import * as api from '@/lib/api';
import type { Product } from '@/types';

// Mock authStore at module level so dynamic import picks it up.
// Each test mutates useAuthStore.getState()'s return via the mock.
const mockAuthState = { isAuthenticated: false };
vi.mock('@/stores/authStore', () => ({
  useAuthStore: {
    getState: () => mockAuthState,
  },
}));

const mockProduct: Product = {
  id: 1,
  name: 'Test Sneaker',
  slug: 'test-sneaker',
  description: '',
  brand: 'Test',
  price: 5000,
  xp_reward: 100,
  in_stock: true,
  status: 'in_stock',
  stock_quantity: 20,
  category_id: 12,
  category_name: 'Кроссовки',
  images: [],
  sizes: [
    { label: '42', stock_quantity: 5 },
    { label: '43', stock_quantity: 3 },
    { label: '44', stock_quantity: 0 },
  ],
  color: '#000',
  model: '',
  material: '',
  size_chart: [],
  care: [],
  created_by: 'admin',
  created_at: '2024-01-01',
  updated_at: '2024-01-01',
};

const mockPreorderProduct: Product = {
  ...mockProduct,
  id: 99,
  slug: 'test-preorder',
  status: 'preorder',
};

beforeEach(() => {
  useCartStore.setState({ items: [] });
  mockAuthState.isAuthenticated = false;
  vi.restoreAllMocks();
});

describe('cartStore.addItem', () => {
  it('adds an in-stock item', () => {
    useCartStore.getState().addItem(mockProduct, '42');
    const items = useCartStore.getState().items;
    expect(items).toHaveLength(1);
    expect(items[0].product.id).toBe(1);
    expect(items[0].size).toBe('42');
    expect(items[0].quantity).toBe(1);
  });

  it('adds a preorder item with measurements', () => {
    useCartStore.getState().addItem(
      mockPreorderProduct,
      'preorder',
      1,
      { height: 175, weight: 70 },
    );
    const items = useCartStore.getState().items;
    expect(items).toHaveLength(1);
    expect(items[0].measurements).toEqual({ height: 175, weight: 70 });
  });

  it('stacks quantity on same product+size', () => {
    useCartStore.getState().addItem(mockProduct, '42', 1);
    useCartStore.getState().addItem(mockProduct, '42', 2);
    expect(useCartStore.getState().items[0].quantity).toBe(3);
  });
});

describe('cartStore.split', () => {
  it('splits in-stock from preorder items', () => {
    useCartStore.getState().addItem(mockProduct, '42');
    useCartStore.getState().addItem(
      mockPreorderProduct,
      'preorder',
      1,
      { foot_length: 27 },
    );
    const items = useCartStore.getState().items;
    const inStock = items.filter((i) => i.product.status === 'in_stock');
    const preorder = items.filter((i) => i.product.status !== 'in_stock');
    expect(inStock).toHaveLength(1);
    expect(preorder).toHaveLength(1);
    expect(preorder[0].measurements).toEqual({ foot_length: 27 });
  });
});

// ── loadCartFromServer ──

describe('loadCartFromServer', () => {
  it('skips when not authenticated', async () => {
    const fetchCartSpy = vi.spyOn(api, 'fetchCustomerCart');
    mockAuthState.isAuthenticated = false;
    await loadCartFromServer();
    expect(fetchCartSpy).not.toHaveBeenCalled();
  });

  it('skips when local cart is non-empty (local wins)', async () => {
    useCartStore.getState().addItem(mockProduct, '42');
    const fetchCartSpy = vi.spyOn(api, 'fetchCustomerCart');
    mockAuthState.isAuthenticated = true;
    await loadCartFromServer();
    expect(fetchCartSpy).not.toHaveBeenCalled();
  });

  it('does nothing when server cart is empty', async () => {
    const fetchCartSpy = vi
      .spyOn(api, 'fetchCustomerCart')
      .mockResolvedValue({ items: [] });
    const fetchProdSpy = vi.spyOn(api, 'fetchStoreProduct');
    mockAuthState.isAuthenticated = true;
    await loadCartFromServer();
    expect(fetchCartSpy).toHaveBeenCalledOnce();
    expect(fetchProdSpy).not.toHaveBeenCalled();
    expect(useCartStore.getState().items).toHaveLength(0);
  });

  it('hydrates cart items from server', async () => {
    vi.spyOn(api, 'fetchCustomerCart').mockResolvedValue({
      items: [
        { product_id: 1, size_label: '42', quantity: 2, product_slug: 'test-sneaker', measurements: { height: 175 } },
      ],
    });
    vi.spyOn(api, 'fetchStoreProduct').mockResolvedValue(mockProduct);
    mockAuthState.isAuthenticated = true;
    await loadCartFromServer();
    const items = useCartStore.getState().items;
    expect(items).toHaveLength(1);
    expect(items[0].product.slug).toBe('test-sneaker');
    expect(items[0].size).toBe('42');
    expect(items[0].quantity).toBe(2);
    expect(items[0].measurements).toEqual({ height: 175 });
  });

  it('skips items when product fetch fails (deleted product)', async () => {
    vi.spyOn(api, 'fetchCustomerCart').mockResolvedValue({
      items: [
        { product_id: 1, size_label: '42', quantity: 1, product_slug: 'deleted' },
        { product_id: 2, size_label: 'M', quantity: 3, product_slug: 'ok-slug' },
      ],
    });
    const fetchProdSpy = vi
      .spyOn(api, 'fetchStoreProduct')
      .mockRejectedValueOnce(new Error('404'))
      .mockResolvedValueOnce(mockProduct);
    mockAuthState.isAuthenticated = true;
    await loadCartFromServer();
    const items = useCartStore.getState().items;
    expect(items).toHaveLength(1); // only the second item survives
    expect(items[0].size).toBe('M');
    expect(items[0].quantity).toBe(3);
    expect(fetchProdSpy).toHaveBeenCalledTimes(2);
  });

  it('survives network error silently', async () => {
    vi.spyOn(api, 'fetchCustomerCart').mockRejectedValue(new Error('Network error'));
    mockAuthState.isAuthenticated = true;
    await expect(loadCartFromServer()).resolves.toBeUndefined();
    expect(useCartStore.getState().items).toHaveLength(0);
  });
});
