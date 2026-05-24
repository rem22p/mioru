import { describe, it, expect, beforeEach } from 'vitest';
import { useUiStore } from './uiStore';

describe('uiStore', () => {
  beforeEach(() => {
    localStorage.clear();
    useUiStore.setState({ scale: 13, sidebarCollapsed: false });
  });

  it('initializes with scale 13 by default', () => {
    const state = useUiStore.getState();
    expect(state.scale).toBe(13);
  });

  it('sidebarCollapsed is false by default', () => {
    expect(useUiStore.getState().sidebarCollapsed).toBe(false);
  });

  it('setScale clamps between 11 and 18', () => {
    const { setScale } = useUiStore.getState();
    setScale(5);
    expect(useUiStore.getState().scale).toBe(11);
    setScale(20);
    expect(useUiStore.getState().scale).toBe(18);
    setScale(14);
    expect(useUiStore.getState().scale).toBe(14);
  });

  it('setScale persists to localStorage', () => {
    const { setScale } = useUiStore.getState();
    setScale(16);
    expect(localStorage.getItem('ui_scale')).toBe('16');
  });

  it('toggleSidebar switches collapsed state', () => {
    const { toggleSidebar } = useUiStore.getState();
    toggleSidebar();
    expect(useUiStore.getState().sidebarCollapsed).toBe(true);
    toggleSidebar();
    expect(useUiStore.getState().sidebarCollapsed).toBe(false);
  });
});
