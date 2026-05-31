import { useEffect } from "react";
import { motion } from "framer-motion";
import { Link } from "react-router-dom";
import { useCatalogStore } from "@/stores/catalogStore";
import { useCartStore } from "@/stores/cartStore";
import { ShoppingBag } from "lucide-react";
import { useTranslation } from "react-i18next";

export default function FeaturedProducts() {
  const { t } = useTranslation();
  const { products, fetchProducts } = useCatalogStore();
  const addItem = useCartStore((state) => state.addItem);

  useEffect(() => {
    if (products.length === 0) {
      fetchProducts({ per_page: "6" });
    }
  }, [products.length, fetchProducts]);

  const featured = products.slice(0, 6);

  return (
    <section className="py-24">
      <div className="mx-auto max-w-7xl px-6 lg:px-8">
        <motion.div
          initial={{ opacity: 0, y: 40 }}
          whileInView={{ opacity: 1, y: 0 }}
          viewport={{ once: true }}
          transition={{ duration: 0.8 }}
          className="mb-16 flex items-end justify-between"
        >
          <div>
            <p className="text-xs font-mono uppercase tracking-[0.3em] text-[#558b5c]">
              {t("home.featured.badge")}
            </p>
            <h2 className="mt-4 text-4xl font-bold tracking-tighter text-[var(--color-text-primary)] sm:text-5xl">
              {t("home.featured.title")}
            </h2>
          </div>
          <Link
            to="/catalog"
            className="hidden text-sm text-[var(--color-text-secondary)] transition-colors hover:text-[#44944A] sm:block"
          >
            {t("home.featured.allProducts")}
          </Link>
        </motion.div>

        {featured.length === 0 ? (
          <div className="grid grid-cols-2 gap-4 sm:grid-cols-3 lg:grid-cols-6">
            {Array.from({ length: 6 }).map((_, i) => (
              <div
                key={i}
                className="relative aspect-[3/4] overflow-hidden rounded-xl bg-[var(--color-bg-card)] border border-[var(--color-border-custom)] animate-pulse"
              >
                <div className="absolute inset-0 flex items-center justify-center">
                  <span className="text-5xl opacity-20">👤</span>
                </div>
              </div>
            ))}
          </div>
        ) : (
          <div className="grid grid-cols-2 gap-4 sm:grid-cols-3 lg:grid-cols-6">
            {featured.map((product, index) => (
              <motion.div
                key={product.id}
                initial={{ opacity: 0, y: 40 }}
                whileInView={{ opacity: 1, y: 0 }}
                viewport={{ once: true }}
                transition={{ duration: 0.5, delay: index * 0.1 }}
                className="group"
              >
                <Link to={`/product/${product.slug}`}>
                  <div className="card-hover overflow-hidden rounded-xl bg-[var(--color-bg-card)] border border-[var(--color-border-custom)]">
                    <div className="relative aspect-[3/4] overflow-hidden">
                      {/* Product image placeholder */}
                      <div className="absolute inset-0 flex items-center justify-center">
                        <span className="text-3xl sm:text-5xl transition-transform duration-500 group-hover:scale-110">
                          📦
                        </span>
                      </div>

                      {/* Overlay on hover */}
                      <div className="absolute inset-0 bg-gradient-to-t from-[var(--color-bg-primary)] via-transparent to-transparent opacity-60 pointer-events-none" />

                      {/* Quick add button */}
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
                    </div>

                    {/* Bottom info */}
                    <div className="px-3 py-2.5 border-t border-[var(--color-border-custom)]">
                      <p className="text-[11px] font-mono uppercase tracking-wider text-[#558b5c]">
                        {product.category_name}
                      </p>
                      <h3 className="mt-0.5 text-sm font-medium text-[var(--color-text-primary)] line-clamp-1">
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
        )}
      </div>
    </section>
  );
}
