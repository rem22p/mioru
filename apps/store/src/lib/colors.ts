// Color palette for the catalog's color filter chips.
//
// The store's `products.color` column stores a short lowercase name
// (English: "black", "white", …). We map each known name to a hex so
// the filter chip can be rendered as a solid color block (style "C"
// per design) with a contrast-correct text colour.
//
// Anything not in the map falls back to a neutral grey and shows the
// raw name as the chip label, so adding new colours to the DB never
// crashes the UI — it just looks plain until a designer adds the
// matching entry here.

export const COLOR_HEX: Record<string, string> = {
  // 4 colours currently present in mioru_test (see
  // SELECT DISTINCT color FROM products). Extend as the catalog grows.
  black:  "#000000",
  white:  "#ffffff",
  pink:   "#ec4899", // hot pink — readable on the storefront's dark card bg
  yellow: "#facc15", // warm yellow
};

export function colorHex(name: string): string | null {
  return COLOR_HEX[name.toLowerCase()] ?? null;
}

/**
 * contrastTextFor picks black or white text for a hex background using
 * WCAG-style relative luminance. Anything above 0.5 luminance gets
 * black text, anything below gets white text.
 *
 * Examples:
 *   contrastTextFor("#ffffff") === "#000"  // white → black text
 *   contrastTextFor("#000000") === "#fff"  // black → white text
 *   contrastTextFor("#facc15") === "#000"  // yellow → black text
 *   contrastTextFor("#ec4899") === "#fff"  // hot pink → white text
 */
export function contrastTextFor(hex: string): "#000" | "#fff" {
  const normalised = hex.replace("#", "");
  if (normalised.length !== 6) return "#fff";
  const rgb = parseInt(normalised, 16);
  if (Number.isNaN(rgb)) return "#fff";
  const r = (rgb >> 16) & 0xff;
  const g = (rgb >> 8) & 0xff;
  const b = rgb & 0xff;
  // Standard WCAG relative-luminance, no gamma correction (cheap
  // and good enough for a chip label — not used for fine a11y text).
  const lum = (0.2126 * r + 0.7152 * g + 0.0722 * b) / 255;
  return lum > 0.5 ? "#000" : "#fff";
}
