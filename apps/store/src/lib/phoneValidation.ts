/**
 * Phone-number validation shared between Checkout and CustomOrder forms.
 *
 * Mirror of `backend/api/internal/handler/customer.go::phoneRE`
 * (`^\+?\d{7,15}$`). If you change the pattern here, change it
 * there too — and run `grep -rn "phoneRE\|PHONE_RE" backend/ apps/`
 * to confirm no stale copy survives. Frontend mirrors backend so
 * the user gets the same error message on submit that the API
 * would return; trimming before testing absorbs the common
 * paste-from-clipboard-with-leading-space case.
 */
export const PHONE_RE = /^\+?\d{7,15}$/;

export function isValidPhone(raw: string): boolean {
  return PHONE_RE.test(raw.trim());
}