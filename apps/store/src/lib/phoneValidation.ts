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
 *
 * A value that is not a Moldova/PMR number is NOT reinterpreted as one: a
 * legacy "+79161234567" (valid under the pre-KAN-53 contract) must not be
 * truncated into the plausible-but-different "+37379161234" — the user would
 * see someone else's number with no hint that it changed. Such input yields
 * "" so the field stays empty and has to be filled deliberately.
 */
export function phoneDigits(raw: string): string {
  const d = raw.replace(/\D/g, "");
  if (d.length <= PHONE_DIGITS) return d; // already subscriber digits
  if (d.startsWith("373")) return d.slice(3, 3 + PHONE_DIGITS);
  return "";
}

/** Compose the canonical full phone string from subscriber digits. */
export function toFullPhone(digits: string): string {
  return digits ? PHONE_PREFIX + digits : "";
}

/**
 * A phone kept on the profile is usable in a form only when it already matches
 * the KAN-53 format. Accounts created before it can hold "+79161234567", which
 * `PhoneInput` renders as an empty field — putting that raw value into form
 * state makes what the user sees disagree with what the form submits (an
 * empty-looking field that the server rejects, or a "use my phone" button that
 * visibly does nothing). Anything unusable collapses to "".
 */
export function usableStoredPhone(raw: string | null | undefined): string {
  const v = (raw ?? "").trim();
  return isValidPhone(v) ? v : "";
}
