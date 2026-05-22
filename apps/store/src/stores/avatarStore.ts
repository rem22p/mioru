import { create } from 'zustand';
import { persist } from 'zustand/middleware';
import type { AvatarParams } from '@/types';

interface AvatarStore {
  params: AvatarParams;
  currentPose: string;
  setParams: (params: Partial<AvatarParams>) => void;
  setPose: (pose: string) => void;
}

const defaultParams: AvatarParams = {
  gender: 'male',
  height: 175,
  weight: 70,
  fatPercentage: 20,
  musclePercentage: 30,
};

export const useAvatarStore = create<AvatarStore>()(
  persist(
    (set) => ({
      params: defaultParams,
      currentPose: 'idle',
      setParams: (newParams) =>
        set((state) => ({ params: { ...state.params, ...newParams } })),
      setPose: (pose) => set({ currentPose: pose }),
    }),
    {
      name: 'mioru-avatar',
    }
  )
);
