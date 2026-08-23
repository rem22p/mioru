import { useTranslation } from "react-i18next";
import { useCurrencyStore } from "@/stores/currencyStore";
import type { Currency } from "@/lib/currency";

interface PillOption<T extends string> {
  value: T;
  label: string;
}

interface PillToggleProps<T extends string> {
  options: PillOption<T>[];
  current: T;
  onChange: (value: T) => void;
  testId: string;
  ariaLabel: string;
  className?: string;
}

/**
 * Segmented pill toggle (see KAN-56 mock): rounded-full border,
 * active segment filled with the primary text color, inactive segments muted.
 * Colors adapt to dark/light theme via CSS variables.
 */
function PillToggle<T extends string>({
  options,
  current,
  onChange,
  testId,
  ariaLabel,
  className = "",
}: PillToggleProps<T>) {
  return (
    <div
      data-testid={testId}
      role="group"
      aria-label={ariaLabel}
      className={`inline-flex items-center rounded-full border border-[var(--color-border-light)] p-0.5 ${className}`}
    >
      {options.map((opt) => {
        const active = opt.value === current;
        return (
          <button
            key={opt.value}
            type="button"
            data-testid={`${testId}-${opt.value.toLowerCase()}`}
            aria-pressed={active}
            onClick={() => onChange(opt.value)}
            className={`rounded-full px-3 py-1 text-xs font-bold tracking-wide transition-colors ${
              active
                ? "bg-[var(--color-text-primary)] text-[var(--color-bg-primary)]"
                : "text-[var(--color-text-muted)] hover:text-[var(--color-text-primary)]"
            }`}
          >
            {opt.label}
          </button>
        );
      })}
    </div>
  );
}

const languageOptions: PillOption<string>[] = [
  { value: "ru", label: "RU" },
  { value: "ro", label: "RO" },
  { value: "en", label: "EN" },
];

export function LanguageToggle({ className }: { className?: string }) {
  const { t, i18n } = useTranslation();
  const raw = (i18n.resolvedLanguage || i18n.language || "ru").slice(0, 2);
  const current = languageOptions.some((o) => o.value === raw) ? raw : "ru";
  return (
    <PillToggle
      options={languageOptions}
      current={current}
      onChange={(lng) => i18n.changeLanguage(lng)}
      testId="language-toggle"
      ariaLabel={t("nav.language")}
      className={className}
    />
  );
}

const currencyOptions: PillOption<Currency>[] = [
  { value: "PMR", label: "RUB" },
  { value: "MDL", label: "MDL" },
];

export function CurrencyToggle({ className }: { className?: string }) {
  const { t } = useTranslation();
  const { currency, setCurrency } = useCurrencyStore();
  return (
    <PillToggle
      options={currencyOptions}
      current={currency}
      onChange={setCurrency}
      testId="currency-toggle"
      ariaLabel={t("nav.currency")}
      className={className}
    />
  );
}
