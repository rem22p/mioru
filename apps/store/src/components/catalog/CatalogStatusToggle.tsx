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
      <div className="relative inline-flex rounded-full bg-[var(--color-bg-card)] border border-[var(--color-border-custom)] p-1 select-none">
        {/* Sliding active background */}
        <motion.div
          layout
          transition={{ type: "spring", stiffness: 400, damping: 30 }}
          className="absolute top-1 bottom-1 rounded-full bg-[#44944A] shadow-sm"
          style={{
            left: isInStock ? "0.25rem" : "50%",
            right: isInStock ? "50%" : "0.25rem",
          }}
        />
        {/* Left segment */}
        <button
          onClick={() => onChange("in_stock")}
          className={`relative z-10 px-8 py-3 text-sm font-semibold rounded-full transition-colors min-w-[140px] text-center uppercase tracking-wider ${
            isInStock
              ? "text-black"
              : "text-[var(--color-text-secondary)] hover:text-[var(--color-text-primary)]"
          }`}
        >
          {t("catalog.toggle.inStock")}
        </button>
        {/* Right segment */}
        <button
          onClick={() => onChange("preorder")}
          className={`relative z-10 px-8 py-3 text-sm font-semibold rounded-full transition-colors min-w-[140px] text-center uppercase tracking-wider ${
            !isInStock
              ? "text-black"
              : "text-[var(--color-text-secondary)] hover:text-[var(--color-text-primary)]"
          }`}
        >
          {t("catalog.toggle.preorder")}
        </button>
      </div>
    </div>
  );
}
