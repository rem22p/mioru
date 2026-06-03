import { useState } from "react";
import { motion } from "framer-motion";
import { Link } from "react-router-dom";
import { useCatalogStore } from "@/stores/catalogStore";
import { getThumbUrl } from "@/lib/api";
import { ArrowUpRight } from "lucide-react";
import { useTranslation } from "react-i18next";

function categoryEmoji(slug: string): string {
  const map: Record<string, string> = {
    sneakers: "👟",
    slides: "🩴",
    tshirts: "👕",
    shorts: "🩳",
    bracelets: "⛓️",
  };
  return map[slug] || "📦";
}

function CategoryCard({ category, index }: { category: { id: number; name: string; slug: string; cover_image?: string | null }; index: number }) {
  const [imgError, setImgError] = useState(false);
  const imageUrl = category.cover_image ? getThumbUrl(category.cover_image) : null;
  const showEmoji = !imageUrl || imgError;

  return (
    <motion.div
      initial={{ opacity: 0, y: 40 }}
      whileInView={{ opacity: 1, y: 0 }}
      viewport={{ once: true }}
      transition={{ duration: 0.6, delay: index * 0.1 }}
    >
      <Link to={`/catalog/${category.slug}`}>
        <div className="card-hover group relative h-[400px] w-[300px] overflow-hidden rounded-2xl bg-[var(--color-bg-secondary)] border border-[var(--color-border-custom)]">
          {/* Product image */}
          {imageUrl && !imgError && (
            <img
              src={imageUrl}
              alt={category.name}
              className="absolute inset-0 h-full w-full object-cover transition-transform duration-700 group-hover:scale-105"
              loading="lazy"
              onError={() => setImgError(true)}
            />
          )}

          {/* Emoji fallback — only when no image or image failed */}
          {showEmoji && (
            <div className="absolute inset-0 flex items-center justify-center">
              <span className="text-8xl opacity-30 transition-transform duration-500 group-hover:scale-110">
                {categoryEmoji(category.slug)}
              </span>
            </div>
          )}

          {/* Background gradient */}
          <div className="absolute inset-0 bg-gradient-to-t from-black/60 via-transparent to-transparent" />

          {/* Content */}
          <div className="absolute bottom-0 left-0 right-0 p-6">
            <div className="flex items-center justify-between">
              <div>
                <h3 className="text-xl font-bold text-[var(--color-text-primary)]">
                  {category.name}
                </h3>
                <p className="mt-1 text-xs font-mono text-[var(--color-text-muted)]">
                  {category.slug}
                </p>
              </div>
              <div className="flex h-10 w-10 items-center justify-center rounded-full border border-[var(--color-border-light)] transition-all group-hover:border-[#44944A] group-hover:bg-[#44944A]">
                <ArrowUpRight className="h-4 w-4 text-[var(--color-text-secondary)] transition-colors group-hover:text-black" />
              </div>
            </div>
          </div>

          {/* Hover line */}
          <div className="absolute bottom-0 left-0 h-1 w-0 bg-[#44944A] transition-all duration-500 group-hover:w-full" />
        </div>
      </Link>
    </motion.div>
  );
}

export default function HorizontalCategories() {
  const { t } = useTranslation();
  const { categories } = useCatalogStore();

  // Top-level categories only
  const topCats = categories;

  return (
    <section className="relative py-24">
      {/* Section header */}
      <div className="mx-auto max-w-7xl px-6 lg:px-8">
        <motion.div
          initial={{ opacity: 0, y: 40 }}
          whileInView={{ opacity: 1, y: 0 }}
          viewport={{ once: true }}
          transition={{ duration: 0.8 }}
          className="mb-12 flex items-end justify-between"
        >
          <div>
            <p className="text-xs font-mono uppercase tracking-[0.3em] text-[#558b5c]">
              {t("home.categories.badge")}
            </p>
            <h2 className="mt-4 text-4xl font-bold tracking-tighter text-[var(--color-text-primary)] sm:text-5xl">
              {t("home.categories.title")}
            </h2>
          </div>
          <Link
            to="/catalog"
            className="hidden text-sm text-[var(--color-text-secondary)] transition-colors hover:text-[#44944A] sm:block"
          >
            {t("home.categories.allProducts")}
          </Link>
        </motion.div>
      </div>

      {/* Horizontal scroll */}
      <div className="horizontal-scroll mx-auto max-w-7xl px-6 pb-4 lg:px-8 pt-2">
        {topCats.map((category, index) => (
          <CategoryCard key={category.id} category={category} index={index} />
        ))}
      </div>
    </section>
  );
}
