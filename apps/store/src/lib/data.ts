import type { User, Order } from "@/types";

// Static mock data for the profile page (orders not yet in the backend).

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
