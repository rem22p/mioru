import { useState } from "react";
import { useNavigate, useSearchParams } from "react-router-dom";
import { motion } from "framer-motion";
import { useAuthStore } from "@/stores/authStore";
import { useTranslation } from "react-i18next";
import { Helmet } from "@dr.pogodin/react-helmet";
import { User, Mail, Lock, ArrowRight, Eye, EyeOff } from "lucide-react";

export default function AuthPage() {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const [searchParams] = useSearchParams();
  const redirect = searchParams.get("redirect") || "/catalog";

  const { login, register, loading, error, clearError } = useAuthStore();

  const [mode, setMode] = useState<"login" | "register">("login");
  const [showPassword, setShowPassword] = useState(false);
  const [form, setForm] = useState({
    email: "",
    password: "",
    first_name: "",
    last_name: "",
  });

  const update = (field: string, value: string) => {
    if (error) clearError();
    setForm((p) => ({ ...p, [field]: value }));
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    try {
      if (mode === "login") {
        await login(form.email, form.password);
      } else {
        await register({
          email: form.email,
          password: form.password,
          first_name: form.first_name,
          last_name: form.last_name,
        });
      }
      navigate(redirect, { replace: true });
    } catch {
      // error is set by store
    }
  };

  const inputClass =
    "w-full rounded-xl bg-[var(--color-bg-card)] border border-[var(--color-border-custom)] px-4 py-3 text-sm text-[var(--color-text-primary)] outline-none focus:border-[#44944A] transition-colors";

  return (
    <div className="px-6 py-24 lg:px-8">
      <Helmet>
        <title>{mode === "login" ? "Вход" : "Регистрация"} — MIORU</title>
      </Helmet>
      <div className="mx-auto max-w-md">
        <motion.div
          initial={{ opacity: 0, y: 40 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: 0.8 }}
        >
          {/* Tabs */}
          <div className="flex rounded-xl bg-[var(--color-bg-card)] border border-[var(--color-border-custom)] p-1 mb-8">
            <button
              onClick={() => { setMode("login"); clearError(); setShowPassword(false); }}
              className={`flex-1 rounded-lg py-2.5 text-sm font-semibold transition-all ${
                mode === "login"
                  ? "bg-[#44944A] text-black"
                  : "text-[var(--color-text-secondary)] hover:text-[var(--color-text-primary)]"
              }`}
            >
              {t("auth.login")}
            </button>
            <button
              onClick={() => { setMode("register"); clearError(); }}
              className={`flex-1 rounded-lg py-2.5 text-sm font-semibold transition-all ${
                mode === "register"
                  ? "bg-[#44944A] text-black"
                  : "text-[var(--color-text-secondary)] hover:text-[var(--color-text-primary)]"
              }`}
            >
              {t("auth.register")}
            </button>
          </div>

          <form onSubmit={handleSubmit} className="space-y-4">
            {mode === "register" && (
              <>
                <div className="relative">
                  <User className="absolute left-4 top-1/2 -translate-y-1/2 h-4 w-4 text-[var(--color-text-muted)]" />
                  <input
                    type="text"
                    value={form.first_name}
                    onChange={(e) => update("first_name", e.target.value)}
                    className={`${inputClass} pl-11`}
                    placeholder={t("auth.firstName")}
                    required
                  />
                </div>
                <div className="relative">
                  <User className="absolute left-4 top-1/2 -translate-y-1/2 h-4 w-4 text-[var(--color-text-muted)]" />
                  <input
                    type="text"
                    value={form.last_name}
                    onChange={(e) => update("last_name", e.target.value)}
                    className={`${inputClass} pl-11`}
                    placeholder={t("auth.lastName")}
                  />
                </div>
              </>
            )}

            <div className="relative">
              <Mail className="absolute left-4 top-1/2 -translate-y-1/2 h-4 w-4 text-[var(--color-text-muted)]" />
              <input
                type="email"
                value={form.email}
                onChange={(e) => update("email", e.target.value)}
                className={`${inputClass} pl-11`}
                placeholder="email@example.com"
                required
              />
            </div>

            <div className="relative">
              <Lock className="absolute left-4 top-1/2 -translate-y-1/2 h-4 w-4 text-[var(--color-text-muted)]" />
              <input
                type={showPassword ? "text" : "password"}
                value={form.password}
                onChange={(e) => update("password", e.target.value)}
                className={`${inputClass} pl-11 pr-11`}
                placeholder={t("auth.password")}
                required
                minLength={8}
              />
              <button
                type="button"
                onClick={() => setShowPassword(!showPassword)}
                className="absolute right-4 top-1/2 -translate-y-1/2 text-[var(--color-text-muted)] hover:text-[var(--color-text-primary)] transition-colors"
                tabIndex={-1}
              >
                {showPassword ? <EyeOff className="h-4 w-4" /> : <Eye className="h-4 w-4" />}
              </button>
            </div>

            {error && (
              <p className="text-sm text-red-400 text-center">{error}</p>
            )}

            <button
              type="submit"
              disabled={loading}
              className="w-full flex items-center justify-center gap-2 rounded-xl bg-[#44944A] px-6 py-4 text-sm font-semibold text-black transition-all hover:shadow-[0_0_30px_rgba(68,148,74,0.3)] disabled:opacity-60"
            >
              {loading
                ? t("common.loading")
                : mode === "login"
                  ? t("auth.login")
                  : t("auth.register")}
              {!loading && <ArrowRight className="h-4 w-4" />}
            </button>
          </form>
        </motion.div>
      </div>
    </div>
  );
}
