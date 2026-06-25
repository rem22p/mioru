// apps/admin/src/workspaces/Telegram.tsx
//
// Telegram admin workspace: shows the live state of the
// notifier (bot token set? username? last 24h counters?),
// lets the manager send a "🧪 test" message to every
// configured manager chat, and lists every send attempt in
// reverse chronological order.
//
// All three cards (status, send test, history) follow the
// same `motion.div` + `rounded-2xl bg-card border-custom`
// envelope used in Orders.tsx and Customers.tsx so this
// workspace feels like part of the same app, not a
// one-off. The history table is Users.tsx-style (table
// + uppercase muted headers + `px-6 py-3.5`).
//
// Visual language matches the rest of the admin:
//
//   * Outer wrapper is `p-6 lg:p-8 max-w-7xl mx-auto` (same
//     as Orders/Customers).
//   * Page header is `text-2xl font-bold` (no subtitle — the
//     same choice Orders.tsx made; the Status card is the
//     "subtitle" here).
//   * Status colours: green for OK, red for FAIL, gray for
//     "never sent / not configured", matching the project-wide
//     Tailwind 100/800 + dark:900/30 + 300 palette used by
//     Orders' status badges.

import { useEffect, useState, useCallback } from "react";
import { motion } from "framer-motion";
import { RefreshCw, Send, AlertCircle, CheckCircle2, Clock, Inbox } from "lucide-react";
import {
  fetchTelegramDiagnose,
  fetchTelegramMessages,
  sendTelegramTest,
  type TelegramDiagnose,
  type TelegramMessageRow,
  type TelegramTestResult,
} from "@/lib/api";

