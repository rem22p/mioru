import { motion } from 'framer-motion';
import { LucideIcon } from 'lucide-react';

interface PlaceholderProps {
  icon: LucideIcon;
  title: string;
  desc: string;
}

export default function Placeholder({ icon: Icon, title, desc }: PlaceholderProps) {
  return (
    <motion.div
      initial={{ opacity: 0, y: 40 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ duration: 0.8, ease: [0.16, 1, 0.3, 1] }}
      className="flex flex-col items-center justify-center h-full text-center p-8 min-h-[60vh]"
    >
      <div className="w-20 h-20 rounded-2xl bg-[var(--color-bg-card)] border border-[var(--color-border-custom)] flex items-center justify-center mb-6">
        <Icon className="h-10 w-10 text-[var(--color-text-muted)]" />
      </div>
      <h2 className="text-xl font-bold tracking-tighter text-[var(--color-text-primary)] mb-2">
        {title}
      </h2>
      <p className="text-sm text-[var(--color-text-secondary)]">{desc}</p>
    </motion.div>
  );
}
