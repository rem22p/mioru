import { useState, useEffect } from "react";
import { useParams, Link } from "react-router-dom";
import { fetchStoreProduct } from "@/lib/api";
import type { Product } from "@/types";
import ProductPageClient from "./ProductPageClient";
import { ArrowLeft } from "lucide-react";
import { useTranslation } from "react-i18next";

export default function ProductPage() {
  const { slug } = useParams<{ slug: string }>();
  const { t } = useTranslation();
  const [product, setProduct] = useState<Product | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [notFound, setNotFound] = useState(false);

  useEffect(() => {
    if (!slug) return;
    let cancelled = false;

    setLoading(true);
    setError(null);
    setNotFound(false);
    setProduct(null);

    fetchStoreProduct(slug)
      .then((data) => {
        if (!cancelled) {
          setProduct(data);
          setLoading(false);
        }
      })
      .catch((err) => {
        if (!cancelled) {
          const message =
            err instanceof Error ? err.message : "Failed to fetch product";
          if (
            message.toLowerCase().includes("not found") ||
            message.includes("404")
          ) {
            setNotFound(true);
          } else {
            setError(message);
          }
          setLoading(false);
        }
      });

    return () => {
      cancelled = true;
    };
  }, [slug]);

  if (loading) {
    return (
      <div className="flex items-center justify-center min-h-[60vh]">
        <div className="text-center">
          <p className="text-[var(--color-text-muted)] font-mono text-sm">
            {t("common.loading")}
          </p>
        </div>
      </div>
    );
  }

  if (notFound) {
    return (
      <div className="flex items-center justify-center min-h-[60vh] px-6">
        <div className="text-center max-w-md">
          <div className="text-6xl mb-6">🔍</div>
          <h2 className="text-2xl font-bold tracking-tighter text-[var(--color-text-primary)] mb-3">
            {t("product.notFound")}
          </h2>
          <p className="text-sm text-[var(--color-text-secondary)] mb-8">
            {t("product.notFoundDesc")}
          </p>
          <Link
            to="/catalog"
            className="inline-flex items-center gap-2 rounded-xl bg-[#44944A] px-6 py-3 text-sm font-semibold text-black transition-all hover:shadow-[0_0_30px_rgba(68,148,74,0.3)]"
          >
            <ArrowLeft className="h-4 w-4" />
            {t("product.backToCatalog")}
          </Link>
        </div>
      </div>
    );
  }

  if (error) {
    return (
      <div className="flex items-center justify-center min-h-[60vh] px-6">
        <div className="text-center max-w-md">
          <div className="text-6xl mb-6">⚠️</div>
          <h2 className="text-2xl font-bold tracking-tighter text-[var(--color-text-primary)] mb-3">
            {t("common.error")}
          </h2>
          <p className="text-sm text-red-400 mb-8">{error}</p>
          <Link
            to="/catalog"
            className="inline-flex items-center gap-2 rounded-xl bg-[#44944A] px-6 py-3 text-sm font-semibold text-black transition-all hover:shadow-[0_0_30px_rgba(68,148,74,0.3)]"
          >
            <ArrowLeft className="h-4 w-4" />
            {t("product.backToCatalog")}
          </Link>
        </div>
      </div>
    );
  }

  if (!product) {
    return null;
  }

  return <ProductPageClient product={product} />;
}
