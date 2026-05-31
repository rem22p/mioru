import { create } from "zustand";
import { persist } from "zustand/middleware";
import type { Product } from "@/types";
import { getImageUrl, saveCustomerFavorites, fetchCustomerFavorites } from "@/lib/api";

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

// Hydrate favorites from server (called by authStore after login/fetchMe).
export async function hydrateFavoritesFromServer() {
  try {
    const res = await fetchCustomerFavorites();
    const serverIds = new Set(res.product_ids);
    // Keep only local favorites that are also on server.
    const localItems = useFavoritesStore.getState().items;
    const merged = localItems.filter((i) => serverIds.has(i.id));
    useFavoritesStore.setState({ items: merged });
  } catch {
    // best-effort
  }
}

// Push local favorites to server (called by authStore on login).
export async function pushFavoritesToServer() {
  const { useAuthStore } = await import("@/stores/authStore");
  if (!useAuthStore.getState().isAuthenticated) return;
  const productIds = useFavoritesStore.getState().items.map((i) => i.id);
  if (productIds.length > 0) {
    saveCustomerFavorites(productIds).catch(() => {});
  }
}
