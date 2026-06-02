import { create } from "zustand";
import { persist } from "zustand/middleware";
import type { Product } from "@/types";
import { getImageUrl, saveCustomerFavorites } from "@/lib/api";

export interface FavoriteItem {
  id: number;
  slug: string;
  name: string;
  price: number;
  category_name: string;
  imageUrl: string | null;
}

interface FavoritesStore {
  items: FavoriteItem[];
  addFavorite: (product: Product) => void;
  removeFavorite: (productId: number) => void;
  isFavorite: (productId: number) => boolean;
  toggleFavorite: (product: Product) => void;
  clearAll: () => void;
}

// Debounced sync to server when authenticated.
let syncTimer: ReturnType<typeof setTimeout> | null = null;
function scheduleSyncToServer(items: FavoriteItem[]) {
  if (syncTimer) clearTimeout(syncTimer);
  syncTimer = setTimeout(async () => {
    // Dynamic import to avoid circular deps at module level.
    const { useAuthStore } = await import("@/stores/authStore");
    if (!useAuthStore.getState().isAuthenticated) return;
    const productIds = items.map((i) => i.id);
    saveCustomerFavorites(productIds).catch(() => {});
  }, 500);
}

export const useFavoritesStore = create<FavoritesStore>()(
  persist(
    (set, get) => ({
      items: [],
      addFavorite: (product) => {
        if (get().items.some((i) => i.id === product.id)) return;
        const newItems = [
          ...get().items,
          {
            id: product.id,
            slug: product.slug,
            name: product.name,
            price: product.price,
            category_name: product.category_name,
            imageUrl: product.images?.[0]?.url
              ? getImageUrl(product.images[0].url)
              : null,
          },
        ];
        set({ items: newItems });
        scheduleSyncToServer(newItems);
      },
      removeFavorite: (productId) => {
        const newItems = get().items.filter((i) => i.id !== productId);
        set({ items: newItems });
        scheduleSyncToServer(newItems);
      },
      isFavorite: (productId) => get().items.some((i) => i.id === productId),
      toggleFavorite: (product) => {
        if (get().isFavorite(product.id)) {
          get().removeFavorite(product.id);
        } else {
          get().addFavorite(product);
        }
      },
      clearAll: () => {
        set({ items: [] });
      },
    }),
    { name: "mioru-favorites" },
  ),
);

// Push local favorites to server (called by authStore on login). Awaits the
// save for the same reason as the cart push.
//
// There is deliberately no hydrate-from-server counterpart: the server returns
// only product_ids (no product data), so it cannot reconstruct FavoriteItem
// rows on a fresh device. The earlier implementation replaced the list with the
// local∩server intersection, silently dropping favorites that hadn't synced
// yet. The local list (persisted and pushed here) is the source of truth.
// Cross-device restore needs the server to return product details; follow-up.
export async function pushFavoritesToServer() {
  const { useAuthStore } = await import("@/stores/authStore");
  if (!useAuthStore.getState().isAuthenticated) return;
  const productIds = useFavoritesStore.getState().items.map((i) => i.id);
  if (productIds.length === 0) return;
  try {
    await saveCustomerFavorites(productIds);
  } catch {
    // best-effort: the local list stays the source of truth
  }
}
