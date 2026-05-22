import { motion } from 'framer-motion';
import { Link } from 'react-router-dom';
import { ArrowRight } from 'lucide-react';
import { useTranslation } from 'react-i18next';

export default function CTASection() {
  const { t } = useTranslation();

  return (
    <section className="relative py-32 overflow-hidden">
      {/* Background decoration */}
      <div className="absolute inset-0 flex items-center justify-center pointer-events-none">
        <div className="h-[500px] w-[500px] rounded-full bg-[#44944A]/5 blur-[100px]" />
      </div>

      <div className="relative mx-auto max-w-4xl px-6 text-center lg:px-8">
        <motion.div
          initial={{ opacity: 0, y: 40 }}
          whileInView={{ opacity: 1, y: 0 }}
          viewport={{ once: true }}
          transition={{ duration: 0.8 }}
        >
          <p className="text-xs font-mono uppercase tracking-[0.3em] text-[#558b5c]">
            {t('home.cta.badge')}
          </p>
          <h2 className="mt-6 text-4xl font-bold tracking-tighter text-[var(--color-text-primary)] sm:text-6xl lg:text-7xl">
            {t('home.cta.title1')}
            <br />
            <span className="text-[#44944A]">{t('home.cta.title2')}</span>
          </h2>
          <p className="mx-auto mt-6 max-w-lg text-lg text-[var(--color-text-secondary)]">
            {t('home.cta.description')}
          </p>
        </motion.div>

        <motion.div
          initial={{ opacity: 0, y: 30 }}
          whileInView={{ opacity: 1, y: 0 }}
          viewport={{ once: true }}
          transition={{ duration: 0.8, delay: 0.2 }}
          className="mt-12"
        >
          <Link
            to="/avatar"
            className="group inline-flex items-center gap-3 rounded-full bg-white px-8 py-4 text-sm font-semibold text-black transition-all hover:shadow-[0_0_40px_rgba(255,255,255,0.2)]"
          >
            {t('home.cta.button')}
            <ArrowRight className="h-4 w-4 transition-transform group-hover:translate-x-1" />
          </Link>
        </motion.div>
      </div>
    </section>
  );
}
