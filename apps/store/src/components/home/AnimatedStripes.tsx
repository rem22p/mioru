import { motion } from 'framer-motion';
import { useEffect, useRef, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Link } from 'react-router-dom';

export default function AnimatedStripes() {
  const { t } = useTranslation();
  const ref = useRef<HTMLDivElement>(null);
  const [visible, setVisible] = useState(false);

  useEffect(() => {
    const observer = new IntersectionObserver(
      ([entry]) => {
        if (entry.isIntersecting) setVisible(true);
      },
      { threshold: 0.3 }
    );
    if (ref.current) observer.observe(ref.current);
    return () => observer.disconnect();
  }, []);

  const items = t('home.about.items', { returnObjects: true }) as { label: string; desc: string; link: string }[];

  return (
    <section ref={ref} className="py-24 overflow-hidden">
      <div className="mx-auto max-w-7xl px-6 lg:px-8">
        <div className="mb-16">
          <p className="text-xs font-mono uppercase tracking-[0.3em] text-[#558b5c]">
            {t('home.about.badge')}
          </p>
          <motion.h2
            initial={{ opacity: 0, y: 40 }}
            whileInView={{ opacity: 1, y: 0 }}
            viewport={{ once: true }}
            transition={{ duration: 0.8 }}
            className="mt-4 text-4xl font-bold tracking-tighter text-[var(--color-text-primary)] sm:text-6xl"
          >
            {t('home.about.title')}
          </motion.h2>
        </div>

        {/* Animated stripes */}
        <div className="space-y-4">
          {items.map((item, index) => (
            <motion.div
              key={item.label}
              initial={{ opacity: 0, x: -40 }}
              whileInView={{ opacity: 1, x: 0 }}
              viewport={{ once: true }}
              transition={{ duration: 0.6, delay: index * 0.15 }}
              className="group relative"
            >
              <Link
                to={item.link}
                className="flex items-center justify-between border-b border-[var(--color-border-custom)] py-6 transition-colors hover:border-[#44944A] cursor-pointer"
              >
                <div className="flex items-center gap-6">
                  <span className="text-xs font-mono text-[var(--color-text-muted)]">
                    0{index + 1}
                  </span>
                  <h3 className={`text-xl font-semibold transition-colors group-hover:text-[#44944A] ${
                    item.label.includes("скоро") || item.label.includes("coming") || item.label.includes("curând")
                      ? "text-[#44944A]"
                      : "text-[var(--color-text-primary)]"
                  }`}>
                    {item.label}
                  </h3>
                </div>
                <p className="hidden text-sm text-[var(--color-text-muted)] sm:block">
                  {item.desc}
                </p>
              </Link>
              {/* Hover fill */}
              <div className="absolute bottom-0 left-0 h-px w-0 bg-[#44944A] transition-all duration-500 group-hover:w-full" />
            </motion.div>
          ))}
        </div>
      </div>
    </section>
  );
}
