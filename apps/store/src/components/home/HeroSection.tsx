import { motion } from "framer-motion";
import { Link } from "react-router-dom";
import { ArrowRight } from "lucide-react";
import { useTranslation } from "react-i18next";

export default function HeroSection() {
  const { t } = useTranslation();

  return (
    <section className="relative flex min-h-screen items-center justify-center overflow-hidden px-6">
      <div className="mx-auto max-w-7xl text-center">
        <motion.div
          initial={{ opacity: 0, y: 60 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: 1, ease: [0.16, 1, 0.3, 1] }}
        >
          <p className="mb-6 text-xs font-mono uppercase tracking-[0.3em] text-[#558b5c]">
            {t("home.hero.badge")}
          </p>
          <h1 className="text-5xl font-bold tracking-tighter text-[var(--color-text-primary)] sm:text-7xl lg:text-8xl">
            {t("home.hero.title1")}
            <br />
            <span className="text-[#44944A]">{t("home.hero.title2")}</span>
          </h1>
        </motion.div>

        <motion.p
          initial={{ opacity: 0, y: 40 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: 1, delay: 0.2, ease: [0.16, 1, 0.3, 1] }}
          className="mx-auto mt-8 max-w-xl text-lg leading-relaxed text-[var(--color-text-secondary)]"
        >
          {t("home.hero.description")}
        </motion.p>

        <motion.div
          initial={{ opacity: 0, y: 40 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: 1, delay: 0.4, ease: [0.16, 1, 0.3, 1] }}
          className="mt-12 flex flex-col items-center gap-4 sm:flex-row sm:justify-center"
        >
          <Link
            to="/catalog"
            className="group relative overflow-hidden rounded-full bg-[#44944A] px-8 py-4 text-sm font-semibold text-black transition-all hover:shadow-[0_0_40px_rgba(192,254,57,0.3)]"
          >
            <span className="relative z-10 flex items-center gap-2">
              {t("home.hero.cta1")}
              <ArrowRight className="h-4 w-4 transition-transform group-hover:translate-x-1" />
            </span>
          </Link>
          <Link
            to="/avatar"
            className="rounded-full border border-[var(--color-border-light)] px-8 py-4 text-sm font-medium text-[var(--color-text-primary)] transition-all hover:border-[#44944A] hover:text-[#44944A]"
          >
            {t("home.hero.cta2")}
          </Link>
        </motion.div>

        {/* Stats */}
        <motion.div
          initial={{ opacity: 0 }}
          animate={{ opacity: 1 }}
          transition={{ duration: 1, delay: 0.6 }}
          className="mt-20 flex items-center justify-center gap-12"
        >
          {[
            { value: "5", label: t("home.hero.stats.categories") },
            { value: "300+", label: t("home.hero.stats.products") },
            { value: "3D", label: t("home.hero.stats.tryOn") },
          ].map((stat) => (
            <div key={stat.label} className="text-center">
              <div className="text-3xl font-bold text-[var(--color-text-primary)]">
                {stat.value}
              </div>
              <div className="mt-1 text-xs font-mono uppercase tracking-wider text-[var(--color-text-muted)]">
                {stat.label}
              </div>
            </div>
          ))}
        </motion.div>
      </div>
    </section>
  );
}
