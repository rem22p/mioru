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
} from "lucide-react";

interface ProductTableProps {
  products: Product[];
  loading: boolean;
  onEdit: (p: Product) => void;
  onDelete: (p: Product) => void;
  onDuplicate: (p: Product) => void;
}

function SkeletonRow({ index }: { index: number }) {
  return (
    <div
      className="grid grid-cols-12 gap-4 items-center px-4 py-3 border-b border-[var(--color-border-custom)] animate-pulse"
      style={{ animationDelay: `${index * 80}ms` }}
    >
      <div className="col-span-1">
        <div className="w-10 h-10 rounded-lg bg-[var(--color-border-custom)]" />
      </div>
      <div className="col-span-3">
        <div className="h-4 w-3/4 bg-[var(--color-border-custom)] rounded" />
      </div>
      <div className="col-span-2">
        <div className="h-4 w-1/2 bg-[var(--color-border-custom)] rounded" />
      </div>
      <div className="col-span-2">
        <div className="h-4 w-2/3 bg-[var(--color-border-custom)] rounded" />
      </div>
      <div className="col-span-1">
        <div className="h-4 w-16 bg-[var(--color-border-custom)] rounded" />
      </div>
      <div className="col-span-1">
        <div className="h-5 w-16 bg-[var(--color-border-custom)] rounded-full" />
      </div>
      <div className="col-span-2 flex gap-2 justify-end">
        <div className="h-8 w-8 bg-[var(--color-border-custom)] rounded-lg" />
        <div className="h-8 w-8 bg-[var(--color-border-custom)] rounded-lg" />
        <div className="h-8 w-8 bg-[var(--color-border-custom)] rounded-lg" />
      </div>
    </div>
  );
}

function EmptyProducts() {
  return (
    <motion.div
      initial={{ opacity: 0, y: 20 }}
      animate={{ opacity: 1, y: 0 }}
      className="flex flex-col items-center justify-center py-20 text-center"
    >
      <div className="w-20 h-20 rounded-2xl bg-[var(--color-bg-primary)] border border-[var(--color-border-custom)] flex items-center justify-center mb-6">
        <Package className="h-10 w-10 text-[var(--color-text-muted)]" />
      </div>
      <h3 className="text-lg font-bold tracking-tighter text-[var(--color-text-primary)] mb-2">
        Нет товаров
      </h3>
      <p className="text-sm text-[var(--color-text-secondary)] max-w-md">
        Здесь будут отображаться товары вашего магазина. Нажмите &laquo;Добавить
        товар&raquo;, чтобы создать первый.
      </p>
    </motion.div>
  );
}

function formatPrice(price: number): string {
  return new Intl.NumberFormat("ru-RU", {
    style: "currency",
    currency: "MDL",
    minimumFractionDigits: 0,
    maximumFractionDigits: 0,
  }).format(price);
}

const container = {
  hidden: { opacity: 0 },
  show: {
    opacity: 1,
    transition: {
      staggerChildren: 0.04,
    },
  },
};

