import { describe, it, expect, beforeEach } from 'vitest';
import { useThemeStore } from './themeStore';

describe('themeStore', () => {
  beforeEach(() => {
    localStorage.clear();
    document.documentElement.classList.remove('light');
    useThemeStore.setState({ theme: 'dark' });
  });

  it('initializes with dark theme by default', () => {
    const state = useThemeStore.getState();
    expect(state.theme).toBe('dark');
  });

  it('setTheme changes theme and updates DOM', () => {
    const { setTheme } = useThemeStore.getState();
    setTheme('light');
    expect(useThemeStore.getState().theme).toBe('light');
    expect(document.documentElement.classList.contains('light')).toBe(true);
    expect(localStorage.getItem('ui_theme')).toBe('light');
  });

  it('setTheme to dark removes light class', () => {
    const { setTheme } = useThemeStore.getState();
    setTheme('light');
    setTheme('dark');
    expect(useThemeStore.getState().theme).toBe('dark');
    expect(document.documentElement.classList.contains('light')).toBe(false);
  });

  it('toggleTheme switches between dark and light', () => {
    const { toggleTheme } = useThemeStore.getState();
    toggleTheme();
    expect(useThemeStore.getState().theme).toBe('light');
    toggleTheme();
    expect(useThemeStore.getState().theme).toBe('dark');
  });
});
