import { create } from "zustand";
import type { User } from "@/types";
import { me as apiMe, logout as apiLogout } from "@/lib/api";

// Cookie-only auth: the session lives in an HttpOnly cookie set by the
// backend on login. Because JS can't read that cookie, "are we logged in?"
// must be decided by asking the API (a 200 on /api/users/me means yes;
// anything else means no). We can't make that call synchronously, so
// isAuthenticated starts as null and resolves to a concrete boolean after
// fetchUser completes — the layout uses that null to render a brief
// neutral state instead of flashing the login screen on every reload.
interface AuthStore {
  user: User | null;
  isAuthenticated: boolean | null;
  isLoading: boolean;
  error: string | null;
  setUser: (user: User | null) => void;
  // login() is called AFTER a successful POST /api/auth/login — the cookies
  // are already set by the server response, the store just records the user
  // profile that came back in the JSON body.
  login: (user: User) => void;
  logout: () => Promise<void>;
  fetchUser: () => Promise<void>;
  clearError: () => void;
}

export const useAuthStore = create<AuthStore>((set) => ({
  user: null,
  isAuthenticated: null,
  isLoading: false,
  error: null,

  setUser: (user) => set({ user, isAuthenticated: !!user }),

  login: (user) => set({ user, isAuthenticated: true, error: null }),

  logout: async () => {
    // Best-effort cookie clear on the server. We zero local state even if
    // the call errors (e.g. network down) — the user pressed "sign out" and
    // the UI should reflect that immediately; any surviving cookie will be
    // refused next request anyway.
    try {
      await apiLogout();
    } catch {
      /* swallow — see comment above */
    }
    set({ user: null, isAuthenticated: false, error: null });
  },

  fetchUser: async () => {
    set({ isLoading: true, error: null });
    try {
      const user = await apiMe();
      set({ user, isAuthenticated: true, isLoading: false });
    } catch {
      // 401 (no/expired session) and any other failure both collapse to
      // "not logged in" — letting a transient error keep an isAuthenticated
      // = true stale state would route the user to admin pages that then
      // immediately fail.
      set({ user: null, isAuthenticated: false, isLoading: false });
    }
  },

  clearError: () => set({ error: null }),
}));