const rowVariant = {
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
  if (loading) {
    return (
      <div className="rounded-2xl bg-[var(--color-bg-card)] border border-[var(--color-border-custom)] overflow-hidden">
        {/* Header */}
        <div className="grid grid-cols-12 gap-4 px-4 py-3 border-b border-[var(--color-border-custom)] text-xs font-semibold text-[var(--color-text-muted)] uppercase tracking-wider">
          <div className="col-span-1" />
          <div className="col-span-3">Название</div>
          <div className="col-span-2">Категория</div>
          <div className="col-span-2">Бренд</div>
          <div className="col-span-1">Цена</div>
          <div className="col-span-1">Статус</div>
          <div className="col-span-2 text-right">Действия</div>
        </div>
        {Array.from({ length: 5 }, (_, i) => (
          <SkeletonRow key={i} index={i} />
        ))}
      </div>
    );
  }

  if (products.length === 0) {
    return (
      <div className="rounded-2xl bg-[var(--color-bg-card)] border border-[var(--color-border-custom)] overflow-hidden">
        <EmptyProducts />
      </div>
    );
  }

  return (
    <div className="rounded-2xl bg-[var(--color-bg-card)] border border-[var(--color-border-custom)] overflow-hidden">
      {/* Header */}
      <div className="grid grid-cols-12 gap-4 px-4 py-3 border-b border-[var(--color-border-custom)] text-xs font-semibold text-[var(--color-text-muted)] uppercase tracking-wider">
        <div className="col-span-1" />
        <div className="col-span-3">Название</div>
        <div className="col-span-2">Категория</div>
        <div className="col-span-2">Бренд</div>
        <div className="col-span-1">Цена</div>
        <div className="col-span-1">Статус</div>
        <div className="col-span-2 text-right">Действия</div>
      </div>

      {/* Rows */}
      <motion.div variants={container} initial="hidden" animate="show">
        {products.map((product, i) => (
          <motion.div
            key={product.id}
            variants={rowVariant}
            className="grid grid-cols-12 gap-4 items-center px-4 py-3 border-b border-[var(--color-border-custom)] last:border-b-0 hover:bg-[var(--color-bg-primary)]/50 transition-colors cursor-pointer group"
            onClick={() => onEdit(product)}
          >
            {/* Image */}
            <div className="col-span-1">
              {product.images?.[0]?.url ? (
                <img
                  src={getImageUrl(product.images[0].url)}
                  alt={product.name}
                  className="w-10 h-10 rounded-lg object-cover border border-[var(--color-border-custom)]"
                  loading="lazy"
                />
              ) : (
                <div className="w-10 h-10 rounded-lg bg-[var(--color-bg-primary)] border border-[var(--color-border-custom)] flex items-center justify-center">
                  <ImageIcon className="h-4 w-4 text-[var(--color-text-muted)]" />
                </div>
              )}
            </div>

            {/* Name */}
            <div className="col-span-3 min-w-0">
              <p className="text-sm font-medium text-[var(--color-text-primary)] truncate group-hover:text-[#44944A] transition-colors">
                {product.name}
              </p>
              {product.color && (
                <p className="text-xs text-[var(--color-text-muted)]">
                  {product.color}
                  {product.model ? ` · ${product.model}` : ""}
                </p>
              )}
            </div>

            {/* Category */}
            <div className="col-span-2">
              <span className="text-sm text-[var(--color-text-secondary)] truncate block">
                {product.category_name || "—"}
              </span>
            </div>

            {/* Brand */}
            <div className="col-span-2">
              <span className="text-sm text-[var(--color-text-secondary)] truncate block">
                {product.brand || "—"}
              </span>
            </div>

            {/* Price */}
            <div className="col-span-1">
              <span className="text-sm font-mono font-medium text-[var(--color-text-primary)]">
                {formatPrice(product.price)}
              </span>
            </div>

            {/* Status */}
            <div className="col-span-1">
              {product.status === "in_stock" ? (
                <span className="inline-flex items-center gap-1.5 text-xs font-medium text-green-500 bg-green-500/10 px-2.5 py-1 rounded-full">
                  <CircleCheck className="h-3 w-3" />В наличии
                </span>
              ) : product.status === "pre_order" ? (
                <span className="inline-flex items-center gap-1.5 text-xs font-medium text-yellow-500 bg-yellow-500/10 px-2.5 py-1 rounded-full">
                  <Circle className="h-3 w-3" />
                  Под заказ
                </span>
              ) : (
                <span className="inline-flex items-center gap-1.5 text-xs font-medium text-red-500 bg-red-500/10 px-2.5 py-1 rounded-full">
                  <Circle className="h-3 w-3" />
                  Нет
                </span>
              )}
            </div>

            {/* Actions */}
            <div className="col-span-2 flex items-center justify-end gap-1">
              <button
                onClick={(e) => {
                  e.stopPropagation();
                  onEdit(product);
                }}
                className="h-8 w-8 flex items-center justify-center rounded-lg text-[var(--color-text-muted)] hover:text-[#44944A] hover:bg-[#44944A]/10 transition-colors"
                title="Редактировать"
              >
                <Pencil className="h-4 w-4" />
              </button>
              <button
                onClick={(e) => {
                  e.stopPropagation();
                  onDuplicate(product);
                }}
                className="h-8 w-8 flex items-center justify-center rounded-lg text-[var(--color-text-muted)] hover:text-[var(--color-text-primary)] hover:bg-[var(--color-bg-primary)] transition-colors"
                title="Дублировать"
              >
                <Copy className="h-4 w-4" />
              </button>
              <button
                onClick={(e) => {
                  e.stopPropagation();
                  onDelete(product);
                }}
                className="h-8 w-8 flex items-center justify-center rounded-lg text-[var(--color-text-muted)] hover:text-red-500 hover:bg-red-500/10 transition-colors"
                title="Удалить"
              >
                <Trash2 className="h-4 w-4" />
              </button>
            </div>
          </motion.div>
        ))}
      </motion.div>
    </div>
  );
}
