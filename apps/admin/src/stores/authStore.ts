import { create } from "zustand";
import type { User } from "@/types";
import { me as apiMe } from "@/lib/api";

interface AuthStore {
  user: User | null;
  isAuthenticated: boolean;
  isLoading: boolean;
  error: string | null;
  setUser: (user: User | null) => void;
  login: (token: string) => void;
  logout: () => void;
  fetchUser: () => Promise<void>;
  clearError: () => void;
}

export const useAuthStore = create<AuthStore>((set) => ({
  user: null,
  isAuthenticated: !!localStorage.getItem("token"),
  isLoading: false,
  error: null,

  setUser: (user) => set({ user, isAuthenticated: !!user }),

  login: (token) => {
    localStorage.setItem("token", token);
    set({ isAuthenticated: true, error: null });
  },

  logout: () => {
    localStorage.removeItem("token");
    set({ user: null, isAuthenticated: false, error: null });
  },

  fetchUser: async () => {
    set({ isLoading: true, error: null });
    try {
      const user = await apiMe();
      set({ user, isLoading: false });
    } catch {
      set({ isLoading: false, error: "Failed to fetch user" });
    }
  },

  clearError: () => set({ error: null }),
}));
