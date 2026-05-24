import { useEffect } from "react";
import { motion } from "framer-motion";
import { Link } from "react-router-dom";
import { useCatalogStore } from "@/stores/catalogStore";
import { ArrowUpRight } from "lucide-react";
import { useTranslation } from "react-i18next";

interface RelatedProductsProps {
  relatedProductIds: number[];
  currentProductId: number;
}

export default function RelatedProducts({
  relatedProductIds,
  currentProductId,
}: RelatedProductsProps) {
  const { t } = useTranslation();
  const { products, fetchProducts } = useCatalogStore();

  useEffect(() => {
    if (products.length === 0) {
      fetchProducts();
    }
  }, [products.length, fetchProducts]);

  const related = relatedProductIds.length
    ? products.filter(
        (p) => relatedProductIds.includes(p.id) && p.id !== currentProductId,
      )
    : // Fallback: show 4 random products from the same category
      products.filter((p) => p.id !== currentProductId).slice(0, 4);

  if (related.length === 0) return null;

  return (
    <div className="mt-16">
      <div className="flex items-center justify-between mb-6">
        <h3 className="text-lg font-semibold text-[var(--color-text-primary)]">
          {t("product.relatedProducts")}
        </h3>
        <Link
          to="/catalog"
          className="text-xs font-mono text-[var(--color-text-secondary)] hover:text-[#44944A] transition-colors uppercase tracking-wider flex items-center gap-1"
        >
          {t("product.allProducts")}
          <ArrowUpRight className="h-3.5 w-3.5" />
        </Link>
      </div>

      <div className="grid grid-cols-2 gap-4 sm:grid-cols-3 lg:grid-cols-4">
        {related.map((product, index) => (
          <motion.div
            key={product.id}
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.4, delay: index * 0.1 }}
          >
            <Link to={`/product/${product.slug}`}>
              <div className="group">
                <div className="relative aspect-square overflow-hidden rounded-xl bg-[var(--color-bg-card)] border border-[var(--color-border-custom)] transition-all group-hover:border-[#44944A]/50">
                  <div className="absolute inset-0 flex items-center justify-center">
                    <span className="text-4xl transition-transform duration-500 group-hover:scale-110">
                      📦
                    </span>
                  </div>

                  {/* Hover overlay */}
                  <div className="absolute inset-0 bg-[#44944A]/0 group-hover:bg-[#44944A]/5 transition-colors" />

                  {/* Arrow on hover */}
                  <div className="absolute top-3 right-3 w-8 h-8 rounded-full bg-[var(--color-bg-primary)]/60 border border-[var(--color-border-custom)] flex items-center justify-center opacity-0 group-hover:opacity-100 transition-all transform translate-y-1 group-hover:translate-y-0">
                    <ArrowUpRight className="h-4 w-4 text-[var(--color-text-primary)]" />
                  </div>
                </div>

                <div className="mt-3">
                  <p className="text-xs font-mono text-[var(--color-text-muted)] uppercase tracking-wider">
                    {product.category_name}
                  </p>
                  <p className="mt-1 text-sm font-medium text-[var(--color-text-primary)] transition-colors group-hover:text-[#44944A] line-clamp-1">
                    {product.name}
                  </p>
                  <p className="mt-1 text-sm font-bold text-[var(--color-text-primary)]">
                    {product.price.toLocaleString("ru-RU")} ₽
                  </p>
                </div>
              </div>
            </Link>
          </motion.div>
        ))}
      </div>
    </div>
  );
}
