import { Link, useSearchParams, useNavigate } from "react-router-dom";
import { useState, useEffect } from "react";
import { fetchStoreCustomerOrders, type StoreOrder } from "@/lib/api";
import {
  User, Settings, ChevronRight, LogOut,
  MapPin, Truck, CreditCard, ShoppingBag,
} from "lucide-react";
import { motion } from "framer-motion";
import { useTranslation } from "react-i18next";
import { Helmet } from "react-helmet-async";
import { useAuthStore } from "@/stores/authStore";
import AuthSection from "@/components/auth/AuthSection";

export default function ProfilePage() {
  const { t } = useTranslation();
  const { user, isAuthenticated, logout } = useAuthStore();
  const [orders, setOrders] = useState<StoreOrder[]>([]);
  const [searchParams] = useSearchParams();
  const navigate = useNavigate();
  const redirect = searchParams.get("redirect");

  useEffect(() => {
    if (isAuthenticated) {
      fetchStoreCustomerOrders()
        .then((res) => setOrders(res.orders))
        .catch(() => {});
    }
  }, [isAuthenticated]);

  useEffect(() => {
    if (isAuthenticated && redirect) {
      navigate(redirect, { replace: true });
    }
  }, [isAuthenticated, redirect, navigate]);

  if (!isAuthenticated || !user) {
    return (
      <div className="px-6 py-24 lg:px-8">
        <Helmet>
          <title>{t("profile.title")} — MIORU</title>
        </Helmet>
        <div className="mx-auto max-w-md">
          <div className="text-center mb-8">
            <div className="flex justify-center mb-4">
              <div className="flex h-16 w-16 items-center justify-center rounded-full bg-[#44944A]/10">
                <User className="h-8 w-8 text-[#44944A]" />
              </div>
            </div>
            <h2 className="text-xl font-semibold text-[var(--color-text-primary)]">
              {t("profile.title")}
            </h2>
          </div>
          <div className="rounded-2xl bg-[var(--color-bg-card)] border border-[var(--color-border-custom)] p-6">
            <AuthSection />
          </div>
        </div>
      </div>
    );
  }

  return (
    <div className="px-6 py-24 lg:px-8">
      <Helmet>
        <title>{t("profile.title")} — MIORU</title>
        <meta
          name="description"
          content="Ваш личный кабинет MIORU. Управляйте аватаром, отслеживайте заказы и уровень XP."
        />
        <meta property="og:title" content={t("profile.title") + " — MIORU"} />
        <link rel="canonical" href="https://mioru.store/profile" />
      </Helmet>
      <div className="mx-auto max-w-4xl">
        <div className="flex items-center justify-between">
          <motion.h1
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            className="text-4xl font-bold tracking-tighter text-[var(--color-text-primary)] sm:text-5xl"
          >
            {t("profile.title")}
          </motion.h1>
          <button
            onClick={logout}
            className="flex items-center gap-1 text-sm text-[var(--color-text-muted)] hover:text-[var(--color-text-primary)] transition-colors"
          >
            <LogOut className="h-4 w-4" />
            Выйти
          </button>
        </div>

        {/* Profile Card */}
        <motion.div
          initial={{ opacity: 0, y: 20 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ delay: 0.1 }}
          className="mt-10 rounded-2xl bg-[var(--color-bg-card)] border border-[var(--color-border-custom)] p-6"
        >
          <div className="flex items-center gap-4">
            <div className="flex h-16 w-16 items-center justify-center rounded-full bg-[#44944A]/10">
              <User className="h-8 w-8 text-[#44944A]" />
            </div>
            <div>
              <h2 className="text-lg font-semibold text-[var(--color-text-primary)]">
                {user.name || user.email}
              </h2>
              {user.email && (
                <p className="text-sm text-[var(--color-text-secondary)]">
                  {user.email}
                </p>
              )}
            </div>
          </div>
        </motion.div>

        {/* Quick Links */}
        <div className="mt-6">
          <Link
            to="/profile/edit"
            className="flex items-center justify-between rounded-2xl bg-[var(--color-bg-card)] border border-[var(--color-border-custom)] p-5 transition-all hover:border-[#44944A]/50"
          >
            <div className="flex items-center gap-3">
              <Settings className="h-5 w-5 text-[#44944A]" />
              <span className="text-sm font-medium text-[var(--color-text-primary)]">
                {t("profile.edit")}
              </span>
            </div>
            <ChevronRight className="h-4 w-4 text-[var(--color-text-muted)]" />
          </Link>
        </div>

        {/* Orders */}
        <div id="orders" className="mt-12">
          <h3 className="text-lg font-semibold text-[var(--color-text-primary)] mb-6">
            {t("profile.orderHistory")}
          </h3>
          <div className="space-y-4">
            {orders.length === 0 ? (
              <p className="text-sm text-[var(--color-text-muted)] text-center py-4">
                {t("profile.noOrders")}
              </p>
            ) : (
              orders.map((order) => (
                <OrderCard key={order.id} order={order} />
              ))
            )}
          </div>
        </div>
      </div>
    </div>
  );
}

