import { Link, useSearchParams, useNavigate } from "react-router-dom";
import { useState, useEffect } from "react";
import { fetchStoreCustomerOrders, type StoreOrder } from "@/lib/api";
import { VIP_LEVELS } from "@/lib/constants";
import { User, Settings, Package, ChevronRight, LogOut } from "lucide-react";
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

  const currentVip = VIP_LEVELS.reduce((prev, curr) =>
    user.xpBalance >= curr.minXp ? curr : prev,
  );
  const nextVip = VIP_LEVELS.find((v) => v.level > currentVip.level);
  const progress = nextVip
    ? ((user.xpBalance - currentVip.minXp) /
        (nextVip.minXp - currentVip.minXp)) *
      100
    : 100;

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

        {/* XP Progress */}
        <motion.div
          initial={{ opacity: 0, y: 20 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ delay: 0.2 }}
          className="mt-6 rounded-2xl bg-[var(--color-bg-card)] border border-[var(--color-border-custom)] p-6"
        >
          <div className="flex items-center justify-between mb-4">
            <div>
              <h3 className="text-sm font-semibold uppercase tracking-wider text-[var(--color-text-primary)]">
                {t("profile.level")}:{" "}
                <span className="text-[#44944A]">{currentVip.name}</span>
              </h3>
              <p className="text-xs text-[var(--color-text-secondary)] mt-1">
                {t("profile.xp", {
                  xp: (user.xpBalance || 0).toLocaleString("ru-RU"),
                })}
              </p>
            </div>
            {nextVip && (
              <span className="text-xs text-[var(--color-text-muted)] font-mono">
                {t("profile.toNextLevel", {
                  level: nextVip.name,
                  xp: nextVip.minXp.toLocaleString("ru-RU"),
                })}
              </span>
            )}
          </div>
          <div className="h-2 rounded-full bg-[var(--color-bg-primary)] overflow-hidden">
            <div
              className="h-full rounded-full bg-[#44944A] transition-all"
              style={{ width: `${progress}%` }}
            />
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
        <div className="mt-12">
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
                <div
                  key={order.id}
                  className="rounded-2xl bg-[var(--color-bg-card)] border border-[var(--color-border-custom)] p-5"
                >
                  <div className="flex items-center justify-between">
                    <div className="flex items-center gap-3">
                      <Package className="h-5 w-5 text-[var(--color-text-muted)]" />
                      <div>
                        <p className="text-sm font-medium text-[var(--color-text-primary)]">
                          #{order.id}
                        </p>
                        <p className="text-xs text-[var(--color-text-muted)]">
                          {new Date(order.created_at).toLocaleDateString("ru-RU")}
                        </p>
                      </div>
                    </div>
                    <div className="text-right">
                      <p className="text-sm font-medium text-[var(--color-text-primary)]">
                        {(order.total_minor / 100).toLocaleString("ru-RU", {
                          minimumFractionDigits: 2,
                        })}{" "}
                        ₽
                      </p>
                      <p className="text-xs text-[#44944A]">
                        {t(`profile.orderStatus.${order.status}`, order.status)}
                      </p>
                    </div>
                  </div>
                </div>
              ))
            )}
          </div>
        </div>
      </div>
    </div>
  );
}
