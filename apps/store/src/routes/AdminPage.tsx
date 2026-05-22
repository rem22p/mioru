import { useState } from "react";
import { motion } from "framer-motion";
import { products, mockOrders, categories } from "@/lib/data";
import { Plus, Pencil, Trash2, Package, ShoppingBag } from "lucide-react";
import { useTranslation } from "react-i18next";
import { Helmet } from "react-helmet-async";

export default function AdminPage() {
  const { t } = useTranslation();
  const [activeTab, setActiveTab] = useState<"products" | "orders">("products");
  const [showForm, setShowForm] = useState(false);
  const [newProduct, setNewProduct] = useState({
    name: "",
    price: "",
    category: "",
    description: "",
    sizes: [] as string[],
  });

  const toggleSize = (size: string) => {
    setNewProduct((prev) => ({
      ...prev,
      sizes: prev.sizes.includes(size)
        ? prev.sizes.filter((s) => s !== size)
        : [...prev.sizes, size],
    }));
  };

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    alert("Товар добавлен! (Mock)");
    setShowForm(false);
    setNewProduct({
      name: "",
      price: "",
      category: "",
      description: "",
      sizes: [],
    });
  };

  const inputBaseClass =
    "w-full rounded-xl bg-[var(--color-bg-primary)] border border-[var(--color-border-custom)] px-4 py-2 text-base sm:text-sm text-[var(--color-text-primary)] outline-none focus:border-[#44944A]";

  return (
    <div className="px-6 py-24 lg:px-8">
      <Helmet>
        <title>Админ панель — MIORU</title>
        <meta name="robots" content="noindex, nofollow" />
      </Helmet>
      <div className="mx-auto max-w-6xl">
        <motion.h1
          initial={{ opacity: 0, y: 20 }}
          animate={{ opacity: 1, y: 0 }}
          className="text-4xl font-bold tracking-tighter text-[var(--color-text-primary)] sm:text-5xl"
        >
          {t("admin.title")}
        </motion.h1>

        <div className="mt-10 flex gap-6 border-b border-[var(--color-border-custom)]">
          <button
            onClick={() => setActiveTab("products")}
            className={`flex items-center gap-2 pb-4 text-sm font-medium transition-colors min-h-[44px] ${
              activeTab === "products"
                ? "text-[#44944A] border-b-2 border-[#44944A]"
                : "text-[var(--color-text-secondary)] hover:text-white"
            }`}
          >
            <Package className="h-4 w-4" />
            {t("admin.tabs.products")}
          </button>
          <button
            onClick={() => setActiveTab("orders")}
            className={`flex items-center gap-2 pb-4 text-sm font-medium transition-colors min-h-[44px] ${
              activeTab === "orders"
                ? "text-[#44944A] border-b-2 border-[#44944A]"
                : "text-[var(--color-text-secondary)] hover:text-white"
            }`}
          >
            <ShoppingBag className="h-4 w-4" />
            {t("admin.tabs.orders")}
          </button>
        </div>

        {activeTab === "products" && (
          <div className="mt-8">
            <button
              onClick={() => setShowForm(!showForm)}
              className="flex items-center gap-2 rounded-xl bg-[#44944A] px-4 py-2 text-sm font-semibold text-black transition-all hover:shadow-[0_0_30px_rgba(192,254,57,0.3)] mb-8"
            >
              <Plus className="h-4 w-4" />
              {t("admin.addProduct")}
            </button>

            {showForm && (
              <motion.form
                initial={{ opacity: 0, height: 0 }}
                animate={{ opacity: 1, height: "auto" }}
                className="mb-8 rounded-2xl bg-[var(--color-bg-card)] border border-[var(--color-border-custom)] p-6 space-y-4"
                onSubmit={handleSubmit}
              >
                <div className="grid gap-4 sm:grid-cols-2">
                  <div>
                    <label className="block text-sm font-medium text-[var(--color-text-primary)] mb-1">
                      {t("admin.productForm.name")}
                    </label>
                    <input
                      type="text"
                      value={newProduct.name}
                      onChange={(e) =>
                        setNewProduct({ ...newProduct, name: e.target.value })
                      }
                      className={inputBaseClass}
                      required
                    />
                  </div>
                  <div>
                    <label className="block text-sm font-medium text-[var(--color-text-primary)] mb-1">
                      {t("admin.productForm.price")}
                    </label>
                    <input
                      type="number"
                      value={newProduct.price}
                      onChange={(e) =>
                        setNewProduct({ ...newProduct, price: e.target.value })
                      }
                      className={inputBaseClass}
                      required
                    />
                  </div>
                </div>
                <div>
                  <label className="block text-sm font-medium text-[var(--color-text-primary)] mb-1">
                    {t("admin.productForm.category")}
                  </label>
                  <select
                    value={newProduct.category}
                    onChange={(e) =>
                      setNewProduct({ ...newProduct, category: e.target.value })
                    }
                    className={inputBaseClass}
                    required
                  >
                    <option value="">{t("admin.productForm.category")}</option>
                    {categories.map((cat) => (
                      <option key={cat.id} value={cat.slug}>
                        {cat.name}
                      </option>
                    ))}
                  </select>
                </div>
                <div>
                  <label className="block text-sm font-medium text-[var(--color-text-primary)] mb-1">
                    {t("admin.productForm.description")}
                  </label>
                  <textarea
                    value={newProduct.description}
                    onChange={(e) =>
                      setNewProduct({
                        ...newProduct,
                        description: e.target.value,
                      })
                    }
                    rows={3}
                    className={`${inputBaseClass} resize-none`}
                  />
                </div>
                <div>
                  <label className="block text-sm font-medium text-[var(--color-text-primary)] mb-2">
                    {t("admin.productForm.sizes")}
                  </label>
                  <div className="flex flex-wrap gap-2">
                    {[
                      "XS",
                      "S",
                      "M",
                      "L",
                      "XL",
                      "XXL",
                      "40",
                      "41",
                      "42",
                      "43",
                      "44",
                      "45",
                    ].map((size) => (
                      <button
                        key={size}
                        type="button"
                        onClick={() => toggleSize(size)}
                        className={`rounded-lg border min-w-[44px] min-h-[44px] px-3 py-1 text-sm transition-all flex items-center justify-center ${
                          newProduct.sizes.includes(size)
                            ? "border-[#44944A] bg-[#44944A] text-black"
                            : "border-[var(--color-border-custom)] text-[var(--color-text-secondary)] hover:text-white"
                        }`}
                      >
                        {size}
                      </button>
                    ))}
                  </div>
                </div>
                <div>
                  <label className="block text-sm font-medium text-[var(--color-text-primary)] mb-1">
                    {t("admin.productForm.photo")}
                  </label>
                  <div className="border-2 border-dashed border-[var(--color-border-custom)] rounded-xl p-8 text-center hover:border-[#44944A]/50 transition-colors cursor-pointer">
                    <p className="text-sm text-[var(--color-text-secondary)]">
                      {t("admin.productForm.photoHint")}
                    </p>
                    <p className="text-xs text-[var(--color-text-muted)] mt-1 font-mono">
                      {t("admin.productForm.photoRecommendation")}
                    </p>
                  </div>
                </div>
                <div className="flex gap-4">
                  <button
                    type="submit"
                    className="rounded-xl bg-[#44944A] px-6 py-2 text-sm font-semibold text-black transition-all hover:shadow-[0_0_30px_rgba(192,254,57,0.3)]"
                  >
                    {t("admin.productForm.save")}
                  </button>
                  <button
                    type="button"
                    onClick={() => setShowForm(false)}
                    className="rounded-xl border border-[var(--color-border-custom)] px-6 py-2 text-sm text-[var(--color-text-secondary)] transition-all hover:bg-[var(--color-bg-card)] hover:text-white"
                  >
                    {t("admin.productForm.cancel")}
                  </button>
                </div>
              </motion.form>
            )}

            <div className="rounded-2xl bg-[var(--color-bg-card)] border border-[var(--color-border-custom)] overflow-hidden">
              <div className="overflow-x-auto">
                <table className="w-full">
                  <thead>
                    <tr className="border-b border-[var(--color-border-custom)]">
                      <th className="px-4 py-3 text-left text-xs font-mono uppercase tracking-wider text-[var(--color-text-muted)]">
                        {t("admin.table.product")}
                      </th>
                      <th className="px-4 py-3 text-left text-xs font-mono uppercase tracking-wider text-[var(--color-text-muted)]">
                        {t("admin.table.category")}
                      </th>
                      <th className="px-4 py-3 text-left text-xs font-mono uppercase tracking-wider text-[var(--color-text-muted)]">
                        {t("admin.table.price")}
                      </th>
                      <th className="px-4 py-3 text-right text-xs font-mono uppercase tracking-wider text-[var(--color-text-muted)]">
                        {t("admin.table.actions")}
                      </th>
                    </tr>
                  </thead>
                  <tbody>
                    {products.map((product) => (
                      <tr
                        key={product.id}
                        className="border-b border-[var(--color-border-custom)] last:border-0 hover:bg-[var(--color-bg-primary)]/50 transition-colors"
                      >
                        <td className="px-4 py-3">
                          <div className="flex items-center gap-3">
                            <div className="h-10 w-10 rounded-lg bg-[var(--color-bg-primary)] border border-[var(--color-border-custom)] flex items-center justify-center text-lg">
                              {product.category.slug === "sneakers" && "👟"}
                              {product.category.slug === "slides" && "🩴"}
                              {product.category.slug === "tshirts" && "👕"}
                              {product.category.slug === "shorts" && "🩳"}
                              {product.category.slug === "bracelets" && "⛓️"}
                            </div>
                            <span className="text-sm text-[var(--color-text-primary)]">
                              {product.name}
                            </span>
                          </div>
                        </td>
                        <td className="px-4 py-3">
                          <span className="text-sm text-[var(--color-text-secondary)]">
                            {product.category.name}
                          </span>
                        </td>
                        <td className="px-4 py-3">
                          <span className="text-sm font-medium text-[var(--color-text-primary)]">
                            {product.price.toLocaleString("ru-RU")} ₽
                          </span>
                        </td>
                        <td className="px-4 py-3">
                          <div className="flex items-center justify-end gap-2">
                            <button className="rounded-lg min-w-[44px] min-h-[44px] flex items-center justify-center text-[var(--color-text-secondary)] hover:text-[#44944A] transition-colors">
                              <Pencil className="h-4 w-4" />
                            </button>
                            <button className="rounded-lg min-w-[44px] min-h-[44px] flex items-center justify-center text-[var(--color-text-secondary)] hover:text-red-500 transition-colors">
                              <Trash2 className="h-4 w-4" />
                            </button>
                          </div>
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            </div>
          </div>
        )}

        {activeTab === "orders" && (
          <div className="mt-8">
            <div className="rounded-2xl bg-[var(--color-bg-card)] border border-[var(--color-border-custom)] overflow-hidden">
              <div className="overflow-x-auto">
                <table className="w-full">
                  <thead>
                    <tr className="border-b border-[var(--color-border-custom)]">
                      <th className="px-4 py-3 text-left text-xs font-mono uppercase tracking-wider text-[var(--color-text-muted)]">
                        {t("admin.table.order")}
                      </th>
                      <th className="px-4 py-3 text-left text-xs font-mono uppercase tracking-wider text-[var(--color-text-muted)]">
                        {t("admin.table.date")}
                      </th>
                      <th className="px-4 py-3 text-left text-xs font-mono uppercase tracking-wider text-[var(--color-text-muted)]">
                        {t("admin.table.amount")}
                      </th>
                      <th className="px-4 py-3 text-left text-xs font-mono uppercase tracking-wider text-[var(--color-text-muted)]">
                        {t("admin.table.status")}
                      </th>
                    </tr>
                  </thead>
                  <tbody>
                    {mockOrders.map((order) => (
                      <tr
                        key={order.id}
                        className="border-b border-[var(--color-border-custom)] last:border-0 hover:bg-[var(--color-bg-primary)]/50 transition-colors"
                      >
                        <td className="px-4 py-3 text-sm text-[var(--color-text-primary)]">
                          {order.id}
                        </td>
                        <td className="px-4 py-3 text-sm text-[var(--color-text-secondary)]">
                          {new Date(order.createdAt).toLocaleDateString(
                            "ru-RU",
                          )}
                        </td>
                        <td className="px-4 py-3 text-sm font-medium text-[var(--color-text-primary)]">
                          {order.total.toLocaleString("ru-RU")} ₽
                        </td>
                        <td className="px-4 py-3">
                          <span
                            className={`inline-flex rounded-full px-2 py-1 text-xs font-medium ${
                              order.status === "delivered"
                                ? "bg-[#44944A]/10 text-[#44944A]"
                                : order.status === "shipped"
                                  ? "bg-blue-500/10 text-blue-500"
                                  : "bg-yellow-500/10 text-yellow-500"
                            }`}
                          >
                            {order.status}
                          </span>
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            </div>
          </div>
        )}
      </div>
    </div>
  );
}
