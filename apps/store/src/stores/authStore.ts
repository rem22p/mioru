import { create } from 'zustand';
import type { User } from '@/types';
import {
  fetchStoreLogin,
  fetchStoreRegister,
  fetchStoreLogout,
  fetchStoreCustomerMe,
  type CustomerRegisterData,
  type CustomerLoginData,
} from '@/lib/api';

interface AuthStore {
  user: User | null;
  isAuthenticated: boolean;
  loading: boolean;
  checkAuth: () => Promise<void>;
  login: (email: string, password: string) => Promise<void>;
  register: (data: CustomerRegisterData) => Promise<void>;
  setUser: (user: User) => void;
  logout: () => Promise<void>;
  updateXp: (amount: number) => void;
}

export const useAuthStore = create<AuthStore>((set, get) => ({
  user: null,
  isAuthenticated: false,
  loading: true,

  checkAuth: async () => {
    try {
      const profile = await fetchStoreCustomerMe();
      set({
        user: {
          id: String(profile.id),
          name: [profile.first_name, profile.last_name].filter(Boolean).join(' ') || 'User',
          email: profile.email || null,
          avatarParams: { gender: 'male', height: 170, weight: 70, fatPercentage: 20, musclePercentage: 40 },
          xpBalance: 0,
          vipLevel: 0,
        },
        isAuthenticated: true,
        loading: false,
      });
    } catch {
      set({ user: null, isAuthenticated: false, loading: false });
    }
  },

  login: async (email: string, password: string) => {
    const profile = await fetchStoreLogin({ email, password });
    set({
      user: {
        id: String(profile.id),
        name: [profile.first_name, profile.last_name].filter(Boolean).join(' ') || 'User',
        email: profile.email || null,
        avatarParams: { gender: 'male', height: 170, weight: 70, fatPercentage: 20, musclePercentage: 40 },
        xpBalance: 0,
        vipLevel: 0,
      },
      isAuthenticated: true,
      loading: false,
    });
  },

  register: async (data: CustomerRegisterData) => {
    const profile = await fetchStoreRegister(data);
    set({
      user: {
        id: String(profile.id),
        name: [profile.first_name, profile.last_name].filter(Boolean).join(' ') || 'User',
        email: profile.email || null,
        avatarParams: { gender: 'male', height: 170, weight: 70, fatPercentage: 20, musclePercentage: 40 },
        xpBalance: 0,
        vipLevel: 0,
      },
      isAuthenticated: true,
      loading: false,
    });
  },

  setUser: (user) => set({ user, isAuthenticated: true, loading: false }),

  logout: async () => {
    try {
      await fetchStoreLogout();
    } catch {
      // cookie cleared even if request fails
    }
    set({ user: null, isAuthenticated: false, loading: false });
  },

  updateXp: (amount) =>
    set((state) => ({
      user: state.user
        ? { ...state.user, xpBalance: state.user.xpBalance + amount }
        : null,
    })),
}));
