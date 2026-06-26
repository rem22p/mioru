import { useState, lazy, Suspense, useEffect } from "react";
import { motion } from "framer-motion";
import { Link } from "react-router-dom";
import type { Product } from "@/types";
import { toSizeChart } from "@/types";
import { useCartStore } from "@/stores/cartStore";
import { useFavoritesStore } from "@/stores/favoritesStore";
import { useTranslation } from "react-i18next";
import {
  ShoppingBag,
  ArrowLeft,
  Ruler,
  Check,
  Heart,
  Share2,
  Package,
  X,
} from "lucide-react";
import { useCurrencyStore } from "@/stores/currencyStore";
import { formatPrice } from "@/lib/currency";
import { Helmet } from "@dr.pogodin/react-helmet";

// Lazy-load below-the-fold components
const ProductGallery = lazy(
  () => import("@/components/product/ProductGallery"),
);
const SizeChartModal = lazy(
  () => import("@/components/product/SizeChartModal"),
);
const ProductDetails = lazy(
  () => import("@/components/product/ProductDetails"),
);
const RelatedProducts = lazy(
  () => import("@/components/product/RelatedProducts"),
);

interface ProductPageClientProps {
  product: Product;
}

export default function ProductPageClient({ product }: ProductPageClientProps) {
  const { t } = useTranslation();
  const { currency } = useCurrencyStore();
  const [selectedSize, setSelectedSize] = useState<string>("");
  const [showSizeChart, setShowSizeChart] = useState(false);
  const [showPreorderInfo, setShowPreorderInfo] = useState(false);
  const [addedToCart, setAddedToCart] = useState(false);
  const [copied, setCopied] = useState(false);
  const addItem = useCartStore((state) => state.addItem);
  const updateQuantity = useCartStore((state) => state.updateQuantity);
  const cartItems = useCartStore((state) => state.items);
  const isFav = useFavoritesStore((s) => s.isFavorite(product.id));
  const toggleFav = useFavoritesStore((s) => s.toggleFavorite);

  const isPreorder = product.status !== "in_stock";

  // Auto-select size if product is already in cart
  useEffect(() => {
    const inCart = cartItems.find((item) => item.product.id === product.id);
    if (inCart && !selectedSize) {
      setSelectedSize(inCart.size);
    }
  }, [cartItems, product.id, selectedSize]);

  // Transform backend size_chart rows into UI format
  const sizeChart = toSizeChart(product.size_chart);

  // How many of this product+size are already in the cart
  const cartQty = selectedSize
    ? cartItems
        .filter(
          (item) =>
            item.product.id === product.id && item.size === selectedSize,
        )
        .reduce((sum, item) => sum + item.quantity, 0)
    : 0;

  const stock = product.stock_quantity || 0;
  const available = stock > 0 ? Math.max(0, stock - cartQty) : 999;
  const soldOut = stock > 0 && available <= 0;

  const handleIncrement = () => {
    if (!selectedSize || soldOut) return;
    if (stock > 0 && cartQty >= stock) return;
    addItem(product, selectedSize, 1);
    setAddedToCart(true);
    setTimeout(() => setAddedToCart(false), 800);
  };

  const handleDecrement = () => {
    if (!selectedSize || cartQty <= 0) return;
    updateQuantity(product.id, selectedSize, cartQty - 1);
  };

  return (
    <div className="px-6 py-24 lg:px-8">
      <Helmet>
        <title>
          {product.name} — купить в MIORU | Цена{" "}
          {formatPrice(product.price, currency)}
        </title>
        <meta
          name="description"
          content={`${product.name} — ${product.description.slice(0, 150)}... Цена: ${formatPrice(product.price, currency)}. Виртуальная примерка на 3D-аватаре. Быстрая доставка.`}
        />
        <meta property="og:title" content={`${product.name} — MIORU`} />
        <meta
          property="og:description"
          content={`${product.description.slice(0, 150)}...`}
        />
        <meta property="og:type" content="product" />
        <meta property="og:price:amount" content={String(product.price)} />
        <meta property="og:price:currency" content="RUB" />
        <link
          rel="canonical"
          href={`https://mioru.store/product/${product.slug}`}
        />
      </Helmet>
      <div className="mx-auto max-w-7xl">
        <motion.div
          initial={{ opacity: 0 }}
          animate={{ opacity: 1 }}
          transition={{ duration: 0.5 }}
        >
          {/* Breadcrumb */}
          <Link
            to="/catalog"
            className="inline-flex items-center gap-2 text-sm text-[var(--color-text-secondary)] transition-colors hover:text-[var(--color-text-primary)] mb-8"
          >
            <ArrowLeft className="h-4 w-4" />
            {t("product.backToCatalog")}
          </Link>

          <div className="grid gap-12 lg:grid-cols-2">
            {/* Left: Gallery */}
            <motion.div
              initial={{ opacity: 0, x: -30 }}
              animate={{ opacity: 1, x: 0 }}
              transition={{ duration: 0.6 }}
            >
              <Suspense
                fallback={
                  <div className="aspect-[4/5] rounded-2xl bg-[var(--color-bg-secondary)] border border-[var(--color-border-custom)] animate-pulse" />
                }
              >
                <ProductGallery product={product} />
              </Suspense>
            </motion.div>

            {/* Right: Product Info */}
            <motion.div
              initial={{ opacity: 0, x: 30 }}
              animate={{ opacity: 1, x: 0 }}
              transition={{ duration: 0.6, delay: 0.1 }}
              className="flex flex-col"
            >
              {/* Category & Name */}
              <p className="text-xs font-mono uppercase tracking-[0.3em] text-[#558b5c]">
                {product.category_name}
              </p>
              <h1 className="mt-3 text-4xl font-bold tracking-tighter text-[var(--color-text-primary)] sm:text-5xl">
                {product.name}
              </h1>

              {/* Color + Price Row */}
              <div className="mt-6 flex items-center justify-between">
                <span className="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-full text-xs font-medium bg-[#44944A]/10 text-[#44944A] border border-[#44944A]/20">
                  <span className="w-2 h-2 rounded-full bg-[#44944A]" />
                  {product.color}
                </span>
                <p className="text-3xl font-bold text-[#44944A]">
                  {formatPrice(product.price, currency)}
                </p>
              </div>

              {/* Short Description */}
              <p className="mt-6 text-[var(--color-text-secondary)] leading-relaxed text-sm">
                {product.description}
              </p>

              {/* Size Selector */}
              <div className="mt-8">
                <div className="flex items-center justify-between mb-4">
                  <h3 className="text-sm font-semibold uppercase tracking-wider text-[var(--color-text-primary)]">
                    {t("product.size")}
                  </h3>
                  {isPreorder ? (
                    <button
                      onClick={() => setShowPreorderInfo(true)}
                      className="inline-flex items-center gap-1.5 text-xs transition-colors min-h-[44px] px-2 text-[var(--color-text-secondary)] hover:text-[#44944A]"
                    >
                      <Package className="h-3.5 w-3.5" />
                      {t("product.preorder.button")}
                    </button>
                  ) : (
                    <button
                      onClick={() => sizeChart && setShowSizeChart(true)}
                      className={`inline-flex items-center gap-1.5 text-xs transition-colors min-h-[44px] px-2 ${sizeChart ? "text-[var(--color-text-secondary)] hover:text-[#44944A]" : "text-[var(--color-text-muted)] cursor-default"}`}
                    >
                      <Ruler className="h-3.5 w-3.5" />
                      {t("product.sizeChart")}
                    </button>
                  )}
                </div>
                <div className="flex flex-wrap gap-3">
                  {product.sizes.map((size) => (
                    <button
                      key={size}
                      data-testid="product-size"
                      data-size={size}
                      onClick={() => setSelectedSize(size)}
                      className={`relative rounded-xl border px-5 py-3 text-sm font-medium transition-all min-h-[44px] ${
                        selectedSize === size
                          ? "border-[#44944A] bg-[#44944A] text-black"
                          : "border-[var(--color-border-custom)] text-[var(--color-text-secondary)] hover:border-[var(--color-text-muted)] hover:text-[var(--color-text-primary)]"
                      }`}
                    >
                      {size}
                      {selectedSize === size && (
                        <span className="absolute -top-1.5 -right-1.5 w-5 h-5 rounded-full bg-black flex items-center justify-center">
                          <Check className="h-3 w-3 text-[#44944A]" />
                        </span>
                      )}
                    </button>
                  ))}
                </div>
              </div>

              {/* Actions */}
              <div className="mt-8 flex gap-3">
                {cartQty > 0 ? (
                  <Link
                    to="/cart"
                    data-testid="product-go-to-cart"
                    className="flex flex-1 items-center justify-center gap-2 rounded-xl px-6 py-4 text-sm font-semibold bg-[#44944A] text-black hover:shadow-[0_0_30px_rgba(192,254,57,0.3)] transition-all min-h-[44px]"
                  >
                    <ShoppingBag className="h-4 w-4" />
                    {t("product.inCart")}
                  </Link>
                ) : (
                  <button
                    data-testid="product-add-to-cart"
                    onClick={handleIncrement}
                    disabled={!selectedSize || soldOut}
                    className={`flex flex-1 items-center justify-center gap-2 rounded-xl px-6 py-4 text-sm font-semibold transition-all min-h-[44px] ${
                      addedToCart
                        ? "bg-[#558b5c] text-[var(--color-text-primary)]"
                        : "bg-[#44944A] text-black hover:shadow-[0_0_30px_rgba(192,254,57,0.3)]"
                    } disabled:opacity-50 disabled:cursor-not-allowed`}
                  >
                    {soldOut ? (
                      <>{t("product.soldOut")}</>
                    ) : addedToCart ? (
                      <>
                        <Check className="h-4 w-4" />
                        {t("product.addedToCart")}
                      </>
                    ) : (
                      <>
                        <ShoppingBag className="h-4 w-4" />
                        {selectedSize
                          ? t("product.addToCart")
                          : t("product.selectSize")}
                      </>
                    )}
                  </button>
                )}
                <button
                  onClick={() => toggleFav(product)}
                  className={`flex items-center justify-center gap-2 rounded-xl border px-5 py-4 text-sm transition-all min-h-[44px] min-w-[44px] ${
                    isFav
                      ? "border-[#44944A] text-[#44944A]"
                      : "border-[var(--color-border-custom)] text-[var(--color-text-secondary)] hover:border-[var(--color-text-muted)] hover:text-[var(--color-text-primary)]"
                  }`}
                  aria-label={
                    isFav
                      ? t("product.unwishlist")
                      : t("product.wishlist")
                  }
                >
                  <Heart
                    className={`h-4 w-4 ${isFav ? "fill-[#44944A]" : ""}`}
                  />
                </button>
                <button
                  onClick={async () => {
                    if (navigator.share) {
                      try {
                        await navigator.share({
                          title: product.name,
                          url: window.location.href,
                        });
                        return;
                      } catch {
                        // user cancelled share — fall through to clipboard below
                      }
                    }
                    await navigator.clipboard.writeText(window.location.href);
                    setCopied(true);
                    setTimeout(() => setCopied(false), 1500);
                  }}
                  className={`flex items-center justify-center gap-2 rounded-xl border px-5 py-4 text-sm transition-all min-h-[44px] min-w-[44px] ${
                    copied
                      ? "border-[#44944A] text-[#44944A] bg-[#44944A]/10"
                      : "border-[var(--color-border-custom)] text-[var(--color-text-secondary)] hover:border-[var(--color-text-muted)] hover:text-[var(--color-text-primary)]"
                  }`}
                  aria-label={t("product.share")}
                >
                  {copied ? <Check className="h-4 w-4" /> : <Share2 className="h-4 w-4" />}
                </button>
              </div>

              {/* Quantity selector — only visible when item is in cart */}
              {cartQty > 0 && (
                <div className="mt-3">
                  <div className="flex items-center rounded-xl border border-[#44944A]/40 overflow-hidden">
                    <button
                      onClick={handleDecrement}
                      className="h-12 w-12 flex items-center justify-center text-lg font-medium text-[var(--color-text-secondary)] hover:text-[var(--color-text-primary)] hover:bg-[var(--color-bg-card)] transition-colors"
                    >
                      −
                    </button>
                    <span className="flex-1 text-center text-sm font-bold text-[var(--color-text-primary)]">
                      {cartQty}
                    </span>
                    <button
                      onClick={handleIncrement}
                      disabled={soldOut || (stock > 0 && cartQty >= stock)}
                      className="h-12 w-12 flex items-center justify-center text-lg font-medium text-[var(--color-text-secondary)] hover:text-[var(--color-text-primary)] hover:bg-[var(--color-bg-card)] disabled:opacity-30 transition-colors"
                    >
                      +
                    </button>
                  </div>
                </div>
              )}

              {/* Trust Badges */}
              <div className="mt-8 grid grid-cols-3 gap-3">
                <div className="flex flex-col items-center gap-2 p-4 rounded-xl bg-[var(--color-bg-card)] border border-[var(--color-border-custom)]">
                  <TruckIcon />
                  <p className="text-xs text-[var(--color-text-secondary)] text-center">
                    {t("product.trust.shipping")}
                  </p>
                </div>
                <div className="flex flex-col items-center gap-2 p-4 rounded-xl bg-[var(--color-bg-card)] border border-[var(--color-border-custom)]">
                  <ReturnIcon />
                  <p className="text-xs text-[var(--color-text-secondary)] text-center">
                    {t("product.trust.returns")}
                  </p>
                </div>
                <div className="flex flex-col items-center gap-2 p-4 rounded-xl bg-[var(--color-bg-card)] border border-[var(--color-border-custom)]">
                  <SecureIcon />
                  <p className="text-xs text-[var(--color-text-secondary)] text-center">
                    {t("product.trust.secure")}
                  </p>
                </div>
              </div>
            </motion.div>
          </div>

          {/* Details Tabs */}
          <Suspense
            fallback={
              <div className="mt-12 h-64 bg-[var(--color-bg-card)] rounded-2xl animate-pulse" />
            }
          >
            <ProductDetails product={product} />
          </Suspense>

          {/* Related Products */}
          <Suspense fallback={null}>
            <RelatedProducts
              relatedProductIds={[]}
              currentProductId={product.id}
            />
          </Suspense>
        </motion.div>
      </div>

      {/* Size Chart Modal */}
      {sizeChart && (
        <Suspense fallback={null}>
          <SizeChartModal
            isOpen={showSizeChart}
            onClose={() => setShowSizeChart(false)}
            product={product}
          />
        </Suspense>
      )}

      {/* Preorder Info Modal */}
      {showPreorderInfo && (
        <div className="fixed inset-0 z-50">
          <div
            className="absolute inset-0 bg-black/60 backdrop-blur-sm"
            onClick={() => setShowPreorderInfo(false)}
          />
          <div className="relative min-h-full flex items-center justify-center p-4">
            <motion.div
              initial={{ opacity: 0, scale: 0.95, y: 20 }}
              animate={{ opacity: 1, scale: 1, y: 0 }}
              exit={{ opacity: 0, scale: 0.95, y: 20 }}
              transition={{ duration: 0.2 }}
              className="relative w-full max-w-md mx-auto rounded-2xl bg-[var(--color-bg-card)] border border-[var(--color-border-custom)] shadow-2xl overflow-hidden"
            >
              {/* Header */}
              <div className="flex items-center justify-between px-6 py-4 border-b border-[var(--color-border-custom)]">
                <h2 className="text-base font-semibold text-[var(--color-text-primary)] flex items-center gap-2">
                  <Package className="h-4 w-4 text-[#44944A]" />
                  {t("product.preorder.button")}
                </h2>
                <button
                  onClick={() => setShowPreorderInfo(false)}
                  className="p-2 rounded-lg text-[var(--color-text-muted)] hover:text-[var(--color-text-primary)] hover:bg-[var(--color-bg-secondary)] transition-colors"
                >
                  <X className="h-4 w-4" />
                </button>
              </div>
              {/* Body */}
              <div className="p-6 space-y-4">
                <p className="text-sm text-[var(--color-text-secondary)] leading-relaxed">
                  {t("product.preorder.description")}
                </p>
                <div className="grid gap-3">
                  <TariffCard
                    icon="⚡"
                    name={t("product.preorder.tariffs.express.name")}
                    duration={t("product.preorder.tariffs.express.duration")}
                    accent="border-[var(--color-border-custom)] bg-[var(--color-bg-secondary)]"
                  />
                  <TariffCard
                    icon="🚀"
                    name={t("product.preorder.tariffs.accelerated.name")}
                    duration={t("product.preorder.tariffs.accelerated.duration")}
                    accent="border-[var(--color-border-custom)] bg-[var(--color-bg-secondary)]"
                  />
                  <TariffCard
                    icon="📦"
                    name={t("product.preorder.tariffs.standard.name")}
                    duration={t("product.preorder.tariffs.standard.duration")}
                    accent="border-[var(--color-border-custom)] bg-[var(--color-bg-secondary)]"
                  />
                </div>
                <p className="text-xs font-semibold text-[#44944A] text-center">
                  {t("product.preorder.priceNote")}
                </p>
                <p className="text-xs text-[var(--color-text-muted)] text-center leading-relaxed">
                  {t("product.preorder.telegramInfo")}
                </p>
              </div>
            </motion.div>
          </div>
        </div>
      )}
    </div>
  );
}

