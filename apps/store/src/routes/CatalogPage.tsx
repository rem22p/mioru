import { useState, useMemo, useEffect } from "react";
import { motion, AnimatePresence } from "framer-motion";
import { Link, useParams } from "react-router-dom";
import { useCatalogStore } from "@/stores/catalogStore";
import { useCurrencyStore } from "@/stores/currencyStore";
import { formatPrice } from "@/lib/currency";
import { useCartStore } from "@/stores/cartStore";
import { getThumbUrl, getImageUrl } from "@/lib/api";
import { ShoppingBag, ChevronDown } from "lucide-react";
import { useTranslation } from "react-i18next";
import { Helmet } from "react-helmet-async";

const PER_PAGE = 20;

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
  const { currency } = useCurrencyStore();
  const {
    products,
    categories,
    facets,
    total,
    loading,
    error,
    fetchProducts,
    fetchFacets,
    fetchCategories,
  } = useCatalogStore();

  const [selectedCategory, setSelectedCategory] = useState<string>("all");
  const { categorySlug } = useParams<{ categorySlug?: string }>();
  const [priceMin, setPriceMin] = useState("");
  const [priceMax, setPriceMax] = useState("");
  const [selectedBrands, setSelectedBrands] = useState<Set<string>>(new Set());
  const [selectedColors, setSelectedColors] = useState<Set<string>>(new Set());
  const [selectedSizes, setSelectedSizes] = useState<Set<string>>(new Set());
  const [sortBy, setSortBy] = useState<"price-asc" | "price-desc" | "newest">(
    "newest",
  );
  const [dynamicFiltersOpen, setDynamicFiltersOpen] = useState(false);
  const [filterSubsectionsOpen, setFilterSubsectionsOpen] = useState({
    sizes: false,
    brands: false,
    colors: false,
  });
  const [page, setPage] = useState(1);
  const addItem = useCartStore((state) => state.addItem);

  const handleCategoryChange = (slug: string) => {
    setSelectedCategory(slug);
    setSelectedBrands(new Set());
    setSelectedColors(new Set());
    setSelectedSizes(new Set());
    setPriceMin("");
    setPriceMax("");
    setPage(1);
  };

  const resetFilters = () => {
    setSelectedBrands(new Set());
    setSelectedColors(new Set());
    setSelectedSizes(new Set());
    setPriceMin("");
    setPriceMax("");
    setPage(1);
  };

  // Bootstrap: load category tree once. Products + facets are loaded by the
  // filter-driven effects below.
  useEffect(() => {
    fetchCategories();
  }, [fetchCategories]);

  // When navigated from homepage with a category slug, select it
  useEffect(() => {
    if (categorySlug && categories.length > 0) {
      // Check if the slug exists in the category tree
      const walk = (cats: typeof categories): boolean => {
        for (const c of cats) {
          if (c.slug === categorySlug) return true;
          if (c.children && walk(c.children)) return true;
        }
        return false;
      };
      if (walk(categories)) {
        setSelectedCategory(categorySlug);
      }
    }
  }, [categorySlug, categories]);

  // category_id → slug lookup (used for product cards when no image is set).
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

  // For each category id, collect the set of descendant ids (including itself).
  // When the user picks a parent category, the storefront sends all leaf ids so
  // the backend returns products attached to any of them.
  const descendantIdsBySlug = useMemo(() => {
    const map = new Map<string, number[]>();
    const collect = (node: (typeof categories)[number]): number[] => {
      const ids = [node.id];
      if (node.children) {
        for (const ch of node.children) ids.push(...collect(ch));
      }
      return ids;
    };
    const walk = (cats: typeof categories) => {
      for (const c of cats) {
        map.set(c.slug, collect(c));
        if (c.children) walk(c.children);
      }
    };
    walk(categories);
    return map;
  }, [categories]);

  const categoryIds = useMemo(() => {
    if (selectedCategory === "all") return [];
    return (descendantIdsBySlug.get(selectedCategory) ?? []).map(String);
  }, [selectedCategory, descendantIdsBySlug]);

  const sortParam = useMemo(() => {
    switch (sortBy) {
      case "price-asc":
        return "price";
      case "price-desc":
        return "-price";
      case "newest":
      default:
        return "-created_at";
    }
  }, [sortBy]);

  // Stable keys for the Set-based filter state so deps arrays compare by value.
  const brandKey = useMemo(
    () => [...selectedBrands].sort().join("|"),
    [selectedBrands],
  );
  const colorKey = useMemo(
    () => [...selectedColors].sort().join("|"),
    [selectedColors],
  );
  const sizeKey = useMemo(
    () => [...selectedSizes].sort().join("|"),
    [selectedSizes],
  );
  const categoryIdsKey = categoryIds.join(",");

  // Fetch products whenever any filter, sort, or page changes.
  useEffect(() => {
    fetchProducts({
      category_id: categoryIds.length > 0 ? categoryIds : undefined,
      brand: brandKey ? brandKey.split("|") : undefined,
      color: colorKey ? colorKey.split("|") : undefined,
      size: sizeKey ? sizeKey.split("|") : undefined,
      price_min: priceMin || undefined,
      price_max: priceMax || undefined,
      sort: sortParam,
      page: String(page),
      per_page: String(PER_PAGE),
    });
  }, [
    fetchProducts,
    categoryIdsKey,
    categoryIds,
    brandKey,
    colorKey,
    sizeKey,
    priceMin,
    priceMax,
    sortParam,
    page,
  ]);

  // Facets follow the scope (category + price + search) only — brand/color/size
  // selections do NOT narrow the facet lists, otherwise picking one brand would
  // hide every other brand from the UI.
  useEffect(() => {
    fetchFacets({
      category_id: categoryIds.length > 0 ? categoryIds : undefined,
      price_min: priceMin || undefined,
      price_max: priceMax || undefined,
    });
  }, [fetchFacets, categoryIdsKey, categoryIds, priceMin, priceMax]);

  const totalPages = Math.max(1, Math.ceil(total / PER_PAGE));
  useEffect(() => {
    if (page > totalPages) setPage(totalPages);
  }, [totalPages, page]);

  const availableFilters = useMemo(
    () => ({
      brands: facets.brands,
      colors: facets.colors,
      sizes: [...facets.sizes].sort((a, b) => {
        const order = [
          "XXS", "XS", "S", "M", "L", "XL", "XXL", "XXXL",
        ];
        const ai = order.indexOf(a);
        const bi = order.indexOf(b);
        // Both are named sizes — use predefined order
        if (ai !== -1 && bi !== -1) return ai - bi;
        // Only one is named — put it first
        if (ai !== -1) return -1;
        if (bi !== -1) return 1;
        // Both numeric — compare as numbers
        const na = parseInt(a);
        const nb = parseInt(b);
        if (!isNaN(na) && !isNaN(nb)) return na - nb;
        // One numeric, other is "One size" etc. — numeric first
        if (!isNaN(na)) return -1;
        if (!isNaN(nb)) return 1;
        // Fallback
        return a.localeCompare(b);
      }),
    }),
    [facets],
  );

  const toggleFilter = (
    setter: React.Dispatch<React.SetStateAction<Set<string>>>,
    value: string,
  ) => {
    setter((prev) => {
      const next = new Set(prev);
      if (next.has(value)) next.delete(value);
      else next.add(value);
      return next;
    });
    setPage(1);
  };

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
          <div>
            {/* Filter bar */}
            <motion.div
              initial={{ opacity: 0, y: 20 }}
              animate={{ opacity: 1, y: 0 }}
              transition={{ duration: 0.6, delay: 0.1 }}
              className="mb-8"
            >
              {/* Category chips — horizontal scroll, wrap on mobile */}
              <div className="flex gap-2 overflow-x-auto pb-2 scrollbar-none flex-wrap sm:flex-nowrap">
                <button
                  onClick={() => handleCategoryChange("all")}
                  className={`shrink-0 px-4 py-2 rounded-full text-sm font-medium transition-all ${
                    selectedCategory === "all"
                      ? "bg-[var(--color-text-primary)] text-[var(--color-bg-primary)]"
                      : "bg-[var(--color-bg-card)] text-[var(--color-text-secondary)] border border-[var(--color-border-custom)] hover:text-[var(--color-text-primary)]"
                  }`}
                >
                  {t("catalog.filters.allCategories")}
                </button>
                {categories
                  .filter((c) => !c.parent_id)
                  .map((cat) => {
                    const isActive =
                      selectedCategory === cat.slug ||
                      cat.children?.some((ch) => selectedCategory === ch.slug);
                    return (
                      <button
                        key={cat.id}
                        onClick={() => handleCategoryChange(cat.slug)}
                        className={`shrink-0 px-4 py-2 rounded-full text-sm font-medium transition-all ${
                          isActive
                            ? "bg-[#44944A] text-black shadow-[0_0_20px_rgba(68,148,74,0.4)]"
                            : "bg-[var(--color-bg-card)] text-[var(--color-text-secondary)] border border-[var(--color-border-custom)] hover:text-[var(--color-text-primary)]"
                        }`}
                      >
                        {cat.name}
                      </button>
                    );
                  })}
              </div>

              {/* Subcategory chips — show when parent selected */}
              {selectedCategory !== "all" &&
                (() => {
                  const parent = categories.find(
                    (c) =>
                      !c.parent_id &&
                      c.children?.some(
                        (ch) =>
                          selectedCategory === ch.slug ||
                          c.slug === selectedCategory,
                      ),
                  );
                  const subCats = parent?.children || [];
                  if (subCats.length === 0) return null;
                  return (
                    <div className="flex gap-2 overflow-x-auto pb-2 scrollbar-none flex-wrap sm:flex-nowrap">
                      <span className="shrink-0 self-center text-xs text-[var(--color-text-muted)]">
                        Подкатегории:
                      </span>
                        <button
                          onClick={() => handleCategoryChange(parent!.slug)}
                          className={`shrink-0 px-3 py-1.5 rounded-full text-xs font-medium transition-all ${
                            selectedCategory === parent!.slug
                              ? "bg-[#44944A] text-black shadow-[0_0_15px_rgba(68,148,74,0.3)]"
                              : "bg-[var(--color-bg-card)] text-[var(--color-text-secondary)] border border-[var(--color-border-custom)] hover:text-[var(--color-text-primary)]"
                          }`}
                        >
                          Все
                        </button>
                        {subCats.map((ch) => (
                          <button
                            key={ch.id}
                            onClick={() => handleCategoryChange(ch.slug)}
                            className={`shrink-0 px-3 py-1.5 rounded-full text-xs font-medium transition-all ${
                              selectedCategory === ch.slug
                                ? "bg-[var(--color-text-primary)] text-[var(--color-bg-primary)] shadow-[0_0_15px_rgba(255,255,255,0.12)]"
                                : "bg-[var(--color-bg-card)] text-[var(--color-text-secondary)] border border-[var(--color-border-custom)] hover:text-[var(--color-text-primary)]"
                            }`}
                          >
                            {ch.name}
                          </button>
                        ))}
                      </div>
                  );
                })()}

              {/* Dynamic filters section — only when facets are available */}
              {selectedCategory !== "all" && (availableFilters.sizes.length > 0 || availableFilters.brands.length > 0 || availableFilters.colors.length > 0) && (
                <div className="mt-4">
                  <button
                    onClick={() => setDynamicFiltersOpen(!dynamicFiltersOpen)}
                    className="w-full flex items-center justify-between rounded-xl bg-[var(--color-bg-card)] border border-[var(--color-border-custom)] px-5 py-3 text-sm font-semibold uppercase tracking-wider text-[var(--color-text-primary)] hover:border-[#44944A]/50 transition-colors"
                  >
                    Фильтры
                    <ChevronDown className={`h-4 w-4 transition-transform duration-300 ${dynamicFiltersOpen ? "rotate-180" : ""}`} />
                  </button>
                  {dynamicFiltersOpen && (
                <AnimatePresence mode="wait">
                  <motion.div
                    key={selectedCategory}
                    initial={{ opacity: 0, height: 0 }}
                    animate={{ opacity: 1, height: "auto" }}
                    exit={{ opacity: 0, height: 0 }}
                    className="mb-6 overflow-hidden"
                  >
                    <div className="rounded-2xl bg-[var(--color-bg-card)] border border-[var(--color-border-custom)] p-5 space-y-5 mt-3">
                        <div className="space-y-5">
                          {/* Sizes */}
                          {availableFilters.sizes.length > 0 && (
                            <div>
                              <button
                                onClick={() => setFilterSubsectionsOpen(prev => ({ ...prev, sizes: !prev.sizes }))}
                                className="w-full flex items-center justify-between text-xs font-semibold text-[var(--color-text-primary)] mb-2 hover:text-[#44944A] transition-colors"
                              >
                                Размер
                                <ChevronDown className={`h-3.5 w-3.5 transition-transform duration-300 ${filterSubsectionsOpen.sizes ? "rotate-180" : ""}`} />
                              </button>
                              {filterSubsectionsOpen.sizes && (
                              <div className="flex flex-wrap gap-2">
                                {availableFilters.sizes.map((s) => (
                                  <button
                                    key={s}
                                    onClick={() =>
                                      toggleFilter(setSelectedSizes, s)
                                    }
                                    className={`px-4 py-2 rounded-xl text-sm font-medium transition-all border ${
                                      selectedSizes.has(s)
                                        ? "bg-[#44944A] text-black border-[#44944A]"
                                        : "bg-[var(--color-bg-primary)] text-[var(--color-text-secondary)] border-[var(--color-border-custom)] hover:border-[var(--color-text-muted)] hover:text-[var(--color-text-primary)]"
                                    }`}
                                  >
                                    {s}
                                  </button>
                                ))}
                              </div>
                              )}
                            </div>
                          )}

                          {/* Brands */}
                          {availableFilters.brands.length > 0 && (
                            <div>
                              <button
                                onClick={() => setFilterSubsectionsOpen(prev => ({ ...prev, brands: !prev.brands }))}
                                className="w-full flex items-center justify-between text-xs font-semibold text-[var(--color-text-primary)] mb-2 hover:text-[#44944A] transition-colors"
                              >
                                Бренд
                                <ChevronDown className={`h-3.5 w-3.5 transition-transform duration-300 ${filterSubsectionsOpen.brands ? "rotate-180" : ""}`} />
                              </button>
                              {filterSubsectionsOpen.brands && (
                              <div className="flex flex-wrap gap-2">
                                {availableFilters.brands.map((b) => (
                                  <button
                                    key={b}
                                    onClick={() =>
                                      toggleFilter(setSelectedBrands, b)
                                    }
                                    className={`px-4 py-2 rounded-xl text-sm font-medium transition-all border ${
                                      selectedBrands.has(b)
                                        ? "bg-[#44944A] text-black border-[#44944A]"
                                        : "bg-[var(--color-bg-primary)] text-[var(--color-text-secondary)] border-[var(--color-border-custom)] hover:border-[var(--color-text-muted)] hover:text-[var(--color-text-primary)]"
                                    }`}
                                  >
                                    {b}
                                  </button>
                                ))}
                              </div>
                              )}
                            </div>
                          )}

                          {/* Colors */}
                          {availableFilters.colors.length > 0 && (
                            <div>
                              <button
                                onClick={() => setFilterSubsectionsOpen(prev => ({ ...prev, colors: !prev.colors }))}
                                className="w-full flex items-center justify-between text-xs font-semibold text-[var(--color-text-primary)] mb-2 hover:text-[#44944A] transition-colors"
                              >
                                Цвет
                                <ChevronDown className={`h-3.5 w-3.5 transition-transform duration-300 ${filterSubsectionsOpen.colors ? "rotate-180" : ""}`} />
                              </button>
                              {filterSubsectionsOpen.colors && (
                              <div className="flex flex-wrap gap-2">
                                {availableFilters.colors.map((c) => (
                                  <button
                                    key={c}
                                    onClick={() =>
                                      toggleFilter(setSelectedColors, c)
                                    }
                                    className={`px-4 py-2 rounded-xl text-sm font-medium transition-all border ${
                                      selectedColors.has(c)
                                        ? "bg-[#44944A] text-black border-[#44944A]"
                                        : "bg-[var(--color-bg-primary)] text-[var(--color-text-secondary)] border-[var(--color-border-custom)] hover:border-[var(--color-text-muted)] hover:text-[var(--color-text-primary)]"
                                    }`}
                                  >
                                    {c}
                                  </button>
                                ))}
                              </div>
                              )}
                            </div>
                          )}

                          {(selectedBrands.size > 0 ||
                            selectedColors.size > 0 ||
                            selectedSizes.size > 0 ||
                            priceMin ||
                            priceMax) && (
                            <button
                              type="button"
                              onClick={resetFilters}
                              className="w-full text-center text-xs text-[var(--color-text-muted)] hover:text-red-500 transition-colors py-1"
                            >
                              Сбросить все фильтры
                            </button>
                          )}
                        </div>
                    </div>
                  </motion.div>
                </AnimatePresence>
                  )}
                </div>
              )}

              {/* Sort + price row — 2-col grid on mobile matching product cards */}
              <div className="grid grid-cols-2 gap-4 mt-4">
                <select
                  value={sortBy}
                  onChange={(e) => {
                    setSortBy(e.target.value as typeof sortBy);
                    setPage(1);
                  }}
                  className="rounded-xl bg-[var(--color-bg-card)] border border-[var(--color-border-custom)] px-4 py-2 text-sm text-[var(--color-text-primary)] outline-none focus:border-[#44944A] transition-colors"
                >
                  <option value="newest">{t("catalog.sortBy.newest")}</option>
                  <option value="price-asc">
                    {t("catalog.sortBy.priceAsc")}
                  </option>
                  <option value="price-desc">
                    {t("catalog.sortBy.priceDesc")}
                  </option>
                </select>
                <div className="flex items-center gap-1.5">
                  <input
                    type="number"
                    placeholder="От"
                    value={priceMin}
                    onChange={(e) => {
                      setPriceMin(e.target.value);
                      setPage(1);
                    }}
                    className="flex-1 min-w-0 rounded-xl bg-[var(--color-bg-card)] border border-[var(--color-border-custom)] px-3 py-2 text-sm text-[var(--color-text-primary)] outline-none focus:border-[#44944A] placeholder:text-[var(--color-text-muted)] [appearance:textfield] [&::-webkit-outer-spin-button]:appearance-none [&::-webkit-inner-spin-button]:appearance-none"
                  />
                  <span className="text-[var(--color-text-muted)] text-sm shrink-0">—</span>
                  <input
                    type="number"
                    placeholder="До"
                    value={priceMax}
                    onChange={(e) => {
                      setPriceMax(e.target.value);
                      setPage(1);
                    }}
                    className="flex-1 min-w-0 rounded-xl bg-[var(--color-bg-card)] border border-[var(--color-border-custom)] px-3 py-2 text-sm text-[var(--color-text-primary)] outline-none focus:border-[#44944A] placeholder:text-[var(--color-text-muted)] [appearance:textfield] [&::-webkit-outer-spin-button]:appearance-none [&::-webkit-inner-spin-button]:appearance-none"
                  />
                </div>
              </div>
            </motion.div>

            {/* Product grid */}
            <div className="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 gap-4">
              {products.map((product, index) => (
                <motion.div
                  key={product.id}
                  initial={{ opacity: 0, y: 20 }}
                  animate={{ opacity: 1, y: 0 }}
                  transition={{ duration: 0.4, delay: index * 0.05 }}
                  className="group"
                >
                  <Link to={`/product/${product.slug}`}>
                    <div className="card-hover overflow-hidden rounded-xl bg-[var(--color-bg-secondary)] border border-[var(--color-border-custom)]">
                      <div className="relative aspect-[4/5] overflow-hidden">
                        {product.images?.[0]?.url ? (
                          <img
                            src={getThumbUrl(product.images[0].url)}
                            alt={product.name}
                            className="absolute inset-0 w-full h-full object-cover transition-transform duration-500 group-hover:scale-105"
                            loading="lazy"
                            onError={(e) => {
                              const target = e.currentTarget;
                              if (!target.dataset.fallback) {
                                target.dataset.fallback = "1";
                                target.src = getImageUrl(product.images[0].url);
                              }
                            }}
                          />
                        ) : (
                          <div className="absolute inset-0 flex items-center justify-center">
                            <span className="text-3xl sm:text-5xl transition-transform duration-500 group-hover:scale-110">
                              {categoryEmoji(
                                categorySlugById.get(product.category_id) || "",
                              )}
                            </span>
                          </div>
                        )}
                        <div className="absolute inset-0 bg-gradient-to-t from-[var(--color-bg-primary)] via-transparent to-transparent opacity-60 pointer-events-none" />
                        <button
                          onClick={(e) => {
                            e.preventDefault();
                            addItem(product, product.sizes[0].label);
                          }}
                          className="absolute right-3 top-3 flex h-11 w-11 items-center justify-center rounded-full bg-[#44944A] opacity-0 transition-all duration-300 hover:scale-110 group-hover:opacity-100"
                          aria-label={t("home.featured.addToCart")}
                        >
                          <ShoppingBag className="h-4 w-4 text-black" />
                        </button>
                      </div>
                      <div className="px-3 py-2.5 border-t border-[var(--color-border-custom)]">
                        <p className="text-[11px] font-mono uppercase tracking-wider text-[#558b5c]">
                          {product.category_name}
                        </p>
                        <h3 className="mt-0.5 text-sm font-medium text-[var(--color-text-primary)] line-clamp-1">
                          {product.name}
                        </h3>
                        <p className="mt-1 text-sm font-bold text-[#44944A]">
                          {formatPrice(product.price, currency)}
                        </p>
                      </div>
                    </div>
                  </Link>
                </motion.div>
              ))}
            </div>

            {/* Pagination */}
            {totalPages > 1 && (
              <div className="flex items-center justify-center gap-2 mt-8">
                <button
                  onClick={() => setPage((p) => Math.max(1, p - 1))}
                  disabled={page <= 1}
                  className="px-3 py-2 rounded-xl bg-[var(--color-bg-card)] border border-[var(--color-border-custom)] text-sm text-[var(--color-text-secondary)] hover:text-[var(--color-text-primary)] disabled:opacity-40 transition-colors"
                >
                  ←
                </button>
                {Array.from({ length: totalPages }, (_, i) => i + 1)
                  .filter((p) => {
                    if (totalPages <= 7) return true;
                    if (p === 1 || p === totalPages) return true;
                    if (Math.abs(p - page) <= 1) return true;
                    return false;
                  })
                  .map((p, idx, arr) => (
                    <span key={p}>
                      {idx > 0 && p - (arr[idx - 1] ?? 0) > 1 && (
                        <span className="px-1 text-[var(--color-text-muted)] text-sm">
                          …
                        </span>
                      )}
                      <button
                        onClick={() => setPage(p)}
                        className={`w-9 h-9 rounded-lg text-sm font-medium transition-colors ${
                          p === page
                            ? "bg-[#44944A] text-black"
                            : "bg-[var(--color-bg-card)] text-[var(--color-text-secondary)] hover:text-[var(--color-text-primary)]"
                        }`}
                      >
                        {p}
                      </button>
                    </span>
                  ))}
                <button
                  onClick={() => setPage((p) => Math.min(totalPages, p + 1))}
                  disabled={page >= totalPages}
                  className="px-3 py-2 rounded-xl bg-[var(--color-bg-card)] border border-[var(--color-border-custom)] text-sm text-[var(--color-text-secondary)] hover:text-[var(--color-text-primary)] disabled:opacity-40 transition-colors"
                >
                  →
                </button>
              </div>
            )}
          </div>
        )}
      </div>
    </div>
  );
}
