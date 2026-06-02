import { useEffect } from "react";
import { motion, AnimatePresence } from "framer-motion";
import { X, Ruler } from "lucide-react";
import type { Product } from "@/types";
import { toSizeChart } from "@/types";
import { useTranslation } from "react-i18next";

interface SizeChartModalProps {
  isOpen: boolean;
  onClose: () => void;
  product: Product;
}

export default function SizeChartModal({
  isOpen,
  onClose,
  product,
}: SizeChartModalProps) {
  const { t } = useTranslation();

  // Body scroll lock
  useEffect(() => {
    if (isOpen) {
      document.body.style.overflow = "hidden";
    } else {
      document.body.style.overflow = "";
    }
    return () => {
      document.body.style.overflow = "";
    };
  }, [isOpen]);

  const sizeChart = toSizeChart(product.size_chart);
  if (!sizeChart) return null;

  const { columns, rows } = sizeChart;

  return (
    <AnimatePresence>
      {isOpen && (
        <motion.div
          initial={{ opacity: 0 }}
          animate={{ opacity: 1 }}
          exit={{ opacity: 0 }}
          className="fixed inset-0 z-50 flex items-center justify-center p-4"
        >
          {/* Backdrop */}
          <div
            className="absolute inset-0 bg-black/70 backdrop-blur-sm"
            onClick={onClose}
          />

          {/* Modal */}
          <motion.div
            initial={{ opacity: 0, scale: 0.95, y: 20 }}
            animate={{ opacity: 1, scale: 1, y: 0 }}
            exit={{ opacity: 0, scale: 0.95, y: 20 }}
            transition={{ duration: 0.2 }}
            className="relative w-full max-w-2xl bg-[var(--color-bg-card)] border border-[var(--color-border-custom)] rounded-2xl overflow-hidden max-h-[90vh] flex flex-col"
          >
            {/* Header */}
            <div className="flex items-center justify-between p-6 border-b border-[var(--color-border-custom)] shrink-0">
              <div className="flex items-center gap-3">
                <Ruler className="h-5 w-5 text-[#44944A]" />
                <div>
                  <h3 className="text-lg font-semibold text-[var(--color-text-primary)]">
                    {t("product.sizeChart")}
                  </h3>
                  <p className="text-xs text-[var(--color-text-secondary)] mt-0.5">
                    {product.name}
                  </p>
                </div>
              </div>
              <button
                onClick={onClose}
                className="min-w-[44px] min-h-[44px] rounded-lg border border-[var(--color-border-custom)] flex items-center justify-center text-[var(--color-text-secondary)] hover:text-[var(--color-text-primary)] hover:border-[var(--color-text-muted)] transition-all"
                aria-label={t("common.close")}
              >
                <X className="h-4 w-4" />
              </button>
            </div>

            {/* Scrollable content */}
            <div className="p-6 overflow-y-auto">
              <table className="w-full">
                <thead>
                  <tr className="border-b border-[var(--color-border-custom)]">
                    {columns.map((col) => (
                      <th
                        key={col.key}
                        className="text-left py-3 px-4 text-xs font-mono uppercase tracking-wider text-[var(--color-text-secondary)]"
                      >
                        {col.label}
                      </th>
                    ))}
                  </tr>
                </thead>
                <tbody>
                  {rows.map((row, idx) => (
                    <tr
                      key={idx}
                      className={`border-b border-[var(--color-border-custom)]/50 ${
                        idx % 2 === 0 ? "bg-[var(--color-bg-primary)]/50" : ""
                      }`}
                    >
                      {columns.map((col) => (
                        <td
                          key={col.key}
                          className="py-3 px-4 text-sm text-[var(--color-text-primary)]"
                        >
                          {row[col.key] ?? "—"}
                        </td>
                      ))}
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </motion.div>
        </motion.div>
      )}
    </AnimatePresence>
  );
}
