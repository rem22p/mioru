import { create } from 'zustand';

type Theme = 'dark' | 'light';

function getInitialTheme(): Theme {
  const saved = localStorage.getItem('ui_theme');
  if (saved === 'light' || saved === 'dark') return saved;
  return 'dark';
}

function applyTheme(theme: Theme) {
  document.documentElement.classList.toggle('light', theme === 'light');
  localStorage.setItem('ui_theme', theme);
}

interface ThemeStore {
  theme: Theme;
  setTheme: (theme: Theme) => void;
  toggleTheme: () => void;
}

export const useThemeStore = create<ThemeStore>((set) => ({
  theme: getInitialTheme(),
  setTheme: (theme) => {
    applyTheme(theme);
    set({ theme });
  },
  toggleTheme: () => {
    set((state) => {
      const next = state.theme === 'dark' ? 'light' : 'dark';
      applyTheme(next);
      return { theme: next };
    });
  },
}));
