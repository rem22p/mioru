/**
 * Phone-number validation shared between Checkout and CustomOrder forms.
 *
 * Mirror of backend's phoneRE: accepts international numbers
 * with optional + prefix followed by 7-15 digits.
 * Keep both in sync.
 */
export const PHONE_RE = /^\+?\d{7,15}$/;

export function isValidPhone(raw: string): boolean {
  return PHONE_RE.test(raw.trim());
}
