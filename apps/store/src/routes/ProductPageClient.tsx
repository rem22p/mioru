import { useState, lazy, Suspense } from "react";
import { motion } from "framer-motion";
import { Link } from "react-router-dom";
import { Product } from "@/types";
import { useCartStore } from "@/stores/cartStore";
import { useTranslation } from "react-i18next";
import {
  ShoppingBag,
  ArrowLeft,
  Star,
  Ruler,
  Check,
  Heart,
  Share2,
} from "lucide-react";
import { Helmet } from "react-helmet-async";

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
const ReviewsSection = lazy(
  () => import("@/components/product/ReviewsSection"),
);
const RelatedProducts = lazy(
  () => import("@/components/product/RelatedProducts"),
);

interface ProductPageClientProps {
  product: Product;
}

export default function ProductPageClient({ product }: ProductPageClientProps) {
  const { t } = useTranslation();
  const [selectedSize, setSelectedSize] = useState<string>("");
  const [showSizeChart, setShowSizeChart] = useState(false);
  const [isWishlisted, setIsWishlisted] = useState(false);
  const [addedToCart, setAddedToCart] = useState(false);
  const addItem = useCartStore((state) => state.addItem);

  const averageRating =
    product.reviews.length > 0
      ? Math.round(
          product.reviews.reduce((s, r) => s + r.rating, 0) /
            product.reviews.length,
        )
      : 0;

  const handleAddToCart = () => {
    if (selectedSize) {
      addItem(product, selectedSize);
      setAddedToCart(true);
      setTimeout(() => setAddedToCart(false), 2000);
    }
  };

  return (
    <div className="px-6 py-24 lg:px-8">
      <Helmet>
        <title>
          {product.name} — купить в MIORU | Цена{" "}
          {product.price.toLocaleString("ru-RU")} ₽
        </title>
        <meta
          name="description"
          content={`${product.name} — ${product.description.slice(0, 150)}... Цена: ${product.price.toLocaleString("ru-RU")} ₽. Виртуальная примерка на 3D-аватаре. Быстрая доставка.`}
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
            className="inline-flex items-center gap-2 text-sm text-[var(--color-text-secondary)] transition-colors hover:text-white mb-8"
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
                  <div className="aspect-square rounded-2xl bg-[var(--color-bg-card)] border border-[var(--color-border-custom)] animate-pulse" />
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
                {product.category.name}
              </p>
              <h1 className="mt-3 text-4xl font-bold tracking-tighter text-[var(--color-text-primary)] sm:text-5xl">
                {product.name}
              </h1>

              {/* Rating & Price Row */}
              <div className="mt-6 flex items-center justify-between">
                <div className="flex items-center gap-3">
                  <div className="flex items-center gap-1">
                    {[1, 2, 3, 4, 5].map((star) => (
                      <Star
                        key={star}
                        className={`h-4 w-4 ${
                          star <= averageRating
                            ? "fill-[#44944A] text-[#44944A]"
                            : "fill-transparent text-[var(--color-border-light)]"
                        }`}
                      />
                    ))}
                  </div>
                  <span className="text-xs text-[var(--color-text-secondary)]">
                    {t("product.reviews", { count: product.reviews.length })}
                  </span>
                </div>
                <p className="text-3xl font-bold text-[#44944A]">
                  {product.price.toLocaleString("ru-RU")} ₽
                </p>
              </div>

              {/* XP Badge */}
              <div className="mt-4 inline-flex items-center gap-2 rounded-full bg-[#44944A]/10 px-4 py-2 text-xs font-mono text-[#44944A] w-fit">
                <Star className="h-3 w-3 fill-[#44944A]" />
                {t("product.xpReward", { xp: product.xpReward })}
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
                  <button
                    onClick={() => setShowSizeChart(true)}
                    className="inline-flex items-center gap-1.5 text-xs text-[var(--color-text-secondary)] hover:text-[#44944A] transition-colors min-h-[44px] px-2"
                  >
                    <Ruler className="h-3.5 w-3.5" />
                    {t("product.sizeChart")}
                  </button>
                </div>
                <div className="flex flex-wrap gap-3">
                  {product.sizes.map((size) => (
                    <button
                      key={size}
                      onClick={() => setSelectedSize(size)}
                      className={`relative rounded-xl border px-5 py-3 text-sm font-medium transition-all min-h-[44px] ${
                        selectedSize === size
                          ? "border-[#44944A] bg-[#44944A] text-black"
                          : "border-[var(--color-border-custom)] text-[var(--color-text-secondary)] hover:border-[var(--color-text-muted)] hover:text-white"
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
                <button
                  onClick={handleAddToCart}
                  disabled={!selectedSize}
                  className={`flex flex-1 items-center justify-center gap-2 rounded-xl px-6 py-4 text-sm font-semibold transition-all min-h-[44px] ${
                    addedToCart
                      ? "bg-[#558b5c] text-[var(--color-text-primary)]"
                      : "bg-[#44944A] text-black hover:shadow-[0_0_30px_rgba(192,254,57,0.3)]"
                  } disabled:opacity-50 disabled:cursor-not-allowed`}
                >
                  {addedToCart ? (
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
                <button
                  onClick={() => setIsWishlisted(!isWishlisted)}
                  className={`flex items-center justify-center gap-2 rounded-xl border px-5 py-4 text-sm transition-all min-h-[44px] min-w-[44px] ${
                    isWishlisted
                      ? "border-[#44944A] text-[#44944A]"
                      : "border-[var(--color-border-custom)] text-[var(--color-text-secondary)] hover:border-[var(--color-text-muted)] hover:text-white"
                  }`}
                  aria-label={
                    isWishlisted
                      ? t("product.unwishlist")
                      : t("product.wishlist")
                  }
                >
                  <Heart
                    className={`h-4 w-4 ${isWishlisted ? "fill-[#44944A]" : ""}`}
                  />
                </button>
                <button
                  onClick={() => {
                    if (typeof window !== "undefined") {
                      if (navigator.share) {
                        navigator.share({
                          title: product.name,
                          url: window.location.href,
                        });
                      } else {
                        navigator.clipboard.writeText(window.location.href);
                      }
                    }
                  }}
                  className="flex items-center justify-center gap-2 rounded-xl border border-[var(--color-border-custom)] px-5 py-4 text-sm text-[var(--color-text-secondary)] transition-all hover:border-[var(--color-text-muted)] hover:text-white min-h-[44px] min-w-[44px]"
                  aria-label={t("product.share")}
                >
                  <Share2 className="h-4 w-4" />
                </button>
              </div>

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

          {/* Reviews */}
          <Suspense fallback={null}>
            <ReviewsSection reviews={product.reviews} />
          </Suspense>

          {/* Related Products */}
          <Suspense fallback={null}>
            <RelatedProducts
              relatedProductIds={product.relatedProductIds}
              currentProductId={product.id}
            />
          </Suspense>
        </motion.div>
      </div>

      {/* Size Chart Modal */}
      <Suspense fallback={null}>
        <SizeChartModal
          isOpen={showSizeChart}
          onClose={() => setShowSizeChart(false)}
          product={product}
        />
      </Suspense>
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
