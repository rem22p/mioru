import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen } from '@testing-library/react';
import { BrowserRouter } from 'react-router-dom';
import { useAuthStore } from '@/stores/authStore';
import Sidebar from './Sidebar';

const mockNav = vi.fn();
vi.mock('react-router-dom', async () => {
  const actual = await vi.importActual('react-router-dom');
  return { ...actual, useNavigate: () => mockNav };
});

describe('Sidebar', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    useAuthStore.setState({
      user: {
        id: '1',
        username: 'testuser',
        first_name: 'Test',
        last_name: 'User',
        email: 'test@test.com',
        display_name: 'Test User',
        avatar_color: '#f85149',
      },
      isAuthenticated: true,
      isLoading: false,
      error: null,
    });
  });

  it('renders workspace navigation items', () => {
    render(
      <BrowserRouter>
        <Sidebar collapsed={false} onToggle={vi.fn()} />
      </BrowserRouter>
    );
    expect(screen.getByText('Товары')).toBeDefined();
    expect(screen.getByText('Бухгалтерия')).toBeDefined();
  });

  it('renders user info', () => {
    render(
      <BrowserRouter>
        <Sidebar collapsed={false} onToggle={vi.fn()} />
      </BrowserRouter>
    );
    expect(screen.getByText('Test User')).toBeDefined();
  });

  it('hides labels when collapsed', () => {
    render(
      <BrowserRouter>
        <Sidebar collapsed={true} onToggle={vi.fn()} />
      </BrowserRouter>
    );
    expect(screen.queryByText('Товары')).toBeNull();
  });
});
