import { useState, useMemo, useEffect, useRef } from "react";
import { motion, AnimatePresence } from "framer-motion";
import { Link, useNavigate, useParams, useSearchParams } from "react-router-dom";
import { useCatalogStore } from "@/stores/catalogStore";
import { useCurrencyStore } from "@/stores/currencyStore";
import { formatPrice } from "@/lib/currency";
import { useFavoritesStore } from "@/stores/favoritesStore";
import { getThumbUrl, getImageUrl } from "@/lib/api";
import { colorHex } from "@/lib/colors";
import { Search, SlidersHorizontal, X, Heart } from "lucide-react";
import { useTranslation } from "react-i18next";
import { Helmet } from "@dr.pogodin/react-helmet";

const PER_PAGE = 100;

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

/** Availability filter values (KAN-55: «Наличие» lives in the filter panel). */
type Availability = "all" | "in_stock" | "preorder";

export default function CatalogPage() {
  const { t } = useTranslation();
  const { currency } = useCurrencyStore();
  const navigate = useNavigate();
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

  const { categorySlug } = useParams<{ categorySlug?: string }>();
  const [searchParams, setSearchParams] = useSearchParams();

  // ── KAN-55: the URL is the single source of truth for filters, so they
  // survive navigation (back/forward), refresh and link sharing —
  // «ФИЛЬТРЫ НЕ ДОЛЖНЫ СЛЕТАТЬ».
  const selectedCategory = categorySlug || "all";
  const status = ((): Availability => {
    const raw = searchParams.get("status");
    return raw === "in_stock" || raw === "preorder" ? raw : "all";
  })();
  const searchQuery = searchParams.get("search") || "";
  const priceMin = searchParams.get("price_min") || "";
  const priceMax = searchParams.get("price_max") || "";
  const sortRaw = searchParams.get("sort") || "popular";

  const csv = (key: string): string[] => {
    const v = searchParams.get(key);
    return v ? v.split("|").filter(Boolean) : [];
  };
  const selectedBrands = useMemo(
    () => new Set(csv("brand")),
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [searchParams],
  );
  const selectedColors = useMemo(
    () => new Set(csv("color")),
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [searchParams],
  );
  const selectedSizes = useMemo(
    () => new Set(csv("size")),
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [searchParams],
  );

  const setParam = (key: string, value: string | null) => {
    const sp = new URLSearchParams(searchParams);
    if (value === null || value === "") sp.delete(key);
    else sp.set(key, value);
    setPage(1); // reset page on filter change (CLAUDE.md rule)
    setSearchParams(sp);
  };
  const toggleCSV = (key: string, value: string) => {
    const list = new Set(csv(key));
    if (list.has(value)) list.delete(value);
    else list.add(value);
    const sp = new URLSearchParams(searchParams);
    const arr = [...list].sort();
    if (arr.length) sp.set(key, arr.join("|"));
    else sp.delete(key);
    setPage(1); // reset page on filter change
    setSearchParams(sp);
  };

  // Sort: the URL carries the backend sort token; the select shows the
  // friendly value.
  const sortBy: "price-asc" | "price-desc" | "newest" | "popular" =
    sortRaw === "price"
      ? "price-asc"
      : sortRaw === "-price"
        ? "price-desc"
        : sortRaw === "newest"
          ? "newest"
          : "popular";
  const setSortBy = (next: typeof sortBy) => {
    const token =
      next === "price-asc"
        ? "price"
        : next === "price-desc"
          ? "-price"
          : next === "newest"
            ? "newest"
            : "popular";
    setParam("sort", token === "popular" ? null : token);
  };

  // Search input keeps immediate feedback; the debounced value lands in the
  // URL (which drives the fetch).
  const [searchInput, setSearchInput] = useState(searchQuery);
  const debounceRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const [page, setPage] = useState(1);
  useEffect(() => {
    setSearchInput(searchQuery);
    // A pending debounce would re-push the old input into the URL after
    // popstate rolled it back (the back button would be undone).
    if (debounceRef.current) clearTimeout(debounceRef.current);
  }, [searchQuery]);
  useEffect(() => () => {
    if (debounceRef.current) clearTimeout(debounceRef.current);
  }, []);
  const onSearchChange = (v: string) => {
    setSearchInput(v);
    if (debounceRef.current) clearTimeout(debounceRef.current);
    debounceRef.current = setTimeout(() => setParam("search", v || null), 300);
  };

  const [panelOpen, setPanelOpen] = useState(false);

  // Subscribe to `items` so the component re-renders on toggle. Selecting
  // only `isFavorite` (the function) is a no-op: Zustand sees the same
  // function reference and skips the update, so the heart never flips.
  const items = useFavoritesStore((state) => state.items);
  const toggleFavorite = useFavoritesStore((state) => state.toggleFavorite);
  const isFavorite = (id: number) => items.some((i) => i.id === id);

  // Bootstrap: load category tree once. Products + facets are loaded by the
  // filter-driven effects below.
  useEffect(() => {
    fetchCategories();
  }, [fetchCategories]);

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
      case "popular":
        return "popular";
      case "newest":
      default:
        return "-created_at";
    }
  }, [sortBy]);

  // Stable keys for deps arrays so effects compare by value.
  const brandKey = [...selectedBrands].sort().join("|");
  const colorKey = [...selectedColors].sort().join("|");
  const sizeKey = [...selectedSizes].sort().join("|");
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
      search: searchQuery || undefined,
      status: status === "all" ? undefined : status,
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
    status,
    searchQuery,
  ]);

  // Facets follow the scope (category + price + search) only — brand/color/size
  // selections do NOT narrow the facet lists, otherwise picking one brand would
  // hide every other brand from the UI.
  useEffect(() => {
    fetchFacets({
      category_id: categoryIds.length > 0 ? categoryIds : undefined,
      price_min: priceMin || undefined,
      price_max: priceMax || undefined,
      search: searchQuery || undefined,
      // Facets must follow the availability toggle, otherwise the panel
      // offers brands/colors/sizes that yield an empty grid.
      status: status === "all" ? undefined : status,
    });
  }, [fetchFacets, categoryIdsKey, categoryIds, priceMin, priceMax, searchQuery, status]);

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

  const toggleFilter = (key: string, value: string) => {
    toggleCSV(key, value);
  };

  // KAN-55: the number of active filter selections for the badge on the
  // «Фильтр» button.
  const activeFilterCount =
    selectedBrands.size +
    selectedColors.size +
    selectedSizes.size +
    (status !== "all" ? 1 : 0) +
    (priceMin ? 1 : 0) +
    (priceMax ? 1 : 0);

  const setCategorySlug = (slug: string) => {
    setPage(1); // reset page on filter change
    const sp = new URLSearchParams(searchParams);
    const qs = sp.toString();
    if (slug === "all") navigate(`/catalog${qs ? `?${qs}` : ""}`);
    else navigate(`/catalog/${slug}${qs ? `?${qs}` : ""}`);
  };

  const resetFilters = () => {
    setPage(1); // reset page on filter change
    const sp = new URLSearchParams(searchParams);
    for (const k of ["brand", "color", "size", "price_min", "price_max", "status", "q", "search"]) {
      sp.delete(k);
    }
    setSearchParams(sp);
  };

  // Cascading category selects (KAN-55): parent from the route slug, child
  // optional. Selecting a child navigates to the child slug.
  const currentParent = useMemo(() => {
    if (selectedCategory === "all") return null;
    return (
      categories.find(
        (c) =>
          !c.parent_id &&
          (c.slug === selectedCategory ||
            c.children?.some((ch) => ch.slug === selectedCategory)),
      ) || null
    );
  }, [categories, selectedCategory]);
  const currentChildSlug =
    currentParent?.children?.find((ch) => ch.slug === selectedCategory)?.slug || "";

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
            {/* KAN-55 filter bar: search → category chips (with counts) →
                sort + filter button */}
            <motion.div
              initial={{ opacity: 0, y: 20 }}
              animate={{ opacity: 1, y: 0 }}
              transition={{ duration: 0.6, delay: 0.1 }}
              className="mb-8 space-y-3"
            >
              {/* KAN-55: standard semi-transparent search field — always
                  visible, no green button. */}
              <div className="relative">
                <Search className="absolute left-4 top-1/2 -translate-y-1/2 h-4 w-4 text-[var(--color-text-muted)] pointer-events-none" />
                <input
                  type="text"
                  maxLength={200}
                  value={searchInput}
                  onChange={(e) => onSearchChange(e.target.value)}
                  onKeyDown={(e) => {
                    if (e.key === "Escape") onSearchChange("");
                  }}
                  placeholder={t("catalog.searchPlaceholder")}
                  data-testid="catalog-search-input"
                  className="w-full rounded-xl border border-[var(--color-border-custom)] bg-[var(--color-bg-card)]/60 px-10 py-2.5 text-sm text-[var(--color-text-primary)] outline-none focus:border-[#44944A]/60 focus:bg-[var(--color-bg-card)] placeholder:text-[var(--color-text-muted)] transition-colors"
                />
                {searchInput && (
                  <button
                    type="button"
                    onClick={() => onSearchChange("")}
                    aria-label={t("catalog.searchClear")}
                    className="absolute right-3 top-1/2 -translate-y-1/2 text-[var(--color-text-muted)] hover:text-[var(--color-text-primary)] transition-colors"
                  >
                    <X className="h-4 w-4" />
                  </button>
                )}
              </div>

              {/* Category chips — horizontal scroll, with product counts */}
              <div className="flex gap-2 overflow-x-auto pb-2 scrollbar-none items-center">
                <button
                  onClick={() => setCategorySlug("all")}
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
                        onClick={() => setCategorySlug(cat.slug)}
                        className={`shrink-0 px-4 py-2 rounded-full text-sm font-medium transition-all ${
                          isActive
                            ? "bg-[#44944A] text-black shadow-[0_0_20px_rgba(68,148,74,0.4)]"
                            : "bg-[var(--color-bg-card)] text-[var(--color-text-secondary)] border border-[var(--color-border-custom)] hover:text-[var(--color-text-primary)]"
                        }`}
                      >
                        {cat.name}
                        {(cat.products_count ?? 0) > 0 && (
                          <span className="ml-1.5 opacity-60">{cat.products_count}</span>
                        )}
                      </button>
                    );
                  })}
              </div>

              {/* Sort + filter button — KAN-55: «Фильтр» replaces От/До */}
              <div className="flex items-center gap-2">
                <select
                  value={sortBy}
                  onChange={(e) => setSortBy(e.target.value as typeof sortBy)}
                  className="flex-1 rounded-xl bg-[var(--color-bg-card)] border border-[var(--color-border-custom)] px-4 py-2 text-sm text-[var(--color-text-primary)] outline-none focus:border-[#44944A] transition-colors"
                >
                  <option value="popular">{t("catalog.sortBy.popular")}</option>
                  <option value="newest">{t("catalog.sortBy.newest")}</option>
                  <option value="price-asc">
                    {t("catalog.sortBy.priceAsc")}
                  </option>
                  <option value="price-desc">
                    {t("catalog.sortBy.priceDesc")}
                  </option>
                </select>
                <button
                  type="button"
                  onClick={() => setPanelOpen(!panelOpen)}
                  data-testid="catalog-filter-button"
                  aria-expanded={panelOpen}
                  className={`relative shrink-0 flex items-center gap-2 rounded-xl border px-4 py-2 text-sm font-semibold transition-colors ${
                    panelOpen || activeFilterCount > 0
                      ? "border-[#44944A] text-[#44944A] bg-[#44944A]/10"
                      : "border-[var(--color-border-custom)] text-[var(--color-text-primary)] bg-[var(--color-bg-card)] hover:border-[#44944A]/50"
                  }`}
                >
                  <SlidersHorizontal className="h-4 w-4" />
                  {t("catalog.filters.button")}
                  {activeFilterCount > 0 && (
                    <span
                      data-testid="catalog-filter-badge"
                      className="flex h-5 min-w-5 items-center justify-center rounded-full bg-[#44944A] px-1 text-[11px] font-bold text-black"
                    >
                      {activeFilterCount}
                    </span>
                  )}
                </button>
              </div>

              {/* KAN-55 filter panel — available on every category bucket,
                  including «Все категории» */}
              <AnimatePresence initial={false}>
                {panelOpen && (
                  <motion.div
                    key="catalog-filter-panel"
                    initial={{ opacity: 0, height: 0 }}
                    animate={{ opacity: 1, height: "auto" }}
                    exit={{ opacity: 0, height: 0 }}
                    className="overflow-hidden"
                  >
                    <div
                      data-testid="catalog-filter-panel"
                      className="rounded-2xl bg-[var(--color-bg-card)] border border-[var(--color-border-custom)] p-5 space-y-5 mt-3"
                    >
                      {/* Наличие */}
                      <div>
                        <p className="text-xs font-semibold uppercase tracking-wider text-[var(--color-text-primary)] mb-2">
                          {t("catalog.panel.availability")}
                        </p>
                        <div
                          role="group"
                          data-testid="catalog-availability"
                          className="flex gap-2"
                        >
                          {(
                            [
                              { v: "all", label: t("catalog.panel.availabilityAll") },
                              { v: "in_stock", label: t("catalog.panel.availabilityInStock") },
                              { v: "preorder", label: t("catalog.panel.availabilityPreorder") },
                            ] as const
                          ).map((o) => (
                            <button
                              key={o.v}
                              type="button"
                              data-testid={`catalog-availability-${o.v}`}
                              aria-pressed={status === o.v}
                              onClick={() => setParam("status", o.v === "all" ? null : o.v)}
                              className={`px-3 py-1.5 rounded-lg text-xs sm:text-sm font-medium transition-colors border ${
                                status === o.v
                                  ? "bg-[#44944A] text-black border-[#44944A]"
                                  : "bg-[var(--color-bg-primary)] text-[var(--color-text-secondary)] border-[var(--color-border-custom)] hover:text-[var(--color-text-primary)]"
                              }`}
                            >
                              {o.label}
                            </button>
                          ))}
                        </div>
                      </div>

                      {/* Категория — cascading: parent → child */}
                      <div>
                        <p className="text-xs font-semibold uppercase tracking-wider text-[var(--color-text-primary)] mb-2">
                          {t("catalog.panel.category")}
                        </p>
                        <div className="grid gap-2 sm:grid-cols-2">
                          <select
                            value={currentParent?.slug ?? ""}
                            onChange={(e) =>
                              setCategorySlug(e.target.value || "all")
                            }
                            data-testid="catalog-category-parent-select"
                            className="rounded-xl bg-[var(--color-bg-primary)] border border-[var(--color-border-custom)] px-3 py-2 text-sm text-[var(--color-text-primary)] outline-none focus:border-[#44944A] transition-colors"
                          >
                            <option value="">{t("catalog.filters.allCategories")}</option>
                            {categories
                              .filter((c) => !c.parent_id)
                              .map((c) => (
                                <option key={c.id} value={c.slug}>
                                  {c.name}
                                </option>
                              ))}
                          </select>
                          {currentParent &&
                            (currentParent.children?.length || 0) > 0 && (
                              <select
                                value={currentChildSlug}
                                onChange={(e) =>
                                  setCategorySlug(e.target.value || currentParent.slug)
                                }
                                data-testid="catalog-category-child-select"
                                className="rounded-xl bg-[var(--color-bg-primary)] border border-[var(--color-border-custom)] px-3 py-2 text-sm text-[var(--color-text-primary)] outline-none focus:border-[#44944A] transition-colors"
                              >
                                <option value="">{t("catalog.panel.allSubcategories")}</option>
                                {currentParent.children!.map((ch) => (
                                  <option key={ch.id} value={ch.slug}>
                                    {ch.name}
                                  </option>
                                ))}
                              </select>
                            )}
                        </div>
                      </div>

                      {/* Бренд */}
                      {availableFilters.brands.length > 0 && (
                        <div>
                          <p className="text-xs font-semibold uppercase tracking-wider text-[var(--color-text-primary)] mb-2">
                            {t("catalog.panel.brand")}
                          </p>
                          <div className="flex flex-wrap gap-2">
                            {availableFilters.brands.map((b) => (
                              <button
                                key={b}
                                onClick={() => toggleFilter("brand", b)}
                                className={`px-3 py-1.5 rounded-xl text-sm font-medium transition-all border ${
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

                      {/* Цвет */}
                      {availableFilters.colors.length > 0 && (
                        <div>
                          <p className="text-xs font-semibold uppercase tracking-wider text-[var(--color-text-primary)] mb-2">
                            {t("catalog.panel.color")}
                          </p>
                          <div className="flex flex-wrap gap-2">
                            {availableFilters.colors.map((c) => {
                              const hex = colorHex(c) ?? "#888888";
                              return (
                                <button
                                  key={c}
                                  onClick={() => toggleFilter("color", c)}
                                  className={`px-3 py-1.5 rounded-xl text-sm font-medium transition-all border inline-flex items-center gap-2 ${
                                    selectedColors.has(c)
                                      ? "bg-[#44944A] text-black border-[#44944A]"
                                      : "bg-[var(--color-bg-primary)] text-[var(--color-text-secondary)] border-[var(--color-border-custom)] hover:border-[var(--color-text-muted)] hover:text-[var(--color-text-primary)]"
                                  }`}
                                >
                                  <span>{t(`catalog.colorFilter.${c}`, c)}</span>
                                  <span
                                    aria-hidden="true"
                                    style={{ background: hex }}
                                    className="inline-block h-4 w-4 rounded-md border border-black/20 shrink-0"
                                  />
                                </button>
                              );
                            })}
                          </div>
                        </div>
                      )}

                      {/* Размер */}
                      {availableFilters.sizes.length > 0 && (
                        <div>
                          <p className="text-xs font-semibold uppercase tracking-wider text-[var(--color-text-primary)] mb-2">
                            {t("catalog.panel.size")}
                          </p>
                          <div className="flex flex-wrap gap-2">
                            {availableFilters.sizes.map((s) => (
                              <button
                                key={s}
                                onClick={() => toggleFilter("size", s)}
                                className={`px-3 py-1.5 rounded-xl text-sm font-medium transition-all border ${
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

                      {/* Цена */}
                      <div>
                        <p className="text-xs font-semibold uppercase tracking-wider text-[var(--color-text-primary)] mb-2">
                          {t("catalog.panel.price")}
                        </p>
                        <div className="flex items-center gap-1.5 max-w-xs">
                          <input
                            key={`pmin-${priceMin}`}
                            type="number"
                            min={0}
                            placeholder={t("catalog.panel.priceFrom")}
                            defaultValue={priceMin}
                            onBlur={(e) => {
                              // The backend parses Atoi — non-positive or
                              // fractional values are silently dropped.
                              const v = e.target.value.trim();
                              const n = Number(v);
                              setParam("price_min", Number.isInteger(n) && n > 0 ? v : null);
                            }}
                            className="flex-1 min-w-0 rounded-xl bg-[var(--color-bg-primary)] border border-[var(--color-border-custom)] px-3 py-2 text-sm text-[var(--color-text-primary)] outline-none focus:border-[#44944A] placeholder:text-[var(--color-text-muted)] [appearance:textfield] [&::-webkit-outer-spin-button]:appearance-none [&::-webkit-inner-spin-button]:appearance-none"
                          />
                          <span className="text-[var(--color-text-muted)] text-sm shrink-0">—</span>
                          <input
                            key={`pmax-${priceMax}`}
                            type="number"
                            min={0}
                            placeholder={t("catalog.panel.priceTo")}
                            defaultValue={priceMax}
                            onBlur={(e) => {
                              const v = e.target.value.trim();
                              const n = Number(v);
                              setParam("price_max", Number.isInteger(n) && n > 0 ? v : null);
                            }}
                            className="flex-1 min-w-0 rounded-xl bg-[var(--color-bg-primary)] border border-[var(--color-border-custom)] px-3 py-2 text-sm text-[var(--color-text-primary)] outline-none focus:border-[#44944A] placeholder:text-[var(--color-text-muted)] [appearance:textfield] [&::-webkit-outer-spin-button]:appearance-none [&::-webkit-inner-spin-button]:appearance-none"
                          />
                        </div>
                      </div>

                      {activeFilterCount > 0 && (
                        <button
                          type="button"
                          onClick={resetFilters}
                          data-testid="catalog-filter-reset"
                          className="w-full text-center text-xs text-[var(--color-text-muted)] hover:text-red-500 transition-colors py-1"
                        >
                          {t("catalog.panel.reset")}
                        </button>
                      )}
                    </div>
                  </motion.div>
                )}
              </AnimatePresence>
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
                  <Link
                    to={`/product/${product.slug}`}
                    data-testid="catalog-product-card"
                    data-slug={product.slug}
                  >
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
                        <button
                          onClick={(e) => {
                            e.preventDefault();
                            toggleFavorite(product);
                          }}
                          aria-label={
                            isFavorite(product.id)
                              ? t("nav.favorites") + " — remove"
                              : t("nav.favorites") + " — add"
                          }
                          className="absolute right-3 top-3 flex h-11 w-11 items-center justify-center rounded-full bg-[#44944A] opacity-0 transition-all duration-300 hover:scale-110 group-hover:opacity-100"
                        >
                          <Heart
                            className={
                              isFavorite(product.id)
                                ? "h-4 w-4 text-black fill-black"
                                : "h-4 w-4 text-black"
                            }
                          />
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
