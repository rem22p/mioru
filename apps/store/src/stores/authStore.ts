import { create } from 'zustand';
import type { User } from '@/types';

interface AuthStore {
  user: User | null;
  isAuthenticated: boolean;
  login: (user: User) => void;
  logout: () => void;
  updateXp: (amount: number) => void;
}

export const useAuthStore = create<AuthStore>((set) => ({
  user: null,
  isAuthenticated: false,
  login: (user) => set({ user, isAuthenticated: true }),
  logout: () => set({ user: null, isAuthenticated: false }),
  updateXp: (amount) =>
    set((state) => ({
      user: state.user
        ? { ...state.user, xpBalance: state.user.xpBalance + amount }
        : null,
    })),
}));
