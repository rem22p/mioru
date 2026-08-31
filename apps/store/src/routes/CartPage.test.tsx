import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { HelmetProvider } from "@dr.pogodin/react-helmet";
import i18n from "@/i18n";
import CartPage from "./CartPage";
import { useCartStore } from "@/stores/cartStore";
import { useAuthStore } from "@/stores/authStore";
import type { Product } from "@/types";
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

const product: Product = {
  id: 1,
  slug: "midnight-runner",
  category_id: 12,
  category_name: "Кроссовки",
  brand: "Nike",
  name: "Midnight Runner",
  price: 7990,
  color: "Чёрный",
  model: "MR-1",
  material: "Текстиль/резина",
  care: ["Чистить мягкой щёткой"],
  description: "Беговые кроссовки",
  xp_reward: 79,
  in_stock: true,
  status: "in_stock",
  stock_quantity: 25,
  created_by: "e2e",
  created_at: "2024-01-01T00:00:00Z",
  updated_at: "2024-01-01T00:00:00Z",
  sizes: [{ label: "42", stock_quantity: 5 }],
  size_chart: [],
  images: [{ id: 1, url: "/uploads/a.jpg", sort_order: 1 }],
};

describe("CartPage — localized quantity labels (#83 F2)", () => {
  beforeEach(async () => {
    await i18n.changeLanguage("ro");
    useCartStore.setState({
      items: [{ product, size: "42", quantity: 1 }],
    });
    useAuthStore.setState({ isAuthenticated: false });
  });

  afterEach(async () => {
    await i18n.changeLanguage("ru");
    useCartStore.setState({ items: [] });
  });

  it("announces the ro labels, not the hard-coded Russian ones", () => {
    render(
      <HelmetProvider>
        <MemoryRouter>
          <CartPage />
        </MemoryRouter>
      </HelmetProvider>,
    );
    const decrease = screen.getByRole("button", { name: "Micșorează" });
    const increase = screen.getByRole("button", { name: "Mărește" });
    expect(decrease).not.toHaveAccessibleName("Уменьшить");
    expect(increase).not.toHaveAccessibleName("Увеличить");
  });
});
