import { useState, useEffect, useRef, useCallback } from "react";
import { createPortal } from "react-dom";
import { motion } from "framer-motion";
import type { Product, Category, SizeChartEntry } from "@/types";
import { useProductStore } from "@/stores/productStore";
import ProductPreview from "./ProductPreview";
import {
  SIZE_OPTIONS_CLOTHING,
  SIZE_OPTIONS_SHOES,
  SIZE_OPTIONS_ACCESSORIES,
} from "@/lib/constants";
import {
  createProduct,
  updateProduct,
  uploadImage,
  getImageUrl,
} from "@/lib/api";
import {
  X,
  Upload,
  Plus,
  Trash2,
  Image as ImageIcon,
  Save,
  Eye,
  GripVertical,
  ChevronDown,
} from "lucide-react";

interface ProductFormProps {
  product: Product | null;
  onClose: () => void;
  onSaved: () => void;
}

function getCategoryPath(
  categoryId: number,
  cats: readonly Category[],
): number[] {
  const cat = cats.find((c) => c.id === categoryId);
  if (!cat) return [];
  if (cat.parent_id) {
    return [...getCategoryPath(cat.parent_id, cats), cat.id];
  }
  return [cat.id];
}

const TEXT_FIELD_STYLE =
  "w-full rounded-xl bg-[var(--color-bg-primary)] border border-[var(--color-border-custom)] px-4 py-2.5 text-sm text-[var(--color-text-primary)] outline-none focus:border-[#44944A] placeholder:text-[var(--color-text-muted)] transition-colors";