export default function Telegram() {
  const [diagnose, setDiagnose] = useState<TelegramDiagnose | null>(null);
  const [messages, setMessages] = useState<TelegramMessageRow[]>([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [statusFilter, setStatusFilter] = useState<"all" | "sent" | "failed" | "pending">("all");
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [testing, setTesting] = useState(false);
  const [testResults, setTestResults] = useState<TelegramTestResult[] | null>(null);

  const load = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const [d, m] = await Promise.all([
        fetchTelegramDiagnose(),
        fetchTelegramMessages(page, 20, statusFilter === "all" ? undefined : statusFilter),
      ]);
      setDiagnose(d);
      setMessages(m.messages);
      setTotal(m.total);
    } catch (e) {
      setError(e instanceof Error ? e.message : "ошибка загрузки");
    } finally {
      setLoading(false);
    }
  }, [page, statusFilter]);

  useEffect(() => {
    void load();
  }, [load]);

  const handleTest = async () => {
    setTesting(true);
    setTestResults(null);
    try {
      const r = await sendTelegramTest();
      setTestResults(r.results);
    } catch (e) {
      setError(e instanceof Error ? e.message : "ошибка отправки");
    } finally {
      setTesting(false);
      void load();
    }
  };

  return (
    <div className="p-6 lg:p-8 max-w-7xl mx-auto" data-testid="telegram-page">
      {/* Header — matches Orders.tsx: text-2xl + refresh, no subtitle */}
      <div className="flex items-center justify-between mb-6">
        <h1 className="text-2xl font-bold text-[var(--color-text-primary)]">
          Telegram
        </h1>
        <button
          onClick={() => void load()}
          className="flex items-center gap-2 rounded-xl bg-[var(--color-bg-card)] border border-[var(--color-border-custom)] px-4 py-2 text-sm text-[var(--color-text-secondary)] hover:text-[var(--color-text-primary)] transition-colors"
          data-testid="telegram-refresh"
        >
          <RefreshCw className={`h-4 w-4 ${loading ? "animate-spin" : ""}`} />
          Обновить
        </button>
      </div>

      {error && (
        <div className="mb-4 rounded-2xl bg-red-500/10 border border-red-500/30 px-4 py-3 text-sm text-red-400">
          {error}
        </div>
      )}

      {/* Status card */}
      <motion.div
        initial={{ opacity: 0, y: 8 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ duration: 0.4, delay: 0.05, ease: [0.16, 1, 0.3, 1] }}
        className="rounded-2xl bg-[var(--color-bg-card)] border border-[var(--color-border-custom)] p-6 mb-4"
        data-testid="telegram-status-card"
      >
        {diagnose ? (
          <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
            <StatusTile
              label="Бот"
              icon={diagnose.bot_token_set ? CheckCircle2 : AlertCircle}
              ok={diagnose.bot_token_set}
              value={diagnose.bot_token_set ? `@${diagnose.bot_username || "?"}` : "TELEGRAM_BOT_TOKEN не задан"}
            />
            <StatusTile
              label="Manager чатов"
              icon={Send}
              ok={diagnose.manager_chat_count > 0}
              value={
                diagnose.manager_chat_count > 0
                  ? String(diagnose.manager_chat_count)
                  : "TELEGRAM_MANAGER_CHAT_IDS не заданы"
              }
            />
            <StatusTile
              label="Последняя отправка"
              icon={Clock}
              ok={diagnose.last_send_status === "ok"}
              value={
                diagnose.last_send_at
                  ? `${formatDateTime(diagnose.last_send_at)} — ${statusLabel(diagnose.last_send_status)}`
                  : "никогда"
              }
              error={diagnose.last_send_error}
            />
            <StatusTile
              label="24 часа"
              icon={Inbox}
              ok={diagnose.last_24h_failed === 0}
              value={`${diagnose.last_24h_total} (${diagnose.last_24h_sent} ✓ / ${diagnose.last_24h_failed} ✗)`}
            />
          </div>
        ) : (
          <div className="flex items-center justify-center py-8 text-[var(--color-text-muted)]">
            <RefreshCw className="h-6 w-6 animate-spin" />
          </div>
        )}
      </motion.div>

      {/* Send test */}
      <motion.div
        initial={{ opacity: 0, y: 8 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ duration: 0.4, delay: 0.1, ease: [0.16, 1, 0.3, 1] }}
        className="rounded-2xl bg-[var(--color-bg-card)] border border-[var(--color-border-custom)] p-6 mb-4"
        data-testid="telegram-test-card"
      >
        <div className="flex items-center justify-between gap-3 flex-wrap">
          <div>
            <h2 className="text-sm font-semibold text-[var(--color-text-primary)]">
              Отправить тестовое сообщение
            </h2>
            <p className="text-xs text-[var(--color-text-muted)] mt-0.5">
              Проверяет бота, chat_id и сетевой доступ. Сообщение появится в истории ниже.
            </p>
          </div>
          <button
            onClick={handleTest}
            disabled={testing || !diagnose?.bot_token_set || diagnose?.manager_chat_count === 0}
            data-testid="telegram-send-test"
            className="flex items-center gap-2 rounded-xl bg-[#44944A] hover:bg-[#3a7d3f] disabled:opacity-50 disabled:cursor-not-allowed px-4 py-2 text-sm font-medium text-white transition-colors"
          >
            <Send className={`h-4 w-4 ${testing ? "animate-pulse" : ""}`} />
            {testing ? "Отправляю…" : "Отправить тест"}
          </button>
        </div>
        {testResults && (
          <div className="mt-4 space-y-2">
            {testResults.map((r) => (
              <div
                key={r.chat_id}
                data-testid={`telegram-test-result-${r.chat_id}`}
                className={`rounded-xl border px-3 py-2 text-sm ${
                  r.ok
                    ? "border-[#44944A]/30 bg-[#44944A]/10 text-[#44944A]"
                    : "border-red-500/30 bg-red-500/10 text-red-400"
                }`}
              >
                <span className="font-mono">chat {r.chat_id}</span>: {r.ok ? "OK" : `FAIL — ${r.error}`} ({r.duration_ms} ms)
              </div>
            ))}
          </div>
        )}
      </motion.div>

      {/* History */}
      <motion.div
        initial={{ opacity: 0, y: 8 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ duration: 0.4, delay: 0.15, ease: [0.16, 1, 0.3, 1] }}
        className="rounded-2xl bg-[var(--color-bg-card)] border border-[var(--color-border-custom)] overflow-hidden"
        data-testid="telegram-history-card"
      >
        <div className="px-6 py-4 border-b border-[var(--color-border-custom)] flex items-center justify-between flex-wrap gap-3">
          <h2 className="text-sm font-semibold text-[var(--color-text-primary)]">
            История сообщений
            <span className="ml-2 text-xs font-normal text-[var(--color-text-muted)]">
              {total}
            </span>
          </h2>
          <div className="flex items-center gap-1 text-xs">
            {(["all", "sent", "failed", "pending"] as const).map((f) => (
              <button
                key={f}
                onClick={() => {
                  setStatusFilter(f);
                  setPage(1);
                }}
                data-testid={`telegram-filter-${f}`}
                className={`px-3 py-1.5 rounded-full transition-colors ${
                  statusFilter === f
                    ? "bg-[var(--color-text-primary)] text-[var(--color-bg-primary)]"
                    : "text-[var(--color-text-muted)] hover:text-[var(--color-text-primary)]"
                }`}
              >
                {f === "all" ? "Все" : f === "sent" ? "Отправлено" : f === "failed" ? "Ошибка" : "В процессе"}
              </button>
            ))}
          </div>
        </div>

        {messages.length === 0 ? (
          <div className="flex flex-col items-center justify-center py-16 text-[var(--color-text-muted)]" data-testid="telegram-empty">
            <Inbox className="h-12 w-12 mb-4" />
            <p className="text-sm">{loading ? "Загрузка…" : "Нет сообщений"}</p>
          </div>
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full">
              <thead>
                <tr className="border-b border-[var(--color-border-custom)]">
                  <th className="px-6 py-3.5 text-left text-xs font-semibold text-[var(--color-text-muted)] uppercase tracking-wider">
                    Время
                  </th>
                  <th className="px-6 py-3.5 text-left text-xs font-semibold text-[var(--color-text-muted)] uppercase tracking-wider">
                    Чат
                  </th>
                  <th className="px-6 py-3.5 text-left text-xs font-semibold text-[var(--color-text-muted)] uppercase tracking-wider">
                    Текст
                  </th>
                  <th className="px-6 py-3.5 text-left text-xs font-semibold text-[var(--color-text-muted)] uppercase tracking-wider">
                    Статус
                  </th>
                </tr>
              </thead>
              <tbody>
                {messages.map((m) => (
                  <tr
                    key={m.id}
                    data-testid={`telegram-row-${m.id}`}
                    className="border-b border-[var(--color-border-custom)] last:border-0 hover:bg-[var(--color-bg-primary)]/50 transition-colors"
                  >
                    <td className="px-6 py-4 text-xs text-[var(--color-text-muted)] whitespace-nowrap tabular-nums font-mono">
                      {formatDateTime(m.sent_at)}
                    </td>
                    <td className="px-6 py-4 text-sm text-[var(--color-text-secondary)] font-mono">
                      {m.chat_id}
                    </td>
                    <td className="px-6 py-4 text-sm text-[var(--color-text-primary)] max-w-md">
                      <div className="line-clamp-2">{m.text}</div>
                      {m.error && (
                        <div className="text-xs text-red-400 mt-1 line-clamp-1">{m.error}</div>
                      )}
                    </td>
                    <td className="px-6 py-4">
                      <span
                        className={`inline-flex items-center gap-1.5 text-xs font-medium px-2 py-0.5 rounded-full ${statusBadgeClass(m.status)}`}
                      >
                        {statusBadgeLabel(m.status)}
                        {m.http_status && (
                          <span className="font-mono opacity-70">HTTP {m.http_status}</span>
                        )}
                      </span>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}

        {/* Pagination footer */}
        {total > 20 && (
          <div className="px-6 py-4 border-t border-[var(--color-border-custom)] flex items-center justify-between text-sm text-[var(--color-text-muted)]">
            <span>
              Показано {(page - 1) * 20 + 1}–{Math.min(page * 20, total)} из {total}
            </span>
            <div className="flex items-center gap-2">
              <button
                onClick={() => setPage((p) => Math.max(1, p - 1))}
                disabled={page === 1}
                data-testid="telegram-prev"
                className="px-3 py-1.5 rounded-xl bg-[var(--color-bg-card)] border border-[var(--color-border-custom)] text-sm text-[var(--color-text-secondary)] hover:text-[var(--color-text-primary)] disabled:opacity-30 disabled:cursor-not-allowed transition-colors"
              >
                ←
              </button>
              <span className="tabular-nums">{page}</span>
              <button
                onClick={() => setPage((p) => p + 1)}
                disabled={page * 20 >= total}
                data-testid="telegram-next"
                className="px-3 py-1.5 rounded-xl bg-[var(--color-bg-card)] border border-[var(--color-border-custom)] text-sm text-[var(--color-text-secondary)] hover:text-[var(--color-text-primary)] disabled:opacity-30 disabled:cursor-not-allowed transition-colors"
              >
                →
              </button>
            </div>
          </div>
        )}
      </motion.div>
    </div>
  );
}

interface StatusTileProps {
  label: string;
  icon: React.ComponentType<{ className?: string }>;
  ok: boolean;
  value: string;
  error?: string;
}

function StatusTile({ label, icon: Icon, ok, value, error }: StatusTileProps) {
  return (
    <div
      className="rounded-xl bg-[var(--color-bg-primary)] border border-[var(--color-border-light)] p-4"
      data-testid={`telegram-tile-${label.toLowerCase().replace(/\s/g, "-")}`}
    >
      <div className="flex items-center gap-2 text-xs font-medium text-[var(--color-text-muted)] uppercase tracking-wider">
        <Icon className="h-3.5 w-3.5" />
        {label}
      </div>
      <div
        className={`mt-2 text-sm font-medium ${ok ? "text-[#44944A]" : "text-red-400"} truncate`}
        title={value}
      >
        {value}
      </div>
      {error && !ok && (
        <div className="mt-1 text-xs text-red-400/80 line-clamp-2" title={error}>
          {error}
        </div>
      )}
    </div>
  );
}

function formatDateTime(s: string): string {
  if (!s) return "—";
  // Backend returns ISO timestamps. We render in the manager's
  // local time without a timezone suffix so the "Last send"
  // tile reads "16.06.2026 15:29" — easier to scan than a
  // full RFC3339 string.
  return new Date(s).toLocaleString("ru-RU", {
    year: "numeric",
    month: "2-digit",
    day: "2-digit",
    hour: "2-digit",
    minute: "2-digit",
  });
}

function statusLabel(s: TelegramDiagnose["last_send_status"]): string {
  switch (s) {
    case "ok":
      return "OK";
    case "failed":
      return "ошибка";
    case "never":
      return "не отправлялось";
    default:
      return "неизвестно";
  }
}

function statusBadgeClass(status: TelegramMessageRow["status"]): string {
  switch (status) {
    case "sent":
      return "bg-[#44944A]/10 text-[#44944A]";
    case "failed":
      return "bg-red-500/10 text-red-400";
    case "pending":
      return "bg-amber-500/10 text-amber-400";
  }
}

function statusBadgeLabel(status: TelegramMessageRow["status"]): string {
  switch (status) {
    case "sent":
      return "Отправлено";
    case "failed":
      return "Ошибка";
    case "pending":
      return "В процессе";
  }
}
