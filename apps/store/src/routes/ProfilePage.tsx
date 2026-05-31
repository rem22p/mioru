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

  const nextLevel = VIP_LEVELS.find((lvl) => lvl.minXp > (user.xpBalance || 0));
  const xpProgress = nextLevel
    ? Math.min(
        100,
        Math.round(
          ((user.xpBalance || 0) / nextLevel.minXp) * 100,
        ),
      )
    : 100;

  return (
    <div className="px-6 py-24 lg:px-8">
      <Helmet>
        <title>{t("profile.title")} — MIORU</title>
      </Helmet>
      <div className="mx-auto max-w-2xl space-y-10">
        <motion.div
          initial={{ opacity: 0, y: 30 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: 0.6 }}
          className="rounded-2xl bg-[var(--color-bg-card)] border border-[var(--color-border-custom)] p-8"
        >
          <div className="flex items-center gap-4 mb-6">
            <div
              className="flex h-14 w-14 items-center justify-center rounded-full text-lg font-bold text-white"
              style={{ backgroundColor: user.avatarParams ? "#44944A" : "#44944A" }}
            >
              {(user.name || user.email || "?")[0].toUpperCase()}
            </div>
            <div>
              <h2 className="text-lg font-semibold text-[var(--color-text-primary)]">
                {user.name || user.email}
              </h2>
              <p className="text-sm text-[var(--color-text-muted)]">
                {user.email}
              </p>
            </div>
          </div>

          <div className="mb-6">
            <div className="flex items-center justify-between mb-2">
              <span className="text-sm text-[var(--color-text-secondary)]">
                {t("profile.level")} {nextLevel ? nextLevel.level : "MAX"}
              </span>
              <span className="text-xs text-[var(--color-text-muted)]">
                {t("profile.xp", { xp: user.xpBalance || 0 })}
              </span>
            </div>
            <div className="h-2 rounded-full bg-[var(--color-bg-hover)] overflow-hidden">
              <div
                className="h-full rounded-full bg-[#44944A] transition-all duration-500"
                style={{ width: `${xpProgress}%` }}
              />
            </div>
            {nextLevel && (
              <p className="text-xs text-[var(--color-text-muted)] mt-1">
                {t("profile.toNextLevel", {
                  level: nextLevel.level,
                  xp: nextLevel.minXp - (user.xpBalance || 0),
                })}
              </p>
            )}
          </div>

          <div className="space-y-2">
            <Link
              to="/avatar"
              className="flex items-center justify-between rounded-xl bg-[var(--color-bg-hover)] px-4 py-3 text-sm text-[var(--color-text-primary)] hover:brightness-110 transition"
            >
              <span className="flex items-center gap-3">
                <User className="h-4 w-4 text-[var(--color-text-muted)]" />
                {t("profile.myAvatar")}
              </span>
              <ChevronRight className="h-4 w-4 text-[var(--color-text-muted)]" />
            </Link>
            <Link
              to="/profile"
              className="flex items-center justify-between rounded-xl bg-[var(--color-bg-hover)] px-4 py-3 text-sm text-[var(--color-text-primary)] hover:brightness-110 transition"
            >
              <span className="flex items-center gap-3">
                <Settings className="h-4 w-4 text-[var(--color-text-muted)]" />
                {t("profile.title")}
              </span>
              <ChevronRight className="h-4 w-4 text-[var(--color-text-muted)]" />
            </Link>
          </div>

          <button
            onClick={logout}
            className="mt-4 w-full flex items-center justify-center gap-2 rounded-xl bg-red-500/10 px-4 py-3 text-sm text-red-400 hover:bg-red-500/20 transition"
          >
            <LogOut className="h-4 w-4" />
            {t("auth.logout")}
          </button>
        </motion.div>

        <div>
          <h3 className="text-lg font-semibold text-[var(--color-text-primary)] mb-4">
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
