import { describe, it, expect, beforeEach, vi } from "vitest";

// Mock the whole API layer: authStore (login/register/me), cartStore
// (saveCustomerCart) and favoritesStore (saveCustomerFavorites, getImageUrl)
// all import from here.
vi.mock("@/lib/api", () => ({
  fetchStoreLogin: vi.fn(),
  fetchStoreRegister: vi.fn(),
  fetchStoreLogout: vi.fn(),
  fetchStoreCustomerMe: vi.fn(),
  saveCustomerCart: vi.fn(() => Promise.resolve({ ok: true })),
  saveCustomerFavorites: vi.fn(() => Promise.resolve({ ok: true })),
  getImageUrl: (p: string) => p,
}));

import * as api from "@/lib/api";
import { useAuthStore } from "./authStore";
import { useCartStore, pushCartToServer } from "./cartStore";
import { useFavoritesStore } from "./favoritesStore";
import type { Product } from "@/types";

const product: Product = {
  id: 1,
  slug: "test-sneaker",
  category_id: 12,
  category_name: "Кроссовки",
  brand: "Test",
  name: "Test Sneaker",
  price: 5000,
  color: "#000",
  model: "",
  fit: "",
  material: "",
  care: [],
  description: "",
  xp_reward: 100,
  in_stock: true,
  status: "active",
  stock_quantity: 20,
  created_by: "admin",
  created_at: "2024-01-01",
  updated_at: "2024-01-01",
  sizes: [{label: "42", stock_quantity: 5}, {label: "43", stock_quantity: 3}, {label: "44", stock_quantity: 0}],
  size_chart: [],
  images: [],
};

const profile = {
  id: 1,
  email: "a@b.c",
  first_name: "A",
  last_name: "B",
  phone: "",
  avatar_color: "",
};

beforeEach(() => {
  vi.clearAllMocks();
  useAuthStore.setState({
    user: null,
    isAuthenticated: false,
    loading: false,
    error: null,
  });
  useCartStore.setState({ items: [] });
  useFavoritesStore.setState({ items: [] });
});

describe("authStore login cart sync", () => {
  // Regression for the checkout price bug: a forced login (guest → /checkout)
  // must not change the cart the user just built. The old code pulled a
  // stale/partial server cart over the local one, which silently changed the
  // total on the confirmation step.
  it("keeps the local cart and its total unchanged after login", async () => {
    vi.mocked(api.fetchStoreLogin).mockResolvedValue(profile);
    useCartStore.setState({ items: [{ product, size: "42", quantity: 2 }] });
    const totalBefore = useCartStore.getState().totalPrice();

    await useAuthStore.getState().login("a@b.c", "pw");

    expect(useAuthStore.getState().isAuthenticated).toBe(true);
    const items = useCartStore.getState().items;
    expect(items).toHaveLength(1);
    expect(items[0].quantity).toBe(2);
    expect(useCartStore.getState().totalPrice()).toBe(totalBefore);
    expect(totalBefore).toBe(10000);
  });

  it("pushes the local cart to the server on login", async () => {
    vi.mocked(api.fetchStoreLogin).mockResolvedValue(profile);
    useCartStore.setState({ items: [{ product, size: "42", quantity: 2 }] });

    await useAuthStore.getState().login("a@b.c", "pw");

    expect(api.saveCustomerCart).toHaveBeenCalledWith([
      { product_id: 1, size_label: "42", quantity: 2 },
    ]);
  });
});

describe("pushCartToServer", () => {
  it("awaits the save and sends the right payload when authenticated", async () => {
    useAuthStore.setState({ isAuthenticated: true });
    useCartStore.setState({ items: [{ product, size: "43", quantity: 3 }] });

    await pushCartToServer();

    expect(api.saveCustomerCart).toHaveBeenCalledWith([
      { product_id: 1, size_label: "43", quantity: 3 },
    ]);
  });

  it("no-ops on an empty cart", async () => {
    useAuthStore.setState({ isAuthenticated: true });
    useCartStore.setState({ items: [] });

    await pushCartToServer();

    expect(api.saveCustomerCart).not.toHaveBeenCalled();
  });

  it("no-ops when not authenticated", async () => {
    useAuthStore.setState({ isAuthenticated: false });
    useCartStore.setState({ items: [{ product, size: "42", quantity: 1 }] });

    await pushCartToServer();

    expect(api.saveCustomerCart).not.toHaveBeenCalled();
  });
});

describe("telegram binding mapping", () => {
  // The wire uses snake_case (first_name); the User type uses camelCase.
  // Passing the payload through unmapped silently loses the name and the
  // profile renders "—" for accounts without a @username.
  it("maps first_name from the wire onto telegram.firstName", async () => {
    vi.mocked(api.fetchStoreLogin).mockResolvedValue({
      ...profile,
      telegram: { linked: true, first_name: "Neo" },
    });

    await useAuthStore.getState().login("a@b.c", "pw");

    const tg = useAuthStore.getState().user?.telegram;
    expect(tg?.linked).toBe(true);
    expect(tg?.firstName).toBe("Neo");
  });

  it("keeps username and linked=false from /me", async () => {
    vi.mocked(api.fetchStoreCustomerMe).mockResolvedValue({
      ...profile,
      telegram: { linked: false },
    });

    await useAuthStore.getState().fetchMe();

    expect(useAuthStore.getState().user?.telegram).toEqual({
      linked: false,
      username: undefined,
      firstName: undefined,
    });
  });

  it("leaves telegram undefined when the backend omits it", async () => {
    vi.mocked(api.fetchStoreLogin).mockResolvedValue(profile);

    await useAuthStore.getState().login("a@b.c", "pw");

    expect(useAuthStore.getState().user?.telegram).toBeUndefined();
  });
});
