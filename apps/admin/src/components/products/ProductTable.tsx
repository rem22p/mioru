import { useState } from "react";
import { motion } from "framer-motion";
import type { Product } from "@/types";
import { getImageUrl } from "@/lib/api";
import {
  Pencil,
  Copy,
  Trash2,
  Package,
  Circle,
  CircleCheck,
  Image as ImageIcon,
  Tag,
  Layers,
} from "lucide-react";

interface ProductTableProps {
  products: Product[];
  loading: boolean;
  onEdit: (p: Product) => void;
  onDelete: (p: Product) => void;
  onDuplicate: (p: Product) => void;
}

function formatPrice(price: number): string {
  return new Intl.NumberFormat("ru-RU", {
    style: "currency",
    currency: "MDL",
    minimumFractionDigits: 0,
    maximumFractionDigits: 0,
  }).format(price);
}

function SkeletonCard() {
  return (
    <div className="rounded-2xl bg-[var(--color-bg-card)] border border-[var(--color-border-custom)] overflow-hidden animate-pulse">
      <div className="aspect-[4/5] bg-[var(--color-border-custom)]" />
      <div className="p-4 space-y-3">
        <div className="h-4 w-3/4 bg-[var(--color-border-custom)] rounded" />
        <div className="h-3 w-1/2 bg-[var(--color-border-custom)] rounded" />
        <div className="flex gap-2">
          <div className="h-5 w-16 bg-[var(--color-border-custom)] rounded-full" />
          <div className="h-5 w-14 bg-[var(--color-border-custom)] rounded-full" />
        </div>
        <div className="h-5 w-20 bg-[var(--color-border-custom)] rounded" />
      </div>
    </div>
  );
}

function EmptyProducts() {
  return (
    <motion.div
      initial={{ opacity: 0, y: 20 }}
      animate={{ opacity: 1, y: 0 }}
      className="flex flex-col items-center justify-center py-20 text-center col-span-full"
    >
      <div className="w-20 h-20 rounded-2xl bg-[var(--color-bg-primary)] border border-[var(--color-border-custom)] flex items-center justify-center mb-6">
        <Package className="h-10 w-10 text-[var(--color-text-muted)]" />
      </div>
      <h3 className="text-lg font-bold tracking-tighter text-[var(--color-text-primary)] mb-2">
        Нет товаров
      </h3>
      <p className="text-sm text-[var(--color-text-secondary)] max-w-md">
        Здесь будут отображаться товары вашего магазина. Нажмите «Добавить
        товар», чтобы создать первый.
      </p>
    </motion.div>
  );
}

function thumbUrl(url: string): string {
  return getImageUrl(url).replace("/uploads/", "/uploads/thumb_");
}

const cardVariant = {
  hidden: { opacity: 0, y: 12 },
  show: { opacity: 1, y: 0 },
};

