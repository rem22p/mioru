import { create } from "zustand";
import { persist } from "zustand/middleware";
import type { CartItem, Product } from "@/types";
import {
  saveCustomerCart,
  fetchCustomerCart,
  fetchStoreProduct,
  type CartSyncItem,
} from "@/lib/api";

interface CartStore {
  items: CartItem[];
  addItem: (product: Product, size: string, qty?: number, measurements?: Record<string, number>) => void;
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
    ...(i.measurements ? { measurements: i.measurements } : {}),
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
      addItem: (product, size, qty = 1, measurements) => {
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
          newItems = [...items, { product, size, quantity: qty, measurements }];
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

// Load cart from server on login if local cart is empty (cross-device scenario).
// Fetches full product data for each cart item by slug so the local CartItem
// can be fully hydrated. Called by authStore after login/register.
export async function loadCartFromServer() {
  const { useAuthStore } = await import("@/stores/authStore");
  if (!useAuthStore.getState().isAuthenticated) return;
  const store = useCartStore.getState();
  // Don't overwrite a non-empty local cart — local wins on conflict.
  if (store.items.length > 0) return;
  try {
    const res = await fetchCustomerCart();
    if (!res || !res.items || res.items.length === 0) return;
    // Fetch all products in parallel — avoid N+1 sequential delays.
    const settled = await Promise.allSettled(
      res.items
        .filter((ci) => ci.product_slug)
        .map(async (ci) => {
          const product = await fetchStoreProduct(ci.product_slug!);
          return { product, size: ci.size_label, quantity: ci.quantity, measurements: ci.measurements } as CartItem;
        }),
    );
    const newItems: CartItem[] = [];
    for (const r of settled) {
      if (r.status === "fulfilled") newItems.push(r.value);
      // Rejected → product may have been deleted, skip silently.
    }
    if (newItems.length > 0) {
      useCartStore.setState({ items: newItems });
    }
  } catch {
    // best-effort: server cart load can fail (network/server error)
  }
}
