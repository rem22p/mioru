import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { HelmetProvider } from "@dr.pogodin/react-helmet";
import CheckoutPage from "./CheckoutPage";
import { useAuthStore } from "@/stores/authStore";
import { useCartStore } from "@/stores/cartStore";
import type { User } from "@/types";
import "@testing-library/jest-dom/vitest";

// ── Mocks (non-store infra only) ──────────────────────────────────────────

const tMap: Record<string, string> = {
  "checkout.phone": "Телефон",
  "checkout.phonePlaceholder": "+373 ...",
  "checkout.fields.city": "Город",
  "checkout.delivery.title": "Способ доставки",
  "checkout.delivery.personal": "Встреча",
  "checkout.delivery.free": "Бесплатно",
  "checkout.delivery.address": "Доставка",
  "checkout.delivery.price25": "25",
  "checkout.delivery.bus": "Автобус",
  "checkout.delivery.priceUpTo50": "до 50",
  "checkout.delivery.express": "Экспресс",
  "checkout.delivery.moldovaPost": "Почта",
  "checkout.back": "Назад",
  "checkout.next": "Далее",
  "checkout.confirm": "Подтвердить заказ",
  "checkout.telegramRequired": "Подключите Telegram для заказа",
  "checkout.telegramRequiredCta": "Подключить",
  "checkout.orderSummary": "Заказ",
  "checkout.total": "Итого",
  "checkout.summary.city": "Город",
  "checkout.summary.delivery": "Доставка",
  "checkout.summary.payment": "Оплата",
  "checkout.payment.card": "Карта",
  "checkout.payment.cardDesc": "Онлайн",
  "checkout.payment.cod": "Наличные",
  "checkout.payment.codDesc": "При получении",
  "checkout.creditCard": "Карта онлайн",
  "checkout.steps.address": "Адрес",
  "checkout.steps.payment": "Оплата",
  "checkout.steps.confirmation": "Подтверждение",
  "checkout.useMyPhone": "Использовать мой",
};

// framer-motion AnimatePresence mode="wait" needs requestAnimationFrame to
// finish the exit animation before mounting the next step — jsdom has no
// rAF loop, so step 2 never appears in tests. Stub it to plain passthrough
// components (standard practice for framer-motion in unit tests).
vi.mock("framer-motion", () => ({
  motion: new Proxy(
    {},
    {
      get: (_t, tag: string) =>
        ({ div: (p: React.HTMLAttributes<HTMLDivElement>) => <div {...p} /> })[tag] ||
        ((p: React.HTMLAttributes<HTMLElement>) => <div {...p} />),
    },
  ),
  AnimatePresence: ({ children }: { children: React.ReactNode }) => <>{children}</>,
}));

vi.mock("react-i18next", () => ({
  useTranslation: () => ({ t: (k: string) => tMap[k] || k }),
}));

const mockNavigate = vi.fn();
vi.mock("react-router-dom", async (importOriginal) => {
  const actual = await importOriginal<typeof import("react-router-dom")>();
  return {
    ...actual,
    useNavigate: () => mockNavigate,
  };
});

// CityAutocomplete pulls a city list over the network — stub it to a
// plain controlled input in tests.
vi.mock("@/components/ui/CityAutocomplete", () => ({
  default: ({
    value,
    onChange,
    testId,
  }: {
    value: string;
    onChange: (v: string) => void;
    testId?: string;
  }) => (
    <input
      data-testid={testId || "city-autocomplete"}
      value={value}
      onChange={(e) => onChange(e.target.value)}
    />
  ),
}));

vi.mock("@/stores/currencyStore", () => ({
  useCurrencyStore: vi.fn((selector?: (s: unknown) => unknown) => {
    const state = { currency: "MDL" };
    return selector ? selector(state) : state;
  }),
}));

vi.mock("@/lib/currency", () => ({
  formatPrice: (n: number) => `${n} MDL`,
}));

