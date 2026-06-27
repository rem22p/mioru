import { motion } from "framer-motion";
import { useTranslation } from "react-i18next";

export type CatalogStatus = "in_stock" | "preorder";

interface Props {
  value: CatalogStatus;
  onChange: (next: CatalogStatus) => void;
}

export default function CatalogStatusToggle({ value, onChange }: Props) {
  const { t } = useTranslation();
  const isInStock = value === "in_stock";

  return (
    <div className="mt-4">
      <div className="relative inline-flex rounded-full bg-[var(--color-bg-secondary)] border-2 border-[var(--color-border-custom)] p-1 select-none shadow-inner">
        {/* Sliding active background — white/green depending on side */}
        <motion.div
          layout
          transition={{ type: "spring", stiffness: 350, damping: 28 }}
          className="absolute top-1 bottom-1 rounded-full shadow-md"
          style={{
            left: isInStock ? "0.375rem" : "50%",
            right: isInStock ? "50%" : "0.375rem",
            background: isInStock
              ? "#44944A"
              : "#ffffff",
          }}
        />
        {/* В наличии */}
        <button
          onClick={() => onChange("in_stock")}
          className={`relative z-10 px-6 py-2.5 text-xl font-bold rounded-full transition-all duration-200 min-w-[140px] text-center ${
            isInStock
              ? "text-white scale-[1.02]"
              : "text-[var(--color-text-muted)] hover:text-[var(--color-text-primary)]"
          }`}
        >
          {t("catalog.toggle.inStock")}
        </button>
        {/* Под заказ */}
        <button
          onClick={() => onChange("preorder")}
          className={`relative z-10 px-6 py-2.5 text-xl font-bold rounded-full transition-all duration-200 min-w-[140px] text-center ${
            !isInStock
              ? "text-black scale-[1.02]"
              : "text-[var(--color-text-muted)] hover:text-[var(--color-text-primary)]"
          }`}
        >
          {t("catalog.toggle.preorder")}
        </button>
      </div>
    </div>
  );
}
