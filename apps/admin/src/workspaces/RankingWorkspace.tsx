import { useState, useEffect, useCallback } from "react";
import { GripVertical, Save, Loader2 } from "lucide-react";
import { api } from "@/lib/api";

type RankingContext = "in_stock" | "preorder";

interface Product {
  id: number;
  name: string;
  brand: string;
  category_name: string;
  status: string;
  images: { url: string }[];
}

interface ListResponse {
  products: Product[];
  total: number;
}

async function saveRanks(products: Product[], ctx: RankingContext) {
  const key = ctx === "preorder" ? "popularity_rank_preorder" : "popularity_rank";
  const ranks = products.map((p, i) => ({ id: p.id, rank: i + 1, key }));
  const res = await fetch("/api/admin/products/rank", {
    method: "PUT",
    credentials: "include",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(ranks),
  });
  if (!res.ok) throw new Error(await res.text());
}

export default function RankingWorkspace() {
  const [tab, setTab] = useState<RankingContext>("in_stock");
  const [products, setProducts] = useState<Product[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [saving, setSaving] = useState(false);
  const [saved, setSaved] = useState(false);

  const load = useCallback(async () => {
    setLoading(true);
    setError("");
    try {
      const statusParam = tab === "preorder" ? "&status=preorder" : "&status=in_stock";
      const data = await api<ListResponse>(
        `/api/admin/products?per_page=200&sort=popular${statusParam}`
      );
      setProducts(data.products || []);
    } catch (e: any) {
      setError(e?.message || "Ошибка загрузки");
    }
    setLoading(false);
  }, [tab]);

  useEffect(() => { load(); }, [load]);

  const save = async () => {
    setSaving(true);
    setSaved(false);
    setError("");
    try {
      await saveRanks(products, tab);
      setSaved(true);
      setTimeout(() => setSaved(false), 3000);
    } catch (e: any) {
      setError(e?.message || "Ошибка сохранения");
    }
    setSaving(false);
  };

  const move = (from: number, to: number) => {
    const next = [...products];
    const [item] = next.splice(from, 1);
    next.splice(to, 0, item);
    setProducts(next);
    setSaved(false);
  };

  const tabLabel = tab === "in_stock" ? "В наличии" : "Под заказ";

  return (
    <div className="p-6 lg:p-8">
      <div className="flex items-center justify-between mb-6">
        <div>
          <h1 className="text-2xl font-bold tracking-tighter text-[var(--color-text-primary)]">
            Сортировка товаров
          </h1>
          <p className="text-sm text-[var(--color-text-muted)] mt-1">
            Перетащите строки чтобы задать порядок для «Самых популярных».
          </p>
        </div>
        <button
          onClick={save}
          disabled={saving}
          className="flex items-center gap-2 rounded-xl bg-[#44944A] px-5 py-2.5 text-sm font-semibold text-black hover:bg-[#3a7d3f] disabled:opacity-50 transition-colors"
        >
          <Save className="h-4 w-4" />
          {saving ? "Сохранение..." : saved ? "✓ Сохранено" : "Сохранить порядок"}
        </button>
      </div>

      {/* Tabs */}
      <div className="flex gap-2 mb-4">
        {(["in_stock", "preorder"] as RankingContext[]).map((t) => (
          <button
            key={t}
            onClick={() => { setTab(t); setSaved(false); }}
            className={`px-4 py-2 rounded-xl text-sm font-medium transition-colors ${
              tab === t
                ? "bg-[#44944A] text-black"
                : "bg-[var(--color-bg-card)] border border-[var(--color-border-custom)] text-[var(--color-text-secondary)] hover:text-[var(--color-text-primary)]"
            }`}
          >
            {t === "in_stock" ? "В наличии" : "Под заказ"}
          </button>
        ))}
      </div>

      {error && (
        <div className="mb-4 p-3 rounded-xl bg-red-500/10 border border-red-500/20 text-sm text-red-500">{error}</div>
      )}
      {saved && (
        <div className="mb-4 p-3 rounded-xl bg-[#44944A]/10 border border-[#44944A]/20 text-sm text-[#44944A] font-medium">
          Порядок для «{tabLabel}» сохранён!
        </div>
      )}

      {loading ? (
        <div className="flex items-center justify-center min-h-[200px]">
          <Loader2 className="h-6 w-6 text-[var(--color-text-muted)] animate-spin" />
        </div>
      ) : (
        <div className="rounded-2xl border border-[var(--color-border-custom)] overflow-hidden">
          <div className="grid grid-cols-[48px_56px_1fr_160px_140px] gap-3 px-4 py-3 bg-[var(--color-bg-secondary)] text-xs font-semibold text-[var(--color-text-muted)] uppercase tracking-wider">
            <span>№</span>
            <span>Фото</span>
            <span>Название</span>
            <span>Категория</span>
            <span>Бренд</span>
          </div>

          {products.map((product, index) => (
            <div
              key={product.id}
              draggable
              onDragStart={(e) => { e.dataTransfer.effectAllowed = "move"; e.dataTransfer.setData("text/plain", String(index)); }}
              onDragOver={(e) => { e.preventDefault(); e.dataTransfer.dropEffect = "move"; }}
              onDrop={(e) => {
                e.preventDefault();
                const from = parseInt(e.dataTransfer.getData("text/plain"), 10);
                if (!isNaN(from) && from !== index) move(from, index);
              }}
              className="grid grid-cols-[48px_56px_1fr_160px_140px] gap-3 px-4 py-3 items-center border-t border-[var(--color-border-custom)] hover:bg-[var(--color-bg-secondary)] transition-colors cursor-grab active:cursor-grabbing"
            >
              <span className="flex items-center justify-center gap-1 text-xs text-[var(--color-text-muted)]">
                <GripVertical className="h-4 w-4" />
                {index + 1}
              </span>
              <div className="w-10 h-10 rounded-lg bg-[var(--color-bg-secondary)] overflow-hidden flex-shrink-0">
                {product.images?.[0]?.url ? (
                  <img src={product.images[0].url} alt="" className="w-full h-full object-cover" />
                ) : (
                  <div className="w-full h-full flex items-center justify-center text-[10px] text-[var(--color-text-muted)]">—</div>
                )}
              </div>
              <span className="text-sm font-medium text-[var(--color-text-primary)] truncate">{product.name}</span>
              <span className="text-xs text-[var(--color-text-muted)] truncate">{product.category_name}</span>
              <span className="text-xs text-[var(--color-text-muted)] truncate">{product.brand}</span>
            </div>
          ))}

          {products.length === 0 && (
            <div className="px-4 py-12 text-center text-[var(--color-text-muted)]">Нет товаров</div>
          )}
        </div>
      )}
    </div>
  );
}
