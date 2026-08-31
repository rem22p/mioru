import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { HelmetProvider } from "@dr.pogodin/react-helmet";
import ProfilePage from "./ProfilePage";
import { useAuthStore } from "@/stores/authStore";
import { useCurrencyStore } from "@/stores/currencyStore";
import { fetchStoreCustomerOrders, type StoreOrder } from "@/lib/api";
import type { User } from "@/types";
import "@testing-library/jest-dom/vitest";

vi.mock("framer-motion", () => ({
  motion: new Proxy(
    {},
    {
      get: () =>
        ((p: React.HTMLAttributes<HTMLElement>) => (
          <div {...p} />
        )) as never,
    },
  ),
}));

vi.mock("react-i18next", () => ({
  useTranslation: () => ({ t: (k: string) => k }),
}));

vi.mock("@/lib/api", () => ({
  fetchStoreCustomerOrders: vi.fn(),
  getImageUrl: (u: string) => u,
}));

const user: User = {
  id: "1",
  name: "E2E",
  email: "e2e@example.com",
  firstName: "E2E",
  lastName: "",
  phone: "+37360000000",
  avatarParams: {} as unknown as User["avatarParams"],
  xpBalance: 0,
  vipLevel: 0,
  telegram: { linked: false },
};

const individualOrder: StoreOrder = {
  id: 77,
  order_code: "M-77",
  total_minor: 199000,
  status: "pending",
  type: "individual",
  city: "Тирасполь",
  phone: "+37377777777",
  delivery_method: "personal",
  payment_method: "cod",
  street: "",
  house: "",
  apartment: "",
  comment: "",
  category: "shoes",
  foot_length: 27.5,
  photos: ["/uploads/p1.jpg"],
  created_at: "2026-08-30T10:00:00Z",
};

function renderPage(order: StoreOrder) {
  vi.mocked(fetchStoreCustomerOrders).mockResolvedValue({
    orders: [order],
    total: 1,
    page: 1,
    per_page: 20,
  });
  useAuthStore.setState({ isAuthenticated: true, user, loading: false, error: null });
  useCurrencyStore.setState({ currency: "MDL" } as never);
  return render(
    <HelmetProvider>
      <MemoryRouter>
        <ProfilePage />
      </MemoryRouter>
    </HelmetProvider>,
  );
}

describe("ProfilePage — individual order card (#88 F1)", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("renders category, foot length and photos for an individual order", async () => {
    renderPage(individualOrder);
    const row = await screen.findByTestId("order-row");
    expect(row).toBeInTheDocument();

    // Expanded details render after the header click (bubbles to its onClick).
    fireEvent.click(screen.getByText("M-77"));

    await waitFor(() => {
      expect(screen.getByText("customOrder.categories.shoes")).toBeInTheDocument();
      expect(screen.getByText("profile.orderDetail.footLengthCm")).toBeInTheDocument();
      expect(screen.getByText("profile.orderDetail.photos")).toBeInTheDocument();
      expect(screen.getByAltText("")).toBeInTheDocument();
    });
  });
});
