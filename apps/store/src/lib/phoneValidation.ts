/**
 * Phone-number validation shared between Checkout, CustomOrder and
 * EditProfile forms.
 *
 * KAN-53: strict Moldova/PMR format — a fixed `+373` prefix followed by
 * exactly 8 digits (both manager examples, e.g. `+373 60000000`, are 8
 * digits; the "9 digits" in the task text is a typo). The `+373` prefix is
 * mandatory.
 *
 * Mirror of backend's phoneRE (internal/handler/customer.go). Keep in sync.
 */
export const PHONE_RE = /^\+373\d{8}$/;

/** Moldova/PMR country calling code with the leading plus. */
export const PHONE_PREFIX = "+373";

/** Number of subscriber digits after the `+373` prefix. */
export const PHONE_DIGITS = 8;

export function isValidPhone(raw: string): boolean {
  return PHONE_RE.test(raw.trim());
}

/**
 * Extract the subscriber digits (after `+373`) from any stored/typed value.
 * Tolerates "+373...", "373...", a bare "60000000", spaces and dashes.
 * Capped at {@link PHONE_DIGITS}.
 */
export function phoneDigits(raw: string): string {
  let d = raw.replace(/\D/g, "");
  if (d.startsWith("373")) d = d.slice(3);
  return d.slice(0, PHONE_DIGITS);
}

/** Compose the canonical full phone string from subscriber digits. */
export function toFullPhone(digits: string): string {
  return digits ? PHONE_PREFIX + digits : "";
}
