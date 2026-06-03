import { Link } from "react-router-dom";
import { motion } from "framer-motion";
import { Heart, Trash2, ArrowRight } from "lucide-react";
import { useFavoritesStore } from "@/stores/favoritesStore";
import { useCurrencyStore } from "@/stores/currencyStore";
import { formatPrice } from "@/lib/currency";
import { useTranslation } from "react-i18next";
import { Helmet } from "react-helmet-async";

export default function FavoritesPage() {
  const { t } = useTranslation();
  const { currency } = useCurrencyStore();
  const items = useFavoritesStore((s) => s.items);
  const removeFavorite = useFavoritesStore((s) => s.removeFavorite);

  if (items.length === 0) {
    return (
      <div className="px-6 py-24 lg:px-8">
        <Helmet>
          <title>{t("nav.favorites")} — MIORU</title>
        </Helmet>
        <div className="flex flex-col items-center justify-center min-h-[40vh] text-center">
          <Heart className="h-16 w-16 text-[var(--color-border-light)] mb-6" />
          <h1 className="text-2xl font-bold text-[var(--color-text-primary)]">
            {t("nav.favorites")}
          </h1>
          <p className="mt-2 text-[var(--color-text-secondary)] text-sm">
            {t("favorites.empty")}
          </p>
          <Link
            to="/catalog"
            className="mt-6 inline-flex items-center gap-2 rounded-xl bg-[#44944A] px-6 py-3 text-sm font-semibold text-black transition-all hover:shadow-[0_0_30px_rgba(68,148,74,0.3)]"
          >
            {t("cart.toCatalog")}
            <ArrowRight className="h-4 w-4" />
          </Link>
        </div>
      </div>
    );
  }

  return (
    <div className="px-6 py-24 lg:px-8">
      <Helmet>
        <title>{t("nav.favorites")} — MIORU</title>
      </Helmet>
      <div className="mx-auto max-w-7xl">
        <motion.h1
          initial={{ opacity: 0, y: 20 }}
          animate={{ opacity: 1, y: 0 }}
          className="text-4xl font-bold tracking-tighter text-[var(--color-text-primary)] sm:text-5xl mb-10"
        >
          {t("nav.favorites")}
        </motion.h1>

        <div className="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 gap-4">
          {items.map((item, index) => (
            <motion.div
              key={item.id}
              initial={{ opacity: 0, y: 20 }}
              animate={{ opacity: 1, y: 0 }}
              transition={{ duration: 0.4, delay: index * 0.05 }}
              className="group"
            >
              <div className="card-hover overflow-hidden rounded-xl bg-[var(--color-bg-card)] border border-[var(--color-border-custom)]">
                <Link to={`/product/${item.slug}`}>
                  <div className="relative aspect-[4/5] overflow-hidden">
                    {item.imageUrl ? (
                      <img
                        src={item.imageUrl}
                        alt={item.name}
                        className="absolute inset-0 w-full h-full object-cover transition-transform duration-500 group-hover:scale-105"
                        loading="lazy"
                      />
                    ) : (
                      <div className="absolute inset-0 flex items-center justify-center">
                        <span className="text-3xl sm:text-5xl transition-transform duration-500 group-hover:scale-110">
                          📦
                        </span>
                      </div>
                    )}
                    <div className="absolute inset-0 bg-gradient-to-t from-[var(--color-bg-primary)] via-transparent to-transparent opacity-60 pointer-events-none" />

                    {/* Delete button — top right */}
                    <button
                      onClick={(e) => {
                        e.preventDefault();
                        e.stopPropagation();
                        removeFavorite(item.id);
                      }}
                      className="absolute top-3 right-3 flex h-9 w-9 items-center justify-center rounded-full bg-[var(--color-bg-primary)]/70 border border-[var(--color-border-custom)] text-[var(--color-text-muted)] hover:text-red-500 hover:border-red-500/50 hover:bg-[var(--color-bg-primary)] transition-all opacity-0 group-hover:opacity-100"
                      aria-label={t("common.delete")}
                    >
                      <Trash2 className="h-4 w-4" />
                    </button>
                  </div>
                </Link>

                <div className="px-3 py-2.5 border-t border-[var(--color-border-custom)]">
                  <p className="text-[11px] font-mono uppercase tracking-wider text-[#558b5c]">
                    {item.category_name}
                  </p>
                  <h3 className="mt-0.5 text-sm font-medium text-[var(--color-text-primary)] line-clamp-1">
                    {item.name}
                  </h3>
                  <p className="mt-1 text-sm font-bold text-[#44944A]">
                    {formatPrice(item.price, currency)}
                  </p>
                </div>
              </div>
            </motion.div>
          ))}
        </div>
      </div>
    </div>
  );
}
