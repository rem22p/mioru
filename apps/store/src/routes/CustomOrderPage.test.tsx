import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, fireEvent, waitFor, act } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { HelmetProvider } from "@dr.pogodin/react-helmet";
import CustomOrderPage from "./CustomOrderPage";
import { useAuthStore } from "@/stores/authStore";
import { createOrder, uploadOrderPhoto } from "@/lib/api";
import type { User } from "@/types";
import "@testing-library/jest-dom/vitest";

// ── Mocks (non-store infra only) ──────────────────────────────────────────

vi.mock("framer-motion", () => ({
  motion: new Proxy(
    {},
    {
      get: (_t, tag: string) =>
        ({
          div: (p: React.HTMLAttributes<HTMLDivElement>) => <div {...p} />,
          form: (p: React.FormHTMLAttributes<HTMLFormElement>) => <form {...p} />,
        })[tag] || ((p: React.HTMLAttributes<HTMLElement>) => <div {...p} />),
    },
  ),
  AnimatePresence: ({ children }: { children: React.ReactNode }) => <>{children}</>,
}));

vi.mock("react-i18next", () => ({
  useTranslation: () => ({ t: (k: string) => k }),
}));

const mockNavigate = vi.fn();
vi.mock("react-router-dom", async (importOriginal) => {
  const actual = await importOriginal<typeof import("react-router-dom")>();
  return { ...actual, useNavigate: () => mockNavigate };
});

vi.mock("@/components/ui/CityAutocomplete", () => ({
  default: ({ value, onChange }: { value: string; onChange: (v: string) => void }) => (
    <input
      data-testid="city-autocomplete"
      value={value}
      onChange={(e) => onChange(e.target.value)}
    />
  ),
}));

vi.mock("@/lib/api", () => ({
  createOrder: vi.fn(),
  uploadOrderPhoto: vi.fn(),
}));

// ── Fixtures ──────────────────────────────────────────────────────────────

function makeUser(telegramLinked: boolean): User {
  return {
    id: "1",
    name: "E2E",
    email: "e2e@example.com",
    firstName: "E2E",
    lastName: "",
    phone: "+37360000000",
    avatarParams: {} as unknown as User["avatarParams"],
    xpBalance: 0,
    vipLevel: 0,
    telegram: telegramLinked
      ? { linked: true, username: "e2e_tg", firstName: "E2E" }
      : { linked: false },
  };
}

function renderPage(telegramLinked: boolean) {
  useAuthStore.setState({
    isAuthenticated: true,
    user: makeUser(telegramLinked),
    loading: false,
    error: null,
  });
  return render(
    <HelmetProvider>
      <MemoryRouter>
        <CustomOrderPage />
      </MemoryRouter>
    </HelmetProvider>,
  );
}

// Fills everything `canSubmit` requires, so the Telegram gate is the only
// remaining reason the button could stay disabled. KAN-52: clothing is
// selected so the height/weight fields exist.
function fillForm(container: HTMLElement) {
  const fileInput = container.querySelector('input[type="file"]') as HTMLInputElement;
  fireEvent.change(fileInput, {
    target: { files: [new File(["x"], "photo.png", { type: "image/png" })] },
  });
  fireEvent.click(screen.getByTestId("custom-order-category-clothing"));
  fireEvent.change(screen.getByPlaceholderText("175"), { target: { value: "175" } });
  fireEvent.change(screen.getByTestId("city-autocomplete"), { target: { value: "Тирасполь" } });
  fireEvent.change(screen.getByTestId("custom-order-phone"), {
    target: { value: "+37360000000" },
  });
  fireEvent.click(screen.getByTestId("delivery-personal"));
}

describe("CustomOrderPage — Telegram order gate", () => {
  beforeEach(() => {
    mockNavigate.mockClear();
    vi.mocked(createOrder).mockClear();
    vi.mocked(uploadOrderPhoto).mockClear();
    // jsdom has no object URLs; the photo preview calls it on every pick.
    globalThis.URL.createObjectURL = vi.fn(() => "blob:preview");
    useAuthStore.setState({ isAuthenticated: true, user: null, loading: false, error: null });
  });

  it("blocks submit + shows the banner when Telegram is not linked", () => {
    const { container } = renderPage(false);
    fillForm(container);

    expect(screen.getByTestId("custom-order-telegram-required")).toBeVisible();
    expect(screen.getByTestId("custom-order-submit")).toBeDisabled();

    fireEvent.click(screen.getByTestId("custom-order-submit"));
    // The 403 lands only after every photo is already in UPLOAD_DIR, so the
    // upload must not start at all while the gate is closed.
    expect(uploadOrderPhoto).not.toHaveBeenCalled();
    expect(createOrder).not.toHaveBeenCalled();
  });

  it("enables submit and hides the banner when Telegram is linked", () => {
    const { container } = renderPage(true);
    fillForm(container);

    expect(screen.queryByTestId("custom-order-telegram-required")).not.toBeInTheDocument();
    expect(screen.getByTestId("custom-order-submit")).toBeEnabled();
  });
});

