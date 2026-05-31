import { create } from "zustand";
import { persist } from "zustand/middleware";
import type { CartItem, Product } from "@/types";
import { saveCustomerCart, fetchCustomerCart, type CartSyncItem } from "@/lib/api";

interface CartStore {
  items: CartItem[];
  addItem: (product: Product, size: string, qty?: number) => void;
  removeItem: (productId: number | string, size: string) => void;
  updateQuantity: (
    productId: number | string,
    size: string,
    quantity: number,
  ) => void;
  clearCart: () => void;
  totalItems: () => number;
  totalPrice: () => number;
}

// Debounced sync to server when authenticated.
let syncTimer: ReturnType<typeof setTimeout> | null = null;
function scheduleSyncToServer(items: CartItem[]) {
  if (syncTimer) clearTimeout(syncTimer);
  syncTimer = setTimeout(async () => {
    const { useAuthStore } = await import("@/stores/authStore");
    if (!useAuthStore.getState().isAuthenticated) return;
    const payload: CartSyncItem[] = items.map((i) => ({
      product_id: i.product.id,
      size_label: i.size,
      quantity: i.quantity,
    }));
    saveCustomerCart(payload).catch(() => {});
  }, 500);
}

export const useCartStore = create<CartStore>()(
  persist(
    (set, get) => ({
      items: [],
      addItem: (product, size, qty = 1) => {
        const items = get().items;
        const existing = items.find(
          (item) => item.product.id === product.id && item.size === size,
        );
        let newItems: CartItem[];
        if (existing) {
          newItems = items.map((item) =>
            item.product.id === product.id && item.size === size
              ? { ...item, quantity: item.quantity + qty }
              : item,
          );
        } else {
          newItems = [...items, { product, size, quantity: qty }];
        }
        set({ items: newItems });
        scheduleSyncToServer(newItems);
      },
      removeItem: (productId, size) => {
        const newItems = get().items.filter(
          (item) => !(item.product.id === productId && item.size === size),
        );
        set({ items: newItems });
        scheduleSyncToServer(newItems);
      },
      updateQuantity: (productId, size, quantity) => {
        let newItems: CartItem[];
        if (quantity <= 0) {
          newItems = get().items.filter(
            (item) => !(item.product.id === productId && item.size === size),
          );
        } else {
          newItems = get().items.map((item) =>
            item.product.id === productId && item.size === size
              ? { ...item, quantity }
              : item,
          );
        }
        set({ items: newItems });
        scheduleSyncToServer(newItems);
      },
      clearCart: () => set({ items: [] }),
      totalItems: () =>
        get().items.reduce((sum, item) => sum + item.quantity, 0),
      totalPrice: () =>
        get().items.reduce(
          (sum, item) => sum + item.product.price * item.quantity,
          0,
        ),
    }),
    {
      name: "mioru-cart",
    },
  ),
);

// Hydrate cart from server (called by authStore after login/fetchMe).
// Preserves local items that also exist on server (matched by product_id + size).
export async function hydrateCartFromServer() {
  try {
    const res = await fetchCustomerCart();
    const localItems = useCartStore.getState().items;
    const merged = res.items
      .map((ci) => {
        const existing = localItems.find(
          (e) => e.product.id === ci.product_id && e.size === ci.size_label,
        );
        return existing
          ? { ...existing, quantity: ci.quantity }
          : null;
      })
      .filter(Boolean) as CartItem[];
    useCartStore.setState({ items: merged });
  } catch {
    // best-effort
  }
}

// Push local cart to server (called by authStore on login).
export async function pushCartToServer() {
  const { useAuthStore } = await import("@/stores/authStore");
  if (!useAuthStore.getState().isAuthenticated) return;
  const items = useCartStore.getState().items;
  if (items.length > 0) {
    const payload: CartSyncItem[] = items.map((i) => ({
      product_id: i.product.id,
      size_label: i.size,
      quantity: i.quantity,
    }));
    saveCustomerCart(payload).catch(() => {});
  }
}
