import { create } from "zustand";
import type { Currency } from "@/lib/currency";

interface CurrencyState {
  currency: Currency;
  setCurrency: (c: Currency) => void;
}

const stored = (typeof window !== "undefined" ? localStorage.getItem("mioru-currency") : null) as Currency | null;

export const useCurrencyStore = create<CurrencyState>((set) => ({
  currency: stored === "MDL" ? "MDL" : "PMR",
  setCurrency: (c) => {
    localStorage.setItem("mioru-currency", c);
    set({ currency: c });
  },
}));
