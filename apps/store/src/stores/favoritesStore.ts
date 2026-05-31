import { create } from "zustand";
import { persist } from "zustand/middleware";
import type { Product } from "@/types";
import { getImageUrl } from "@/lib/api";

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
}

export const useFavoritesStore = create<FavoritesStore>()(
  persist(
    (set, get) => ({
      items: [],
      addFavorite: (product) => {
        if (get().items.some((i) => i.id === product.id)) return;
        set({
          items: [
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
          ],
        });
      },
      removeFavorite: (productId) => {
        set({ items: get().items.filter((i) => i.id !== productId) });
      },
      isFavorite: (productId) => get().items.some((i) => i.id === productId),
      toggleFavorite: (product) => {
        if (get().isFavorite(product.id)) {
          get().removeFavorite(product.id);
        } else {
          get().addFavorite(product);
        }
      },
    }),
    { name: "mioru-favorites" },
  ),
);
