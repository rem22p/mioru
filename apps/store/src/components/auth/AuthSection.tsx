import { useState, type FormEvent } from "react";
import { useTranslation } from "react-i18next";
import { motion } from "framer-motion";
import { Mail, Lock, User, Phone } from "lucide-react";
import { useAuthStore } from "@/stores/authStore";
import TelegramLoginButton from "@/components/auth/TelegramLoginButton";

const TELEGRAM_BOT_NAME = import.meta.env.VITE_TELEGRAM_BOT_NAME || "";

export default function AuthSection() {
  const { t } = useTranslation();
  const { login, register } = useAuthStore();

  const [tab, setTab] = useState<"login" | "register">("login");
  const [error, setError] = useState("");
  const [submitting, setSubmitting] = useState(false);

  // Login fields
  const [loginEmail, setLoginEmail] = useState("");
  const [loginPassword, setLoginPassword] = useState("");

  // Register fields
  const [regEmail, setRegEmail] = useState("");
  const [regPassword, setRegPassword] = useState("");
  const [regFirstName, setRegFirstName] = useState("");
  const [regLastName, setRegLastName] = useState("");
  const [regPhone, setRegPhone] = useState("");

  async function handleLogin(e: FormEvent) {
    e.preventDefault();
    setError("");
    if (!loginEmail || !loginPassword) {
      setError("Заполните все поля");
      return;
    }
    setSubmitting(true);
    try {
      await login(loginEmail, loginPassword);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Ошибка входа");
    } finally {
      setSubmitting(false);
    }
  }

  async function handleRegister(e: FormEvent) {
    e.preventDefault();
    setError("");
    if (!regEmail || !regPassword || !regFirstName) {
      setError("Заполните обязательные поля");
      return;
    }
    if (regPassword.length < 8) {
      setError("Пароль минимум 8 символов");
      return;
    }
    setSubmitting(true);
    try {
      await register({
        email: regEmail,
        password: regPassword,
        first_name: regFirstName,
        last_name: regLastName,
        phone: regPhone || undefined,
      });
    } catch (err) {
      setError(err instanceof Error ? err.message : "Ошибка регистрации");
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <motion.div
      initial={{ opacity: 0, y: 20 }}
      animate={{ opacity: 1, y: 0 }}
      className="w-full max-w-sm mx-auto"
    >
      {/* Tabs */}
      <div className="flex rounded-xl bg-[var(--color-bg-primary)] p-1 mb-6">
        <button
          onClick={() => { setTab("login"); setError(""); }}
          className={`flex-1 py-2 text-sm font-medium rounded-lg transition-all ${
            tab === "login"
              ? "bg-[var(--color-bg-card)] text-[var(--color-text-primary)] shadow-sm"
              : "text-[var(--color-text-muted)] hover:text-[var(--color-text-secondary)]"
          }`}
        >
          Вход
        </button>
        <button
          onClick={() => { setTab("register"); setError(""); }}
          className={`flex-1 py-2 text-sm font-medium rounded-lg transition-all ${
            tab === "register"
              ? "bg-[var(--color-bg-card)] text-[var(--color-text-primary)] shadow-sm"
              : "text-[var(--color-text-muted)] hover:text-[var(--color-text-secondary)]"
          }`}
        >
          Регистрация
        </button>
      </div>

      {/* Error */}
      {error && (
        <p className="text-sm text-red-500 mb-4 text-center">{error}</p>
      )}

      {/* Login form */}
      {tab === "login" && (
        <form onSubmit={handleLogin} className="space-y-4">
          <div className="relative">
            <Mail className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-[var(--color-text-muted)]" />
            <input
              type="email"
              value={loginEmail}
              onChange={(e) => setLoginEmail(e.target.value)}
              placeholder="Email"
              className="w-full pl-10 pr-4 py-3 rounded-xl bg-[var(--color-bg-primary)] border border-[var(--color-border-custom)] text-sm text-[var(--color-text-primary)] placeholder:text-[var(--color-text-muted)] focus:outline-none focus:border-[#44944A]/50 transition-colors"
            />
          </div>
          <div className="relative">
            <Lock className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-[var(--color-text-muted)]" />
            <input
              type="password"
              value={loginPassword}
              onChange={(e) => setLoginPassword(e.target.value)}
              placeholder="Пароль"
              className="w-full pl-10 pr-4 py-3 rounded-xl bg-[var(--color-bg-primary)] border border-[var(--color-border-custom)] text-sm text-[var(--color-text-primary)] placeholder:text-[var(--color-text-muted)] focus:outline-none focus:border-[#44944A]/50 transition-colors"
            />
          </div>
          <button
            type="submit"
            disabled={submitting}
            className="w-full py-3 rounded-xl bg-[#44944A] text-white text-sm font-medium hover:bg-[#44944A]/90 disabled:opacity-50 transition-colors"
          >
            {submitting ? "..." : "Войти"}
          </button>
        </form>
      )}

      {/* Register form */}
      {tab === "register" && (
        <form onSubmit={handleRegister} className="space-y-3">
          <div className="relative">
            <User className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-[var(--color-text-muted)]" />
            <input
              type="text"
              value={regFirstName}
              onChange={(e) => setRegFirstName(e.target.value)}
              placeholder="Имя *"
              className="w-full pl-10 pr-4 py-3 rounded-xl bg-[var(--color-bg-primary)] border border-[var(--color-border-custom)] text-sm text-[var(--color-text-primary)] placeholder:text-[var(--color-text-muted)] focus:outline-none focus:border-[#44944A]/50 transition-colors"
            />
          </div>
          <div className="relative">
            <User className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-[var(--color-text-muted)]" />
            <input
              type="text"
              value={regLastName}
              onChange={(e) => setRegLastName(e.target.value)}
              placeholder="Фамилия"
              className="w-full pl-10 pr-4 py-3 rounded-xl bg-[var(--color-bg-primary)] border border-[var(--color-border-custom)] text-sm text-[var(--color-text-primary)] placeholder:text-[var(--color-text-muted)] focus:outline-none focus:border-[#44944A]/50 transition-colors"
            />
          </div>
          <div className="relative">
            <Mail className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-[var(--color-text-muted)]" />
            <input
              type="email"
              value={regEmail}
              onChange={(e) => setRegEmail(e.target.value)}
              placeholder="Email *"
              className="w-full pl-10 pr-4 py-3 rounded-xl bg-[var(--color-bg-primary)] border border-[var(--color-border-custom)] text-sm text-[var(--color-text-primary)] placeholder:text-[var(--color-text-muted)] focus:outline-none focus:border-[#44944A]/50 transition-colors"
            />
          </div>
          <div className="relative">
            <Phone className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-[var(--color-text-muted)]" />
            <input
              type="tel"
              value={regPhone}
              onChange={(e) => setRegPhone(e.target.value)}
              placeholder="Телефон"
              className="w-full pl-10 pr-4 py-3 rounded-xl bg-[var(--color-bg-primary)] border border-[var(--color-border-custom)] text-sm text-[var(--color-text-primary)] placeholder:text-[var(--color-text-muted)] focus:outline-none focus:border-[#44944A]/50 transition-colors"
            />
          </div>
          <div className="relative">
            <Lock className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-[var(--color-text-muted)]" />
            <input
              type="password"
              value={regPassword}
              onChange={(e) => setRegPassword(e.target.value)}
              placeholder="Пароль (мин. 8 символов) *"
              className="w-full pl-10 pr-4 py-3 rounded-xl bg-[var(--color-bg-primary)] border border-[var(--color-border-custom)] text-sm text-[var(--color-text-primary)] placeholder:text-[var(--color-text-muted)] focus:outline-none focus:border-[#44944A]/50 transition-colors"
            />
          </div>
          <button
            type="submit"
            disabled={submitting}
            className="w-full py-3 rounded-xl bg-[#44944A] text-white text-sm font-medium hover:bg-[#44944A]/90 disabled:opacity-50 transition-colors"
          >
            {submitting ? "..." : "Зарегистрироваться"}
          </button>
        </form>
      )}

      {/* Divider + Telegram */}
      <div className="mt-6">
        <div className="flex items-center gap-3 mb-4">
          <div className="flex-1 h-px bg-[var(--color-border-custom)]" />
          <span className="text-xs text-[var(--color-text-muted)]">
            {t("auth.orLoginWith")}
          </span>
          <div className="flex-1 h-px bg-[var(--color-border-custom)]" />
        </div>
        {TELEGRAM_BOT_NAME ? (
          <div className="flex justify-center">
            <TelegramLoginButton botName={TELEGRAM_BOT_NAME} />
          </div>
        ) : (
          <p className="text-xs text-center text-[var(--color-text-muted)]">
            {t("auth.telegramNotConfigured")}
          </p>
        )}
      </div>
    </motion.div>
  );
}
