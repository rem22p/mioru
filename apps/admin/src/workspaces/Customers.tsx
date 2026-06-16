// apps/admin/src/workspaces/Customers.tsx
//
// Admin "Customers" workspace: a paginated, searchable list of every
// customer (the people who buy through the storefront) with
// rolled-up order stats and a Telegram link indicator.
//
// Visual language matches the `Users` workspace verbatim —
// `motion.div` rounded-2xl table card, `px-6 py-3.5` uppercase
// muted headers, `px-6 py-4` body cells, `hover:bg-…/50`
// row hover, no border between rows. The Customers table sits
// next to the Users table in the sidebar so they should
// read as the same kind of list. We diverge only in the
// column set (Telegram, lifetime spend, last order) — the
// chrome around them is the same.
//
// The avatars use the shadcn `<Avatar>` primitive (which wraps
// `@radix-ui/react-avatar`) so the avatar's clipped circular
// shape and the font-bold initial behaviour are inherited
// from the project-wide Avatar component rather than a
// one-off `<div className="rounded-full">` we re-invented in
// the first cut. Same goes for the Filter input shape — it
// matches the Orders.tsx filter `select` rounded-xl.
import { useState, useEffect, useCallback, useMemo, useRef } from "react";
import { useNavigate } from "react-router-dom";
import { motion } from "framer-motion";
import {
  fetchAdminCustomers,
  type AdminCustomer,
} from "@/lib/api";
import { Avatar, AvatarFallback } from "@/components/ui/avatar";
import { RefreshCw, Mail, Phone, Send, Copy, Check, Users as UsersIcon } from "lucide-react";

const PER_PAGE = 20;

function fmtMoney(minor: number): string {
  return new Intl.NumberFormat("ru-RU", {
    style: "currency",
    currency: "MDL",
    maximumFractionDigits: 0,
  }).format(minor / 100);
}

function fmtDate(s: string): string {
  if (!s) return "—";
  return new Date(s).toLocaleDateString("ru-RU", {
    year: "numeric",
    month: "short",
    day: "numeric",
  });
}

// Russian plural rule for the "клиент/клиента/клиентов" word.
// Rendered as a *single* string so the count label can't be
// truncated by a flex `truncate` between the number and the
// word (which was the "2 клиент..." bug in the first cut).
function pluralizeClients(n: number): string {
  const mod10 = n % 10;
  const mod100 = n % 100;
  if (mod10 === 1 && mod100 !== 11) return "клиент";
  if (mod10 >= 2 && mod10 <= 4 && (mod100 < 10 || mod100 >= 20))
    return "клиента";
  return "клиентов";
}

function initials(c: AdminCustomer): string {
  const first = c.first_name?.[0] || "";
  const last = c.last_name?.[0] || "";
  return (first + last).toUpperCase() || c.email[0]?.toUpperCase() || "?";
}

