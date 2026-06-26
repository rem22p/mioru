import { create } from "zustand";
import { persist } from "zustand/middleware";
import type { CartItem, Product } from "@/types";
import { saveCustomerCart, type CartSyncItem } from "@/lib/api";

interface CartStore {
  items: CartItem[];
  addItem: (product: Product, size: string, qty?: number, height?: number, weight?: number) => void;
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

function itemToSyncPayload(i: CartItem): CartSyncItem {
  return {
    product_id: i.product.id,
    size_label: i.size,
    quantity: i.quantity,
    ...(i.height != null ? { height_cm: i.height } : {}),
    ...(i.weight != null ? { weight_kg: i.weight } : {}),
  };
}

// Debounced sync to server when authenticated.
let syncTimer: ReturnType<typeof setTimeout> | null = null;
function scheduleSyncToServer(items: CartItem[]) {
  if (syncTimer) clearTimeout(syncTimer);
  syncTimer = setTimeout(async () => {
    const { useAuthStore } = await import("@/stores/authStore");
    if (!useAuthStore.getState().isAuthenticated) return;
    const payload: CartSyncItem[] = items.map(itemToSyncPayload);
    saveCustomerCart(payload).catch(() => {});
  }, 500);
}

export const useCartStore = create<CartStore>()(
  persist(
    (set, get) => ({
      items: [],
      addItem: (product, size, qty = 1, height, weight) => {
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
          newItems = [...items, { product, size, quantity: qty, height, weight }];
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

// Push local cart to server (called by authStore on login).
export async function pushCartToServer() {
  const { useAuthStore } = await import("@/stores/authStore");
  if (!useAuthStore.getState().isAuthenticated) return;
  const items = useCartStore.getState().items;
  if (items.length === 0) return;
  const payload: CartSyncItem[] = items.map(itemToSyncPayload);
  try {
    await saveCustomerCart(payload);
  } catch {
    // best-effort: the local cart stays the source of truth
  }
}
