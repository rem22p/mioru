import { useEffect } from 'react';
import { useAuthStore } from '@/stores/authStore';

export function useAuth() {
  const { user, isAuthenticated, isLoading, fetchUser, logout } = useAuthStore();

  useEffect(() => {
    if (isAuthenticated && !user && !isLoading) {
      fetchUser();
    }
  }, [isAuthenticated, user, isLoading, fetchUser]);

  return { user, isAuthenticated, isLoading, logout };
}