export default function ProductForm({
  product,
  onClose,
  onSaved,
}: ProductFormProps) {
  const { categories, categoriesLoading } = useProductStore();
  const cats = categories;
  const categoriesUnavailable = !categoriesLoading && cats.length === 0;

  const isEdit = !!product;

  // Basic fields
  const [name, setName] = useState(product?.name || "");
  const [slug, setSlug] = useState(product?.slug || "");
  const [description, setDescription] = useState(product?.description || "");
  const [brand, setBrand] = useState(product?.brand || "");
  const [price, setPrice] = useState(
    product?.price ? String(product.price) : "",
  );
  const [xpReward, setXpReward] = useState(
    product?.xp_reward ? String(product.xp_reward) : "0",
  );
  const [inStock, setInStock] = useState(product?.in_stock ?? true);
  const [status, setStatus] = useState(product?.status || "in_stock");
  const [stockQuantity, setStockQuantity] = useState(
    product?.stock_quantity ? String(product.stock_quantity) : "0",
  );
  const [selectedCategoryId, setSelectedCategoryId] = useState<number | "">(
    product?.category_id || "",
  );

  // Dynamic fields
  const [color, setColor] = useState(product?.color || "");
  const [model, setModel] = useState(product?.model || "");
  const [fit, setFit] = useState(product?.fit || "");
  const [material, setMaterial] = useState(product?.material || "");

  // Sizes — per-label with stock quantity
  const [selectedSizes, setSelectedSizes] = useState<{label: string; stock: number}[]>(
    (product?.sizes || []).map((s: string | {label: string; stock_quantity: number}) => ({
      label: typeof s === "string" ? s : s.label,
      stock: typeof s === "string" ? (product?.stock_quantity || 0) : (s.stock_quantity || 0),
    })),
  );

  // Size chart
  const [sizeChart, setSizeChart] = useState<SizeChartEntry[]>(
    product?.size_chart && product.size_chart.length > 0
      ? product.size_chart
      : [
          {
            label: "",
            chest: "",
            waist: "",
            hips: "",
            length: "",
            foot_length: "",
            wrist: "",
          },
        ],
  );

  // Care instructions
  const [careInstructions, setCareInstructions] = useState<string[]>(
    product?.care || [""],
  );

  // Images
  const [images, setImages] = useState<
    { id: string; url: string; file?: File }[]
  >(product?.images?.map((img) => ({ id: img.id, url: img.url })) || []);
  const [uploading, setUploading] = useState(false);
  const [dragOver, setDragOver] = useState(false);

  // State
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState("");
  const [previewOpen, setPreviewOpen] = useState(false);
  const fileInputRef = useRef<HTMLInputElement>(null);

  // Derive which criteria to show
  const selectedCategory =
    typeof selectedCategoryId === "number"
      ? cats.find((c) => c.id === selectedCategoryId)
      : null;
  const parentCategory = selectedCategory?.parent_id
    ? cats.find((c) => c.id === selectedCategory.parent_id)
    : selectedCategory;
  const criteria = parentCategory?.criteria || [];

  const showSize = criteria.includes("size");
  const showBrand = criteria.includes("brand");
  const showColor = criteria.includes("color");
  const showModel = criteria.includes("model");

  // Determine size options
  const getSizeOptions = (): readonly string[] => {
    if (!selectedCategoryId) return [];
    const path = getCategoryPath(selectedCategoryId, cats);
    if (path.length === 0) return [];
    const rootId = path[0];
    if (rootId === 11) return SIZE_OPTIONS_SHOES;
    if (rootId === 16) return SIZE_OPTIONS_ACCESSORIES;
    if (rootId === 1 || rootId === 23) return SIZE_OPTIONS_CLOTHING;
    return [];
  };

  const sizeOptions = getSizeOptions();

  // Slug auto-generation
  const generateSlug = (val: string) => {
    return val
      .toLowerCase()
      .replace(/[^a-zа-яё0-9\s-]/g, "")
      .replace(/[\s]+/g, "-")
      .replace(/-+/g, "-")
      .trim();
  };

  const handleNameChange = (val: string) => {
    setName(val);
    if (!isEdit) {
      setSlug(generateSlug(val));
    }
  };

  // Image upload
  const handleImageUpload = async (files: FileList | null) => {
    if (!files) return;
    setUploading(true);
    for (const file of Array.from(files)) {
      if (file.type !== "image/png") continue;
      try {
        const result = await uploadImage(file);
        setImages((prev) => [
          ...prev,
          { id: `img-${Date.now()}-${Math.random()}`, url: result.url, file },
        ]);
      } catch {
        // ignore
      }
    }
    setUploading(false);
  };

  const handleDragOver = (e: React.DragEvent) => {
    e.preventDefault();
    setDragOver(true);
  };
  const handleDragLeave = () => setDragOver(false);
  const handleDrop = (e: React.DragEvent) => {
    e.preventDefault();
    setDragOver(false);
    handleImageUpload(e.dataTransfer.files);
  };

  const removeImage = (id: string) => {
    setImages((prev) => prev.filter((img) => img.id !== id));
  };

  const toggleSize = (sizeLabel: string) => {
    setSelectedSizes((prev) => {
      const exists = prev.find((s) => s.label === sizeLabel);
      if (exists) return prev.filter((s) => s.label !== sizeLabel);
      return [...prev, { label: sizeLabel, stock: 0 }];
    });
  };

  const updateSizeStock = (label: string, stock: number) => {
    setSelectedSizes((prev) =>
      prev.map((s) => (s.label === label ? { ...s, stock } : s)),
    );
  };

  // Size chart
  const addSizeChartRow = () => {
    setSizeChart((prev) => [
      ...prev,
      {
        label: "",
        chest: "",
        waist: "",
        hips: "",
        length: "",
        foot_length: "",
        wrist: "",
      },
    ]);
  };

  const removeSizeChartRow = (index: number) => {
    setSizeChart((prev) => prev.filter((_, i) => i !== index));
  };

  const updateSizeChartRow = (
    index: number,
    field: keyof SizeChartEntry,
    value: string,
  ) => {
    setSizeChart((prev) =>
      prev.map((row, i) => (i === index ? { ...row, [field]: value } : row)),
    );
  };

  // Care instructions
  const addCareInstruction = () => setCareInstructions((prev) => [...prev, ""]);
  const removeCareInstruction = (index: number) =>
    setCareInstructions((prev) => prev.filter((_, i) => i !== index));
  const updateCareInstruction = (index: number, value: string) =>
    setCareInstructions((prev) =>
      prev.map((c, i) => (i === index ? value : c)),
    );

  // Submit
  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError("");

    if (!name.trim()) {
      setError("Введите название товара");
      return;
    }
    if (!selectedCategoryId) {
      setError("Выберите категорию");
      return;
    }
    if (!price || isNaN(Number(price)) || Number(price) < 0) {
      setError("Введите корректную цену");
      return;
    }

    setSaving(true);
    try {
      const fd = new FormData();
      fd.append("name", name.trim());
      fd.append("slug", slug || generateSlug(name));
      fd.append("description", description);
      if (showBrand || brand) fd.append("brand", brand);
      fd.append("price", price);
      fd.append("xp_reward", xpReward || "0");
      fd.append("category_id", String(selectedCategoryId));
      fd.append("in_stock", inStock ? "1" : "0");
      fd.append("status", status);
      fd.append("stock_quantity", stockQuantity || "0");
      if (showColor || color) fd.append("color", color);
      if (model) fd.append("model", model);
      if (fit) fd.append("fit", fit);
      if (material) fd.append("material", material);

      // Sizes
      selectedSizes.forEach((s) => fd.append("sizes[]", s.label));
      selectedSizes.forEach((s) => fd.append("size_stocks[]", String(s.stock)));

      // Size chart
      sizeChart
        .filter((row) => row.label.trim())
        .forEach((row, i) => {
          fd.append(`size_chart[${i}][label]`, row.label);
          if (row.chest) fd.append(`size_chart[${i}][chest]`, row.chest);
          if (row.waist) fd.append(`size_chart[${i}][waist]`, row.waist);
          if (row.hips) fd.append(`size_chart[${i}][hips]`, row.hips);
          if (row.length) fd.append(`size_chart[${i}][length]`, row.length);
          if (row.foot_length)
            fd.append(`size_chart[${i}][foot_length]`, row.foot_length);
          if (row.wrist) fd.append(`size_chart[${i}][wrist]`, row.wrist);
        });

      // Care instructions
      careInstructions
        .filter((c) => c.trim())
        .forEach((c) => fd.append("care[]", c));

      // Images — send existing URLs + new files
      images.forEach((img) => {
        if (img.file) {
          fd.append("images", img.file);
        } else if (img.url) {
          fd.append("existing_images[]", img.url);
        }
      });

      if (isEdit && product) {
        await updateProduct(product.slug, fd);
      } else {
        await createProduct(fd);
      }
      onSaved();
    } catch (err) {
      const msg = err instanceof Error ? err.message : "Ошибка сохранения";
      setError(msg);
    } finally {
      setSaving(false);
    }
  };

  // Build category tree for select
  const parentCats = cats.filter((c) => c.parent_id === null);
  const getSubcategories = (parentId: number) =>
    cats.filter((c) => c.parent_id === parentId);

  return (
    <>
      <motion.div
        initial={{ opacity: 0 }}
        animate={{ opacity: 1 }}
        exit={{ opacity: 0 }}
        transition={{ duration: 0.2 }}
        className="fixed inset-0 z-50 flex items-start justify-center bg-black/60 backdrop-blur-sm overflow-y-auto"
        onClick={onClose}
      >
        <motion.div
          initial={{ opacity: 0, scale: 0.95, y: 40 }}
          animate={{ opacity: 1, scale: 1, y: 0 }}
          exit={{ opacity: 0, scale: 0.95, y: 40 }}
          transition={{ duration: 0.4, ease: [0.16, 1, 0.3, 1] }}
          onClick={(e) => e.stopPropagation()}
          className="w-full max-w-3xl my-8 mx-4 rounded-2xl bg-[var(--color-bg-card)] border border-[var(--color-border-custom)] shadow-2xl"
        >
          {/* Header */}
          <div className="flex items-center justify-between p-6 border-b border-[var(--color-border-custom)] sticky top-0 bg-[var(--color-bg-card)] rounded-t-2xl z-10">
            <h2 className="text-xl font-bold tracking-tighter text-[var(--color-text-primary)]">
              {isEdit ? "Редактировать товар" : "Новый товар"}
            </h2>
            <div className="flex items-center gap-2">
              <button
                type="button"
                onClick={() => setPreviewOpen(true)}
                className="flex items-center gap-1.5 h-9 px-3 rounded-lg text-[var(--color-text-muted)] hover:text-[var(--color-accent)] hover:bg-[var(--color-bg-secondary)] transition-colors text-sm"
              >
                <Eye className="h-4 w-4" />
                Предпросмотр
              </button>
              <button
                onClick={onClose}
                className="h-8 w-8 flex items-center justify-center rounded-lg text-[var(--color-text-muted)] hover:text-[var(--color-text-primary)] hover:bg-[var(--color-bg-secondary)] transition-colors"
              >
                <X className="h-5 w-5" />
              </button>
            </div>
          </div>

          {/* Form */}
          <form onSubmit={handleSubmit} className="p-6 space-y-6">
            {/* Basic info */}
            <div className="space-y-4">
              <h3 className="text-sm font-semibold text-[var(--color-text-secondary)] uppercase tracking-wider">
                Основная информация
              </h3>

              <div>
                <label className="block text-sm font-medium text-[var(--color-text-secondary)] mb-1.5">
                  Название *
                </label>
                <input
                  type="text"
                  value={name}
                  onChange={(e) => handleNameChange(e.target.value)}
                  placeholder="Название товара"
                  data-testid="pf-name"
                  className={TEXT_FIELD_STYLE}
                  autoFocus
                />
              </div>

              <div>
                <label className="block text-sm font-medium text-[var(--color-text-secondary)] mb-1.5">
                  Slug
                </label>
                <input
                  type="text"
                  value={slug}
                  onChange={(e) => setSlug(e.target.value)}
                  placeholder="nazvanie-tovara"
                  data-testid="pf-slug"
                  className={`${TEXT_FIELD_STYLE} font-mono text-xs`}
                />
              </div>

              <div>
                <label className="block text-sm font-medium text-[var(--color-text-secondary)] mb-1.5">
                  Описание
                </label>
                <textarea
                  value={description}
                  onChange={(e) => setDescription(e.target.value)}
                  placeholder="Описание товара..."
                  rows={4}
                  className={`${TEXT_FIELD_STYLE} resize-none`}
                />
              </div>

              {/* Category */}
              <div>
                <label className="block text-sm font-medium text-[var(--color-text-secondary)] mb-1.5">
                  Категория *
                </label>
                <select
                  value={String(selectedCategoryId)}
                  onChange={(e) =>
                    setSelectedCategoryId(
                      e.target.value ? Number(e.target.value) : "",
                    )
                  }
                  disabled={categoriesUnavailable}
                  data-testid="pf-category"
                  className={TEXT_FIELD_STYLE}
                >
                  <option value="">Выберите категорию</option>
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
                {categoriesUnavailable && (
                  <p className="mt-1.5 text-sm text-[var(--color-danger,#f85149)]">
                    Не удалось загрузить категории — обновите страницу.
                  </p>
                )}
              </div>

              {/* Dynamic: Brand */}
              {showBrand && (
                <div>
                  <label className="block text-sm font-medium text-[var(--color-text-secondary)] mb-1.5">
                    Бренд
                  </label>
                  <input
                    type="text"
                    value={brand}
                    onChange={(e) => setBrand(e.target.value)}
                    placeholder="Nike, Adidas..."
                    className={TEXT_FIELD_STYLE}
                  />
                </div>
              )}

              {/* Dynamic: Color */}
              {showColor && (
                <div>
                  <label className="block text-sm font-medium text-[var(--color-text-secondary)] mb-1.5">
                    Цвет
                  </label>
                  <input
                    type="text"
                    value={color}
                    onChange={(e) => setColor(e.target.value)}
                    placeholder="Чёрный, белый..."
                    className={TEXT_FIELD_STYLE}
                  />
                </div>
              )}

              {/* Model — only for shoes */}
              {showModel && (
                <div>
                  <label className="block text-sm font-medium text-[var(--color-text-secondary)] mb-1.5">
                    Модель
                  </label>
                  <input
                    type="text"
                    value={model}
                    onChange={(e) => setModel(e.target.value)}
                    placeholder="Air Force, Samba..."
                    className={TEXT_FIELD_STYLE}
                  />
                </div>
              )}

              {/* Fit */}
              <div>
                <label className="block text-sm font-medium text-[var(--color-text-secondary)] mb-1.5">
                  Посадка (fit)
                </label>
                <select
                  value={fit}
                  onChange={(e) => setFit(e.target.value)}
                  className={TEXT_FIELD_STYLE}
                >
                  <option value="">Не выбрано</option>
                  <option value="slim">Slim</option>
                  <option value="regular">Regular</option>
                  <option value="loose">Loose / Oversized</option>
                  <option value="tailored">Tailored</option>
                </select>
              </div>

              {/* Material */}
              <div>
                <label className="block text-sm font-medium text-[var(--color-text-secondary)] mb-1.5">
                  Материал
                </label>
                <input
                  type="text"
                  value={material}
                  onChange={(e) => setMaterial(e.target.value)}
                  placeholder="100% хлопок"
                  className={TEXT_FIELD_STYLE}
                />
              </div>

              {/* Price row */}
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <label className="block text-sm font-medium text-[var(--color-text-secondary)] mb-1.5">
                    Цена (MDL) *
                  </label>
                  <input
                    type="number"
                    value={price}
                    onChange={(e) => setPrice(e.target.value)}
                    placeholder="0"
                    min="0"
                    data-testid="pf-price"
                    className={`${TEXT_FIELD_STYLE} font-mono`}
                  />
                </div>
                <div>
                  <label className="block text-sm font-medium text-[var(--color-text-secondary)] mb-1.5">
                    XP награда
                  </label>
                  <input
                    type="number"
                    value={xpReward}
                    onChange={(e) => setXpReward(e.target.value)}
                    placeholder="0"
                    min="0"
                    className={`${TEXT_FIELD_STYLE} font-mono`}
                  />
                </div>
              </div>

              {/* Status + Quantity */}
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <label className="block text-sm font-medium text-[var(--color-text-secondary)] mb-1.5">
                    Статус
                  </label>
                  <select
                    value={status}
                    onChange={(e) => {
                      setStatus(e.target.value);
                      setInStock(e.target.value === "in_stock");
                    }}
                    className={TEXT_FIELD_STYLE}
                  >
                    <option value="in_stock">В наличии</option>
                    <option value="pre_order">Под заказ</option>
                    <option value="none">Нет</option>
                  </select>
                </div>
                <div>
                  <label className="block text-sm font-medium text-[var(--color-text-secondary)] mb-1.5">
                    Количество
                  </label>
                  <input
                    type="number"
                    value={stockQuantity}
                    onChange={(e) => setStockQuantity(e.target.value)}
                    placeholder="0"
                    min="0"
                    className={`${TEXT_FIELD_STYLE} font-mono`}
                  />
                </div>
              </div>
            </div>

            {/* Sizes */}
            {showSize && sizeOptions.length > 0 && (
              <div className="space-y-4">
                <h3 className="text-sm font-semibold text-[var(--color-text-secondary)] uppercase tracking-wider">
                  Размеры
                </h3>
                <div className="flex flex-wrap gap-2">
                  {sizeOptions.map((size) => {
                    const sz = selectedSizes.find((s) => s.label === size);
                    const isSelected = !!sz;
                    return (
                      <button
                        key={size}
                        type="button"
                        onClick={() => toggleSize(size)}
                        className={`px-4 py-2 rounded-xl text-sm font-medium transition-all border ${
                          isSelected
                            ? "bg-[#44944A] text-black border-[#44944A]"
                            : "bg-[var(--color-bg-primary)] text-[var(--color-text-secondary)] border-[var(--color-border-custom)] hover:border-[var(--color-text-muted)]"
                        }`}
                      >
                        {size}
                      </button>
                    );
                  })}
                </div>
                {selectedSizes.length > 0 && (
                  <div className="space-y-2">
                    <h4 className="text-xs font-semibold text-[var(--color-text-muted)] uppercase tracking-wider">Количество на размер</h4>
                    {selectedSizes.map((sz) => (
                      <div key={sz.label} className="flex items-center gap-3">
                        <span className="w-8 text-sm font-medium text-[var(--color-text-primary)]">{sz.label}</span>
                        <input
                          type="number"
                          value={sz.stock}
                          onChange={(e) => updateSizeStock(sz.label, parseInt(e.target.value) || 0)}
                          min="0"
                          placeholder="0"
                          className={`${TEXT_FIELD_STYLE} w-24 font-mono`}
                        />
                      </div>
                    ))}
                  </div>
                )}
              </div>
            )}

            {/* Size chart */}
            <div className="space-y-4">
              <div className="flex items-center justify-between">
                <h3 className="text-sm font-semibold text-[var(--color-text-secondary)] uppercase tracking-wider">
                  Размерная сетка
                </h3>
                <button
                  type="button"
                  onClick={addSizeChartRow}
                  className="flex items-center gap-1.5 text-xs font-medium text-[#44944A] hover:underline"
                >
                  <Plus className="h-3.5 w-3.5" />
                  Добавить строку
                </button>
              </div>
              <div className="overflow-x-auto">
                <table className="w-full text-sm">
                  <thead>
                    <tr className="border-b border-[var(--color-border-custom)]">
                      <th className="text-left py-2 px-2 text-xs font-semibold text-[var(--color-text-muted)] uppercase">
                        Размер
                      </th>
                      <th className="text-left py-2 px-2 text-xs font-semibold text-[var(--color-text-muted)] uppercase">
                        Грудь
                      </th>
                      <th className="text-left py-2 px-2 text-xs font-semibold text-[var(--color-text-muted)] uppercase">
                        Талия
                      </th>
                      <th className="text-left py-2 px-2 text-xs font-semibold text-[var(--color-text-muted)] uppercase">
                        Бёдра
                      </th>
                      <th className="text-left py-2 px-2 text-xs font-semibold text-[var(--color-text-muted)] uppercase">
                        Длина
                      </th>
                      <th className="text-left py-2 px-2 text-xs font-semibold text-[var(--color-text-muted)] uppercase">
                        Длина стопы
                      </th>
                      <th className="text-left py-2 px-2 text-xs font-semibold text-[var(--color-text-muted)] uppercase">
                        Запястье
                      </th>
                      <th className="w-10" />
                    </tr>
                  </thead>
                  <tbody>
                    {sizeChart.map((row, i) => (
                      <tr
                        key={i}
                        className="border-b border-[var(--color-border-custom)] last:border-b-0"
                      >
                        <td className="py-1.5 px-1">
                          <input
                            type="text"
                            value={row.label || ""}
                            onChange={(e) =>
                              updateSizeChartRow(i, "label", e.target.value)
                            }
                            placeholder="M"
                            className="w-full rounded-lg bg-[var(--color-bg-primary)] border border-[var(--color-border-custom)] px-2 py-1.5 text-xs text-[var(--color-text-primary)] outline-none focus:border-[#44944A]"
                          />
                        </td>
                        <td className="py-1.5 px-1">
                          <input
                            type="text"
                            value={row.chest || ""}
                            onChange={(e) =>
                              updateSizeChartRow(i, "chest", e.target.value)
                            }
                            placeholder="96"
                            className="w-full rounded-lg bg-[var(--color-bg-primary)] border border-[var(--color-border-custom)] px-2 py-1.5 text-xs text-[var(--color-text-primary)] outline-none focus:border-[#44944A] font-mono"
                          />
                        </td>
                        <td className="py-1.5 px-1">
                          <input
                            type="text"
                            value={row.waist || ""}
                            onChange={(e) =>
                              updateSizeChartRow(i, "waist", e.target.value)
                            }
                            placeholder="76"
                            className="w-full rounded-lg bg-[var(--color-bg-primary)] border border-[var(--color-border-custom)] px-2 py-1.5 text-xs text-[var(--color-text-primary)] outline-none focus:border-[#44944A] font-mono"
                          />
                        </td>
                        <td className="py-1.5 px-1">
                          <input
                            type="text"
                            value={row.hips || ""}
                            onChange={(e) =>
                              updateSizeChartRow(i, "hips", e.target.value)
                            }
                            placeholder="100"
                            className="w-full rounded-lg bg-[var(--color-bg-primary)] border border-[var(--color-border-custom)] px-2 py-1.5 text-xs text-[var(--color-text-primary)] outline-none focus:border-[#44944A] font-mono"
                          />
                        </td>
                        <td className="py-1.5 px-1">
                          <input
                            type="text"
                            value={row.length || ""}
                            onChange={(e) =>
                              updateSizeChartRow(i, "length", e.target.value)
                            }
                            placeholder="72"
                            className="w-full rounded-lg bg-[var(--color-bg-primary)] border border-[var(--color-border-custom)] px-2 py-1.5 text-xs text-[var(--color-text-primary)] outline-none focus:border-[#44944A] font-mono"
                          />
                        </td>
                        <td className="py-1.5 px-1">
                          <input
                            type="text"
                            value={row.foot_length || ""}
                            onChange={(e) =>
                              updateSizeChartRow(
                                i,
                                "foot_length",
                                e.target.value,
                              )
                            }
                            placeholder="26.5"
                            className="w-full rounded-lg bg-[var(--color-bg-primary)] border border-[var(--color-border-custom)] px-2 py-1.5 text-xs text-[var(--color-text-primary)] outline-none focus:border-[#44944A] font-mono"
                          />
                        </td>
                        <td className="py-1.5 px-1">
                          <input
                            type="text"
                            value={row.wrist || ""}
                            onChange={(e) =>
                              updateSizeChartRow(i, "wrist", e.target.value)
                            }
                            placeholder="17"
                            className="w-full rounded-lg bg-[var(--color-bg-primary)] border border-[var(--color-border-custom)] px-2 py-1.5 text-xs text-[var(--color-text-primary)] outline-none focus:border-[#44944A] font-mono"
                          />
                        </td>
                        <td className="py-1.5 px-1">
                          <button
                            type="button"
                            onClick={() => removeSizeChartRow(i)}
                            className="h-6 w-6 flex items-center justify-center rounded text-[var(--color-text-muted)] hover:text-red-500 transition-colors"
                          >
                            <Trash2 className="h-3.5 w-3.5" />
                          </button>
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            </div>

            {/* Images */}
            <div className="space-y-4">
              <h3 className="text-sm font-semibold text-[var(--color-text-secondary)] uppercase tracking-wider">
                Изображения
              </h3>

              <div
                onDragOver={handleDragOver}
                onDragLeave={handleDragLeave}
                onDrop={handleDrop}
                className={`relative border-2 border-dashed rounded-2xl p-8 text-center transition-colors cursor-pointer ${
                  dragOver
                    ? "border-[#44944A] bg-[#44944A]/5"
                    : "border-[var(--color-border-custom)] hover:border-[var(--color-text-muted)]"
                }`}
                onClick={() => fileInputRef.current?.click()}
              >
                <Upload className="h-8 w-8 text-[var(--color-text-muted)] mx-auto mb-3" />
                <p className="text-sm text-[var(--color-text-secondary)]">
                  Перетащите изображения или нажмите для загрузки
                </p>
                <p className="text-xs text-[var(--color-text-muted)] mt-1">
                  PNG, JPG, WebP до 10MB
                </p>
                <input
                  ref={fileInputRef}
                  type="file"
                  accept="image/png"
                  multiple
                  data-testid="pf-image-input"
                  className="hidden"
                  onChange={(e) => handleImageUpload(e.target.files)}
                />
                {uploading && (
                  <div className="absolute inset-0 bg-[var(--color-bg-card)]/80 rounded-2xl flex items-center justify-center">
                    <div className="w-6 h-6 border-2 border-[#44944A] border-t-transparent rounded-full animate-spin" />
                  </div>
                )}
              </div>

              {images.length > 0 && (
                <div className="grid grid-cols-3 sm:grid-cols-4 md:grid-cols-6 gap-3">
                  {images.map((img, i) => (
                    <div
                      key={img.id}
                      className="relative group rounded-xl overflow-hidden border border-[var(--color-border-custom)] aspect-square"
                    >
                      <img
                        src={
                          img.file
                            ? URL.createObjectURL(img.file)
                            : getImageUrl(img.url)
                        }
                        alt={`img-${i}`}
                        className="w-full h-full object-cover"
                      />
                      <button
                        type="button"
                        onClick={() => removeImage(img.id)}
                        className="absolute top-1 right-1 h-6 w-6 rounded-lg bg-black/60 text-white flex items-center justify-center opacity-0 group-hover:opacity-100 transition-opacity hover:bg-red-500"
                      >
                        <X className="h-3.5 w-3.5" />
                      </button>
                      {i === 0 && (
                        <span className="absolute bottom-1 left-1 text-[10px] font-medium bg-black/60 text-white px-1.5 py-0.5 rounded">
                          Обложка
                        </span>
                      )}
                    </div>
                  ))}
                </div>
              )}
            </div>

            {/* Care instructions */}
            <div className="space-y-4">
              <div className="flex items-center justify-between">
                <h3 className="text-sm font-semibold text-[var(--color-text-secondary)] uppercase tracking-wider">
                  Уход
                </h3>
                <button
                  type="button"
                  onClick={addCareInstruction}
                  className="flex items-center gap-1.5 text-xs font-medium text-[#44944A] hover:underline"
                >
                  <Plus className="h-3.5 w-3.5" />
                  Добавить
                </button>
              </div>
              <div className="space-y-2">
                {careInstructions.map((inst, i) => (
                  <div key={i} className="flex gap-2">
                    <input
                      type="text"
                      value={inst}
                      onChange={(e) => updateCareInstruction(i, e.target.value)}
                      placeholder="Стирать при 30°C"
                      className={`${TEXT_FIELD_STYLE} flex-1`}
                    />
                    <button
                      type="button"
                      onClick={() => removeCareInstruction(i)}
                      className="h-10 w-10 flex items-center justify-center rounded-xl text-[var(--color-text-muted)] hover:text-red-500 hover:bg-red-500/10 transition-colors"
                    >
                      <Trash2 className="h-4 w-4" />
                    </button>
                  </div>
                ))}
              </div>
            </div>

            {/* Error */}
            {error && (
              <div
                data-testid="pf-error"
                className="p-4 rounded-xl bg-red-500/10 border border-red-500/20 text-sm text-red-500"
              >
                {error}
              </div>
            )}

            {/* Actions */}
            <div className="flex gap-3 justify-end pt-4 border-t border-[var(--color-border-custom)]">
              <button
                type="button"
                onClick={onClose}
                className="rounded-xl bg-[var(--color-bg-primary)] border border-[var(--color-border-custom)] px-5 py-2.5 text-sm font-medium text-[var(--color-text-secondary)] hover:text-[var(--color-text-primary)] transition-colors"
              >
                Отмена
              </button>
              <button
                type="submit"
                disabled={saving}
                data-testid="pf-submit"
                className="flex items-center gap-2 rounded-xl bg-[#44944A] px-5 py-2.5 text-sm font-semibold text-black transition-all hover:shadow-[0_0_30px_rgba(68,148,74,0.3)] disabled:opacity-50 disabled:cursor-not-allowed"
              >
                <Save className="h-4 w-4" />
                {saving
                  ? "Сохранение..."
                  : isEdit
                    ? "Сохранить"
                    : "Создать товар"}
              </button>
            </div>
          </form>
        </motion.div>
      </motion.div>

      {/* Preview via Portal — renders at body level to avoid scroll issues */}
      {previewOpen &&
        createPortal(
          <ProductPreview
            data={{
              name,
              price: Number(price) || 0,
              description,
              brand,
              material,
              care: careInstructions,
              sizes: selectedSizes,
              color,
              fit,
              categoryName: selectedCategory?.name || "",
              images,
              sizeChart,
              inStock,
            }}
            onClose={() => setPreviewOpen(false)}
          />,
          document.body,
        )}
    </>
  );
}
