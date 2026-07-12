// apps/admin/src/workspaces/CustomerDetail.tsx
//
// Per-customer admin view: profile header + lifetime stats +
// order history.
//
// Visual language matches the project-wide conventions:
//
//   * Header row mirrors `Profile.tsx` (the admin's own
//     profile page): `flex items-center gap-4`, a
//     shadcn `<Avatar>` on the left, name + meta on the
//     right. We diverge only in the meta row — Profile uses
//     `@username · email`, here we use `email · phone ·
//     registration date` so the manager can copy/contact
//     without scrolling.
//
//   * The order history below is a Users-style table
//     (`<motion.div>` rounded-2xl, `px-6 py-3.5` uppercase
//     muted headers, `px-6 py-4` body cells, `hover:bg-…/50`).
//     Order history is a flat list of rows, exactly like
//     the Users list — there's no expandable card to
//     justify the `<OrderCard>` pattern from Orders.tsx.
//
// Cart and favorites are deliberately not shown: the
// manager asked for "клиенты с историей заказов" (clients
// with order history) and surfacing what's in a cart
// right now would invite "did you mean to buy X?"
// messages the storefront doesn't support. The backend
// also doesn't return them. Add later if requested.
import { useState, useEffect, useCallback } from "react";
import { useParams, useNavigate } from "react-router-dom";
import { motion } from "framer-motion";
import {
  fetchAdminCustomerDetail,
  type AdminCustomerDetail as CustomerDetailData,
} from "@/lib/api";
import { Avatar, AvatarFallback } from "@/components/ui/avatar";
import {
  RefreshCw,
  Mail,
  Phone,
  Send,
  Copy,
  Check,
  ShoppingBag,
  TrendingUp,
  Clock,
  UserCircle,
  Users as UsersIcon,
} from "lucide-react";

function fmtMoney(minor: number): string {
  return new Intl.NumberFormat("ru-RU", {
    style: "currency",
    currency: "MDL",
    maximumFractionDigits: 0,
  }).format(minor / 100);
}

function fmtDateTime(s: string): string {
  if (!s) return "—";
  return new Date(s).toLocaleString("ru-RU", {
    year: "numeric",
    month: "short",
    day: "numeric",
    hour: "2-digit",
    minute: "2-digit",
  });
}

function fmtDate(s: string): string {
  if (!s) return "—";
  return new Date(s).toLocaleDateString("ru-RU", {
    year: "numeric",
    month: "short",
    day: "numeric",
  });
}

// Same status palette as Orders.tsx — Tailwind's standard
// 100/800 light-mode + 900/30 + 300 dark-mode. Project-wide
// convention; both light and dark themes read correctly.
const STATUS_LABELS: Record<string, string> = {
  pending: "Ожидает",
  processing: "В обработке",
  shipped: "Отправлен",
  delivered: "Доставлен",
  cancelled: "Отменён",
};

const STATUS_ACCENT: Record<string, string> = {
  pending: "bg-amber-100 text-amber-800 dark:bg-amber-900/30 dark:text-amber-300",
  processing: "bg-blue-100 text-blue-800 dark:bg-blue-900/30 dark:text-blue-300",
  shipped: "bg-purple-100 text-purple-800 dark:bg-purple-900/30 dark:text-purple-300",
  delivered: "bg-emerald-100 text-emerald-800 dark:bg-emerald-900/30 dark:text-emerald-300",
  cancelled: "bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-300",
};

function initials(d: CustomerDetailData): string {
  const first = d.first_name?.[0] || "";
  const last = d.last_name?.[0] || "";
  return (first + last).toUpperCase() || d.email[0]?.toUpperCase() || "?";
}