vi.mock("@/lib/api", () => ({
  createOrder: vi.fn(),
  saveCustomerCart: vi.fn(),
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
    avatarParams: {
      gender: "male",
      hairStyle: 0,
      hairColor: "#000000",
      skinTone: 0,
      top: 0,
      bottom: 0,
      accessories: 0,
      // avatarParams requires more fields — the test never reads them,
      // so cast through unknown.
    } as unknown as User["avatarParams"],
    xpBalance: 0,
    vipLevel: 0,
    telegram: telegramLinked
      ? { linked: true, username: "e2e_tg", firstName: "E2E" }
      : { linked: false },
  };
}

const cartItem = {
  product: {
    id: 1,
    slug: "tee",
    name: "Футболка",
    price: 100,
    status: "in_stock",
  } as unknown as import("@/types").Product,
  size: "M",
  quantity: 1,
} as unknown as import("@/types").CartItem;

function seedStores(telegramLinked: boolean) {
  useAuthStore.setState({
    isAuthenticated: true,
    user: makeUser(telegramLinked),
    loading: false,
    error: null,
  });
  useCartStore.setState({ items: [cartItem] });
}

// ── Helper: walk to the confirm step (steps 1 and 2) ─────────────────────

async function reachConfirmStep() {
  fireEvent.change(screen.getByTestId("checkout-phone"), {
    target: { value: "+37360000000" },
  });
  fireEvent.change(screen.getByTestId("checkout-city"), {
    target: { value: "Тирасполь" },
  });
  // "personal" delivery is the first radio and enabled for Tiraspol.
  fireEvent.click(screen.getAllByRole("radio")[0]);
  fireEvent.click(screen.getByTestId("checkout-next"));
  // Step 2 — card payment is the default and always allowed.
  fireEvent.click(screen.getByTestId("checkout-next"));
}

describe("CheckoutPage — Telegram order gate", () => {
  beforeEach(() => {
    mockNavigate.mockClear();
    useCartStore.setState({ items: [] });
    useAuthStore.setState({ isAuthenticated: false, user: null, loading: false, error: null });
  });

  it("blocks confirm + shows the banner when Telegram is not linked", async () => {
    seedStores(false);

    render(
      <HelmetProvider>
        <MemoryRouter>
          <CheckoutPage />
        </MemoryRouter>
      </HelmetProvider>,
    );

    await reachConfirmStep();

    // Banner explains why the button is disabled.
    expect(screen.getByTestId("checkout-telegram-required")).toBeVisible();
    expect(screen.getByTestId("checkout-confirm")).toBeDisabled();
    // A disabled button swallows clicks — submitOrder (and its redirect)
    // must never fire while the gate is closed.
    fireEvent.click(screen.getByTestId("checkout-confirm"));
    expect(mockNavigate).not.toHaveBeenCalled();
  });

  it("enables confirm and hides the banner when Telegram is linked", async () => {
    seedStores(true);

    render(
      <HelmetProvider>
        <MemoryRouter>
          <CheckoutPage />
        </MemoryRouter>
      </HelmetProvider>,
    );

    await reachConfirmStep();

    expect(screen.queryByTestId("checkout-telegram-required")).not.toBeInTheDocument();
    expect(screen.getByTestId("checkout-confirm")).toBeEnabled();
  });
});

describe("CheckoutPage — guest access (KAN: form opens without login)", () => {
  beforeEach(() => {
    mockNavigate.mockClear();
    useCartStore.setState({ items: [cartItem] });
    useAuthStore.setState({ isAuthenticated: false, user: null, loading: false, error: null });
  });

  it("guest can open the checkout form — no redirect to /profile", () => {
    render(
      <HelmetProvider>
        <MemoryRouter>
          <CheckoutPage />
        </MemoryRouter>
      </HelmetProvider>,
    );
    expect(mockNavigate).not.toHaveBeenCalledWith(
      "/profile?redirect=/checkout",
      expect.anything(),
    );
    // Step 1 (address) is visible — the guest can start filling the form
    expect(screen.getByTestId("checkout-phone")).toBeInTheDocument();
  });
});
