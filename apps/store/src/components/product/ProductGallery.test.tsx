import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { render, screen } from "@testing-library/react";
import i18n from "@/i18n";
import ProductGallery from "./ProductGallery";
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
  images: [
    { id: 1, url: "/uploads/a.jpg", sort_order: 1 },
    { id: 2, url: "/uploads/b.jpg", sort_order: 2 },
  ],
};

describe("ProductGallery — localized nav labels (#83 F2)", () => {
  beforeEach(async () => {
    await i18n.changeLanguage("ro");
  });

  afterEach(async () => {
    await i18n.changeLanguage("ru");
  });

  it("announces the ro labels, not the hard-coded Russian ones", () => {
    render(<ProductGallery product={product} />);
    const prev = screen.getByRole("button", { name: "Fotografia anterioară" });
    const next = screen.getByRole("button", { name: "Fotografia următoare" });
    expect(prev).not.toHaveAccessibleName("Предыдущее фото");
    expect(next).not.toHaveAccessibleName("Следующее фото");
  });
});
