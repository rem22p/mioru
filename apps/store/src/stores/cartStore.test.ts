import { describe, it, expect, beforeEach } from 'vitest';
import { useCartStore } from '@/stores/cartStore';
import type { Product } from '@/types';

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
  fit: '',
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
