import { useState, useEffect } from "react";
import { Link } from "react-router-dom";
import { fetchAdminOrders, updateOrderStatus, getImageUrl, type AdminOrder } from "@/lib/api";
import { Package, RefreshCw, X, MapPin, Truck, CreditCard, User, Clock, ShoppingBag, Phone, Copy, Check } from "lucide-react";

const MEASUREMENT_LABELS: Record<string, string> = {
  height: "Рост",
  weight: "Вес",
  foot_length: "Длина стопы",
  head_circumference: "Обхват головы",
  waist: "Талия",
};

const STATUS_OPTIONS = ["pending", "processing", "shipped", "delivered"];
const STATUS_LABELS: Record<string, string> = {
  pending: "Ожидает",
  processing: "В обработке",
  shipped: "Отправлен",
  delivered: "Доставлен",
};
const STATUS_ACCENT: Record<string, string> = {
  pending: "bg-amber-100 text-amber-800 dark:bg-amber-900/30 dark:text-amber-300",
  processing: "bg-blue-100 text-blue-800 dark:bg-blue-900/30 dark:text-blue-300",
  shipped: "bg-purple-100 text-purple-800 dark:bg-purple-900/30 dark:text-purple-300",
  delivered: "bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-300",
};

const DELIVERY_LABELS: Record<string, string> = {
  personal: "Личная встреча",
  address: "Доставка по адресу",
  bus: "Маршрутка",
  express: "Экспресс-почта",
  moldovaPost: "Почта Молдовы",
};

const TYPE_LABELS: Record<string, string> = {
  cart: "Корзина",
  individual: "Индивидуальный",
};

const DELIVERY_TIME_LABELS: Record<string, string> = {
  fast: "Как можно быстрее",
  medium: "В течение недели",
  slow: "В течение месяца",
};