function TariffCard({
  icon,
  name,
  duration,
  accent,
}: {
  icon: string;
  name: string;
  duration: string;
  accent: string;
}) {
  return (
    <div
      className={`flex items-center gap-4 px-4 py-3 rounded-lg border ${accent}`}
    >
      <span className="text-xl">{icon}</span>
      <div>
        <p className="text-sm font-semibold text-[var(--color-text-primary)]">
          {name}
        </p>
        <p className="text-xs text-[var(--color-text-muted)]">{duration}</p>
      </div>
    </div>
  );
}

function TruckIcon() {
  return (
    <svg
      width="20"
      height="20"
      viewBox="0 0 24 24"
      fill="none"
      stroke="#44944A"
      strokeWidth="1.5"
      strokeLinecap="round"
      strokeLinejoin="round"
    >
      <rect x="1" y="3" width="15" height="13" />
      <polygon points="16 8 20 8 23 11 23 16 16 16 16 8" />
      <circle cx="5.5" cy="18.5" r="2.5" />
      <circle cx="18.5" cy="18.5" r="2.5" />
    </svg>
  );
}

function ReturnIcon() {
  return (
    <svg
      width="20"
      height="20"
      viewBox="0 0 24 24"
      fill="none"
      stroke="#44944A"
      strokeWidth="1.5"
      strokeLinecap="round"
      strokeLinejoin="round"
    >
      <path d="M3 12a9 9 0 1 0 9-9 9.75 9.75 0 0 0-6.74 2.74L3 8" />
      <path d="M3 3v5h5" />
    </svg>
  );
}

function SecureIcon() {
  return (
    <svg
      width="20"
      height="20"
      viewBox="0 0 24 24"
      fill="none"
      stroke="#44944A"
      strokeWidth="1.5"
      strokeLinecap="round"
      strokeLinejoin="round"
    >
      <rect x="3" y="11" width="18" height="11" rx="2" ry="2" />
      <path d="M7 11V7a5 5 0 0 1 10 0v4" />
    </svg>
  );
}