describe("CustomOrderPage — delivery time removed (KAN-59 follow-up)", () => {
  beforeEach(() => {
    useAuthStore.setState({ isAuthenticated: true, user: makeUser(true), loading: false, error: null });
  });

  it("has no delivery-time radios — the section is gone entirely", () => {
    renderPage(true);
    const timeRadios = screen
      .getAllByRole("radio")
      .filter((r) => r.getAttribute("name") === "deliveryTime");
    expect(timeRadios.length).toBe(0);
  });

  it("keeps the delivery-method radios (personal/address/…)", () => {
    renderPage(true);
    const methodRadios = screen
      .getAllByRole("radio")
      .filter((r) => r.getAttribute("name") === "deliveryMethod");
    expect(methodRadios.length).toBeGreaterThan(0);
  });
});

describe("CustomOrderPage — guest access (KAN: form opens without login)", () => {
  beforeEach(() => {
    mockNavigate.mockClear();
    globalThis.URL.createObjectURL = vi.fn(() => "blob:preview");
    useAuthStore.setState({ isAuthenticated: false, user: null, loading: false, error: null });
  });

  it("guest can open the form — no redirect to /profile", () => {
    useAuthStore.setState({ isAuthenticated: false, user: null, loading: false, error: null });
    render(
      <HelmetProvider>
        <MemoryRouter>
          <CustomOrderPage />
        </MemoryRouter>
      </HelmetProvider>,
    );
    // The form is shown, not a login bounce
    expect(mockNavigate).not.toHaveBeenCalledWith(
      "/profile?redirect=/custom-order",
      expect.anything(),
    );
    expect(screen.getByTestId("custom-order-submit")).toBeInTheDocument();
  });

  it("guest cannot submit — button stays disabled until login + Telegram", () => {
    useAuthStore.setState({ isAuthenticated: false, user: null, loading: false, error: null });
    render(
      <HelmetProvider>
        <MemoryRouter>
          <CustomOrderPage />
        </MemoryRouter>
      </HelmetProvider>,
    );
    expect(screen.getByTestId("custom-order-submit")).toBeDisabled();
  });
});

describe("CustomOrderPage — legacy profile phone", () => {
  beforeEach(() => {
    mockNavigate.mockClear();
    globalThis.URL.createObjectURL = vi.fn(() => "blob:preview");
  });

  function seedWithPhone(phone: string) {
    useAuthStore.setState({
      isAuthenticated: true,
      user: { ...makeUser(true), phone },
      loading: false,
      error: null,
    });
    render(
      <HelmetProvider>
        <MemoryRouter>
          <CustomOrderPage />
        </MemoryRouter>
      </HelmetProvider>,
    );
  }

  it("offers the shortcut for a +373 profile phone and fills the field", () => {
    seedWithPhone("+37360000000");

    fireEvent.click(screen.getByTestId("custom-order-use-my-phone"));

    expect(screen.getByTestId("custom-order-phone")).toHaveValue("60000000");
  });

  it("hides the shortcut when the profile still holds a pre-KAN-53 number", () => {
    seedWithPhone("+79161234567");

    expect(screen.queryByTestId("custom-order-use-my-phone")).not.toBeInTheDocument();
  });
});

describe("CustomOrderPage — category choice (KAN-52)", () => {
  beforeEach(() => {
    mockNavigate.mockClear();
    vi.mocked(createOrder).mockClear();
    vi.mocked(uploadOrderPhoto).mockClear();
    globalThis.URL.createObjectURL = vi.fn(() => "blob:preview");
    useAuthStore.setState({ isAuthenticated: true, user: makeUser(true), loading: false, error: null });
  });

  const baseFields = (container: HTMLElement) => {
    const fileInput = container.querySelector('input[type="file"]') as HTMLInputElement;
    fireEvent.change(fileInput, {
      target: { files: [new File(["x"], "photo.png", { type: "image/png" })] },
    });
    fireEvent.change(screen.getByTestId("city-autocomplete"), { target: { value: "Тирасполь" } });
    fireEvent.change(screen.getByTestId("custom-order-phone"), {
      target: { value: "+37360000000" },
    });
    fireEvent.click(screen.getByTestId("delivery-personal"));
  };

  it("requires a category — submit stays disabled without one", () => {
    const { container } = renderPage(true);
    baseFields(container);
    expect(screen.getByTestId("custom-order-submit")).toBeDisabled();
  });

  it("shoes swap height/weight for the insole field", () => {
    const { container } = renderPage(true);
    baseFields(container);
    fireEvent.click(screen.getByTestId("custom-order-category-shoes"));

    expect(screen.getByTestId("custom-order-foot-length")).toBeInTheDocument();
    expect(screen.queryByPlaceholderText("175")).not.toBeInTheDocument();
  });

  it("accessories hide all measurement fields", () => {
    const { container } = renderPage(true);
    baseFields(container);
    fireEvent.click(screen.getByTestId("custom-order-category-accessories"));

    expect(screen.queryByPlaceholderText("175")).not.toBeInTheDocument();
    expect(screen.queryByTestId("custom-order-foot-length")).not.toBeInTheDocument();
  });

  it("sends category + foot_length for a shoes order and drops height/weight", async () => {
    vi.mocked(uploadOrderPhoto).mockResolvedValue("/uploads/p1.jpg");
    vi.mocked(createOrder).mockResolvedValue({ id: 1, status: "pending", created_at: "now" });

    const { container } = renderPage(true);
    baseFields(container);
    fireEvent.click(screen.getByTestId("custom-order-category-shoes"));
    fireEvent.change(screen.getByTestId("custom-order-foot-length"), {
      target: { value: "27.5" },
    });
    fireEvent.click(screen.getByTestId("custom-order-submit"));

    await waitFor(() => expect(createOrder).toHaveBeenCalled());
    const payload = vi.mocked(createOrder).mock.calls[0][0] as unknown as Record<string, unknown>;
    expect(payload.category).toBe("shoes");
    expect(payload.foot_length).toBe(27.5);
    expect(payload.height).toBeUndefined();
    expect(payload.weight).toBeUndefined();
  });
});

