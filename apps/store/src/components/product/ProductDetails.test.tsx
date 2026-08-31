import { describe, it, expect, vi } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import ProductDetails from "./ProductDetails";
import type { Product } from "@/types";
import "@testing-library/jest-dom/vitest";

// #80: t() supports returnObjects so the delivery/returns lists arrive as
// translated arrays, matching how the component consumes them.
vi.mock("react-i18next", () => ({
  useTranslation: () => ({
    t: (key: string, opts?: { returnObjects: boolean }) => {
      const map: Record<string, string> = {
        "product.tabs.material": "Состав и уход",
        "product.tabs.delivery": "Доставка и возврат",
        "product.details.material": "Материал",
        "product.details.careTitle": "Рекомендации по уходу",
        "product.details.deliveryTitle": "Доставка",
        "product.details.returnsTitle": "Возврат",
      };
      if (key === "product.details.deliveryItems" && opts?.returnObjects) {
        return ["Личная встреча — Тирасполь", "Маршрутка — ПМР + Кишинёв"];
      }
      if (key === "product.details.returnsItems" && opts?.returnObjects) {
        return ["24 часа на возврат", "Товар должен быть с бирками"];
      }
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

describe("ProductDetails (KAN-57 + #80)", () => {
  it("shows the care/material tab by default (no description tab)", () => {
    render(<ProductDetails product={product} />);
    // Default open: "Состав и уход" content is visible immediately
    expect(screen.getByText(product.material)).toBeInTheDocument();
    expect(screen.getByText(product.care[0])).toBeInTheDocument();
  });

  it("does not render a description tab", () => {
    render(<ProductDetails product={product} />);
    expect(screen.queryByText(/Описание/i)).not.toBeInTheDocument();
  });

  it("switches to the delivery tab on click", () => {
    render(<ProductDetails product={product} />);
    fireEvent.click(screen.getByTestId("product-details-tab-delivery"));
    // The delivery tab content renders the translated list items
    expect(screen.getByText(/Личная встреча/)).toBeInTheDocument();
  });

  it("tab row keeps buttons inside the viewport (no overflow mechanism)", () => {
    // #77 hardening: the row scrolls instead of stretching the document.
    render(<ProductDetails product={product} />);
    const row = screen.getByTestId("product-details-tabs");
    expect(row.className).toContain("overflow-x-auto");
  });
});
