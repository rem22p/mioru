import { motion } from 'framer-motion';
import type { Product } from '@/types';
import { AlertTriangle, X } from 'lucide-react';

interface DeleteDialogProps {
  product: Product;
  onCancel: () => void;
  onConfirm: () => void;
}

export default function DeleteDialog({ product, onCancel, onConfirm }: DeleteDialogProps) {
  return (
    <motion.div
      initial={{ opacity: 0 }}
      animate={{ opacity: 1 }}
      exit={{ opacity: 0 }}
      transition={{ duration: 0.2 }}
      className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-sm p-4"
      onClick={onCancel}
    >
      <motion.div
        initial={{ opacity: 0, scale: 0.95, y: 20 }}
        animate={{ opacity: 1, scale: 1, y: 0 }}
        exit={{ opacity: 0, scale: 0.95, y: 20 }}
        transition={{ duration: 0.3, ease: [0.16, 1, 0.3, 1] }}
        onClick={(e) => e.stopPropagation()}
        className="w-full max-w-md rounded-2xl bg-[var(--color-bg-card)] border border-[var(--color-border-custom)] p-6 shadow-2xl"
      >
        <div className="flex items-start justify-between mb-4">
          <div className="w-12 h-12 rounded-full bg-red-500/10 flex items-center justify-center">
            <AlertTriangle className="h-6 w-6 text-red-500" />
          </div>
          <button
            onClick={onCancel}
            className="text-[var(--color-text-muted)] hover:text-[var(--color-text-primary)] transition-colors"
          >
            <X className="h-5 w-5" />
          </button>
        </div>

        <h3 className="text-lg font-bold tracking-tighter text-[var(--color-text-primary)] mb-2">
          Удалить товар?
        </h3>
        <p className="text-sm text-[var(--color-text-secondary)] mb-6">
          Вы уверены, что хотите удалить{' '}
          <span className="font-semibold text-[var(--color-text-primary)]">
            &laquo;{product.name}&raquo;
          </span>
          ? Это действие нельзя будет отменить.
        </p>

        <div className="flex gap-3 justify-end">
          <button
            onClick={onCancel}
            className="rounded-xl bg-[var(--color-bg-primary)] border border-[var(--color-border-custom)] px-5 py-2.5 text-sm font-medium text-[var(--color-text-secondary)] hover:text-[var(--color-text-primary)] transition-colors"
          >
            Отмена
          </button>
          <button
            onClick={onConfirm}
            data-testid="product-delete-confirm"
            className="rounded-xl bg-red-500 px-5 py-2.5 text-sm font-semibold text-white hover:bg-red-600 transition-colors"
          >
            Удалить
          </button>
        </div>
      </motion.div>
    </motion.div>
  );
}