export default function Customers() {
  const navigate = useNavigate();
  const [customers, setCustomers] = useState<AdminCustomer[]>([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [search, setSearch] = useState("");
  const [searchInput, setSearchInput] = useState("");
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [copiedChatID, setCopiedChatID] = useState<string | null>(null);
  const debounceRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  // Debounce the search input so we don't issue a fetch on every
  // keystroke — 250ms is short enough to feel instant and long
  // enough that a mid-word fetch doesn't flicker the table.
  useEffect(() => {
    if (debounceRef.current) {
      clearTimeout(debounceRef.current);
    }
    debounceRef.current = setTimeout(() => {
      setSearch(searchInput);
      setPage(1);
    }, 250);
    return () => {
      if (debounceRef.current) clearTimeout(debounceRef.current);
    };
  }, [searchInput]);

  const load = useCallback(async () => {
    setLoading(true);
    setError("");
    try {
      const resp = await fetchAdminCustomers(page, PER_PAGE, search);
      setCustomers(resp.customers);
      setTotal(resp.total);
    } catch (e) {
      setError(e instanceof Error ? e.message : "Не удалось загрузить клиентов");
    } finally {
      setLoading(false);
    }
  }, [page, search]);

  useEffect(() => {
    load();
  }, [load]);

  const totalPages = useMemo(
    () => Math.max(1, Math.ceil(total / PER_PAGE)),
    [total],
  );

  // Same clipboard fallback ladder as Orders.tsx. Kept inline
  // because (a) extracting it before there's a third caller
  // would be YAGNI and (b) the chat_id copy needs the per-row
  // `setCopiedChatID` to flip the icon, so a shared helper
  // would have to take a setter anyway.
  const copyChatID = async (chatID: string) => {
    let ok = false;
    try {
      if (navigator.clipboard && window.isSecureContext) {
        await navigator.clipboard.writeText(chatID);
        ok = true;
      }
    } catch {
      // fall through to legacy path
    }
    if (!ok) {
      const ta = document.createElement("textarea");
      ta.value = chatID;
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
      setCopiedChatID(chatID);
      window.setTimeout(() => {
        setCopiedChatID((prev) => (prev === chatID ? null : prev));
      }, 1500);
    }
  };

  return (
    <div className="p-6 lg:p-8 max-w-7xl mx-auto" data-testid="customers-page">
      {/* Header — same flex justify-between + text-2xl as Orders.tsx
          and Users.tsx. No subtitle (Users doesn't have one
          either); the page title is enough. */}
      <div className="flex items-center justify-between mb-6">
        <h1 className="text-2xl font-bold text-[var(--color-text-primary)]">
          Клиенты
        </h1>
        <button
          onClick={load}
          data-testid="customers-refresh"
          className="flex items-center gap-2 rounded-xl bg-[var(--color-bg-card)] border border-[var(--color-border-custom)] px-4 py-2 text-sm text-[var(--color-text-secondary)] hover:text-[var(--color-text-primary)] transition-colors"
        >
          <RefreshCw className={`h-4 w-4 ${loading ? "animate-spin" : ""}`} />
          Обновить
        </button>
      </div>

      {/* Search + count — same row shape as the Users workspace
          (filter `select`s on the left, total count on the
          right). The `whitespace-nowrap` on the count is what
          kept the "2 клиент..." bug from coming back when the
          parent ran out of room. */}
      <div className="flex items-center gap-3 mb-6 flex-wrap">
        <div className="relative flex-1 min-w-[200px]">
          <Mail className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-[var(--color-text-muted)] pointer-events-none" />
          <input
            type="text"
            placeholder="Поиск по email или имени…"
            value={searchInput}
            onChange={(e) => setSearchInput(e.target.value)}
            data-testid="customers-search"
            aria-label="Поиск клиента"
            className="w-full pl-10 pr-4 py-2 rounded-xl bg-[var(--color-bg-card)] border border-[var(--color-border-custom)] text-sm text-[var(--color-text-primary)] placeholder:text-[var(--color-text-muted)] focus:outline-none focus:border-[var(--color-text-muted)] transition-colors"
          />
        </div>
        <span
          data-testid="customers-total"
          className="text-sm text-[var(--color-text-muted)] ml-auto whitespace-nowrap"
        >
          {total} {pluralizeClients(total)}
        </span>
      </div>

      {error && (
        <p
          data-testid="customers-error"
          className="text-sm text-red-400 mb-4 p-3 rounded-xl bg-red-500/10 border border-red-500/20"
        >
          {error}
        </p>
      )}

      {/* Table — same envelope and cell padding as the Users
          table: motion.div rounded-2xl card, no internal
          borders between rows, hover `bg-…/50`, clickable
          rows that navigate to the detail page. The `table
          w-full` + fixed header + `tbody` skeleton is the
          exact Users shape; we just have a different column
          set (Telegram, lifetime spend, last order instead
          of role/created_at/delete). */}
      {loading ? (
        <div
          data-testid="customers-loading"
          className="flex items-center justify-center py-20"
        >
          <RefreshCw className="h-6 w-6 text-[var(--color-text-muted)] animate-spin" />
        </div>
      ) : customers.length === 0 ? (
        <motion.div
          initial={{ opacity: 0, y: 20 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: 0.4 }}
          data-testid="customers-empty"
          className="flex flex-col items-center justify-center py-20 text-center"
        >
          <UsersIcon className="h-12 w-12 text-[var(--color-text-muted)] mb-4" />
          <p className="text-[var(--color-text-muted)]">
            {search ? "Ничего не найдено" : "Клиентов пока нет"}
          </p>
        </motion.div>
      ) : (
        <motion.div
          initial={{ opacity: 0, y: 20 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: 0.4 }}
          data-testid="customers-table"
          className="rounded-2xl bg-[var(--color-bg-card)] border border-[var(--color-border-custom)] overflow-hidden"
        >
          <div className="overflow-x-auto">
            <table className="w-full">
              <thead>
                <tr className="border-b border-[var(--color-border-custom)]">
                  <th className="px-6 py-3.5 text-left text-xs font-semibold text-[var(--color-text-muted)] uppercase tracking-wider">
                    Клиент
                  </th>
                  <th className="px-6 py-3.5 text-left text-xs font-semibold text-[var(--color-text-muted)] uppercase tracking-wider">
                    Контакты
                  </th>
                  <th className="px-6 py-3.5 text-left text-xs font-semibold text-[var(--color-text-muted)] uppercase tracking-wider">
                    Telegram
                  </th>
                  <th className="px-6 py-3.5 text-right text-xs font-semibold text-[var(--color-text-muted)] uppercase tracking-wider">
                    Заказы
                  </th>
                  <th className="px-6 py-3.5 text-right text-xs font-semibold text-[var(--color-text-muted)] uppercase tracking-wider">
                    Потрачено
                  </th>
                  <th className="px-6 py-3.5 text-left text-xs font-semibold text-[var(--color-text-muted)] uppercase tracking-wider">
                    Последний заказ
                  </th>
                </tr>
              </thead>
              <tbody>
                {customers.map((c) => (
                  <tr
                    key={c.id}
                    onClick={() => navigate(`/customers/${c.id}`)}
                    data-testid={`customer-row-${c.id}`}
                    className="hover:bg-[var(--color-bg-primary)]/50 transition-colors cursor-pointer"
                  >
                    {/* Identity column: Avatar (shadcn) + name +
                        email. The shadcn Avatar wraps the
                        background-coloured circle and the
                        initial text in a single primitive so
                        we don't have to maintain
                        `flex items-center justify-center
                        rounded-full overflow-hidden text-sm
                        font-bold text-white flex-shrink-0`
                        on every screen. The email is a
                        muted-text second line, matching
                        Users.tsx where the username is the
                        muted second line. */}
                    <td className="px-6 py-4">
                      <div className="flex items-center gap-3">
                        <Avatar>
                          <AvatarFallback
                            style={{ backgroundColor: c.avatar_color || "#44944A" }}
                            className="text-sm font-bold text-white"
                          >
                            {initials(c)}
                          </AvatarFallback>
                        </Avatar>
                        <div className="min-w-0">
                          <div className="text-sm font-medium text-[var(--color-text-primary)] truncate max-w-[200px]">
                            {[c.first_name, c.last_name].filter(Boolean).join(" ") || c.email}
                          </div>
                          <div className="text-xs text-[var(--color-text-muted)] truncate max-w-[200px]">
                            {c.email}
                          </div>
                        </div>
                      </div>
                    </td>

                    {/* Phone column — emerald-600 to match the
                        "phone is a known good channel" affordance
                        used in OrderCard. `font-mono
                        whitespace-nowrap` keeps the digits and
                        the `+` together even when the column is
                        narrow. Empty state ("нет телефона")
                        uses the muted text token. */}
                    <td className="px-6 py-4 text-sm">
                      {c.phone ? (
                        <span className="inline-flex items-center gap-1.5 text-emerald-600 dark:text-emerald-400 font-mono whitespace-nowrap">
                          <Phone className="h-3.5 w-3.5 shrink-0" />
                          {c.phone}
                        </span>
                      ) : (
                        <span className="text-[var(--color-text-muted)] inline-flex items-center gap-1.5">
                          <Phone className="h-3.5 w-3.5 shrink-0" />
                          —
                        </span>
                      )}
                    </td>

                    {/* Telegram column — the @username is a
                        deep link; the chat_id is reachable via
                        a copy icon (the icon swaps to a
                        emerald check for ~1.5s after a
                        successful copy). Both the link and
                        the copy button are `e.stopPropagation()`
                        so clicking them doesn't trigger the
                        row's navigate-to-detail. */}
                    <td
                      className="px-6 py-4 text-sm"
                      onClick={(e) => e.stopPropagation()}
                    >
                      {c.telegram_linked ? (
                        <div className="flex items-center gap-1.5">
                          <Send className="h-3.5 w-3.5 text-sky-600 dark:text-sky-400 shrink-0" />
                          {c.telegram_username ? (
                            <a
                              href={`https://t.me/${c.telegram_username}`}
                              target="_blank"
                              rel="noopener noreferrer"
                              className="text-sky-600 dark:text-sky-400 hover:underline truncate max-w-[140px]"
                            >
                              @{c.telegram_username}
                            </a>
                          ) : (
                            <span className="text-[var(--color-text-secondary)]">
                              привязан
                            </span>
                          )}
                          <button
                            type="button"
                            onClick={() => copyChatID(c.telegram_chat_id)}
                            aria-label="Скопировать chat_id"
                            data-testid={`customer-tg-copy-${c.id}`}
                            className="text-[var(--color-text-muted)] hover:text-[var(--color-text-primary)] transition-colors inline-flex items-center"
                          >
                            {copiedChatID === c.telegram_chat_id ? (
                              <Check className="h-3.5 w-3.5 text-emerald-600 dark:text-emerald-400" />
                            ) : (
                              <Copy className="h-3.5 w-3.5" />
                            )}
                          </button>
                        </div>
                      ) : (
                        <span className="text-[var(--color-text-muted)] inline-flex items-center gap-1.5">
                          <Send className="h-3.5 w-3.5 shrink-0" />
                          —
                        </span>
                      )}
                    </td>

                    {/* Right-aligned stat columns.
                        `tabular-nums` keeps the digit widths
                        uniform so the column reads as a
                        numerical list across rows. */}
                    <td className="px-6 py-4 text-sm text-right font-mono tabular-nums text-[var(--color-text-primary)] whitespace-nowrap">
                      {c.orders_count}
                    </td>
                    <td className="px-6 py-4 text-sm text-right font-mono tabular-nums text-[var(--color-text-primary)] whitespace-nowrap">
                      {c.total_spent_minor > 0 ? fmtMoney(c.total_spent_minor) : "—"}
                    </td>
                    <td className="px-6 py-4 text-sm text-[var(--color-text-muted)] whitespace-nowrap">
                      {c.last_order_at ? fmtDate(c.last_order_at) : "—"}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </motion.div>
      )}

      {/* Pagination — same `←`/`→` text arrow buttons as
          Orders.tsx. `px-3 py-2 rounded-xl` rectangles, no
          per-row border, only renders when there's more than
          one page. */}
      {totalPages > 1 && (
        <div
          data-testid="customers-pagination"
          className="flex items-center justify-center gap-2 mt-8"
        >
          <button
            onClick={() => setPage((p) => Math.max(1, p - 1))}
            disabled={page <= 1}
            data-testid="customers-pagination-prev"
            className="px-3 py-2 rounded-xl bg-[var(--color-bg-card)] border border-[var(--color-border-custom)] text-sm text-[var(--color-text-secondary)] disabled:opacity-30"
          >
            ←
          </button>
          <span
            data-testid="customers-pagination-info"
            className="text-sm text-[var(--color-text-muted)]"
          >
            {page} / {totalPages}
          </span>
          <button
            onClick={() => setPage((p) => Math.min(totalPages, p + 1))}
            disabled={page >= totalPages}
            data-testid="customers-pagination-next"
            className="px-3 py-2 rounded-xl bg-[var(--color-bg-card)] border border-[var(--color-border-custom)] text-sm text-[var(--color-text-secondary)] disabled:opacity-30"
          >
            →
          </button>
        </div>
      )}
    </div>
  );
}
