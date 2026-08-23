/**
 * Component tests for ProductForm's draft / close-guard / upload flows.
 *
 * These guard the behaviours reworked in review of PR #58:
 * - close-guard arms on real edits (not only after a restore), and stays
 *   silent for a pristine form;
 * - opening a form never persists a spurious draft (no phantom restore prompt);
 * - Esc closes the preview overlay before it closes the whole form;
 * - per-file upload validation surfaces rejects in `pf-image-errors`.
 *
 * The API layer, product store, and the (untouched) ProductPreview are mocked
 * so the test exercises ProductForm's own logic in isolation.
 */
import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import ProductForm from "./ProductForm";
import {
  STORAGE_PREFIX,
  type DraftPayload,
  type StoredDraft,
} from "@/hooks/useProductDraft.storage";

const uploadImage = vi.fn();
const createProduct = vi.fn().mockResolvedValue({});

vi.mock("@/lib/api", () => ({
  createProduct: (...args: unknown[]) => createProduct(...args),
  updateProduct: vi.fn().mockResolvedValue({}),
  uploadImage: (...args: unknown[]) => uploadImage(...args),
  getImageUrl: (u: string) => u,
}));

vi.mock("@/stores/productStore", () => ({
  useProductStore: () => ({
    categories: [
      {
        id: 1,
        parent_id: null,
        name: "Одежда",
        slug: "odezhda",
        criteria: ["size", "brand", "color"],
      },
    ],
    categoriesLoading: false,
  }),
}));

vi.mock("./ProductPreview", () => ({
  default: ({ onClose }: { onClose: () => void }) => (
    <div data-testid="pf-preview-stub">
      <button onClick={onClose}>close-preview</button>
    </div>
  ),
}));

const NEW_SLOT_KEY = `${STORAGE_PREFIX}new`;

const emptyPayload = (over: Partial<DraftPayload> = {}): DraftPayload => ({
  name: "",
  slug: "",
  description: "",
  brands: [],
  price: "",
  xpReward: "",
  inStock: true,
  status: "in_stock",
  selectedCategoryId: "",
  color: "",
  material: "",
  selectedSizes: [],
  sizeChart: [],
  careInstructions: [],
  images: [],
  ...over,
});

beforeEach(() => {
  localStorage.clear();
  vi.clearAllMocks();
});

describe("ProductForm — open / autosave", () => {
  it("opening a pristine new form persists no draft and shows no restore prompt", async () => {
    vi.useFakeTimers();
    try {
      render(<ProductForm product={null} onClose={vi.fn()} onSaved={vi.fn()} />);
      screen.getByTestId("pf-name");
      expect(screen.queryByTestId("pf-restore-dialog")).toBeNull();
      // Past the 500ms autosave debounce: a pristine open must not have queued
      // any write — otherwise the next open pops a phantom restore prompt.
      await vi.advanceTimersByTimeAsync(600);
      expect(localStorage.getItem(NEW_SLOT_KEY)).toBeNull();
    } finally {
      vi.useRealTimers();
    }
  });

  it("offers to restore a pre-existing draft and populates the form on confirm", async () => {
    const stored: StoredDraft = {
      data: emptyPayload({ name: "Restored Shirt", slug: "restored-shirt" }),
      savedAt: "2026-07-01T10:00:00.000Z",
    };
    localStorage.setItem(NEW_SLOT_KEY, JSON.stringify(stored));

    render(<ProductForm product={null} onClose={vi.fn()} onSaved={vi.fn()} />);
    await screen.findByTestId("pf-restore-dialog");
    await userEvent.click(screen.getByTestId("pf-restore-confirm"));

    expect(screen.getByTestId("pf-name")).toHaveValue("Restored Shirt");
  });
});

describe("ProductForm — close guard", () => {
  it("closes immediately when nothing was edited", async () => {
    const onClose = vi.fn();
    render(<ProductForm product={null} onClose={onClose} onSaved={vi.fn()} />);
    await screen.findByTestId("pf-name");

    await userEvent.click(screen.getByTestId("pf-close-x"));

    expect(onClose).toHaveBeenCalledTimes(1);
    expect(screen.queryByTestId("pf-close-dialog")).toBeNull();
  });

  it("prompts before closing once the form has been edited", async () => {
    const onClose = vi.fn();
    render(<ProductForm product={null} onClose={onClose} onSaved={vi.fn()} />);
    await screen.findByTestId("pf-name");

    await userEvent.type(screen.getByTestId("pf-name"), "Cool Shirt");
    await userEvent.click(screen.getByTestId("pf-close-x"));

    expect(await screen.findByTestId("pf-close-dialog")).toBeInTheDocument();
    expect(onClose).not.toHaveBeenCalled();

    await userEvent.click(screen.getByTestId("pf-close-confirm"));
    expect(onClose).toHaveBeenCalledTimes(1);
  });
});