export default function ProductTable({
  products,
  loading,
  onEdit,
  onDelete,
  onDuplicate,
}: ProductTableProps) {
  const [hoveredId, setHoveredId] = useState<number | null>(null);

  if (loading) {
    return (
      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-4">
        {Array.from({ length: 8 }, (_, i) => (
          <SkeletonCard key={i} />
        ))}
      </div>
    );
  }

  if (products.length === 0) {
    return <EmptyProducts />;
  }

  return (
    <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-4">
      {products.map((product, i) => (
        <motion.div
          key={product.id}
          data-testid="product-row"
          data-slug={product.slug}
          variants={cardVariant}
          initial="hidden"
          animate="show"
          transition={{ delay: i * 0.03 }}
          onMouseEnter={() => setHoveredId(product.id)}
          onMouseLeave={() => setHoveredId(null)}
          className="rounded-2xl bg-[var(--color-bg-card)] border border-[var(--color-border-custom)] overflow-hidden hover:border-[var(--color-text-muted)] transition-all cursor-pointer group"
          onClick={() => onEdit(product)}
        >
          {/* Image */}
          <div className="aspect-[4/5] relative bg-[var(--color-bg-secondary)] overflow-hidden">
            {product.images?.[0]?.url ? (
              <img
                src={thumbUrl(product.images[0].url)}
                alt={product.name}
                className="w-full h-full object-cover transition-transform duration-300 group-hover:scale-105"
                loading="lazy"
                onError={(e) => {
                  (e.target as HTMLImageElement).src = getImageUrl(
                    product.images[0].url,
                  );
                }}
              />
            ) : (
              <div className="w-full h-full flex items-center justify-center">
                <ImageIcon className="h-10 w-10 text-[var(--color-text-muted)]" />
              </div>
            )}

            {/* Hover actions overlay */}
            {hoveredId === product.id && (
              <motion.div
                initial={{ opacity: 0 }}
                animate={{ opacity: 1 }}
                className="absolute inset-0 bg-black/40 flex items-center justify-center gap-2"
              >
                <button
                  onClick={(e) => {
                    e.stopPropagation();
                    onEdit(product);
                  }}
                  data-testid="product-edit"
                  className="h-10 w-10 rounded-xl bg-white/90 text-gray-800 flex items-center justify-center hover:bg-white transition-colors"
                  title="Редактировать"
                >
                  <Pencil className="h-4 w-4" />
                </button>
                <button
                  onClick={(e) => {
                    e.stopPropagation();
                    onDuplicate(product);
                  }}
                  data-testid="product-duplicate"
                  className="h-10 w-10 rounded-xl bg-white/90 text-gray-800 flex items-center justify-center hover:bg-white transition-colors"
                  title="Дублировать"
                >
                  <Copy className="h-4 w-4" />
                </button>
                <button
                  onClick={(e) => {
                    e.stopPropagation();
                    onDelete(product);
                  }}
                  data-testid="product-delete"
                  className="h-10 w-10 rounded-xl bg-white/90 text-red-500 flex items-center justify-center hover:bg-white transition-colors"
                  title="Удалить"
                >
                  <Trash2 className="h-4 w-4" />
                </button>
              </motion.div>
            )}
          </div>

          {/* Info */}
          <div className="p-4 space-y-2.5">
            {/* Name */}
            <h3 className="text-sm font-semibold text-[var(--color-text-primary)] leading-tight line-clamp-2 group-hover:text-[#44944A] transition-colors">
              {product.name}
            </h3>

            {/* Color + Model */}
            {(product.color || product.model) && (
              <p className="text-xs text-[var(--color-text-muted)] truncate">
                {[product.color, product.model].filter(Boolean).join(" · ")}
              </p>
            )}

            {/* Badges: category + brand */}
            <div className="flex items-center gap-1.5 flex-wrap">
              {product.category_name && (
                <span className="inline-flex items-center gap-1 text-[10px] font-medium text-[var(--color-text-muted)] bg-[var(--color-bg-secondary)] px-2 py-0.5 rounded-full">
                  <Layers className="h-2.5 w-2.5" />
                  {product.category_name}
                </span>
              )}
              {product.brand && (
                <span className="inline-flex items-center gap-1 text-[10px] font-medium text-[var(--color-text-muted)] bg-[var(--color-bg-secondary)] px-2 py-0.5 rounded-full">
                  <Tag className="h-2.5 w-2.5" />
                  {product.brand}
                </span>
              )}
            </div>

            {/* Price + Status */}
            <div className="flex items-center justify-between pt-1">
              <span className="text-base font-bold text-[var(--color-text-primary)] tabular-nums">
                {formatPrice(product.price)}
              </span>

              {product.status === "in_stock" ? (
                <span className="inline-flex items-center gap-1 text-[10px] font-medium text-green-500 bg-green-500/10 px-2 py-0.5 rounded-full">
                  <CircleCheck className="h-2.5 w-2.5" />В наличии
                  {product.stock_quantity > 0 && ` (${product.stock_quantity})`}
                </span>
              ) : product.status === "pre_order" ? (
                <span className="inline-flex items-center gap-1 text-[10px] font-medium text-yellow-500 bg-yellow-500/10 px-2 py-0.5 rounded-full">
                  <Circle className="h-2.5 w-2.5" />
                  Под заказ
                </span>
              ) : (
                <span className="inline-flex items-center gap-1 text-[10px] font-medium text-red-500 bg-red-500/10 px-2 py-0.5 rounded-full">
                  <Circle className="h-2.5 w-2.5" />
                  Нет
                </span>
              )}
            </div>

            {/* Sizes */}
            {product.sizes && product.sizes.length > 0 && (
              <div className="flex items-center gap-1 flex-wrap pt-0.5">
                {product.sizes.map((size) => (
                  <span
                    key={typeof size === "string" ? size : size.label}
                    className="text-[10px] font-medium text-[var(--color-text-muted)] bg-[var(--color-bg-secondary)] border border-[var(--color-border-custom)] px-1.5 py-0.5 rounded"
                  >
                    {typeof size === "string" ? size : size.label}
                  </span>
                ))}
              </div>
            )}
          </div>
        </motion.div>
      ))}
    </div>
  );
}
