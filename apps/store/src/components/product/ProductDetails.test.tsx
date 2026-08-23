import { describe, it, expect, vi } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import ProductDetails from "./ProductDetails";
import type { Product } from "@/types";
import "@testing-library/jest-dom/vitest";

vi.mock("react-i18next", () => ({
  useTranslation: () => ({
    t: (key: string) => {
      const map: Record<string, string> = {
        "product.tabs.material": "Состав и уход",
        "product.tabs.delivery": "Доставка и возврат",
      };
      return map[key] ?? key;
    },
  }),
}));

const product: Product = {
  id: 1,
  slug: "crocs-mcqueen",
  category_id: 1,
  category_name: "Тапочки",
  brand: "Crocs",
  name: "Cars x Crocs Classic Clog",
  price: 649,
  color: "Красный",
  model: "",
  fit: "",
  material: "Фирменный легкий полимерный материал Croslite™",
  care: ["Промывайте в прохладной воде с мылом", "Не сушить на батарее"],
  description: "Тапочки Crocs с принтом Тачки",
  xp_reward: 0,
  in_stock: true,
  status: "in_stock",
  stock_quantity: 5,
  created_by: "admin",
  created_at: "2026-01-01T00:00:00Z",
  updated_at: "2026-01-01T00:00:00Z",
  sizes: [],
  size_chart: [],
  images: [],
};

describe("ProductDetails (KAN-57)", () => {
  it("shows the care/material tab by default (no description tab)", () => {
    render(<ProductDetails product={product} />);
    // Default open: "Состав и уход" content is visible immediately
    expect(screen.getByText(product.material)).toBeInTheDocument();
    expect(
      screen.getByText(/Рекомендации по уходу/i),
    ).toBeInTheDocument();
  });

  it("does not render a description tab", () => {
    render(<ProductDetails product={product} />);
    expect(screen.queryByText(/Описание/i)).not.toBeInTheDocument();
  });

  it("switches to the delivery tab on click", () => {
    render(<ProductDetails product={product} />);
    fireEvent.click(screen.getByText("Доставка и возврат"));
    // The delivery tab content renders the returns list
    expect(screen.getByText(/24 часа на возврат/i)).toBeInTheDocument();
  });
});
