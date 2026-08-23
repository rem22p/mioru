import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
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
// remaining reason the button could stay disabled.
function fillForm(container: HTMLElement) {
  const fileInput = container.querySelector('input[type="file"]') as HTMLInputElement;
  fireEvent.change(fileInput, {
    target: { files: [new File(["x"], "photo.png", { type: "image/png" })] },
  });
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

describe("CustomOrderPage — delivery time (KAN-59)", () => {
  beforeEach(() => {
    useAuthStore.setState({ isAuthenticated: true, user: makeUser(true), loading: false, error: null });
  });

  it("delivery-time radios are rendered but disabled (manager quotes the price)", () => {
    renderPage(true);
    // All three tariff radios exist but are not selectable.
    const radios = screen
      .getAllByRole("radio")
      .filter((r) => r.getAttribute("name") === "deliveryTime");
    expect(radios.length).toBe(3);
    for (const r of radios) {
      expect(r).toBeDisabled();
    }
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
