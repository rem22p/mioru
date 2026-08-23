import { useState } from "react";
import { useNavigate, Link } from "react-router-dom";
import { motion } from "framer-motion";
import { ArrowLeft, Save, Lock } from "lucide-react";
import { useTranslation } from "react-i18next";
import { Helmet } from "@dr.pogodin/react-helmet";
import { useAuthStore } from "@/stores/authStore";
import { fetchStoreCustomerUpdate } from "@/lib/api";
import PhoneInput from "@/components/PhoneInput";

export default function EditProfilePage() {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const { user } = useAuthStore();

  const [firstName, setFirstName] = useState(user?.firstName || "");
  const [lastName, setLastName] = useState(user?.lastName || "");
  const [phone, setPhone] = useState(user?.phone || "");
  const [password, setPassword] = useState("");
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState("");

  if (!user) {
    navigate("/profile", { replace: true });
    return null;
  }

  const handleSave = async (e: React.FormEvent) => {
    e.preventDefault();
    setError("");

    if (!password) {
      setError(t("profile.passwordRequired"));
      return;
    }

    setSaving(true);
    try {
      await fetchStoreCustomerUpdate({
        first_name: firstName,
        last_name: lastName,
        phone,
        current_password: password,
      });
      await useAuthStore.getState().fetchMe();
      navigate("/profile");
    } catch (err) {
      setError(err instanceof Error ? err.message : t("common.error"));
    } finally {
      setSaving(false);
    }
  };

  const inputClass =
    "w-full rounded-xl bg-[var(--color-bg-primary)] border border-[var(--color-border-custom)] px-4 py-3 text-sm text-[var(--color-text-primary)] placeholder:text-[var(--color-text-muted)] focus:outline-none focus:border-[#44944A]/50 transition-colors";

  return (
    <div className="px-6 py-24 lg:px-8">
      <Helmet>
        <title>{t("profile.edit")} — MIORU</title>
      </Helmet>
      <div className="mx-auto max-w-lg">
        <Link
          to="/profile"
          className="inline-flex items-center gap-2 text-sm text-[var(--color-text-secondary)] hover:text-[var(--color-text-primary)] mb-8 transition-colors"
        >
          <ArrowLeft className="h-4 w-4" />
          {t("product.backToCatalog")}
        </Link>

        <motion.h1
          initial={{ opacity: 0, y: 20 }}
          animate={{ opacity: 1, y: 0 }}
          className="text-3xl font-bold tracking-tighter text-[var(--color-text-primary)] mb-8"
        >
          {t("profile.edit")}
        </motion.h1>

        <motion.form
          initial={{ opacity: 0, y: 20 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ delay: 0.1 }}
          onSubmit={handleSave}
          className="rounded-2xl bg-[var(--color-bg-card)] border border-[var(--color-border-custom)] p-6 space-y-5"
        >
          <div>
            <label className="block text-xs font-mono uppercase tracking-wider text-[var(--color-text-secondary)] mb-2">
              {t("auth.firstName")}
            </label>
            <input
              type="text"
              value={firstName}
              onChange={(e) => setFirstName(e.target.value)}
              className={inputClass}
              placeholder={t("profile.namePlaceholder")}
              required
            />
          </div>

          <div>
            <label className="block text-xs font-mono uppercase tracking-wider text-[var(--color-text-secondary)] mb-2">
              {t("auth.lastName")}
            </label>
            <input
              type="text"
              value={lastName}
              onChange={(e) => setLastName(e.target.value)}
              className={inputClass}
              placeholder={t("profile.lastNamePlaceholder")}
            />
          </div>

          <div>
            <label className="block text-xs font-mono uppercase tracking-wider text-[var(--color-text-secondary)] mb-2">
              {t("profile.phone")}
            </label>
            <PhoneInput
              value={phone}
              onChange={setPhone}
              className={inputClass}
              placeholder={t("checkout.phonePlaceholder")}
            />
          </div>

          <div className="border-t border-[var(--color-border-custom)] pt-5">
            <label className="block text-xs font-mono uppercase tracking-wider text-[var(--color-text-secondary)] mb-2">
              {t("profile.confirmPassword")}
            </label>
            <div className="relative">
              <Lock className="absolute left-4 top-1/2 -translate-y-1/2 h-4 w-4 text-[var(--color-text-muted)]" />
              <input
                type="password"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                className={`${inputClass} pl-11`}
                placeholder={t("auth.password")}
                required
              />
            </div>
            <p className="mt-2 text-xs text-[var(--color-text-muted)]">
              {t("profile.confirmPasswordHint")}
            </p>
          </div>

          {error && (
            <p className="text-sm text-red-400 text-center">{error}</p>
          )}

          <button
            type="submit"
            disabled={saving}
            className="w-full flex items-center justify-center gap-2 rounded-xl bg-[#44944A] px-6 py-4 text-sm font-semibold text-black transition-all hover:shadow-[0_0_30px_rgba(68,148,74,0.3)] disabled:opacity-60"
          >
            {saving ? (
              t("common.loading")
            ) : (
              <>
                <Save className="h-4 w-4" />
                {t("common.save")}
              </>
            )}
          </button>
        </motion.form>
      </div>
    </div>
  );
}
