import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { BrowserRouter } from 'react-router-dom';
import Login from './Login';

const mockNav = vi.fn();
vi.mock('react-router-dom', async () => {
  const actual = await vi.importActual('react-router-dom');
  return { ...actual, useNavigate: () => mockNav };
});

describe('Login page', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    localStorage.clear();
  });

  it('renders login form with MIORU branding', () => {
    render(<BrowserRouter><Login /></BrowserRouter>);
    expect(screen.getByText('MIORU')).toBeDefined();
    expect(screen.getByPlaceholderText('Никнейм')).toBeDefined();
    expect(screen.getByPlaceholderText('Пароль')).toBeDefined();
  });

  it('shows error when submitting empty form', async () => {
    const user = userEvent.setup();
    render(<BrowserRouter><Login /></BrowserRouter>);
    await user.click(screen.getByRole('button', { name: /войти/i }));
    expect(screen.getByText('Заполните все поля')).toBeDefined();
  });

  it('has link to register page', () => {
    render(<BrowserRouter><Login /></BrowserRouter>);
    expect(screen.getByText('Регистрация')).toBeDefined();
  });

  it('has link to forgot password page', () => {
    render(<BrowserRouter><Login /></BrowserRouter>);
    expect(screen.getByText('Забыли пароль?')).toBeDefined();
  });

  it('has login button', () => {
    render(<BrowserRouter><Login /></BrowserRouter>);
    expect(screen.getByRole('button', { name: /войти/i })).toBeDefined();
  });
});
