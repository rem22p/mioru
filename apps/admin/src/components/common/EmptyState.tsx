import { motion } from 'framer-motion';
import { FolderOpen } from 'lucide-react';

export default function EmptyState() {
  return (
    <motion.div
      initial={{ opacity: 0, y: 40 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ duration: 0.8, ease: [0.16, 1, 0.3, 1] }}
      className="flex flex-col items-center justify-center h-full text-center p-8"
    >
      <div className="w-20 h-20 rounded-2xl bg-[var(--color-bg-card)] border border-[var(--color-border-custom)] flex items-center justify-center mb-6">
        <FolderOpen className="h-10 w-10 text-[var(--color-text-muted)]" />
      </div>
      <h2 className="text-xl font-bold tracking-tighter text-[var(--color-text-primary)] mb-2">
        Выбери рабочее пространство
      </h2>
      <p className="text-sm text-[var(--color-text-secondary)]">
        Выбери раздел в меню слева, чтобы начать работу
      </p>
    </motion.div>
  );
}