describe("CustomOrderPage — insole length input (KAN-52 F2/F3 review fixes)", () => {
  beforeEach(() => {
    mockNavigate.mockClear();
    vi.mocked(createOrder).mockClear();
    vi.mocked(uploadOrderPhoto).mockClear();
    globalThis.URL.createObjectURL = vi.fn(() => "blob:preview");
    useAuthStore.setState({ isAuthenticated: true, user: makeUser(true), loading: false, error: null });
  });

  const baseFields = (container: HTMLElement) => {
    const fileInput = container.querySelector('input[type="file"]') as HTMLInputElement;
    fireEvent.change(fileInput, {
      target: { files: [new File(["x"], "photo.png", { type: "image/png" })] },
    });
    fireEvent.change(screen.getByTestId("city-autocomplete"), { target: { value: "Тирасполь" } });
    fireEvent.change(screen.getByTestId("custom-order-phone"), {
      target: { value: "+37360000000" },
    });
    fireEvent.click(screen.getByTestId("delivery-personal"));
  };

  it("normalises the comma decimal separator (27,5 → 27.5)", () => {
    const { container } = renderPage(true);
    baseFields(container);
    fireEvent.click(screen.getByTestId("custom-order-category-shoes"));
    fireEvent.change(screen.getByTestId("custom-order-foot-length"), { target: { value: "27,5" } });
    expect((screen.getByTestId("custom-order-foot-length") as HTMLInputElement).value).toBe("27.5");
  });

  it("clamps out-of-range values on blur", async () => {
    const { container } = renderPage(true);
    baseFields(container);
    fireEvent.click(screen.getByTestId("custom-order-category-shoes"));
    // The framer-motion mock recreates its component functions on every
    // render, so the subtree remounts after each event — re-query the
    // node after every change/blur instead of caching it.
    const typeAndBlur = (v: string): HTMLInputElement => {
      const el = screen.getByTestId("custom-order-foot-length") as HTMLInputElement;
      el.focus();
      fireEvent.change(el, { target: { value: v } });
      const afterChange = screen.getByTestId("custom-order-foot-length") as HTMLInputElement;
      afterChange.focus();
      act(() => {
        afterChange.dispatchEvent(new FocusEvent("focusout", { bubbles: true }));
      });
      return screen.getByTestId("custom-order-foot-length") as HTMLInputElement;
    };
    expect(typeAndBlur("41").value).toBe("40");
    expect(typeAndBlur("5").value).toBe("10");
  });

  it("rejects a lone decimal point with a form error instead of submitting", async () => {
    vi.mocked(uploadOrderPhoto).mockResolvedValue("/uploads/p1.jpg");
    vi.mocked(createOrder).mockResolvedValue({ id: 1, status: "pending", created_at: "now" });
    const { container } = renderPage(true);
    baseFields(container);
    fireEvent.click(screen.getByTestId("custom-order-category-shoes"));
    fireEvent.change(screen.getByTestId("custom-order-foot-length"), { target: { value: "." } });
    fireEvent.click(screen.getByTestId("custom-order-submit"));
    expect(createOrder).not.toHaveBeenCalled();
  });

  it("sends 27.5 for a comma-typed shoes order", async () => {
    vi.mocked(uploadOrderPhoto).mockResolvedValue("/uploads/p1.jpg");
    vi.mocked(createOrder).mockResolvedValue({ id: 1, status: "pending", created_at: "now" });
    const { container } = renderPage(true);
    baseFields(container);
    fireEvent.click(screen.getByTestId("custom-order-category-shoes"));
    fireEvent.change(screen.getByTestId("custom-order-foot-length"), { target: { value: "27,5" } });
    fireEvent.click(screen.getByTestId("custom-order-submit"));
    await waitFor(() => expect(createOrder).toHaveBeenCalled());
    const payload = vi.mocked(createOrder).mock.calls[0][0] as unknown as Record<string, unknown>;
    expect(payload.foot_length).toBe(27.5);
  });
});