// ── Order Card ──

const STATUS_ACCENT: Record<string, string> = {
  pending: "text-amber-400 bg-amber-400/10",
  processing: "text-blue-400 bg-blue-400/10",
  shipped: "text-purple-400 bg-purple-400/10",
  delivered: "text-green-400 bg-green-400/10",
};

function OrderCard({ order: o }: { order: StoreOrder }) {
  const { t } = useTranslation();
  const [expanded, setExpanded] = useState(false);
  const totalRub = (o.total_minor / 100).toFixed(2);

  const statusLabel = t(`profile.orderStatus.${o.status}`, o.status);
  const typeLabel = t(`profile.orderType.${o.type}`, o.type);
  const deliveryLabel = t(`profile.delivery.${o.delivery_method}`, o.delivery_method);

  return (
    <motion.div
      initial={{ opacity: 0, y: 20 }}
      animate={{ opacity: 1, y: 0 }}
      className="rounded-2xl bg-[var(--color-bg-card)] border border-[var(--color-border-custom)] overflow-hidden hover:border-[var(--color-text-muted)] transition-colors"
    >
      {/* Card header — click to expand */}
      <div
        className="p-4 sm:p-5 flex items-start justify-between gap-3 cursor-pointer select-none"
        onClick={() => setExpanded(!expanded)}
      >
        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-2 flex-wrap mb-1.5">
            <span className="text-lg font-bold text-[var(--color-text-primary)] font-mono">
              #{o.id}
            </span>
            <span
              className={`text-xs font-medium px-2 py-0.5 rounded-full ${
                o.type === "individual"
                  ? "text-pink-400 bg-pink-400/10"
                  : "text-gray-400 bg-gray-400/10"
              }`}
            >
              {typeLabel}
            </span>
            <span
              className={`text-xs font-medium px-2 py-0.5 rounded-full ${
                STATUS_ACCENT[o.status] || ""
              }`}
            >
              {statusLabel}
            </span>
          </div>

          {/* Quick info */}
          <div className="flex items-center gap-3 mt-2 text-xs text-[var(--color-text-muted)] flex-wrap">
            <span className="flex items-center gap-1">
              <MapPin className="h-3 w-3" />
              {o.city}
            </span>
            <span className="flex items-center gap-1">
              <Truck className="h-3 w-3" />
              {deliveryLabel}
            </span>
            <span className="flex items-center gap-1">
              <CreditCard className="h-3 w-3" />
              {o.payment_method === "card" ? t("checkout.payment.card") : t("checkout.payment.cod")}
            </span>
            <span className="flex items-center gap-1">
              <ShoppingBag className="h-3 w-3" />
              {o.items?.length || 0} {t("profile.itemsCount")}
            </span>
          </div>
        </div>

        <div className="text-right shrink-0">
          <span className="text-lg font-bold text-[var(--color-text-primary)]">
            {totalRub} ₽
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
                    </div>
                    <div className="flex items-center gap-3 shrink-0 ml-3">
                      <span className="text-[var(--color-text-muted)] text-xs">
                        ×{item.quantity}
                      </span>
                      <span className="text-[var(--color-text-primary)] font-medium tabular-nums">
                        {(item.price_minor / 100).toFixed(2)} ₽
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
                    {o.delivery_time.map((timeKey) =>
                      t(`profile.deliveryTime.${timeKey}`, timeKey)
                    ).join(", ")}
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
          {o.type === "cart" && (o.street || o.house) && (
            <div className="p-3 rounded-xl bg-[var(--color-bg-secondary)]">
              <span className="text-xs text-[var(--color-text-muted)]">Адрес доставки</span>
              <p className="text-sm text-[var(--color-text-primary)] mt-0.5">
                {[o.street, o.house, o.apartment ? `кв. ${o.apartment}` : null]
                  .filter(Boolean)
                  .join(", ")}
              </p>
            </div>
          )}

          {/* Comment for cart orders */}
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
    </motion.div>
  );
}
