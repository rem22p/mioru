import type { User, Order, Product, Category } from "@/types";

// Static mock data is replaced by API calls.
// Products and categories now come from the backend via lib/api.ts and stores/catalogStore.ts.
// This file keeps only the mock user and orders for the profile page (not yet in the backend).

// Kept for backward compatibility (store's deprecated AdminPage)
export const products: Product[] = [];
export const categories: Category[] = [];

export const mockUser: User = {
  id: "1",
  name: "Иван",
  email: "ivan@mioru.store",
  avatarParams: {
    gender: "male",
    height: 180,
    weight: 75,
    fatPercentage: 15,
    musclePercentage: 40,
  },
  xpBalance: 1240,
  vipLevel: 2,
};

export const mockOrders: Order[] = [
  {
    id: "ORD-001",
    userId: "1",
    items: [],
    total: 7100,
    status: "delivered",
    createdAt: "2024-12-01",
    deliveryInfo: {
      name: "Иван",
      phone: "+7 (999) 123-45-67",
      email: "ivan@mioru.store",
      city: "Москва",
      street: "Тверская",
      house: "12",
      apartment: "45",
      paymentMethod: "card",
    },
  },
  {
    id: "ORD-002",
    userId: "1",
    items: [],
    total: 8900,
    status: "shipped",
    createdAt: "2024-12-15",
    deliveryInfo: {
      name: "Иван",
      phone: "+7 (999) 123-45-67",
      email: "ivan@mioru.store",
      city: "Москва",
      street: "Арбат",
      house: "5",
      paymentMethod: "sbp",
    },
  },
];
