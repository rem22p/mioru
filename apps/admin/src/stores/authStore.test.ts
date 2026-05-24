import { describe, it, expect, beforeEach } from 'vitest';
import { useAuthStore } from './authStore';

describe('authStore', () => {
  beforeEach(() => {
    localStorage.clear();
    useAuthStore.setState({
      user: null,
      isAuthenticated: false,
      isLoading: false,
      error: null,
    });
  });

  it('login sets token and isAuthenticated', () => {
    const { login } = useAuthStore.getState();
    login('my-token');
    const state = useAuthStore.getState();
    expect(state.isAuthenticated).toBe(true);
    expect(localStorage.getItem('token')).toBe('my-token');
  });

  it('logout clears token and resets state', () => {
    const { login, logout } = useAuthStore.getState();
    login('my-token');
    logout();
    const state = useAuthStore.getState();
    expect(state.isAuthenticated).toBe(false);
    expect(state.user).toBeNull();
    expect(localStorage.getItem('token')).toBeNull();
  });

  it('setUser updates user and authentication state', () => {
    const { setUser } = useAuthStore.getState();
    const mockUser = {
      id: '1',
      username: 'test',
      first_name: 'Test',
      last_name: 'User',
      email: 'test@test.com',
      display_name: 'Test User',
      avatar_color: '#ff0000',
    };
    setUser(mockUser);
    const state = useAuthStore.getState();
    expect(state.user).toEqual(mockUser);
    expect(state.isAuthenticated).toBe(true);
  });

  it('setUser with null sets isAuthenticated to false', () => {
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
