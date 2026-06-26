import { Link } from "react-router-dom";
import { useCartStore } from "@/stores/cartStore";
import { useAuthStore } from "@/stores/authStore";
import { useFavoritesStore } from "@/stores/favoritesStore";
import { useCurrencyStore } from "@/stores/currencyStore";
import { formatPrice } from "@/lib/currency";
import { Trash2, Plus, Minus, ShoppingBag, ArrowRight, Package } from "lucide-react";
import { getThumbUrl, getImageUrl } from "@/lib/api";
import { motion } from "framer-motion";
import { useTranslation } from "react-i18next";
import { Helmet } from "@dr.pogodin/react-helmet";

export default function CartPage() {
  const { t } = useTranslation();
  const { currency } = useCurrencyStore();
  const items = useCartStore((state) => state.items);
  const removeItem = useCartStore((state) => state.removeItem);
  const updateQuantity = useCartStore((state) => state.updateQuantity);
  const totalItems = useCartStore((state) => state.totalItems());
  const totalPrice = useCartStore((state) => state.totalPrice());
  const clearCart = useCartStore((state) => state.clearCart);
  const isAuthenticated = useAuthStore((s) => s.isAuthenticated);

  const inStockItems = items.filter((i) => i.product.status === "in_stock");
  const preorderItems = items.filter((i) => i.product.status !== "in_stock");
  const hasMixed = inStockItems.length > 0 && preorderItems.length > 0;

  if (items.length === 0) {
    return (
      <div className="flex flex-col items-center justify-center px-6 py-32">
        <Helmet>
          <title>Корзина — MIORU</title>
          <meta
            name="description"
            content="Ваша корзина в MIORU. Проверьте выбранные товары и оформите заказ с виртуальной примеркой."
          />
          <meta property="og:title" content="Корзина — MIORU" />
          <link rel="canonical" href="https://mioru.store/cart" />
        </Helmet>
        <ShoppingBag className="h-16 w-16 text-[var(--color-border-light)]" />
        <h2 className="mt-6 text-2xl font-bold text-[var(--color-text-primary)]">
          {t("cart.empty")}
        </h2>
        <p className="mt-2 text-[var(--color-text-secondary)]">
          {t("cart.emptyDescription")}
        </p>
        <Link
          to="/catalog"
          className="mt-6 inline-flex items-center gap-2 rounded-xl bg-[#44944A] px-6 py-3 text-sm font-semibold text-black transition-all hover:shadow-[0_0_30px_rgba(192,254,57,0.3)]"
        >
          {t("cart.toCatalog")}
          <ArrowRight className="h-4 w-4" />
        </Link>
      </div>
    );
  }

  return (
    <div className="px-6 py-24 lg:px-8">
      <Helmet>
        <title>Корзина — MIORU</title>
        <meta
          name="description"
          content="Ваша корзина в MIORU. Проверьте выбранные товары и оформите заказ с виртуальной примеркой."
        />
        <meta property="og:title" content="Корзина — MIORU" />
        <link rel="canonical" href="https://mioru.store/cart" />
      </Helmet>
      <div className="mx-auto max-w-4xl">
        <motion.h1
          initial={{ opacity: 0, y: 20 }}
          animate={{ opacity: 1, y: 0 }}
          className="text-4xl font-bold tracking-tighter text-[var(--color-text-primary)] sm:text-5xl"
        >
          {t("cart.title")}
        </motion.h1>

        <div className="mt-12 space-y-8">
          {inStockItems.length > 0 && (
            <div className="rounded-2xl bg-[var(--color-bg-card)] border border-[var(--color-border-custom)] overflow-hidden">
              <div className="px-5 py-4 border-b border-[var(--color-border-custom)]">
                <h2 className="text-base font-bold text-[var(--color-text-primary)] flex items-center gap-2.5">
                  <span className="w-8 h-8 rounded-lg bg-[#44944A]/10 flex items-center justify-center">
                    <ShoppingBag className="h-4 w-4 text-[#44944A]" />
                  </span>
                  {t("cart.section.inStock")}
                  <span className="ml-auto text-xs font-normal text-[var(--color-text-muted)]">
                    {inStockItems.length} {t("cart.items")}
                  </span>
                </h2>
              </div>
              <div className="p-4 space-y-3">
                {inStockItems.map((item) => (
                  <CartRow key={`${item.product.id}-${item.size}`} item={item} t={t} currency={currency} updateQuantity={updateQuantity} removeItem={removeItem} />
                ))}
              </div>
            </div>
          )}
          {preorderItems.length > 0 && (
            <div className="rounded-2xl border-2 border-dashed border-[var(--color-border-custom)] overflow-hidden">
              <div className="px-5 py-4 border-b border-[var(--color-border-custom)] bg-[var(--color-bg-secondary)]">
                <h2 className="text-base font-bold text-[var(--color-text-primary)] flex items-center gap-2.5">
                  <span className="w-8 h-8 rounded-lg bg-[var(--color-bg-card)] border border-[var(--color-border-custom)] flex items-center justify-center">
                    <Package className="h-4 w-4 text-[var(--color-text-muted)]" />
                  </span>
                  {t("cart.section.preorder")}
                  <span className="ml-auto text-xs font-normal text-[var(--color-text-muted)]">
                    {preorderItems.length} {t("cart.items")}
                  </span>
                </h2>
              </div>
              <div className="px-5 py-3 text-xs text-[var(--color-text-muted)] leading-relaxed bg-[var(--color-bg-secondary)]">
                {t("cart.section.preorderDesc")}
              </div>
              <div className="p-4 space-y-3 bg-[var(--color-bg-card)]">
                {preorderItems.map((item) => (
                  <CartRow key={`${item.product.id}-${item.size}`} item={item} t={t} currency={currency} updateQuantity={updateQuantity} removeItem={removeItem} />
                ))}
              </div>
            </div>
          )}
        </div>

        {/* Summary */}
        <div className="mt-8 rounded-2xl bg-[var(--color-bg-card)] border border-[var(--color-border-custom)] p-6">
          <div className="space-y-3">
            <div className="flex justify-between text-sm">
              <span className="text-[var(--color-text-secondary)]">
                {t("cart.items")} ({totalItems})
              </span>
              <span className="text-[var(--color-text-primary)]">
                {formatPrice(totalPrice, currency)}
              </span>
            </div>
            <div className="flex justify-between text-sm">
              <span className="text-[var(--color-text-secondary)]">
                {t("cart.shipping")}
              </span>
              <span className="text-[#44944A]">{t("cart.free")}</span>
            </div>
            <div className="border-t border-[var(--color-border-custom)] pt-3">
              <div className="flex justify-between">
                <span className="font-semibold text-[var(--color-text-primary)]">
                  {t("cart.total")}
                </span>
                <span data-testid="cart-total" className="font-bold text-[#44944A] text-lg">
                  {formatPrice(totalPrice, currency)}
                </span>
              </div>
            </div>
          </div>

          {hasMixed && (
            <div className="mt-6 p-4 rounded-xl bg-amber-500/10 border border-amber-500/20 text-center">
              <p className="text-sm text-amber-600 dark:text-amber-400 font-medium">
                {t("checkout.mixedCart")}
              </p>
            </div>
          )}
          <div className="mt-6 flex gap-4">
            <Link
              data-testid="cart-checkout"
              to={hasMixed ? "#" : (isAuthenticated ? "/checkout" : "/profile?redirect=/checkout")}
              onClick={hasMixed ? (e) => e.preventDefault() : undefined}
              className={`flex-1 rounded-xl px-6 py-4 text-center text-sm font-semibold transition-all ${hasMixed ? "bg-[var(--color-bg-secondary)] border border-[var(--color-border-custom)] text-[var(--color-text-muted)] cursor-not-allowed" : "bg-[#44944A] text-black hover:shadow-[0_0_30px_rgba(192,254,57,0.3)]"}`}
            >
              {t("cart.checkout")}
            </Link>
            <button
              onClick={clearCart}
              className="rounded-xl border border-[var(--color-border-custom)] px-6 py-4 text-sm text-[var(--color-text-secondary)] transition-all hover:bg-[var(--color-bg-card)] hover:text-[var(--color-text-primary)]"
            >
              {t("cart.clear")}
            </button>
          </div>
        </div>
      </div>
    </div>
  );
}

