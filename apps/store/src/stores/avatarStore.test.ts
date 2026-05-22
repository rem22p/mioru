import { describe, it, expect, beforeEach } from 'vitest';
import { useAvatarStore } from '@/stores/avatarStore';

describe('avatarStore', () => {
  beforeEach(() => {
    useAvatarStore.setState({
      params: { gender: 'male', height: 175, weight: 70, fatPercentage: 20, musclePercentage: 30 },
      currentPose: 'idle',
    });
  });

  it('has default params', () => {
    const state = useAvatarStore.getState();
    expect(state.params.height).toBe(175);
    expect(state.params.gender).toBe('male');
    expect(state.currentPose).toBe('idle');
  });

  it('updates params partially', () => {
    useAvatarStore.getState().setParams({ height: 190 });
    const params = useAvatarStore.getState().params;
    expect(params.height).toBe(190);
    expect(params.weight).toBe(70); // unchanged
  });

  it('changes pose', () => {
    useAvatarStore.getState().setPose('tpose');
    expect(useAvatarStore.getState().currentPose).toBe('tpose');
  });
});
