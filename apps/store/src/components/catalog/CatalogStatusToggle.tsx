import { useTranslation } from "react-i18next";
import { Switch } from "@/components/ui/switch";

export type CatalogStatus = "in_stock" | "preorder";

interface Props {
  value: CatalogStatus;
  onChange: (next: CatalogStatus) => void;
}

/**
 * CatalogStatusToggle — iPhone-style on/off switch driving the catalog's
 * "В наличии / Под заказ" bucket. The label text and the switch state are
 * tied together: ON (checked) → "В наличии" (the default, the natural
 * state), OFF (unchecked) → "Под заказ".
 *
 * Implementation: shadcn/ui Switch (Radix Switch primitive). No custom
 * motion — the Radix primitive ships the slide animation and a11y (role
 * switch, aria-checked, keyboard). Style is themed to the project's
 * --color-accent token (the same green that drives Buttons / CTAs).
 */
export default function CatalogStatusToggle({ value, onChange }: Props) {
  const { t } = useTranslation();
  const isInStock = value === "in_stock";
  return (
    <div className="mt-4 flex items-center gap-4">
      <h1 className="text-4xl font-bold tracking-tighter text-[var(--color-text-primary)] sm:text-5xl md:text-6xl">
        {isInStock
          ? t("catalog.toggle.inStock")
          : t("catalog.toggle.preorder")}
      </h1>
      <Switch
        checked={isInStock}
        onCheckedChange={(checked) =>
          onChange(checked ? "in_stock" : "preorder")
        }
        aria-label={t("catalog.toggle.inStock") + " / " + t("catalog.toggle.preorder")}
      />
    </div>
  );
}