export default function CustomerDetail() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const [data, setData] = useState<CustomerDetailData | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [copiedChatID, setCopiedChatID] = useState(false);

  const customerID = Number(id);
  const validID = Number.isInteger(customerID) && customerID > 0;

  const load = useCallback(async () => {
    if (!validID) return;
    setLoading(true);
    setError("");
    try {
      const d = await fetchAdminCustomerDetail(customerID);
      setData(d);
    } catch (e) {
      setError(e instanceof Error ? e.message : "Не удалось загрузить клиента");
      setData(null);
    } finally {
      setLoading(false);
    }
  }, [customerID, validID]);

  useEffect(() => {
    load();
  }, [load]);

  // Same clipboard fallback ladder as Customers.tsx. Inline
  // here because the per-row state (`copiedChatID`) is local
  // to this component and there's no second caller yet.
  const copyChatID = async () => {
    if (!data?.telegram_chat_id) return;
    let ok = false;
    try {
      if (navigator.clipboard && window.isSecureContext) {
        await navigator.clipboard.writeText(data.telegram_chat_id);
        ok = true;
      }
    } catch {
      // fall through
    }
    if (!ok) {
      const ta = document.createElement("textarea");
      ta.value = data.telegram_chat_id;
      ta.style.position = "fixed";
      ta.style.opacity = "0";
      document.body.appendChild(ta);
      ta.select();
      try {
        document.execCommand("copy");
        ok = true;
      } catch {
        ok = false;
      }
      document.body.removeChild(ta);
    }
    if (ok) {
      setCopiedChatID(true);
      window.setTimeout(() => setCopiedChatID(false), 1500);
    }
  };

  // ---- early returns (no `data` yet) ----

  if (!validID) {
    return (
      <div className="p-6 lg:p-8 max-w-7xl mx-auto">
        <BackBar />
        <p
          data-testid="customer-error"
          className="text-sm text-red-400 p-3 rounded-xl bg-red-500/10 border border-red-500/20"
        >
          Некорректный идентификатор клиента.
        </p>
      </div>
    );
  }

  if (loading && !data) {
    return (
      <div className="p-6 lg:p-8 max-w-7xl mx-auto">
        <BackBar />
        <div
          data-testid="customer-loading"
          className="flex items-center justify-center py-20"
        >
          <RefreshCw className="h-6 w-6 text-[var(--color-text-muted)] animate-spin" />
        </div>
      </div>
    );
  }

  if (error) {
    return (
      <div className="p-6 lg:p-8 max-w-7xl mx-auto">
        <BackBar />
        <p
          data-testid="customer-error"
          className="text-sm text-red-400 p-3 rounded-xl bg-red-500/10 border border-red-500/20"
        >
          {error}
        </p>
      </div>
    );
  }

  if (!data) return null;

  // ---- main view ----

  return (
    <div className="p-6 lg:p-8 max-w-7xl mx-auto" data-testid="customer-detail-page">
      <BackBar>
        <button
          onClick={load}
          data-testid="customer-refresh"
          className="flex items-center gap-2 rounded-xl bg-[var(--color-bg-card)] border border-[var(--color-border-custom)] px-4 py-2 text-sm text-[var(--color-text-secondary)] hover:text-[var(--color-text-primary)] transition-colors"
        >
          <RefreshCw className={`h-4 w-4 ${loading ? "animate-spin" : ""}`} />
          Обновить
        </button>
      </BackBar>

      {/* Profile header — same flex + Avatar + name/meta shape
          as Profile.tsx. We keep the avatar circle small
          (h-14 w-14) so the row doesn't dominate the
          viewport on tablet. The shadcn `<Avatar>` primitive
          handles the rounded clipping so the inner
          coloured circle can't accidentally overflow. */}
      <motion.div
        initial={{ opacity: 0, y: 20 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ duration: 0.4 }}
        data-testid="customer-profile"
        className="mt-6 flex items-center gap-4 flex-wrap"
      >
        <Avatar className="h-14 w-14">
          <AvatarFallback
            style={{ backgroundColor: data.avatar_color || "#44944A" }}
            className="text-lg font-bold text-white"
          >
            {initials(data)}
          </AvatarFallback>
        </Avatar>
        <div className="min-w-0 flex-1">
          <h1 className="text-xl font-semibold text-[var(--color-text-primary)] truncate">
            {[data.first_name, data.last_name].filter(Boolean).join(" ") || data.email}
          </h1>
          <div className="mt-1 flex flex-wrap items-center gap-x-4 gap-y-1 text-sm text-[var(--color-text-secondary)]">
            <a
              href={`mailto:${data.email}`}
              className="inline-flex items-center gap-1.5 hover:text-[var(--color-text-primary)] transition-colors truncate max-w-[280px]"
            >
              <Mail className="h-3.5 w-3.5 text-[var(--color-text-muted)] shrink-0" />
              <span className="truncate">{data.email}</span>
            </a>
            {data.phone ? (
              <span className="inline-flex items-center gap-1.5 text-emerald-600 dark:text-emerald-400 font-mono whitespace-nowrap">
                <Phone className="h-3.5 w-3.5 shrink-0" />
                {data.phone}
              </span>
            ) : (
              <span className="text-[var(--color-text-muted)] inline-flex items-center gap-1.5">
                <Phone className="h-3.5 w-3.5 shrink-0" />
                нет телефона
              </span>
            )}
            <span className="inline-flex items-center gap-1.5 text-[var(--color-text-muted)] whitespace-nowrap">
              <UserCircle className="h-3.5 w-3.5 shrink-0" />
              с {fmtDate(data.created_at)}
            </span>
          </div>
        </div>
      </motion.div>

      {/* Telegram pinned row — only when linked. Sits on its
          own line under the profile so a long @handle doesn't
          push the registration date off-screen. `whitespace-nowrap`
          on the chat_id copy keeps the icon + label
          together. */}
      {data.telegram_linked && (
        <motion.div
          initial={{ opacity: 0, y: 20 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: 0.4, delay: 0.05 }}
          data-testid="customer-telegram-row"
          className="mt-4 flex flex-wrap items-center gap-x-4 gap-y-1 text-sm"
        >
          <span className="inline-flex items-center gap-1.5">
            <Send className="h-3.5 w-3.5 text-sky-600 dark:text-sky-400 shrink-0" />
            <span className="text-[var(--color-text-muted)]">Telegram:</span>
            {data.telegram_username ? (
              <a
                href={`https://t.me/${data.telegram_username}`}
                target="_blank"
                rel="noopener noreferrer"
                className="text-sky-600 dark:text-sky-400 hover:underline font-medium truncate max-w-[200px]"
              >
                @{data.telegram_username}
              </a>
            ) : (
              <span className="text-[var(--color-text-secondary)]">привязан</span>
            )}
          </span>
          <button
            type="button"
            onClick={copyChatID}
            aria-label="Скопировать chat_id"
            data-testid="customer-detail-tg-copy"
            className="inline-flex items-center gap-1.5 text-[var(--color-text-muted)] hover:text-[var(--color-text-primary)] transition-colors whitespace-nowrap"
          >
            {copiedChatID ? (
              <>
                <Check className="h-3.5 w-3.5 text-emerald-600 dark:text-emerald-400" />
                <span className="text-emerald-600 dark:text-emerald-400">chat_id скопирован</span>
              </>
            ) : (
              <>
                <Copy className="h-3.5 w-3.5" />
                <span>скопировать chat_id</span>
              </>
            )}
          </button>
        </motion.div>
      )}

      {/* Stats row — three small cards. The card itself
          matches the rounded-2xl envelope used by every
          other card in the workspace, but we don't pull in
          the shadcn `<Card>` primitive here because the
          stat content is too simple (icon + label + value)
          to need CardHeader/CardContent. The label uses the
          same `text-[10px] uppercase tracking-wider` muted
          look as the Customers list stat column. */}
      <div className="grid grid-cols-1 sm:grid-cols-3 gap-4 mt-6">
        <StatCard
          label="Заказы"
          value={data.orders_count.toString()}
          icon={<ShoppingBag className="h-3.5 w-3.5" />}
        />
        <StatCard
          label="Потрачено"
          value={data.total_spent_minor > 0 ? fmtMoney(data.total_spent_minor) : "—"}
          icon={<TrendingUp className="h-3.5 w-3.5" />}
        />
        <StatCard
          label="Первый заказ"
          value={fmtDate(data.first_order_at)}
          icon={<Clock className="h-3.5 w-3.5" />}
        />
      </div>

      {/* Order history — Users-style table, same envelope and
          cell padding. Click on a row navigates to /orders
          where the order lives in its full editing context. */}
      <div className="mt-8">
        <h2 className="text-lg font-bold text-[var(--color-text-primary)] mb-4">
          История заказов
        </h2>
        {(data.orders ?? []).length === 0 ? (
          <motion.div
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.4 }}
            data-testid="no-orders"
            className="flex flex-col items-center justify-center py-16 text-center rounded-2xl bg-[var(--color-bg-card)] border border-[var(--color-border-custom)]"
          >
            <UsersIcon className="h-12 w-12 text-[var(--color-text-muted)] mb-4" />
            <p className="text-[var(--color-text-muted)]">Заказов пока нет</p>
          </motion.div>
        ) : (
          <motion.div
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.4 }}
            data-testid="customer-orders-table"
            className="rounded-2xl bg-[var(--color-bg-card)] border border-[var(--color-border-custom)] overflow-hidden"
          >
            <div className="overflow-x-auto">
              <table className="w-full">
                <thead>
                  <tr className="border-b border-[var(--color-border-custom)]">
                    <th className="px-6 py-3.5 text-left text-xs font-semibold text-[var(--color-text-muted)] uppercase tracking-wider">
                      Заказ
                    </th>
                    <th className="px-6 py-3.5 text-left text-xs font-semibold text-[var(--color-text-muted)] uppercase tracking-wider">
                      Тип
                    </th>
                    <th className="px-6 py-3.5 text-right text-xs font-semibold text-[var(--color-text-muted)] uppercase tracking-wider">
                      Позиций
                    </th>
                    <th className="px-6 py-3.5 text-right text-xs font-semibold text-[var(--color-text-muted)] uppercase tracking-wider">
                      Сумма
                    </th>
                    <th className="px-6 py-3.5 text-left text-xs font-semibold text-[var(--color-text-muted)] uppercase tracking-wider">
                      Статус
                    </th>
                    <th className="px-6 py-3.5 text-left text-xs font-semibold text-[var(--color-text-muted)] uppercase tracking-wider">
                      Дата
                    </th>
                  </tr>
                </thead>
                <tbody>
                  {// NOTE: there is no /orders/:id route yet — the admin
                  // Orders workspace renders all orders as cards on one
                  // page. When /orders/:id is added, link here to the
                  // specific order. See PR #52 re-review MEDIUM A.
                  }
                  {(data.orders ?? []).map((o) => (
                    <tr
                      key={o.id}
                      data-testid={`customer-order-row-${o.id}`}
                      onClick={() => navigate("/orders")}
                      className="hover:bg-[var(--color-bg-primary)]/50 transition-colors cursor-pointer"
                    >
                      <td className="px-6 py-4 text-lg font-bold text-[var(--color-text-primary)] font-mono whitespace-nowrap">
                        {o.order_code || `#${o.id}`}
                      </td>
                      <td className="px-6 py-4 text-sm text-[var(--color-text-secondary)] whitespace-nowrap">
                        {o.type === "cart"
                          ? "Корзина"
                          : o.type === "individual"
                            ? "Индивидуальный"
                            : o.type}
                      </td>
                      <td className="px-6 py-4 text-sm text-right text-[var(--color-text-muted)] whitespace-nowrap tabular-nums">
                        {o.items_count}
                      </td>
                      <td className="px-6 py-4 text-sm text-right text-[var(--color-text-primary)] font-mono whitespace-nowrap tabular-nums">
                        {fmtMoney(o.total_minor)}
                      </td>
                      <td className="px-6 py-4 whitespace-nowrap">
                        <span
                          className={`inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium ${
                            STATUS_ACCENT[o.status] ||
                            "bg-gray-100 text-gray-800 dark:bg-gray-900/30 dark:text-gray-300"
                          }`}
                        >
                          {STATUS_LABELS[o.status] || o.status}
                        </span>
                      </td>
                      <td className="px-6 py-4 text-sm text-[var(--color-text-muted)] whitespace-nowrap">
                        {fmtDateTime(o.created_at)}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </motion.div>
        )}
      </div>
    </div>
  );
}

function BackBar({ children }: { children?: React.ReactNode }) {
  const navigate = useNavigate();
  return (
    <div className="flex items-center justify-between mb-6">
      <button
        onClick={() => navigate("/customers")}
        className="flex items-center gap-2 text-sm text-[var(--color-text-secondary)] hover:text-[var(--color-text-primary)] transition-colors"
      >
        ← К списку клиентов
      </button>
      {children}
    </div>
  );
}

interface StatCardProps {
  label: string;
  value: string;
  icon: React.ReactNode;
}

function StatCard({ label, value, icon }: StatCardProps) {
  return (
    <div
      data-testid={`stat-${label.toLowerCase().replace(/\s/g, "-")}`}
      className="rounded-2xl bg-[var(--color-bg-card)] border border-[var(--color-border-custom)] p-4"
    >
      <div className="flex items-center gap-1.5 text-[10px] font-semibold text-[var(--color-text-muted)] uppercase tracking-wider mb-1.5">
        {icon}
        {label}
      </div>
      <div className="text-base font-semibold text-[var(--color-text-primary)]">
        {value}
      </div>
    </div>
  );
}
