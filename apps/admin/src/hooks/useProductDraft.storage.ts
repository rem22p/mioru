/**
 * Pure localStorage helpers for product-form drafts.
 *
 * Split out from `useProductDraft.ts` so the storage logic can be unit-tested
 * without rendering React — `@testing-library/react` v16 on React 19 has a
 * long-standing `React.act is not a function` crash that surfaces the moment
 * `renderHook` is invoked (see `test/setup.ts` for the partial mitigation).
 *
 * Contract:
 * - One slot per logical form: `"new"` for create, `"edit:<slug>"` for edit.
 * - `File` objects are explicitly excluded from `DraftPayload` so callers can't
 *   accidentally try to persist them (structured-clone would silently drop them
 *   anyway, but a typed refusal surfaces bugs earlier).
 * - All functions swallow malformed JSON and `QuotaExceededError` and log to
 *   `console.error` — draft persistence must never break the form.
 */

export const STORAGE_PREFIX = "mioru-admin-product-draft:";

export interface DraftSizeEntry {
  label: string;
  stock: number;
}

export interface DraftSizeChartEntry {
  label: string;
  chest?: string;
  waist?: string;
  hips?: string;
  length?: string;
  foot_length?: string;
  wrist?: string;
  [k: string]: string | undefined;
}

export interface DraftImageRef {
  id: string;
  url: string;
}

export interface DraftPayload {
  name: string;
  slug: string;
  description: string;
  brand: string;
  price: string;
  xpReward: string;
  inStock: boolean;
  status: string;
  stockQuantity: string;
  selectedCategoryId: number | "";
  color: string;
  model: string;
  fit: string;
  material: string;
  selectedSizes: DraftSizeEntry[];
  sizeChart: DraftSizeChartEntry[];
  careInstructions: string[];
  /** Only `id` + `url` — never a live `File` object. */
  images: DraftImageRef[];
}

export interface StoredDraft {
  data: DraftPayload;
  /** ISO 8601 timestamp */
  savedAt: string;
}

export function buildStorageKey(slot: "new" | string): string {
  return `${STORAGE_PREFIX}${slot}`;
}

export function isStoredDraft(value: unknown): value is StoredDraft {
  if (typeof value !== "object" || value === null) return false;
  const v = value as Partial<StoredDraft>;
  return (
    typeof v.savedAt === "string" &&
    typeof v.data === "object" &&
    v.data !== null
  );
}

export function readStoredDraft(slot: "new" | string): StoredDraft | null {
  try {
    const raw = localStorage.getItem(buildStorageKey(slot));
    if (!raw) return null;
    const parsed: unknown = JSON.parse(raw);
    return isStoredDraft(parsed) ? parsed : null;
  } catch (err) {
    console.error(`readStoredDraft: ${buildStorageKey(slot)} unreadable`, err);
    return null;
  }
}

export function writeStoredDraft(slot: "new" | string, payload: DraftPayload): StoredDraft {
  const stored: StoredDraft = {
    data: payload,
    savedAt: new Date().toISOString(),
  };
  try {
    localStorage.setItem(buildStorageKey(slot), JSON.stringify(stored));
  } catch (err) {
    console.error(`writeStoredDraft: ${buildStorageKey(slot)} unwritable`, err);
  }
  return stored;
}

export function clearStoredDraft(slot: "new" | string): void {
  try {
    localStorage.removeItem(buildStorageKey(slot));
  } catch (err) {
    console.error(`clearStoredDraft: ${buildStorageKey(slot)} unremovable`, err);
  }
}

/**
 * The text shown in `window.confirm` when restoring a draft. The date is shown
 * in a compact `ru-RU` form ("25.06.2026, 14:30") so the dialog fits on one
 * line on mobile widths.
 */
export function restorePromptText(savedAt: string): string {
  const saved = new Date(savedAt).toLocaleString("ru-RU", {
    day: "2-digit",
    month: "2-digit",
    year: "numeric",
    hour: "2-digit",
    minute: "2-digit",
  });
  return `У вас есть несохранённый черновик товара от ${saved}. Восстановить его?`;
}
