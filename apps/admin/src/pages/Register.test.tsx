import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen } from '@testing-library/react';
import { BrowserRouter } from 'react-router-dom';
import Register from './Register';

const mockNav = vi.fn();
vi.mock('react-router-dom', async () => {
  const actual = await vi.importActual('react-router-dom');
  return { ...actual, useNavigate: () => mockNav };
});

describe('Register page', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    localStorage.clear();
  });

  it('renders registration form with MIORU branding', () => {
    render(<BrowserRouter><Register /></BrowserRouter>);
    expect(screen.getByText('MIORU')).toBeDefined();
    expect(screen.getByPlaceholderText('Имя')).toBeDefined();
    expect(screen.getByPlaceholderText('Фамилия')).toBeDefined();
    expect(screen.getByPlaceholderText('Email')).toBeDefined();
    expect(screen.getByPlaceholderText('Никнейм')).toBeDefined();
  });

  it('has link to login page', () => {
    render(<BrowserRouter><Register /></BrowserRouter>);
    expect(screen.getByText('Войти')).toBeDefined();
  });

  it('shows error on empty submit', async () => {
    const { userEvent } = await import('@testing-library/user-event');
    const user = userEvent.setup();
    render(<BrowserRouter><Register /></BrowserRouter>);
    const button = screen.getByRole('button', { name: /зарегистрироваться/i });
    await user.click(button);
    expect(screen.getByText('Заполните все поля')).toBeDefined();
  });
});