function CartRow({
  item, t, currency, updateQuantity, removeItem,
}: {
  item: import("@/types").CartItem;
  t: (key: string) => string;
  currency: string;
  updateQuantity: (pid: number | string, size: string, qty: number) => void;
  removeItem: (pid: number | string, size: string) => void;
}) {
  const isPreorder = item.product.status !== "in_stock";
  return (
    <motion.div
      key={`${item.product.id}-${item.size}`}
      data-testid="cart-row"
      data-product-id={item.product.id}
      initial={{ opacity: 0, y: 20 }}
      animate={{ opacity: 1, y: 0 }}
      className="flex items-start gap-3 sm:gap-4 rounded-2xl bg-[var(--color-bg-card)] border border-[var(--color-border-custom)] p-3 sm:p-4"
    >
      <Link to={`/product/${item.product.slug}`} className="relative aspect-[4/5] w-20 sm:w-24 shrink-0 rounded-xl bg-[var(--color-bg-secondary)] border border-[var(--color-border-custom)] overflow-hidden">
        {item.product.images?.[0]?.url ? (
          <img
            src={getThumbUrl(item.product.images[0].url)}
            alt={item.product.name}
            className="absolute inset-0 w-full h-full object-cover"
            onError={(e) => {
              const t = e.currentTarget;
              if (!t.dataset.fallback) {
                t.dataset.fallback = "1";
                t.src = getImageUrl(item.product.images[0].url);
              }
            }}
          />
        ) : (
          <div className="absolute inset-0 flex items-center justify-center text-2xl">📦</div>
        )}
      </Link>
      <div className="flex flex-1 flex-col min-w-0 self-stretch">
        <div className="flex items-start justify-between gap-2">
          <Link to={`/product/${item.product.slug}`} className="min-w-0">
            <h3 className="font-medium text-[var(--color-text-primary)] hover:text-[#44944A] transition-colors text-sm leading-snug line-clamp-2">
              {item.product.name}
            </h3>
          </Link>
          <span className="text-sm font-bold text-[var(--color-text-primary)] whitespace-nowrap shrink-0">
            {formatPrice(item.product.price * item.quantity, currency)}
          </span>
        </div>
        {isPreorder ? (
          <p className="text-xs text-[var(--color-text-muted)] mt-1">
            {item.height != null && `${t("product.preorder.height")}: ${item.height} см`}
            {item.height != null && item.weight != null && " · "}
            {item.weight != null && `${t("product.preorder.weight")}: ${item.weight} кг`}
          </p>
        ) : (
          <p className="text-xs font-mono text-[var(--color-text-muted)] mt-1">
            {t("cart.size")}: {item.size}
          </p>
        )}
        <div className="flex items-center justify-between mt-auto pt-2">
          <div className="flex items-center gap-1">
            <button
              onClick={() => updateQuantity(item.product.id, item.size, item.quantity - 1)}
              className="rounded-lg border border-[var(--color-border-custom)] w-8 h-8 flex items-center justify-center text-[var(--color-text-secondary)] hover:text-[var(--color-text-primary)] transition-colors"
              aria-label="Уменьшить"
            >
              <Minus className="h-3.5 w-3.5" />
            </button>
            <span className="w-7 text-center text-sm font-medium text-[var(--color-text-primary)]">{item.quantity}</span>
            <button
              onClick={() => updateQuantity(item.product.id, item.size, item.quantity + 1)}
              disabled={!isPreorder && (() => {
                const sizeObj = item.product.sizes?.find((s: any) => s.label === item.size);
                const maxStock = sizeObj ? sizeObj.stock_quantity : (item.product.stock_quantity || 0);
                return maxStock > 0 && item.quantity >= maxStock;
              })()}
              className="rounded-lg border border-[var(--color-border-custom)] w-8 h-8 flex items-center justify-center text-[var(--color-text-secondary)] hover:text-[var(--color-text-primary)] disabled:opacity-30 transition-colors"
              aria-label="Увеличить"
            >
              <Plus className="h-3.5 w-3.5" />
            </button>
          </div>
          <button
            onClick={() => removeItem(item.product.id, item.size)}
            className="w-8 h-8 flex items-center justify-center text-[var(--color-text-muted)] hover:text-red-500 transition-colors rounded-lg"
            aria-label={t("common.delete")}
          >
            <Trash2 className="h-4 w-4" />
          </button>
        </div>
      </div>
    </motion.div>
  );
}
