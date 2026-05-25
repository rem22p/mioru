import { useState, useMemo, useEffect } from "react";
import { motion, AnimatePresence } from "framer-motion";
import { Link } from "react-router-dom";
import { useCatalogStore } from "@/stores/catalogStore";
import { useCartStore } from "@/stores/cartStore";
import { getImageUrl } from "@/lib/api";
import { ShoppingBag } from "lucide-react";
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
  const [priceMin, setPriceMin] = useState("");
  const [priceMax, setPriceMax] = useState("");
  const [selectedBrands, setSelectedBrands] = useState<Set<string>>(new Set());
  const [selectedColors, setSelectedColors] = useState<Set<string>>(new Set());
  const [selectedSizes, setSelectedSizes] = useState<Set<string>>(new Set());
  const [sortBy, setSortBy] = useState<"price-asc" | "price-desc" | "newest">(
    "newest",
  );
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

  // Products filtered by category only (before brand/color/size/price)
  const categoryProducts = useMemo(() => {
    if (selectedCategory === "all") return products;
    const catId = flatCategories.find((c) => c.slug === selectedCategory)?.id;
    if (catId == null) return products;
    const ids = new Set<number>();
    const collect = (cats: typeof categories) => {
      for (const c of cats) {
        if (c.id === catId || ids.has(c.parent_id ?? 0)) ids.add(c.id);
        if (c.children) collect(c.children);
      }
    };
    collect(categories);
    ids.add(catId);
    return products.filter((p) => ids.has(p.category_id));
  }, [products, selectedCategory, flatCategories, categories]);

  // Available filter options from category-filtered products
  const availableFilters = useMemo(() => {
    const brands = new Set<string>();
    const colors = new Set<string>();
    const sizes = new Set<string>();
    for (const p of categoryProducts) {
      if (p.brand) brands.add(p.brand);
      if (p.color) colors.add(p.color);
      if (p.sizes) p.sizes.forEach((s) => sizes.add(s));
    }
    return {
      brands: [...brands].sort(),
      colors: [...colors].sort(),
      sizes: [...sizes].sort((a, b) => {
        const na = parseInt(a);
        const nb = parseInt(b);
        if (!isNaN(na) && !isNaN(nb)) return na - nb;
        return a.localeCompare(b);
      }),
    };
  }, [categoryProducts]);

  const toggleFilter = (setter: any, value: string) => {
    setter((prev: Set<string>) => {
      const next = new Set(prev);
      next.has(value) ? next.delete(value) : next.add(value);
      return next;
    });
  };

  const filteredProducts = useMemo(() => {
    let result = categoryProducts;
    if (priceMin) result = result.filter((p) => p.price >= Number(priceMin));
    if (priceMax) result = result.filter((p) => p.price <= Number(priceMax));
    if (selectedBrands.size > 0)
      result = result.filter((p) => selectedBrands.has(p.brand));
    if (selectedColors.size > 0)
      result = result.filter((p) => selectedColors.has(p.color));
    if (selectedSizes.size > 0)
      result = result.filter((p) => p.sizes?.some((s) => selectedSizes.has(s)));
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
  }, [
    categoryProducts,
    sortBy,
    priceMin,
    priceMax,
    selectedBrands,
    selectedColors,
    selectedSizes,
  ]);

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
              {/* Category chips — horizontal scroll */}
              <div className="flex gap-2 overflow-x-auto pb-2 scrollbar-none">
                <button
                  onClick={() => setSelectedCategory("all")}
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
                        onClick={() => setSelectedCategory(cat.slug)}
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
                  const subCats =
                    parent?.children?.filter(
                      (ch) => ch.slug !== selectedCategory,
                    ) || [];
                  if (subCats.length === 0) return null;
                  return (
                    <div className="flex gap-2 overflow-x-auto pb-2 scrollbar-none ml-4">
                      <span className="shrink-0 self-center text-xs text-[var(--color-text-muted)] mr-1">
                        Подкатегории:
                      </span>
                      <button
                        onClick={() => setSelectedCategory(parent!.slug)}
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
                          onClick={() => setSelectedCategory(ch.slug)}
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

              {/* Dynamic filters — after any category selected */}
              {selectedCategory !== "all" &&
                (() => {
                  const catHasChildren = categories.some((c) =>
                    c.children?.some((ch) => ch.slug === selectedCategory),
                  );
                  const parentWithoutChildren = categories.some(
                    (c) => !c.children?.length && c.slug === selectedCategory,
                  );
                  return catHasChildren || parentWithoutChildren;
                })() && (
                  <AnimatePresence mode="wait">
                    <motion.div
                      key={selectedCategory}
                      initial={{ opacity: 0, height: 0 }}
                      animate={{ opacity: 1, height: "auto" }}
                      exit={{ opacity: 0, height: 0 }}
                      className="mb-6 overflow-hidden"
                    >
                      <div className="rounded-2xl bg-[var(--color-bg-card)] border border-[var(--color-border-custom)] p-5 space-y-5">
                        <h3 className="text-sm font-semibold uppercase tracking-wider text-[var(--color-text-muted)]">
                          Фильтры
                        </h3>

                        {/* Sizes */}
                        {availableFilters.sizes.length > 0 && (
                          <div>
                            <h4 className="text-xs font-semibold text-[var(--color-text-primary)] mb-2">
                              Размер
                            </h4>
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
                          </div>
                        )}

                        {/* Brands */}
                        {availableFilters.brands.length > 0 && (
                          <div>
                            <h4 className="text-xs font-semibold text-[var(--color-text-primary)] mb-2">
                              Бренд
                            </h4>
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
                          </div>
                        )}

                        {/* Colors */}
                        {availableFilters.colors.length > 0 && (
                          <div>
                            <h4 className="text-xs font-semibold text-[var(--color-text-primary)] mb-2">
                              Цвет
                            </h4>
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
                          </div>
                        )}
                      </div>
                    </motion.div>
                  </AnimatePresence>
                )}

              {/* Sort + price row */}
              <div className="flex items-center gap-3 mt-4">
                <select
                  value={sortBy}
                  onChange={(e) => setSortBy(e.target.value as typeof sortBy)}
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
                <div className="flex items-center gap-1.5 ml-auto">
                  <input
                    type="number"
                    placeholder="Цена от"
                    value={priceMin}
                    onChange={(e) => setPriceMin(e.target.value)}
                    className="w-24 rounded-xl bg-[var(--color-bg-card)] border border-[var(--color-border-custom)] px-3 py-2 text-sm text-[var(--color-text-primary)] outline-none focus:border-[#44944A] placeholder:text-[var(--color-text-muted)] [appearance:textfield] [&::-webkit-outer-spin-button]:appearance-none [&::-webkit-inner-spin-button]:appearance-none"
                  />
                  <span className="text-[var(--color-text-muted)] text-sm">
                    —
                  </span>
                  <input
                    type="number"
                    placeholder="До"
                    value={priceMax}
                    onChange={(e) => setPriceMax(e.target.value)}
                    className="w-24 rounded-xl bg-[var(--color-bg-card)] border border-[var(--color-border-custom)] px-3 py-2 text-sm text-[var(--color-text-primary)] outline-none focus:border-[#44944A] placeholder:text-[var(--color-text-muted)] [appearance:textfield] [&::-webkit-outer-spin-button]:appearance-none [&::-webkit-inner-spin-button]:appearance-none"
                  />
                </div>
              </div>
            </motion.div>

            {/* Product grid */}
            <div className="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 gap-4">
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
                          <span className="text-3xl sm:text-5xl transition-transform duration-500 group-hover:scale-110">
                            {categoryEmoji(
                              categorySlugById.get(product.category_id) || "",
                            )}
                          </span>
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
                        <p className="text-xs font-mono uppercase tracking-wider text-[#558b5c]">
                          {product.category_name}
                        </p>
                        <h3 className="mt-1 text-base font-medium text-[var(--color-text-primary)] line-clamp-1">
                          {product.name}
                        </h3>
                        <p className="mt-1 text-base font-bold text-[#44944A]">
                          {product.price.toLocaleString("ru-RU")} ₽
                        </p>
                      </div>
                    </div>
                  </Link>
                </motion.div>
              ))}
            </div>
          </div>
        )}
      </div>
    </div>
  );
}