describe("ProductForm — Esc handling", () => {
  it("Esc closes the preview overlay before it closes the form", async () => {
    const onClose = vi.fn();
    render(<ProductForm product={null} onClose={onClose} onSaved={vi.fn()} />);
    await screen.findByTestId("pf-name");

    await userEvent.click(screen.getByTestId("pf-preview-open"));
    expect(screen.getByTestId("pf-preview-stub")).toBeInTheDocument();

    // First Esc is consumed by the preview.
    await userEvent.keyboard("{Escape}");
    expect(screen.queryByTestId("pf-preview-stub")).toBeNull();
    expect(onClose).not.toHaveBeenCalled();

    // Second Esc, no overlay up and form pristine → closes the form.
    await userEvent.keyboard("{Escape}");
    expect(onClose).toHaveBeenCalledTimes(1);
  });
});

describe("ProductForm — image upload validation", () => {
  it("reports an unsupported file and does not upload it", async () => {
    render(<ProductForm product={null} onClose={vi.fn()} onSaved={vi.fn()} />);
    const input = await screen.findByTestId("pf-image-input");

    const bad = new File(["x"], "notes.txt", { type: "text/plain" });
    // fireEvent bypasses the input's `accept` filter that userEvent applies,
    // so the rejection path in handleImageUpload runs.
    fireEvent.change(input, { target: { files: [bad] } });

    const errors = await screen.findByTestId("pf-image-errors");
    expect(errors).toHaveTextContent("notes.txt");
    expect(uploadImage).not.toHaveBeenCalled();
  });
});

describe("ProductForm — brand editor (KAN-14)", () => {
  const renderWithCategory = async () => {
    render(<ProductForm product={null} onClose={vi.fn()} onSaved={vi.fn()} />);
    await screen.findByTestId("pf-name");
    // Select the mocked category whose criteria include "brand".
    fireEvent.change(screen.getByTestId("pf-category"), {
      target: { value: "1" },
    });
    return screen.getByTestId("brand-input");
  };

  it("adds brands as chips and shows the joined card name", async () => {
    const input = await renderWithCategory();
    await userEvent.type(input, "Bape{enter}");
    await userEvent.type(input, "Mastermind{enter}");

    expect(screen.getByTestId("brand-chip-Bape")).toBeInTheDocument();
    expect(screen.getByTestId("brand-chip-Mastermind")).toBeInTheDocument();
    expect(
      screen.getByText("В карточке: Bape x Mastermind"),
    ).toBeInTheDocument();
  });

  it("removes a brand chip via its × button", async () => {
    const input = await renderWithCategory();
    await userEvent.type(input, "Bape{enter}");
    await userEvent.click(screen.getByTestId("brand-remove-Bape"));
    expect(screen.queryByTestId("brand-chip-Bape")).toBeNull();
    expect(screen.queryByText("В карточке: Bape")).toBeNull();
  });

  it("dedupes brands entered twice", async () => {
    const input = await renderWithCategory();
    await userEvent.type(input, "Bape{enter}");
    await userEvent.type(input, "Bape{enter}");
    expect(screen.getAllByTestId("brand-chip-Bape")).toHaveLength(1);
  });

  it("turns a brand typed without Enter into a chip on blur", async () => {
    const input = await renderWithCategory();
    await userEvent.type(input, "Nike");
    fireEvent.blur(input);
    expect(screen.getByTestId("brand-chip-Nike")).toBeInTheDocument();
  });

  const fillRequired = async () => {
    await userEvent.type(screen.getByTestId("pf-name"), "Куртка");
    await userEvent.type(screen.getByTestId("pf-price"), "1200");
  };

  const submittedBrands = async () => {
    await userEvent.click(screen.getByTestId("pf-submit"));
    await waitFor(() => expect(createProduct).toHaveBeenCalled());
    const fd = createProduct.mock.calls[0][0] as FormData;
    return fd.getAll("brands[]");
  };

  it("sends the committed chips as brands[]", async () => {
    const input = await renderWithCategory();
    await fillRequired();
    await userEvent.type(input, "Bape{enter}");
    await userEvent.type(input, "Mastermind{enter}");

    expect(await submittedBrands()).toEqual(["Bape", "Mastermind"]);
  });

  it("does not drop a brand still sitting in the input on submit", async () => {
    const input = await renderWithCategory();
    await fillRequired();
    await userEvent.type(input, "Nike"); // no Enter, straight to "Save"

    expect(await submittedBrands()).toEqual(["Nike"]);
  });
});
