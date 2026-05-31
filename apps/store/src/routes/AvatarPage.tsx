import { useTranslation } from "react-i18next";
import { motion } from "framer-motion";
import { Construction } from "lucide-react";

export default function AvatarPage() {
  const { t } = useTranslation();

  return (
    <div className="px-6 py-24 lg:px-8">
      <div className="mx-auto max-w-7xl">
        <motion.div
          initial={{ opacity: 0, y: 40 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: 0.8 }}
          className="flex flex-col items-center justify-center min-h-[50vh] text-center"
        >
          <Construction className="h-16 w-16 text-[#44944A] mb-6" />
          <h1 className="text-3xl font-bold tracking-tighter text-[var(--color-text-primary)] mb-3">
            {t("avatar.title")}
          </h1>
          <p className="text-[var(--color-text-muted)] font-mono text-sm">
            {t("common.comingSoon")}
          </p>
        </motion.div>
      </div>
    </div>
  );
}
