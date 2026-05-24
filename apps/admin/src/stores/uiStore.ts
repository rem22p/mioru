import { create } from 'zustand';

const MIN = 11;
const MAX = 18;

function getInitialScale(): number {
  return parseInt(localStorage.getItem('ui_scale') || '13');
}

function applyScale(value: number) {
  document.documentElement.style.setProperty('--ui', value + 'px');
  localStorage.setItem('ui_scale', String(value));
}

interface UiStore {
  scale: number;
  sidebarCollapsed: boolean;
  setScale: (scale: number) => void;
  toggleSidebar: () => void;
}

export const useUiStore = create<UiStore>((set) => ({
  scale: getInitialScale(),
  sidebarCollapsed: false,
  setScale: (scale) => {
    const clamped = Math.min(MAX, Math.max(MIN, scale));
    applyScale(clamped);
    set({ scale: clamped });
  },
  toggleSidebar: () => set((state) => ({ sidebarCollapsed: !state.sidebarCollapsed })),
}));
