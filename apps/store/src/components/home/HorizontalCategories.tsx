import { useEffect, useState } from "react";
import { motion } from "framer-motion";
import { Link } from "react-router-dom";
import { useCatalogStore } from "@/stores/catalogStore";
import { fetchStoreProducts, getThumbUrl } from "@/lib/api";
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

export default function HorizontalCategories() {
  const { t } = useTranslation();
  const { categories } = useCatalogStore();
  const [categoryImages, setCategoryImages] = useState<Record<number, string>>({});

  // Top-level categories only
  const topCats = categories;

  // Fetch top product image per category (max stock).
  useEffect(() => {
    let cancelled = false;
    async function load() {
      const images: Record<number, string> = {};
      await Promise.all(
        topCats.map(async (cat) => {
          try {
            const res = await fetchStoreProducts({ category_id: String(cat.id), per_page: "20" });
            let best = null as { stock: number; url: string } | null;
            for (const p of res.products) {
              if (p.images?.[0]?.url) {
                const stock = p.stock_quantity || 0;
                if (!best || stock > best.stock) {
                  best = { stock, url: getThumbUrl(p.images[0].url) };
                }
              }
            }
            if (best) images[cat.id] = best.url;
          } catch {
            // best-effort: keep emoji fallback
          }
        }),
      );
      if (!cancelled) setCategoryImages(images);
    }
    if (topCats.length > 0) load();
    return () => { cancelled = true; };
  }, [topCats]);

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
        {topCats.map((category, index) => {
          const imageUrl = categoryImages[category.id];
          return (
            <motion.div
              key={category.id}
              initial={{ opacity: 0, y: 40 }}
              whileInView={{ opacity: 1, y: 0 }}
              viewport={{ once: true }}
              transition={{ duration: 0.6, delay: index * 0.1 }}
            >
              <Link to={`/catalog/${category.slug}`}>
                <div className="card-hover group relative h-[400px] w-[300px] overflow-hidden rounded-2xl bg-[var(--color-bg-card)] border border-[var(--color-border-custom)]">
                  {/* Product image or emoji fallback */}
                  {imageUrl ? (
                    <img
                      src={imageUrl}
                      alt={category.name}
                      className="absolute inset-0 h-full w-full object-cover transition-transform duration-700 group-hover:scale-105"
                      loading="lazy"
                      onError={(e) => {
                        // Hide broken image, show emoji underneath.
                        e.currentTarget.style.display = "none";
                      }}
                    />
                  ) : null}
                  <div className="absolute inset-0 flex items-center justify-center">
                    <span className="text-8xl opacity-30 transition-transform duration-500 group-hover:scale-110">
                      {categoryEmoji(category.slug)}
                    </span>
                  </div>

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
        })}
      </div>
    </section>
  );
}
