/** PMR to MDL exchange rate: 1 PMR = 0.945 MDL */
export const PMR_TO_MDL = 0.945;

export type Currency = "PMR" | "MDL";

/** Round up to the nearest multiple of 5. */
function ceil5(n: number): number {
  return Math.ceil(n / 5) * 5;
}

/**
 * Convert a price from PMR to the target currency.
 * MDL prices are rounded UP to the nearest 5.
 */
export function convertPrice(pricePMR: number, currency: Currency): number {
  if (currency === "MDL") {
    return ceil5(pricePMR * PMR_TO_MDL);
  }
  return pricePMR;
}

/**
 * Format a price for display with currency symbol.
 * PMR: "1 234 ₽"   MDL: "1 235 L"
 */
export function formatPrice(pricePMR: number, currency: Currency): string {
  const value = convertPrice(pricePMR, currency);
  const formatted = value.toLocaleString("ru-RU");
  return currency === "MDL" ? `${formatted} L` : `${formatted} ₽`;
}
