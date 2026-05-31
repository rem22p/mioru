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
  status: 'active',
  stock_quantity: 20,
  category_id: 12,
  category_name: 'Кроссовки',
  images: [],
  sizes: ['42', '43', '44'],
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

const mockProduct2: Product = {
  ...mockProduct,
  id: 2,
  name: 'Test Boot',
  slug: 'test-boot',
  price: 8000,
};

describe('cartStore', () => {
  beforeEach(() => {
    useCartStore.setState({ items: [] });
  });

  it('adds item to cart', () => {
    useCartStore.getState().addItem(mockProduct, '42');
    const items = useCartStore.getState().items;
    expect(items).toHaveLength(1);
    expect(items[0].product.id).toBe(mockProduct.id);
    expect(items[0].size).toBe('42');
    expect(items[0].quantity).toBe(1);
  });

  it('increments quantity when adding same item and size', () => {
    useCartStore.getState().addItem(mockProduct, '42');
    useCartStore.getState().addItem(mockProduct, '42');
    const items = useCartStore.getState().items;
    expect(items).toHaveLength(1);
    expect(items[0].quantity).toBe(2);
  });

  it('adds item with custom quantity', () => {
    useCartStore.getState().addItem(mockProduct, '42', 5);
    const items = useCartStore.getState().items;
    expect(items).toHaveLength(1);
    expect(items[0].quantity).toBe(5);
  });

  it('adds separate item for different size', () => {
    useCartStore.getState().addItem(mockProduct, '42');
    useCartStore.getState().addItem(mockProduct, '43');
    const items = useCartStore.getState().items;
    expect(items).toHaveLength(2);
  });

  it('removes item from cart', () => {
    useCartStore.getState().addItem(mockProduct, '42');
    useCartStore.getState().removeItem(mockProduct.id, '42');
    expect(useCartStore.getState().items).toHaveLength(0);
  });

  it('updates quantity', () => {
    useCartStore.getState().addItem(mockProduct, '42');
    useCartStore.getState().updateQuantity(mockProduct.id, '42', 5);
    expect(useCartStore.getState().items[0].quantity).toBe(5);
  });

  it('removes item when quantity set to 0', () => {
    useCartStore.getState().addItem(mockProduct, '42');
    useCartStore.getState().updateQuantity(mockProduct.id, '42', 0);
    expect(useCartStore.getState().items).toHaveLength(0);
  });

  it('calculates total items correctly', () => {
    useCartStore.getState().addItem(mockProduct, '42');
    useCartStore.getState().addItem(mockProduct, '42');
    useCartStore.getState().addItem(mockProduct2, '40');
    expect(useCartStore.getState().totalItems()).toBe(3);
  });

  it('calculates total price correctly', () => {
    useCartStore.getState().addItem(mockProduct, '42');
    useCartStore.getState().addItem(mockProduct, '42');
    expect(useCartStore.getState().totalPrice()).toBe(mockProduct.price * 2);
  });

  it('clears cart', () => {
    useCartStore.getState().addItem(mockProduct, '42');
    useCartStore.getState().clearCart();
    expect(useCartStore.getState().items).toHaveLength(0);
  });
});
