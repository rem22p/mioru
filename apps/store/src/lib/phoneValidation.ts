/**
 * Phone-number validation shared between Checkout and CustomOrder forms.
 *
 * Moldova format: +373 XX XXX XXX (with optional spaces).
 * Mirror of backend's phoneRE — keep both in sync.
 */
const PHONE_RE = /^\+373[\d\s]{8,12}$/;

function digitCount(s: string): number {
  return s.replace(/\D/g, "").length;
}

export function isValidPhone(raw: string): boolean {
  const trimmed = raw.trim();
  if (!PHONE_RE.test(trimmed)) return false;
  // After +373 prefix, expect exactly 8 digits
  const digits = trimmed.replace(/\D/g, "");
  return digits.length === 11; // +373 = 3 digits + 8 = 11
}