export default function Orders() {
  const [orders, setOrders] = useState<AdminOrder[]>([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [statusFilter, setStatusFilter] = useState("");
  const [typeFilter, setTypeFilter] = useState("");
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [lightbox, setLightbox] = useState<string | null>(null);
  // Tracks the order id whose phone was just copied to the clipboard,
  // used to flip the icon to a green check for ~1.5s after click.
  const [copiedOrderID, setCopiedOrderID] = useState<number | null>(null);
  const perPage = 12;

  const loadOrders = async () => {
    setLoading(true);
    setError("");
    try {
      const res = await fetchAdminOrders(
        page,
        perPage,
        statusFilter || undefined,
        typeFilter || undefined,
      );
      setOrders(res.orders ?? []);
      setTotal(res.total);
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : "Ошибка загрузки");
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadOrders();
  }, [page, statusFilter, typeFilter]);

  const handleStatusChange = async (orderId: number, newStatus: string) => {
    try {
      await updateOrderStatus(orderId, newStatus);
      setOrders((prev) =>
        prev.map((o) => (o.id === orderId ? { ...o, status: newStatus } : o)),
      );
    } catch (e: unknown) {
      const msg = e instanceof Error ? e.message : "Ошибка обновления";
      console.error("Status change failed:", msg, e);
      setError(msg);
    }
  };

  // copyPhoneToClipboard writes the order's phone to the system clipboard
  // and gives a 1.5s visual confirmation by flipping the per-order icon
  // to a green check. We deliberately fall back to a hidden <textarea>
  // + document.execCommand when navigator.clipboard is unavailable
  // (insecure context, old browser, or when the page is rendered inside
  // an iframe without clipboard permission) — managers running the
  // admin panel over plain http://dev-hostname on some browsers hit this
  // path. Better to copy via the legacy method than silently fail.
  const copyPhoneToClipboard = async (orderId: number, phone: string) => {
    let ok = false;
    if (navigator.clipboard && window.isSecureContext) {
      try {
        await navigator.clipboard.writeText(phone);
        ok = true;
      } catch {
        ok = false;
      }
    }
    if (!ok) {
      try {
        const ta = document.createElement("textarea");
        ta.value = phone;
        ta.style.position = "fixed";
        ta.style.left = "-9999px";
        ta.setAttribute("readonly", "");
        document.body.appendChild(ta);
        ta.select();
        ok = document.execCommand("copy");
        document.body.removeChild(ta);
      } catch {
        ok = false;
      }
    }
    if (ok) {
      setCopiedOrderID(orderId);
      window.setTimeout(() => {
        setCopiedOrderID((prev) => (prev === orderId ? null : prev));
      }, 1500);
    } else {
      // Last-resort UX: still set the state so the manager at least
      // gets a visible reaction to the click, and surface the phone in
      // a prompt they can manually Ctrl+C from.
      setCopiedOrderID(orderId);
      window.prompt("Скопируйте номер вручную:", phone);
      window.setTimeout(() => {
        setCopiedOrderID((prev) => (prev === orderId ? null : prev));
      }, 1500);
    }
  };

  const totalPages = Math.max(1, Math.ceil(total / perPage));

  return (
    <div className="p-6 lg:p-8 max-w-7xl mx-auto" data-testid="orders-page">
      {/* Header */}
      <div className="flex items-center justify-between mb-6">
        <h1 className="text-2xl font-bold text-[var(--color-text-primary)]">
          Заказы
        </h1>
        <button
          onClick={loadOrders}
          data-testid="orders-refresh"
          className="flex items-center gap-2 rounded-xl bg-[var(--color-bg-card)] border border-[var(--color-border-custom)] px-4 py-2 text-sm text-[var(--color-text-secondary)] hover:text-[var(--color-text-primary)] transition-colors"
        >
          <RefreshCw className="h-4 w-4" />
          Обновить
        </button>
      </div>

      {error && (
        <p data-testid="orders-error" className="text-sm text-red-400 mb-4 p-3 rounded-xl bg-red-500/10 border border-red-500/20">
          {error}
        </p>
      )}

      {/* Filters */}
      <div className="flex items-center gap-3 mb-6 flex-wrap">
        <select
          value={statusFilter}
          onChange={(e) => {
            setStatusFilter(e.target.value);
            setPage(1);
          }}
          data-testid="orders-filter-status"
          className="rounded-xl bg-[var(--color-bg-card)] border border-[var(--color-border-custom)] px-4 py-2 text-sm text-[var(--color-text-primary)]"
        >
          <option value="">Все статусы</option>
          {STATUS_OPTIONS.map((s) => (
            <option key={s} value={s}>
              {STATUS_LABELS[s]}
            </option>
          ))}
        </select>

        <select
          value={typeFilter}
          onChange={(e) => {
            setTypeFilter(e.target.value);
            setPage(1);
          }}
          data-testid="orders-filter-type"
          className="rounded-xl bg-[var(--color-bg-card)] border border-[var(--color-border-custom)] px-4 py-2 text-sm text-[var(--color-text-primary)]"
        >
          <option value="">Все типы</option>
          <option value="cart">Корзина</option>
          <option value="individual">Индивидуальный</option>
        </select>

        <span data-testid="orders-total" className="text-sm text-[var(--color-text-muted)] ml-auto">
          {total} заказов
        </span>
      </div>

      {/* Content */}
      {loading ? (
        <div data-testid="orders-loading" className="flex items-center justify-center py-20">
          <RefreshCw className="h-6 w-6 text-[var(--color-text-muted)] animate-spin" />
        </div>
      ) : orders.length === 0 ? (
        <div data-testid="orders-empty" className="flex flex-col items-center justify-center py-20 text-center">
          <Package className="h-12 w-12 text-[var(--color-text-muted)] mb-4" />
          <p className="text-[var(--color-text-muted)]">
            {statusFilter || typeFilter
              ? "Нет заказов с такими фильтрами"
              : "Заказов пока нет"}
          </p>
        </div>
      ) : (
        <>
          <div data-testid="orders-list" className="grid gap-4">
            {orders.map((o) => (
              <OrderCard
                key={o.id}
                order={o}
                onStatusChange={handleStatusChange}
                onPhotoClick={setLightbox}
                onPhoneCopy={copyPhoneToClipboard}
                copiedPhoneID={copiedOrderID}
              />
            ))}
          </div>

          {/* Pagination */}
          {totalPages > 1 && (
            <div data-testid="orders-pagination" className="flex items-center justify-center gap-2 mt-8">
              <button
                onClick={() => setPage((p) => Math.max(1, p - 1))}
                disabled={page <= 1}
                data-testid="orders-pagination-prev"
                className="px-3 py-2 rounded-xl bg-[var(--color-bg-card)] border border-[var(--color-border-custom)] text-sm text-[var(--color-text-secondary)] disabled:opacity-30"
              >
                ←
              </button>
              <span data-testid="orders-pagination-info" className="text-sm text-[var(--color-text-muted)]">
                {page} / {totalPages}
              </span>
              <button
                onClick={() => setPage((p) => Math.min(totalPages, p + 1))}
                disabled={page >= totalPages}
                data-testid="orders-pagination-next"
                className="px-3 py-2 rounded-xl bg-[var(--color-bg-card)] border border-[var(--color-border-custom)] text-sm text-[var(--color-text-secondary)] disabled:opacity-30"
              >
                →
              </button>
            </div>
          )}
        </>
      )}

      {/* Lightbox */}
      {lightbox && (
        <div
          className="fixed inset-0 z-50 flex items-center justify-center bg-black/80 p-4"
          onClick={() => setLightbox(null)}
        >
          <button
            className="absolute top-4 right-4 text-white/60 hover:text-white transition-colors"
            onClick={() => setLightbox(null)}
          >
            <X className="h-8 w-8" />
          </button>
          <img
            src={lightbox}
            alt="Фото"
            className="max-w-full max-h-[90vh] object-contain rounded-lg"
            onClick={(e) => e.stopPropagation()}
          />
        </div>
      )}
    </div>
  );
}

// ── Order Card ──

function OrderCard({
  order: o,
  onStatusChange,
  onPhotoClick,
  onPhoneCopy,
  copiedPhoneID,
}: {
  order: AdminOrder;
  onStatusChange: (id: number, status: string) => void;
  onPhotoClick: (url: string) => void;
  onPhoneCopy: (id: number, phone: string) => void;
  copiedPhoneID: number | null;
}) {
  const [expanded, setExpanded] = useState(false);
  const totalLei = (o.total_minor / 100).toFixed(2);

  return (
    <div data-testid={`orders-row-${o.id}`} className="rounded-2xl bg-[var(--color-bg-card)] border border-[var(--color-border-custom)] overflow-hidden hover:border-[var(--color-text-muted)] transition-colors">
      {/* Card header */}
      <div className="p-4 sm:p-5 flex items-start justify-between gap-3 cursor-pointer select-none"
           onClick={() => setExpanded(!expanded)}>
        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-2 flex-wrap mb-1.5">
            <span className="text-lg font-bold text-[var(--color-text-primary)] font-mono">
              {o.order_code || `#${o.id}`}
            </span>
            <span className={`text-xs font-medium px-2 py-0.5 rounded-full ${
              o.type === "individual"
                ? "bg-pink-100 text-pink-700 dark:bg-pink-900/30 dark:text-pink-300"
                : "bg-gray-100 text-gray-600 dark:bg-gray-800 dark:text-gray-400"
            }`}>
              {TYPE_LABELS[o.type] || o.type}
            </span>
            <select
              value={o.status}
              onClick={(e) => e.stopPropagation()}
              onChange={(e) => onStatusChange(o.id, e.target.value)}
              data-testid={`orders-row-${o.id}-status-select`}
              className={`text-xs font-medium px-2 py-0.5 rounded-full border-0 cursor-pointer ${STATUS_ACCENT[o.status] || ""}`}
            >
              {STATUS_OPTIONS.map((s) => (
                <option key={s} value={s}>
                  {STATUS_LABELS[s]}
                </option>
              ))}
            </select>
          </div>

          {/* Customer — the chip is now a Link to the
              customer detail workspace so the manager can
              jump from an order to the customer's full
              history with one click. We `e.stopPropagation()`
              so clicking the customer name doesn't also
              trigger the parent card's `setExpanded` toggle
              (which is the one behavioural surprise the
              first cut would have hit). */}
          <Link
            to={`/customers/${o.customer_id}`}
            onClick={(e) => e.stopPropagation()}
            data-testid={`order-customer-link-${o.id}`}
            className="inline-flex items-center gap-1.5 text-sm text-[var(--color-text-secondary)] hover:text-[var(--color-text-primary)] transition-colors rounded-md hover:underline underline-offset-4"
          >
            <User className="h-3.5 w-3.5 shrink-0" />
            <span>
              {o.customer_first_name || o.customer_email || `#${o.customer_id}`}
            </span>
            {o.customer_first_name && o.customer_email && (
              <span className="text-[var(--color-text-muted)]">
                ({o.customer_email})
              </span>
            )}
          </Link>

          {/* Phone — manager's primary action: click to copy for outbound
              call. Always rendered so the layout doesn't shift between
              orders. The button is keyboard-accessible (Enter/Space) and
              carries aria-live so screen readers announce the copy
              confirmation. */}
          <div className="flex items-center gap-2 mt-1">
            <button
              type="button"
              onClick={() => onPhoneCopy(o.id, o.phone)}
              data-testid={`order-phone-${o.id}`}
              aria-label={`Скопировать телефон ${o.phone} для связи с клиентом`}
              title="Кликните, чтобы скопировать"
              className="group inline-flex items-center gap-2 rounded-lg border border-[#44944A]/30 bg-[#44944A]/10 px-3 py-1.5 text-sm font-mono font-semibold text-[#44944A] hover:bg-[#44944A]/20 hover:border-[#44944A]/60 transition-colors focus:outline-none focus:ring-2 focus:ring-[#44944A]/40"
            >
              {copiedPhoneID === o.id ? (
                <Check className="h-3.5 w-3.5 shrink-0" aria-hidden="true" />
              ) : (
                <Phone className="h-3.5 w-3.5 shrink-0" aria-hidden="true" />
              )}
              <span className="select-all">{o.phone || "—"}</span>
              {copiedPhoneID === o.id ? (
                <span
                  className="text-[10px] uppercase tracking-wider font-sans font-bold text-[#44944A] opacity-80"
                  aria-live="polite"
                >
                  скопировано
                </span>
              ) : (
                <Copy
                  className="h-3 w-3 shrink-0 opacity-40 group-hover:opacity-80 transition-opacity"
                  aria-hidden="true"
                />
              )}
            </button>
          </div>

          {/* Quick info line */}
          <div className="flex items-center gap-3 mt-2 text-xs text-[var(--color-text-muted)] flex-wrap">
            <span className="flex items-center gap-1">
              <MapPin className="h-3 w-3" />
              {o.city}
            </span>
            <span className="flex items-center gap-1">
              <Truck className="h-3 w-3" />
              {DELIVERY_LABELS[o.delivery_method] || o.delivery_method}
            </span>
            <span className="flex items-center gap-1">
              <CreditCard className="h-3 w-3" />
              {o.payment_method === "card" ? "Карта" : "При получении"}
            </span>
            <span className="flex items-center gap-1">
              <ShoppingBag className="h-3 w-3" />
              {o.items?.length || 0} поз.
            </span>
          </div>
        </div>

        <div className="text-right shrink-0">
          <span className="text-lg font-bold text-[var(--color-text-primary)]">
            {totalLei} лей
          </span>
          <div className="text-xs text-[var(--color-text-muted)] mt-0.5">
            {new Date(o.created_at).toLocaleString("ru-RU", {
              day: "2-digit",
              month: "2-digit",
              hour: "2-digit",
              minute: "2-digit",
            })}
          </div>
        </div>
      </div>

      {/* Expanded details */}
      {expanded && (
        <div className="px-4 sm:px-5 pb-5 border-t border-[var(--color-border-custom)] pt-4 space-y-4">
          {/* Items */}
          {o.items && o.items.length > 0 && (
            <div>
              <h4 className="text-xs font-semibold text-[var(--color-text-muted)] uppercase tracking-wider mb-2">
                Товары
              </h4>
              <div className="space-y-1.5">
                {o.items.map((item) => (
                  <div
                    key={item.id}
                    className="flex items-center justify-between text-sm py-1.5 px-3 rounded-lg bg-[var(--color-bg-secondary)]"
                  >
                    <div className="flex-1 min-w-0">
                      <span className="text-[var(--color-text-primary)]">
                        {item.product_name || `#${item.product_id}`}
                      </span>
                      {item.size_label && (
                        <span className="text-[var(--color-text-muted)] ml-1.5 text-xs">
                          {item.size_label}
                        </span>
                      )}
                      {item.measurements && Object.keys(item.measurements).length > 0 && (
                        <span className="text-[var(--color-text-muted)] ml-1.5 text-xs">
                          {Object.entries(item.measurements).map(([k, v]) => `${MEASUREMENT_LABELS[k] || k}: ${v}`).join(" · ")}
                        </span>
                      )}
                    </div>
                    <div className="flex items-center gap-3 shrink-0 ml-3">
                      <span className="text-[var(--color-text-muted)] text-xs">
                        ×{item.quantity}
                      </span>
                      <span className="text-[var(--color-text-primary)] font-medium tabular-nums">
                        {(item.price_minor / 100).toFixed(2)} лей
                      </span>
                    </div>
                  </div>
                ))}
              </div>
            </div>
          )}

          {/* Individual order details */}
          {o.type === "individual" && (
            <div className="grid gap-3 sm:grid-cols-2">
              {(o.height || o.weight) && (
                <div className="p-3 rounded-xl bg-[var(--color-bg-secondary)]">
                  <span className="text-xs text-[var(--color-text-muted)]">Параметры</span>
                  <p className="text-sm text-[var(--color-text-primary)] mt-0.5">
                    {o.height ? `Рост ${o.height} см` : ""}
                    {o.height && o.weight ? ", " : ""}
                    {o.weight ? `Вес ${o.weight} кг` : ""}
                  </p>
                </div>
              )}
              {o.delivery_time && o.delivery_time.length > 0 && (
                <div className="p-3 rounded-xl bg-[var(--color-bg-secondary)]">
                  <span className="text-xs text-[var(--color-text-muted)]">Сроки</span>
                  <p className="text-sm text-[var(--color-text-primary)] mt-0.5">
                    {o.delivery_time.map((t) => DELIVERY_TIME_LABELS[t] || t).join(", ")}
                  </p>
                </div>
              )}
              {o.comment && (
                <div className="sm:col-span-2 p-3 rounded-xl bg-[var(--color-bg-secondary)]">
                  <span className="text-xs text-[var(--color-text-muted)]">Комментарий</span>
                  <p className="text-sm text-[var(--color-text-primary)] mt-0.5 whitespace-pre-wrap">
                    {o.comment}
                  </p>
                </div>
              )}
            </div>
          )}

          {/* Address for cart orders */}
          {o.type === "cart" && (o.street || o.house || o.apartment) && (
            <div className="p-3 rounded-xl bg-[var(--color-bg-secondary)]">
              <span className="text-xs text-[var(--color-text-muted)]">Адрес доставки</span>
              <p className="text-sm text-[var(--color-text-primary)] mt-0.5">
                {[o.street, o.house, o.apartment ? `кв. ${o.apartment}` : null]
                  .filter(Boolean)
                  .join(", ")}
              </p>
            </div>
          )}

          {/* Photos */}
          {o.photos && o.photos.length > 0 && (
            <div>
              <h4 className="text-xs font-semibold text-[var(--color-text-muted)] uppercase tracking-wider mb-2">
                Фото ({o.photos.length})
              </h4>
              <div className="flex gap-2 flex-wrap">
                {o.photos.map((url, i) => (
                  <button
                    key={i}
                    onClick={() => onPhotoClick(getImageUrl(url))}
                    className="w-20 h-20 rounded-lg overflow-hidden border border-[var(--color-border-custom)] hover:opacity-80 transition-opacity bg-[var(--color-bg-secondary)]"
                  >
                    <img
                      src={getImageUrl(url).replace("/uploads/", "/uploads/thumb_")}
                      alt={`Фото ${i + 1}`}
                      className="w-full h-full object-cover"
                      onError={(e) => {
                        (e.target as HTMLImageElement).src = getImageUrl(url);
                      }}
                    />
                  </button>
                ))}
              </div>
            </div>
          )}

          {/* Delivery info for cart orders with address */}
          {o.type === "cart" && o.comment && (
            <div className="p-3 rounded-xl bg-[var(--color-bg-secondary)]">
              <span className="text-xs text-[var(--color-text-muted)]">Комментарий</span>
              <p className="text-sm text-[var(--color-text-primary)] mt-0.5 whitespace-pre-wrap">
                {o.comment}
              </p>
            </div>
          )}
        </div>
      )}
    </div>
  );
}
