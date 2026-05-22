import { Link } from "react-router-dom";
import { useCartStore } from "@/stores/cartStore";
import { Trash2, Plus, Minus, ShoppingBag, ArrowRight } from "lucide-react";
import { motion } from "framer-motion";
import { useTranslation } from "react-i18next";
import { Helmet } from "react-helmet-async";

export default function CartPage() {
  const { t } = useTranslation();
  const items = useCartStore((state) => state.items);
  const removeItem = useCartStore((state) => state.removeItem);
  const updateQuantity = useCartStore((state) => state.updateQuantity);
  const totalItems = useCartStore((state) => state.totalItems());
  const totalPrice = useCartStore((state) => state.totalPrice());
  const clearCart = useCartStore((state) => state.clearCart);

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
        <p className="mt-2 text-[var(--color-text-secondary)]">{t("cart.emptyDescription")}</p>
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

        <div className="mt-12 space-y-4">
          {items.map((item) => (
            <motion.div
              key={`${item.product.id}-${item.size}`}
              initial={{ opacity: 0, y: 20 }}
              animate={{ opacity: 1, y: 0 }}
              className="flex gap-4 rounded-2xl bg-[var(--color-bg-card)] border border-[var(--color-border-custom)] p-4"
            >
              <div className="h-24 w-24 shrink-0 rounded-xl bg-[var(--color-bg-primary)] border border-[var(--color-border-custom)] flex items-center justify-center text-3xl">
                {item.product.category.slug === "sneakers" && "👟"}
                {item.product.category.slug === "slides" && "🩴"}
                {item.product.category.slug === "tshirts" && "👕"}
                {item.product.category.slug === "shorts" && "🩳"}
                {item.product.category.slug === "bracelets" && "⛓️"}
              </div>

              <div className="flex flex-1 flex-col justify-between">
                <div>
                  <Link to={`/product/${item.product.slug}`}>
                    <h3 className="font-medium text-[var(--color-text-primary)] hover:text-[#44944A] transition-colors">
                      {item.product.name}
                    </h3>
                  </Link>
                  <p className="text-xs font-mono text-[var(--color-text-muted)] mt-1">
                    {t("cart.size")}: {item.size}
                  </p>
                </div>

                <div className="flex items-center justify-between">
                  <div className="flex items-center gap-1">
                    <button
                      onClick={() =>
                        updateQuantity(
                          item.product.id,
                          item.size,
                          item.quantity - 1,
                        )
                      }
                      className="rounded-lg border border-[var(--color-border-custom)] min-w-[44px] min-h-[44px] flex items-center justify-center text-[var(--color-text-secondary)] hover:text-white transition-colors"
                      aria-label="Уменьшить"
                    >
                      <Minus className="h-4 w-4" />
                    </button>
                    <span className="w-10 text-center text-sm font-medium text-[var(--color-text-primary)]">
                      {item.quantity}
                    </span>
                    <button
                      onClick={() =>
                        updateQuantity(
                          item.product.id,
                          item.size,
                          item.quantity + 1,
                        )
                      }
                      className="rounded-lg border border-[var(--color-border-custom)] min-w-[44px] min-h-[44px] flex items-center justify-center text-[var(--color-text-secondary)] hover:text-white transition-colors"
                      aria-label="Увеличить"
                    >
                      <Plus className="h-4 w-4" />
                    </button>
                  </div>

                  <div className="flex items-center gap-4">
                    <span className="text-sm font-bold text-[var(--color-text-primary)]">
                      {(item.product.price * item.quantity).toLocaleString(
                        "ru-RU",
                      )}{" "}
                      ₽
                    </span>
                    <button
                      onClick={() => removeItem(item.product.id, item.size)}
                      className="min-w-[44px] min-h-[44px] flex items-center justify-center text-[var(--color-text-muted)] hover:text-red-500 transition-colors rounded-lg"
                      aria-label={t("common.delete")}
                    >
                      <Trash2 className="h-4 w-4" />
                    </button>
                  </div>
                </div>
              </div>
            </motion.div>
          ))}
        </div>

        {/* Summary */}
        <div className="mt-8 rounded-2xl bg-[var(--color-bg-card)] border border-[var(--color-border-custom)] p-6">
          <div className="space-y-3">
            <div className="flex justify-between text-sm">
              <span className="text-[var(--color-text-secondary)]">
                {t("cart.items")} ({totalItems})
              </span>
              <span className="text-[var(--color-text-primary)]">
                {totalPrice.toLocaleString("ru-RU")} ₽
              </span>
            </div>
            <div className="flex justify-between text-sm">
              <span className="text-[var(--color-text-secondary)]">{t("cart.shipping")}</span>
              <span className="text-[#44944A]">{t("cart.free")}</span>
            </div>
            <div className="border-t border-[var(--color-border-custom)] pt-3">
              <div className="flex justify-between">
                <span className="font-semibold text-[var(--color-text-primary)]">
                  {t("cart.total")}
                </span>
                <span className="font-bold text-[#44944A] text-lg">
                  {totalPrice.toLocaleString("ru-RU")} ₽
                </span>
              </div>
            </div>
          </div>

          <div className="mt-6 flex gap-4">
            <Link
              to="/checkout"
              className="flex-1 rounded-xl bg-[#44944A] px-6 py-4 text-center text-sm font-semibold text-black transition-all hover:shadow-[0_0_30px_rgba(192,254,57,0.3)]"
            >
              {t("cart.checkout")}
            </Link>
            <button
              onClick={clearCart}
              className="rounded-xl border border-[var(--color-border-custom)] px-6 py-4 text-sm text-[var(--color-text-secondary)] transition-all hover:bg-[var(--color-bg-card)] hover:text-white"
            >
              {t("cart.clear")}
            </button>
          </div>
        </div>
      </div>
    </div>
  );
}
