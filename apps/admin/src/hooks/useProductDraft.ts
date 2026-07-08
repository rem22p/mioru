/**
 * useProductDraft — autosave `ProductForm` fields to `localStorage` so an
 * accidental click-outside or Esc can't silently lose 20 minutes of typing.
 *
 * One draft per slot: the key is `"new"` for create and `"edit:<slug>"` for
 * editing an existing product, so editing product A never collides with
 * editing product B.
 *
 * Storage primitives live in `./useProductDraft.storage.ts` so they can be
 * unit-tested without React. This file is the thin React adapter: state +
 * debounce + lifecycle.
 */
import { useCallback, useEffect, useRef, useState } from "react";
import {
  type DraftPayload,
  type StoredDraft,
  clearStoredDraft,
  readStoredDraft,
  writeStoredDraft,
} from "./useProductDraft.storage";

export type {
  DraftPayload,
  StoredDraft,
  DraftImageRef,
  DraftSizeEntry,
  DraftSizeChartEntry,
} from "./useProductDraft.storage";

export interface UseProductDraftOptions {
  debounceMs?: number;
}

export interface UseProductDraftApi {
  draft: StoredDraft | null;
  /** True after the mount-time hydration read has finished. */
  hasHydrated: boolean;
  saveDraft: (payload: DraftPayload) => void;
  clearDraft: () => void;
}

export function useProductDraft(
  slot: "new" | string,
  options: UseProductDraftOptions = {},
): UseProductDraftApi {
  const { debounceMs = 500 } = options;

  const [draft, setDraft] = useState<StoredDraft | null>(null);
  const [hasHydrated, setHasHydrated] = useState(false);
  const timerRef = useRef<number | null>(null);

  // Hydrate from localStorage on mount / when the slot key changes.
  useEffect(() => {
    const initial = readStoredDraft(slot);
    if (initial) setDraft(initial);
    setHasHydrated(true);
  }, [slot]);

  // Cancel any pending debounced write when we unmount or the slot changes —
  // a write queued for the old slot must not land after we've switched.
  useEffect(() => {
    return () => {
      if (timerRef.current !== null) {
        window.clearTimeout(timerRef.current);
        timerRef.current = null;
      }
    };
  }, [slot]);

  const saveDraft = useCallback(
    (payload: DraftPayload) => {
      if (timerRef.current !== null) {
        window.clearTimeout(timerRef.current);
      }
      timerRef.current = window.setTimeout(() => {
        // Write to LS but DO NOT call setDraft — that would re-fire every
        // consumer of `draft` (e.g. the restore-prompt effect) on every
        // keystroke and turn the prompt into a recurring pop-up. The hook
        // only needs `draft` for the initial-mount hydration value; after
        // that it can stay stale because the in-memory form state already
        // IS the latest draft.
        writeStoredDraft(slot, payload);
      }, debounceMs);
    },
    [slot, debounceMs],
  );

  const clearDraft = useCallback(() => {
    if (timerRef.current !== null) {
      window.clearTimeout(timerRef.current);
      timerRef.current = null;
    }
    clearStoredDraft(slot);
    setDraft(null);
  }, [slot]);

  return { draft, hasHydrated, saveDraft, clearDraft };
}
