import { create } from "zustand";
import { persist } from "zustand/middleware";
import type { CartItem, Product } from "@/types";
import { saveCustomerCart, type CartSyncItem } from "@/lib/api";

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

// Push local cart to server (called by authStore on login). Awaits the save so
// callers can rely on the server reflecting the local cart once this resolves —
// the previous fire-and-forget version let later reads race ahead of the write.
//
// There is deliberately no hydrate-from-server counterpart: the server cart
// stores only {product_id, size_label, quantity} with no product data, so it
// cannot reconstruct cart items on a fresh device. The earlier implementation
// replaced the cart with the local∩server intersection, taking the server's
// quantity — which silently dropped items and changed the checkout total right
// after a forced login. The local cart (persisted to localStorage and pushed
// here) is the source of truth. True cross-device hydration needs the server to
// return product details for cart rows; tracked as a follow-up.
export async function pushCartToServer() {
  const { useAuthStore } = await import("@/stores/authStore");
  if (!useAuthStore.getState().isAuthenticated) return;
  const items = useCartStore.getState().items;
  if (items.length === 0) return;
  const payload: CartSyncItem[] = items.map((i) => ({
    product_id: i.product.id,
    size_label: i.size,
    quantity: i.quantity,
  }));
  try {
    await saveCustomerCart(payload);
  } catch {
    // best-effort: the local cart stays the source of truth
  }
}
