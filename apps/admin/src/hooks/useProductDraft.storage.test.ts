/**
 * Tests for the pure localStorage helpers backing `useProductDraft`.
 *
 * The hook itself is exercised through `ProductForm` integration (manual +
 * future component tests). These helpers have all the non-trivial logic —
 * schema validation, error swallowing, debounce targets — and can be tested
 * without React.
 */
import { beforeEach, describe, expect, it, vi } from "vitest";
import {
  STORAGE_PREFIX,
  buildStorageKey,
  readStoredDraft,
  writeStoredDraft,
  clearStoredDraft,
  restorePromptText,
  isStoredDraft,
  type DraftPayload,
} from "./useProductDraft.storage";

const samplePayload = (over: Partial<DraftPayload> = {}): DraftPayload => ({
  name: "Test Shirt",
  slug: "test-shirt",
  description: "A fine shirt.",
  brand: "Acme",
  price: "199",
  xpReward: "5",
  inStock: true,
  status: "in_stock",
  selectedCategoryId: 1,
  color: "red",
  material: "cotton",
  selectedSizes: [{ label: "M", stock: 5 }],
  sizeChart: [{ label: "M", chest: "96", waist: "76", hips: "100", length: "72", foot_length: "", wrist: "" }],
  careInstructions: ["Wash cold"],
  images: [{ id: "img-1", url: "/uploads/shirt.png" }],
  ...over,
});

beforeEach(() => {
  localStorage.clear();
});

describe("buildStorageKey", () => {
  it("uses the new-product slot for create", () => {
    expect(buildStorageKey("new")).toBe(`${STORAGE_PREFIX}new`);
  });

  it("uses the edit:<slug> slot for editing an existing product", () => {
    expect(buildStorageKey("edit:test-shirt")).toBe(
      `${STORAGE_PREFIX}edit:test-shirt`,
    );
  });

  it("keeps 'new' and 'edit:<slug>' from colliding", () => {
    expect(buildStorageKey("new")).not.toBe(buildStorageKey("edit:new"));
  });
});

describe("isStoredDraft", () => {
  it("accepts a structurally valid payload", () => {
    expect(isStoredDraft({ savedAt: "2026-06-25T14:30:00Z", data: samplePayload() })).toBe(true);
  });

  it.each([
    ["null", null],
    ["a primitive", 42],
    ["an empty object", {}],
    ["missing savedAt", { data: samplePayload() }],
    ["missing data", { savedAt: "x" }],
    ["non-string savedAt", { savedAt: 42, data: samplePayload() }],
  ])("rejects %s", (_label, value) => {
    expect(isStoredDraft(value)).toBe(false);
  });
});

describe("readStoredDraft", () => {
  it("returns null when nothing is stored at the slot", () => {
    expect(readStoredDraft("new")).toBeNull();
  });

  it("returns the stored draft when present", () => {
    writeStoredDraft("new", samplePayload({ name: "FromDisk" }));
    const result = readStoredDraft("new");
    expect(result?.data.name).toBe("FromDisk");
    expect(typeof result?.savedAt).toBe("string");
  });

  it("returns null and logs on malformed JSON", () => {
    localStorage.setItem(`${STORAGE_PREFIX}new`, "{not-json");
    const spy = vi.spyOn(console, "error").mockImplementation(() => {});
    expect(readStoredDraft("new")).toBeNull();
    expect(spy).toHaveBeenCalled();
    spy.mockRestore();
  });

  it("returns null when the stored shape fails isStoredDraft", () => {
    localStorage.setItem(`${STORAGE_PREFIX}new`, JSON.stringify({ nope: true }));
    expect(readStoredDraft("new")).toBeNull();
  });

  it("does not read across slots", () => {
    writeStoredDraft("edit:test-shirt", samplePayload({ name: "Edited" }));
    expect(readStoredDraft("new")).toBeNull();
    expect(readStoredDraft("edit:test-shirt")?.data.name).toBe("Edited");
  });
});

describe("writeStoredDraft", () => {
  it("persists the payload to the slot with an ISO savedAt", () => {
    const stored = writeStoredDraft("new", samplePayload());
    expect(stored.savedAt).toMatch(/^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}/);
    const raw = localStorage.getItem(`${STORAGE_PREFIX}new`);
    expect(raw).not.toBeNull();
    expect(JSON.parse(raw!).savedAt).toBe(stored.savedAt);
  });

  it("logs and resolves when localStorage.setItem throws (e.g. quota)", () => {
    // The test setup defines window.localStorage as a local mock object, so
    // `Storage.prototype` is not the prototype chain — spy on the mock
    // directly instead.
    const setItemSpy = vi.spyOn(window.localStorage, "setItem").mockImplementation(() => {
      throw new Error("QuotaExceededError");
    });
    const errSpy = vi.spyOn(console, "error").mockImplementation(() => {});
    // Must not throw — the contract is "never break the form".
    expect(() => writeStoredDraft("new", samplePayload())).not.toThrow();
    expect(setItemSpy).toHaveBeenCalled();
    expect(errSpy).toHaveBeenCalled();
    setItemSpy.mockRestore();
    errSpy.mockRestore();
  });
});

describe("clearStoredDraft", () => {
  it("removes the entry from localStorage", () => {
    writeStoredDraft("new", samplePayload());
    expect(localStorage.getItem(`${STORAGE_PREFIX}new`)).not.toBeNull();
    clearStoredDraft("new");
    expect(localStorage.getItem(`${STORAGE_PREFIX}new`)).toBeNull();
  });

  it("does not touch other slots", () => {
    writeStoredDraft("new", samplePayload());
    writeStoredDraft("edit:test-shirt", samplePayload());
    clearStoredDraft("new");
    expect(readStoredDraft("new")).toBeNull();
    expect(readStoredDraft("edit:test-shirt")).not.toBeNull();
  });

  it("logs and swallows on removeItem failure", () => {
    const rmSpy = vi.spyOn(window.localStorage, "removeItem").mockImplementation(() => {
      throw new Error("blocked");
    });
    const errSpy = vi.spyOn(console, "error").mockImplementation(() => {});
    expect(() => clearStoredDraft("new")).not.toThrow();
    expect(rmSpy).toHaveBeenCalled();
    expect(errSpy).toHaveBeenCalled();
    rmSpy.mockRestore();
    errSpy.mockRestore();
  });
});

describe("restorePromptText", () => {
  it("renders a compact ru-RU timestamp", () => {
    const text = restorePromptText("2026-06-25T14:30:00Z");
    expect(text).toContain("несохранённый черновик товара от");
    // Compact dd.mm.yyyy, hh:mm — not parsed in test (locale-dependent) but
    // we can assert the literal Cyrillic prefix is present.
    expect(text.length).toBeGreaterThan(40);
  });
});
