import { useId } from "react";
import {
  PHONE_DIGITS,
  PHONE_PREFIX,
  phoneDigits,
  toFullPhone,
} from "@/lib/phoneValidation";

interface PhoneInputProps {
  /** Full phone value (e.g. "+37360000000" or ""). */
  value: string;
  /** Called with the canonical full phone ("+373XXXXXXXX") or "" when empty. */
  onChange: (full: string) => void;
  className?: string;
  placeholder?: string;
  "data-testid"?: string;
  id?: string;
}

/**
 * KAN-53 — masked phone input for Moldova/PMR.
 *
 * The `+373` prefix is fixed and always shown; the user types only the
 * subscriber digits. Non-digits are stripped, input is capped at 8 digits,
 * so nothing beyond a valid "+373XXXXXXXX" can ever be entered. The parent
 * always receives the canonical full phone via `onChange`.
 *
 * The field carries no `aria-label`: an own name would override the page's
 * visible label and announce it in one fixed language. Callers pass `id` and
 * point their `<label htmlFor>` at it instead.
 */
export default function PhoneInput({
  value,
  onChange,
  className = "",
  placeholder,
  "data-testid": testId,
  id,
}: PhoneInputProps) {
  const autoId = useId();
  const digits = phoneDigits(value);

  const handleChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const next = phoneDigits(e.target.value);
    onChange(toFullPhone(next));
  };

  return (
    <div
      className={`flex items-center gap-1 ${className}`}
      data-testid={testId ? `${testId}-wrap` : undefined}
    >
      <span
        className="select-none font-medium text-[var(--color-text-primary)]"
        aria-hidden="true"
      >
        {PHONE_PREFIX}
      </span>
      <input
        id={id ?? autoId}
        type="tel"
        inputMode="numeric"
        autoComplete="tel-national"
        maxLength={PHONE_DIGITS}
        value={digits}
        onChange={handleChange}
        placeholder={placeholder}
        data-testid={testId}
        className="min-w-0 flex-1 bg-transparent outline-none placeholder:text-[var(--color-text-muted)]"
      />
    </div>
  );
}
