import { create } from "zustand";
import type { User, AvatarParams } from "@/types";
import {
  fetchStoreLogin,
  fetchStoreRegister,
  fetchStoreLogout,
  fetchStoreCustomerMe,
  type CustomerProfile,
  type CustomerRegisterData,
} from "@/lib/api";

interface AuthStore {
  user: User | null;
  isAuthenticated: boolean;
  loading: boolean;
  error: string | null;
  login: (email: string, password: string) => Promise<void>;
  register: (data: CustomerRegisterData) => Promise<void>;
  logout: () => Promise<void>;
  fetchMe: () => Promise<void>;
  updateXp: (amount: number) => void;
  clearError: () => void;
}

function toUser(c: CustomerProfile): User {
  return {
    id: String(c.id),
    name: [c.first_name, c.last_name].filter(Boolean).join(" ") || c.email,
    email: c.email,
    avatarParams: {} as AvatarParams,
    xpBalance: 0,
    vipLevel: 0,
  };
}

// Push local stores to server, then pull server state back.
// Merges anonymous data with account data on login.
async function syncOnAuth() {
  try {
    const { pushCartToServer, hydrateCartFromServer } = await import(
      "@/stores/cartStore"
    );
    const { pushFavoritesToServer, hydrateFavoritesFromServer } = await import(
      "@/stores/favoritesStore"
    );

    // Push anonymous items to server first.
    await pushCartToServer();
    await pushFavoritesToServer();

    // Then hydrate from server (cross-device merge).
    await hydrateCartFromServer();
    await hydrateFavoritesFromServer();
  } catch {
    // best-effort
  }
}

async function clearLocalStores() {
  try {
    const { useCartStore } = await import("@/stores/cartStore");
    const { useFavoritesStore } = await import("@/stores/favoritesStore");
    useCartStore.getState().clearCart();
    useFavoritesStore.getState().clearAll();
  } catch {
    // best effort
  }
}

export const useAuthStore = create<AuthStore>()((set, get) => ({
  user: null,
  isAuthenticated: false,
  loading: true,
  error: null,

  login: async (email, password) => {
    set({ loading: true, error: null });
    try {
      const customer = await fetchStoreLogin({ email, password });
      set({ user: toUser(customer), isAuthenticated: true, loading: false });
      await syncOnAuth();
    } catch (err) {
      const msg = err instanceof Error ? err.message : "Login failed";
      set({ error: msg, loading: false });
      throw err;
    }
  },

  register: async (data) => {
    set({ loading: true, error: null });
    try {
      const customer = await fetchStoreRegister(data);
      set({ user: toUser(customer), isAuthenticated: true, loading: false });
      await syncOnAuth();
    } catch (err) {
      const msg = err instanceof Error ? err.message : "Registration failed";
      set({ error: msg, loading: false });
      throw err;
    }
  },

  logout: async () => {
    try {
      await fetchStoreLogout();
    } catch {
      // ignore network errors during logout
    }
    await clearLocalStores();
    set({ user: null, isAuthenticated: false, loading: false, error: null });
  },

  fetchMe: async () => {
    set({ loading: true });
    try {
      const customer = await fetchStoreCustomerMe();
      set({ user: toUser(customer), isAuthenticated: true, loading: false });
      // Hydrate from server on app init with existing session.
      const { hydrateCartFromServer } = await import("@/stores/cartStore");
      const { hydrateFavoritesFromServer } = await import(
        "@/stores/favoritesStore"
      );
      await hydrateCartFromServer();
      await hydrateFavoritesFromServer();
    } catch {
      set({ user: null, isAuthenticated: false, loading: false });
    }
  },

  updateXp: (amount) => {
    const user = get().user;
    if (user) {
      set({ user: { ...user, xpBalance: user.xpBalance + amount } });
    }
  },

  clearError: () => set({ error: null }),
}));
