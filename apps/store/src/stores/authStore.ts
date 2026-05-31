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
    set({ user: null, isAuthenticated: false, loading: false, error: null });
  },

  fetchMe: async () => {
    set({ loading: true });
    try {
      const customer = await fetchStoreCustomerMe();
      set({ user: toUser(customer), isAuthenticated: true, loading: false });
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
