import { useEffect } from 'react';
import { motion, AnimatePresence } from 'framer-motion';
import { X, Ruler } from 'lucide-react';
import { Product } from '@/types';
import { useTranslation } from 'react-i18next';

interface SizeChartModalProps {
  isOpen: boolean;
  onClose: () => void;
  product: Product;
}

export default function SizeChartModal({ isOpen, onClose, product }: SizeChartModalProps) {
  const { t } = useTranslation();

  // Body scroll lock
  useEffect(() => {
    if (isOpen) {
      document.body.style.overflow = 'hidden';
    } else {
      document.body.style.overflow = '';
    }
    return () => { document.body.style.overflow = ''; };
  }, [isOpen]);

  if (!product.sizeChart) return null;

  const { columns, rows, unit } = product.sizeChart;

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
                  <h3 className="text-lg font-semibold text-[var(--color-text-primary)]">{t('product.sizeChart')}</h3>
                  <p className="text-xs text-[var(--color-text-secondary)] mt-0.5">{product.name}</p>
                </div>
              </div>
              <button
                onClick={onClose}
                className="min-w-[44px] min-h-[44px] rounded-lg border border-[var(--color-border-custom)] flex items-center justify-center text-[var(--color-text-secondary)] hover:text-[var(--color-text-primary)] hover:border-[var(--color-text-muted)] transition-all"
                aria-label={t('common.close')}
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
                        key={String(col.key)}
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
                      key={row.size}
                      className={`border-b border-[var(--color-border-custom)]/50 ${
                        idx % 2 === 0 ? 'bg-[var(--color-bg-primary)]/50' : ''
                      }`}
                    >
                      {columns.map((col) => (
                        <td
                          key={String(col.key)}
                          className="py-3 px-4 text-sm text-[var(--color-text-primary)]"
                        >
                          {row[col.key] ?? '—'}
                        </td>
                      ))}
                    </tr>
                  ))}
                </tbody>
              </table>

              {/* Fit Info */}
              {product.fit && (
                <div className="mt-6 p-4 rounded-xl bg-[var(--color-bg-primary)] border border-[var(--color-border-custom)]">
                  <p className="text-xs font-mono uppercase tracking-wider text-[var(--color-text-secondary)] mb-1">
                    Посадка
                  </p>
                  <p className="text-sm text-[var(--color-text-primary)]">
                    {product.fit === 'slim' && 'Облегающий крой. Рекомендуем брать размер вверх, если хотите более свободную посадку.'}
                    {product.fit === 'regular' && 'Стандартный крой. Выбирайте свой обычный размер.'}
                    {product.fit === 'oversized' && 'Свободный оверсайз крой. Рекомендуем брать свой обычный размер для заявленного эффекта.'}
                    {product.fit === 'loose' && 'Свободный крой. Рекомендуем брать размер вниз для более прилегающей посадки.'}
                  </p>
                </div>
              )}

              {/* Model Info */}
              {product.modelInfo && (
                <div className="mt-4 p-4 rounded-xl bg-[var(--color-bg-primary)] border border-[var(--color-border-custom)]">
                  <p className="text-xs font-mono uppercase tracking-wider text-[var(--color-text-secondary)] mb-1">
                    {t('product.modelParams')}
                  </p>
                  <p className="text-sm text-[var(--color-text-primary)]">{product.modelInfo}</p>
                </div>
              )}

              <p className="mt-4 text-xs text-[var(--color-text-muted)]">
                Все замеры даны в {unit === 'cm' ? 'сантиметрах' : 'дюймах'}. Допускается отклонение ±1–2 {unit === 'cm' ? 'см' : 'дюйма'}.
              </p>
            </div>
          </motion.div>
        </motion.div>
      )}
    </AnimatePresence>
  );
}
