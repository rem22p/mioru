import { useEffect, useState, useCallback } from "react";
import { motion, AnimatePresence } from "framer-motion";
import { useProductStore } from "@/stores/productStore";
import ProductTable from "@/components/products/ProductTable";
import ProductForm from "@/components/products/ProductForm";
import DeleteDialog from "@/components/products/DeleteDialog";
import type { Product } from "@/types";
import { SORT_OPTIONS } from "@/lib/constants";
import {
  Plus,
  Search,
  X,
  ChevronLeft,
  ChevronRight,
} from "lucide-react";

export default function Products() {
  const {
    products,
    total,
    loading,
    filter,
    categories,
    setFilter,
    resetFilter,
    fetchProducts,
    fetchCategories,
  } = useProductStore();

  const [formOpen, setFormOpen] = useState(false);
  const [editingProduct, setEditingProduct] = useState<Product | null>(null);
  const [deleteTarget, setDeleteTarget] = useState<Product | null>(null);

  const totalPages = Math.max(1, Math.ceil(total / filter.limit));

  useEffect(() => {
    fetchProducts();
    fetchCategories();
  }, [fetchProducts, fetchCategories]);

  const handleFilterChange = useCallback(
    (key: string, value: string) => {
      setFilter({ [key]: value, page: 1 });
    },
    [setFilter],
  );

  useEffect(() => {
    fetchProducts();
  }, [filter, fetchProducts]);

  const handleAdd = () => {
    setEditingProduct(null);
    setFormOpen(true);
  };

  const handleEdit = (product: Product) => {
    setEditingProduct(product);
    setFormOpen(true);
  };

  const handleDelete = (product: Product) => {
    setDeleteTarget(product);
  };

  const handleFormClose = () => {
    setFormOpen(false);
    setEditingProduct(null);
  };

  const handleFormSaved = () => {
    setFormOpen(false);
    setEditingProduct(null);
    fetchProducts();
  };

  const handleDeleteConfirm = async () => {
    if (!deleteTarget) return;
    const { deleteProduct } = await import("@/lib/api");
    try {
      await deleteProduct(deleteTarget.slug);
      setDeleteTarget(null);
      fetchProducts();
    } catch {
      // handled by store
    }
  };

  const handleDuplicate = async (product: Product) => {
    const { fetchProduct, createProduct } = await import("@/lib/api");
    try {
      const full = await fetchProduct(product.slug);
      const fd = new FormData();
      fd.append("name", `${full.name} (копия)`);
      fd.append("description", full.description);
      // KAN-14: duplicate sends the structured brand list (fall back to the
      // display brand for products fetched from a legacy payload).
      const brands = full.brands && full.brands.length > 0 ? full.brands : [full.brand];
      brands.forEach((b) => fd.append("brands[]", b));
      fd.append("price", String(full.price));
      fd.append("category_id", String(full.category_id));
      fd.append("in_stock", String(full.in_stock));
      fd.append("stock_quantity", String(full.stock_quantity));
      if (full.color) fd.append("color", full.color);
      if (full.model) fd.append("model", full.model);
      if (full.material) fd.append("material", full.material);
      full.sizes.forEach((s) => fd.append("sizes[]", s));
      full.care.forEach((c) => fd.append("care[]", c));
      full.size_chart.forEach((sc, i) => {
        fd.append(`size_chart[${i}][label]`, sc.label);
        if (sc.chest) fd.append(`size_chart[${i}][chest]`, sc.chest);
        if (sc.waist) fd.append(`size_chart[${i}][waist]`, sc.waist);
        if (sc.hips) fd.append(`size_chart[${i}][hips]`, sc.hips);
        if (sc.length) fd.append(`size_chart[${i}][length]`, sc.length);
        if (sc.foot_length)
          fd.append(`size_chart[${i}][foot_length]`, sc.foot_length);
        if (sc.wrist) fd.append(`size_chart[${i}][wrist]`, sc.wrist);
      });
      await createProduct(fd);
      fetchProducts();
    } catch {
      // ignore
    }
  };

  // Build category tree for select (categories come from the backend).
  const parentCats = categories.filter((c) => c.parent_id === null);

  const getSubcategories = (parentId: number) =>
    categories.filter((c) => c.parent_id === parentId);

  return (
    <div className="p-6 lg:p-8">
      {/* Header */}
      <motion.div
        initial={{ opacity: 0, y: 20 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ duration: 0.6, ease: [0.16, 1, 0.3, 1] }}
        className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4 mb-8"
      >
        <div>
          <h1 className="text-2xl lg:text-3xl font-bold tracking-tighter text-[var(--color-text-primary)]">
            Товары
          </h1>
          <p className="text-sm text-[var(--color-text-secondary)] mt-1">
            {total} {total === 1 ? "товар" : total < 5 ? "товара" : "товаров"}
          </p>
        </div>
        <button
          onClick={handleAdd}
          data-testid="product-add"
          className="flex items-center gap-2 rounded-xl bg-[#44944A] px-5 py-2.5 text-sm font-semibold text-black transition-all hover:shadow-[0_0_30px_rgba(68,148,74,0.3)]"
        >
          <Plus className="h-4 w-4" />
          Добавить товар
        </button>
      </motion.div>

      {/* Filter Bar */}
      <motion.div
        initial={{ opacity: 0, y: 20 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ duration: 0.6, delay: 0.1, ease: [0.16, 1, 0.3, 1] }}
        className="rounded-2xl bg-[var(--color-bg-card)] border border-[var(--color-border-custom)] p-4 mb-6"
      >
        <div className="flex flex-col lg:flex-row gap-3">
          {/* Search */}
          <div className="relative flex-1 min-w-0">
            <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-[var(--color-text-muted)]" />
            <input
              type="text"
              placeholder="Поиск по названию..."
              data-testid="product-search"
              value={filter.search}
              onChange={(e) => handleFilterChange("search", e.target.value)}
              className="w-full rounded-xl bg-[var(--color-bg-primary)] border border-[var(--color-border-custom)] pl-10 pr-4 py-2.5 text-sm text-[var(--color-text-primary)] outline-none focus:border-[#44944A] placeholder:text-[var(--color-text-muted)] transition-colors"
            />
          </div>

          {/* Category filter */}
          <select
            value={filter.category_id}
            onChange={(e) => handleFilterChange("category_id", e.target.value)}
            className="rounded-xl bg-[var(--color-bg-primary)] border border-[var(--color-border-custom)] px-4 py-2.5 text-sm text-[var(--color-text-primary)] outline-none focus:border-[#44944A] transition-colors min-w-[180px]"
          >
            <option value="">Все категории</option>
            {parentCats.map((pc) => (
              <optgroup key={pc.id} label={pc.name}>
                {getSubcategories(pc.id).length > 0 ? (
                  getSubcategories(pc.id).map((sc) => (
                    <option key={sc.id} value={String(sc.id)}>
                      {sc.name}
                    </option>
                  ))
                ) : (
                  <option value={String(pc.id)}>{pc.name}</option>
                )}
              </optgroup>
            ))}
          </select>

          {/* Brand filter */}
          <input
            type="text"
            placeholder="Бренд..."
            value={filter.brand}
            onChange={(e) => handleFilterChange("brand", e.target.value)}
            className="rounded-xl bg-[var(--color-bg-primary)] border border-[var(--color-border-custom)] px-4 py-2.5 text-sm text-[var(--color-text-primary)] outline-none focus:border-[#44944A] placeholder:text-[var(--color-text-muted)] transition-colors w-36"
          />

          {/* Sort */}
          <select
            value={filter.sort}
            onChange={(e) => handleFilterChange("sort", e.target.value)}
            className="rounded-xl bg-[var(--color-bg-primary)] border border-[var(--color-border-custom)] px-4 py-2.5 text-sm text-[var(--color-text-primary)] outline-none focus:border-[#44944A] transition-colors"
          >
            {SORT_OPTIONS.map((opt) => (
              <option key={opt.value} value={opt.value}>
                {opt.label}
              </option>
            ))}
          </select>

          {/* Reset */}
          {(filter.search || filter.category_id || filter.brand) && (
            <button
              onClick={resetFilter}
              className="flex items-center gap-1.5 rounded-xl bg-[var(--color-bg-primary)] border border-[var(--color-border-custom)] px-4 py-2.5 text-sm text-[var(--color-text-muted)] hover:text-[var(--color-text-primary)] transition-colors"
            >
              <X className="h-4 w-4" />
              Сбросить
            </button>
          )}
        </div>
      </motion.div>

      {/* Table */}
      <ProductTable
        products={products}
        loading={loading}
        onEdit={handleEdit}
        onDelete={handleDelete}
        onDuplicate={handleDuplicate}
      />

      {/* Pagination */}
      {totalPages > 1 && (
        <motion.div
          initial={{ opacity: 0 }}
          animate={{ opacity: 1 }}
          transition={{ delay: 0.3 }}
          className="flex items-center justify-between mt-6 text-sm"
        >
          <p className="text-[var(--color-text-muted)]">
            Страница {filter.page} из {totalPages}
          </p>
          <div className="flex items-center gap-2">
            <button
              onClick={() => setFilter({ page: Math.max(1, filter.page - 1) })}
              disabled={filter.page <= 1}
              className="flex items-center gap-1 rounded-xl bg-[var(--color-bg-card)] border border-[var(--color-border-custom)] px-3 py-2 text-sm text-[var(--color-text-secondary)] hover:text-[var(--color-text-primary)] disabled:opacity-30 transition-colors"
            >
              <ChevronLeft className="h-4 w-4" />
              Назад
            </button>
            <div className="flex gap-1">
              {Array.from({ length: totalPages }, (_, i) => i + 1)
                .filter((p) => {
                  if (totalPages <= 7) return true;
                  if (p === 1 || p === totalPages) return true;
                  if (Math.abs(p - filter.page) <= 1) return true;
                  return false;
                })
                .map((p, idx, arr) => {
                  const showEllipsis = idx > 0 && p - arr[idx - 1] > 1;
                  return (
                    <span key={p} className="flex items-center">
                      {showEllipsis && (
                        <span className="px-1 text-[var(--color-text-muted)]">
                          ...
                        </span>
                      )}
                      <button
                        onClick={() => setFilter({ page: p })}
                        className={`w-9 h-9 rounded-lg text-sm font-medium transition-colors ${
                          p === filter.page
                            ? "bg-[#44944A] text-black"
                            : "text-[var(--color-text-secondary)] hover:text-[var(--color-text-primary)] hover:bg-[var(--color-bg-card)]"
                        }`}
                      >
                        {p}
                      </button>
                    </span>
                  );
                })}
            </div>
            <button
              onClick={() =>
                setFilter({ page: Math.min(totalPages, filter.page + 1) })
              }
              disabled={filter.page >= totalPages}
              className="flex items-center gap-1 rounded-xl bg-[var(--color-bg-card)] border border-[var(--color-border-custom)] px-3 py-2 text-sm text-[var(--color-text-secondary)] hover:text-[var(--color-text-primary)] disabled:opacity-30 transition-colors"
            >
              Вперёд
              <ChevronRight className="h-4 w-4" />
            </button>
          </div>
        </motion.div>
      )}

      {/* Product Form Modal */}
      <AnimatePresence>
        {formOpen && (
          <ProductForm
            product={editingProduct}
            onClose={handleFormClose}
            onSaved={handleFormSaved}
          />
        )}
      </AnimatePresence>

      {/* Delete Dialog */}
      <AnimatePresence>
        {deleteTarget && (
          <DeleteDialog
            product={deleteTarget}
            onCancel={() => setDeleteTarget(null)}
            onConfirm={handleDeleteConfirm}
          />
        )}
      </AnimatePresence>
    </div>
  );
}
