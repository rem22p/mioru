/** MDL to PMR exchange rate: 1 MDL = 0.945 PMR */
export const MDL_TO_PMR = 0.945;

export type Currency = "PMR" | "MDL";

/** Round up to the nearest multiple of 5. */
function ceil5(n: number): number {
  return Math.ceil(n / 5) * 5;
}

/**
 * Convert a price from MDL (stored in DB) to the target display currency.
 * PMR = MDL * 0.945, rounded UP to the nearest 5.
 */
export function convertPrice(priceMDL: number, currency: Currency): number {
  if (currency === "PMR") {
    return ceil5(priceMDL * MDL_TO_PMR);
  }
  return priceMDL;
}

/**
 * Format a price for display with currency symbol.
 * PMR: "475 ₽"   MDL: "500 L"
 */
export function formatPrice(priceMDL: number, currency: Currency): string {
  const value = convertPrice(priceMDL, currency);
  const formatted = value.toLocaleString("ru-RU");
  return currency === "PMR" ? `${formatted} ₽` : `${formatted} L`;
}
