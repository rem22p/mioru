import { describe, it, expect, beforeEach, vi, afterEach } from 'vitest';
import { useAuthStore } from './authStore';
import * as api from '@/lib/api';

const mockUser = {
  id: '1',
  username: 'test',
  first_name: 'Test',
  last_name: 'User',
  email: 'test@test.com',
  display_name: 'Test User',
  avatar_color: '#ff0000',
};

describe('authStore (cookie-only)', () => {
  beforeEach(() => {
    useAuthStore.setState({
      user: null,
      isAuthenticated: null,
      isLoading: false,
      error: null,
    });
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it('login(user) sets the profile and isAuthenticated; no token in localStorage', () => {
    // Sessions live in HttpOnly cookies — the store must never persist a
    // token to localStorage (closes the XSS-exfil path).
    const { login } = useAuthStore.getState();
    login(mockUser);

    const state = useAuthStore.getState();
    expect(state.user).toEqual(mockUser);
    expect(state.isAuthenticated).toBe(true);
    expect(localStorage.getItem('token')).toBeNull();
  });

  it('logout() calls the API and clears state even if the call fails', async () => {
    const spy = vi.spyOn(api, 'logout').mockRejectedValueOnce(new Error('net'));
    useAuthStore.setState({ user: mockUser, isAuthenticated: true });

    await useAuthStore.getState().logout();

    expect(spy).toHaveBeenCalled();
    const state = useAuthStore.getState();
    expect(state.user).toBeNull();
    expect(state.isAuthenticated).toBe(false);
  });

  it('fetchUser() success sets isAuthenticated true', async () => {
    vi.spyOn(api, 'me').mockResolvedValueOnce(mockUser);

    await useAuthStore.getState().fetchUser();

    const state = useAuthStore.getState();
    expect(state.user).toEqual(mockUser);
    expect(state.isAuthenticated).toBe(true);
  });

  it('fetchUser() failure (401) collapses to isAuthenticated false', async () => {
    // The /me endpoint failing is how the SPA learns "no valid session" in
    // cookie-only auth — JS can't peek at the HttpOnly cookie itself, so a
    // failed /me MUST land us as unauthenticated, not in a half-state.
    vi.spyOn(api, 'me').mockRejectedValueOnce(new Error('401'));

    await useAuthStore.getState().fetchUser();

    const state = useAuthStore.getState();
    expect(state.user).toBeNull();
    expect(state.isAuthenticated).toBe(false);
  });

  it('setUser updates user and authentication state', () => {
    const { setUser } = useAuthStore.getState();
    setUser(mockUser);
    const state = useAuthStore.getState();
    expect(state.user).toEqual(mockUser);
    expect(state.isAuthenticated).toBe(true);
  });

  it('setUser(null) sets isAuthenticated to false', () => {
    const { setUser } = useAuthStore.getState();
    setUser(null);
    expect(useAuthStore.getState().isAuthenticated).toBe(false);
  });

  it('clearError resets error to null', () => {
    useAuthStore.setState({ error: 'some error' });
    const { clearError } = useAuthStore.getState();
    clearError();
    expect(useAuthStore.getState().error).toBeNull();
  });
});
