import { useState, useMemo, useEffect } from "react";
import { motion, AnimatePresence } from "framer-motion";
import { Link } from "react-router-dom";
import { useCatalogStore } from "@/stores/catalogStore";
import { useCartStore } from "@/stores/cartStore";
import { getImageUrl } from "@/lib/api";
import { ShoppingBag, SlidersHorizontal, X, ImageIcon } from "lucide-react";
import { useTranslation } from "react-i18next";
import { Helmet } from "react-helmet-async";

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

export default function CatalogPage() {
  const { t } = useTranslation();
  const {
    products,
    categories,
    loading,
    error,
    fetchProducts,
    fetchCategories,
  } = useCatalogStore();

  const [selectedCategory, setSelectedCategory] = useState<string>("all");
  const [priceRange, setPriceRange] = useState<[number, number]>([0, 20000]);
  const [sortBy, setSortBy] = useState<"price-asc" | "price-desc" | "newest">(
    "newest",
  );
  const [mobileFiltersOpen, setMobileFiltersOpen] = useState(false);
  const addItem = useCartStore((state) => state.addItem);

  // Fetch data on mount
  useEffect(() => {
    fetchProducts();
    fetchCategories();
  }, [fetchProducts, fetchCategories]);

  // category_id → slug lookup
  const categorySlugById = useMemo(() => {
    const map = new Map<number, string>();
    const walk = (cats: typeof categories) => {
      for (const c of cats) {
        map.set(c.id, c.slug);
        if (c.children) walk(c.children);
      }
    };
    walk(categories);
    return map;
  }, [categories]);

  // Flatten category tree for filter buttons
  const flatCategories = useMemo(() => {
    const result: typeof categories = [];
    const walk = (cats: typeof categories) => {
      for (const c of cats) {
        result.push(c);
        if (c.children) walk(c.children);
      }
    };
    walk(categories);
    return result;
  }, [categories]);

  const filteredProducts = useMemo(() => {
    let result = products;
    if (selectedCategory !== "all") {
      const catId = flatCategories.find((c) => c.slug === selectedCategory)?.id;
      if (catId != null) {
        result = result.filter((p) => p.category_id === catId);
      }
    }
    result = result.filter(
      (p) => p.price >= priceRange[0] && p.price <= priceRange[1],
    );
    switch (sortBy) {
      case "price-asc":
        result = [...result].sort((a, b) => a.price - b.price);
        break;
      case "price-desc":
        result = [...result].sort((a, b) => b.price - a.price);
        break;
      case "newest":
        result = [...result].sort(
          (a, b) =>
            new Date(b.created_at).getTime() - new Date(a.created_at).getTime(),
        );
        break;
    }
    return result;
  }, [products, selectedCategory, priceRange, sortBy, flatCategories]);

  useEffect(() => {
    if (mobileFiltersOpen) {
      document.body.style.overflow = "hidden";
    } else {
      document.body.style.overflow = "";
    }
    return () => {
      document.body.style.overflow = "";
    };
  }, [mobileFiltersOpen]);

  const inputBaseClass =
    "w-full rounded-lg bg-[var(--color-bg-card)] border border-[var(--color-border-custom)] px-3 py-2 text-base sm:text-sm text-[var(--color-text-primary)] outline-none focus:border-[#44944A]";

  const FilterContent = () => (
    <div className="space-y-8">
      <div>
        <h3 className="text-xs font-semibold uppercase tracking-[0.2em] text-[var(--color-text-muted)]">
          {t("catalog.filters.categories")}
        </h3>
        <div className="mt-4 space-y-1">
          <button
            onClick={() => setSelectedCategory("all")}
            className={`block w-full text-left text-sm py-2 transition-colors ${
              selectedCategory === "all"
                ? "text-[#44944A]"
                : "text-[var(--color-text-secondary)] hover:text-[var(--color-text-primary)]"
            }`}
          >
            {t("catalog.filters.allCategories")}
          </button>
          {flatCategories.map((cat) => (
            <button
              key={cat.id}
              onClick={() => setSelectedCategory(cat.slug)}
              className={`block w-full text-left text-sm py-2 transition-colors ${
                selectedCategory === cat.slug
                  ? "text-[#44944A]"
                  : "text-[var(--color-text-secondary)] hover:text-[var(--color-text-primary)]"
              }`}
            >
              {cat.name}
            </button>
          ))}
        </div>
      </div>
      <div>
        <h3 className="text-xs font-semibold uppercase tracking-[0.2em] text-[var(--color-text-muted)]">
          {t("catalog.filters.price")}
        </h3>
        <div className="mt-4 flex items-center gap-3">
          <input
            type="number"
            value={priceRange[0]}
            onChange={(e) =>
              setPriceRange([Number(e.target.value), priceRange[1]])
            }
            className={inputBaseClass}
            placeholder={t("catalog.filters.from")}
          />
          <span className="text-[var(--color-text-muted)]">—</span>
          <input
            type="number"
            value={priceRange[1]}
            onChange={(e) =>
              setPriceRange([priceRange[0], Number(e.target.value)])
            }
            className={inputBaseClass}
            placeholder={t("catalog.filters.to")}
          />
        </div>
      </div>
    </div>
  );

  return (
    <div className="px-6 py-24 lg:px-8">
      <Helmet>
        <title>Каталог — MIORU | Кроссовки, футболки, шорты, браслеты</title>
        <meta
          name="description"
          content="Каталог одежды и аксессуаров MIORU. Кроссовки, тапки, футболки, шорты и браслеты с виртуальной 3D примеркой. Выбери свой стиль."
        />
        <meta property="og:title" content="Каталог — MIORU" />
        <meta
          property="og:description"
          content="Кроссовки, футболки, шорты и аксессуары с виртуальной примеркой."
        />
        <link rel="canonical" href="https://mioru.store/catalog" />
      </Helmet>
      <div className="mx-auto max-w-7xl">
        <motion.div
          initial={{ opacity: 0, y: 40 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: 0.8 }}
          className="mb-12"
        >
          <p className="text-xs font-mono uppercase tracking-[0.3em] text-[#558b5c]">
            {t("catalog.badge")}
          </p>
          <h1 className="mt-4 text-5xl font-bold tracking-tighter text-[var(--color-text-primary)] sm:text-6xl">
            {t("catalog.title")}
          </h1>
        </motion.div>

        {loading && products.length === 0 ? (
          <div className="flex items-center justify-center py-20">
            <div className="text-center">
              <div className="text-6xl mb-4 animate-pulse">📦</div>
              <p className="text-[var(--color-text-muted)] font-mono text-sm">
                {t("catalog.loading")}
              </p>
            </div>
          </div>
        ) : error ? (
          <div className="flex items-center justify-center py-20">
            <div className="text-center">
              <p className="text-red-400 font-mono text-sm">{error}</p>
            </div>
          </div>
        ) : (
          <div className="flex gap-8">
            <aside className="hidden w-64 shrink-0 md:block">
              <FilterContent />
            </aside>
            <div className="flex-1">
              <div className="mb-6 flex items-center justify-between">
                <p className="text-sm font-mono text-[var(--color-text-muted)]">
                  {t("catalog.count", { count: filteredProducts.length })}
                </p>
                <div className="flex items-center gap-4">
                  <select
                    value={sortBy}
                    onChange={(e) => setSortBy(e.target.value as typeof sortBy)}
                    className="rounded-lg bg-[var(--color-bg-card)] border border-[var(--color-border-custom)] px-3 py-2 text-base sm:text-sm text-[var(--color-text-primary)] outline-none focus:border-[#44944A]"
                  >
                    <option value="newest">{t("catalog.sortBy.newest")}</option>
                    <option value="price-asc">
                      {t("catalog.sortBy.priceAsc")}
                    </option>
                    <option value="price-desc">
                      {t("catalog.sortBy.priceDesc")}
                    </option>
                  </select>
                  <button
                    onClick={() => setMobileFiltersOpen(true)}
                    className="flex items-center gap-2 rounded-lg bg-[var(--color-bg-card)] border border-[var(--color-border-custom)] px-3 py-2 text-sm text-[var(--color-text-primary)] md:hidden min-h-[44px]"
                  >
                    <SlidersHorizontal className="h-4 w-4" />
                    {t("catalog.filters.title")}
                  </button>
                </div>
              </div>
              <div className="grid grid-cols-2 gap-4 sm:grid-cols-3 lg:grid-cols-4">
                {filteredProducts.map((product, index) => (
                  <motion.div
                    key={product.id}
                    initial={{ opacity: 0, y: 20 }}
                    animate={{ opacity: 1, y: 0 }}
                    transition={{ duration: 0.4, delay: index * 0.05 }}
                    className="group"
                  >
                    <Link to={`/product/${product.slug}`}>
                      <div className="card-hover relative aspect-[3/4] overflow-hidden rounded-xl bg-[var(--color-bg-card)] border border-[var(--color-border-custom)]">
                        {product.images?.[0]?.url ? (
                          <img
                            src={getImageUrl(product.images[0].url)}
                            alt={product.name}
                            className="absolute inset-0 w-full h-full object-cover transition-transform duration-500 group-hover:scale-105"
                            loading="lazy"
                          />
                        ) : (
                          <div className="absolute inset-0 flex items-center justify-center">
                            <ImageIcon className="h-12 w-12 text-[var(--color-text-muted)]" />
                          </div>
                        )}
                        <div className="absolute inset-0 bg-gradient-to-t from-[var(--color-bg-primary)] via-transparent to-transparent opacity-60" />
                        <button
                          onClick={(e) => {
                            e.preventDefault();
                            addItem(product, product.sizes[0]);
                          }}
                          className="absolute right-3 top-3 flex h-11 w-11 items-center justify-center rounded-full bg-[#44944A] opacity-0 transition-all duration-300 hover:scale-110 group-hover:opacity-100"
                          aria-label={t("home.featured.addToCart")}
                        >
                          <ShoppingBag className="h-4 w-4 text-black" />
                        </button>
                        <div className="absolute bottom-0 left-0 right-0 p-4">
                          <p className="text-[10px] font-mono uppercase tracking-wider text-[#558b5c]">
                            {product.category_name}
                          </p>
                          <h3 className="mt-1 text-sm font-medium text-[var(--color-text-primary)] line-clamp-1">
                            {product.name}
                          </h3>
                          <p className="mt-1 text-sm font-bold text-[#44944A]">
                            {product.price.toLocaleString("ru-RU")} ₽
                          </p>
                        </div>
                      </div>
                    </Link>
                  </motion.div>
                ))}
              </div>
            </div>
          </div>
        )}
      </div>

      <AnimatePresence>
        {mobileFiltersOpen && (
          <motion.div
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
            exit={{ opacity: 0 }}
            className="fixed inset-0 z-50 md:hidden"
          >
            <div
              className="absolute inset-0 bg-black/50"
              onClick={() => setMobileFiltersOpen(false)}
            />
            <motion.div
              initial={{ x: "100%" }}
              animate={{ x: 0 }}
              exit={{ x: "100%" }}
              transition={{ type: "tween", duration: 0.3 }}
              className="absolute right-0 top-0 h-full w-80 bg-[var(--color-bg-primary)] border-l border-[var(--color-border-custom)] p-6"
            >
              <div className="mb-6 flex items-center justify-between">
                <h2 className="text-lg font-semibold text-[var(--color-text-primary)]">
                  {t("catalog.filters.title")}
                </h2>
                <button
                  onClick={() => setMobileFiltersOpen(false)}
                  className="min-w-[44px] min-h-[44px] flex items-center justify-center rounded-lg"
                  aria-label={t("common.close")}
                >
                  <X className="h-6 w-6 text-[var(--color-text-secondary)]" />
                </button>
              </div>
              <FilterContent />
            </motion.div>
          </motion.div>
        )}
      </AnimatePresence>
    </div>
  );
}
