import { useState, useEffect, useRef } from "react";
import { motion } from "framer-motion";
import { Helmet } from "@dr.pogodin/react-helmet";
import { Link, useNavigate } from "react-router-dom";
import { useTranslation } from "react-i18next";
import { Upload, Send } from "lucide-react";
import CityAutocomplete from "@/components/ui/CityAutocomplete";
import PhoneInput from "@/components/PhoneInput";
import { createOrder, uploadOrderPhoto } from "@/lib/api";
import { useAuthStore } from "@/stores/authStore";
import { isDeliveryBlocked } from "@/lib/deliveryRules";
import { isValidPhone } from "@/lib/phoneValidation";

const deliveryTimeOptions = ["fast", "medium", "slow"] as const;

const deliveryMethods = [
  { key: "personal", price: "Бесплатно", priceColor: "text-[#44944A]" },
  {
    key: "address",
    price: "25 руб",
    priceColor: "text-[var(--color-text-secondary)]",
  },
  {
    key: "bus",
    price: "до 50 руб",
    priceColor: "text-[var(--color-text-secondary)]",
  },
  {
    key: "express",
    price: "до 50 руб",
    priceColor: "text-[var(--color-text-secondary)]",
  },
  {
    key: "moldovaPost",
    price: "до 50 руб",
    priceColor: "text-[var(--color-text-secondary)]",
  },
] as const;

export default function CustomOrderPage() {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const user = useAuthStore((s) => s.user);

  // KAN: guests can open and fill the custom-order form — login is required
  // only on submit (see the !telegramLinked gate below and on handleSubmit).
  // The previous top-level redirect to /profile bounced guests before they
  // could even see the form.

  // Same gate as CheckoutPage: this form posts to the same POST /api/store/orders,
  // which answers 403 TELEGRAM_REQUIRED — and does so only after the photos have
  // already been uploaded, leaving them orphaned in UPLOAD_DIR on every retry.
  const telegramLinked = Boolean(user?.telegram?.linked);

  const [photos, setPhotos] = useState<File[]>([]);
  const [photoPreviews, setPhotoPreviews] = useState<string[]>([]);
  const [height, setHeight] = useState("");
  const [weight, setWeight] = useState("");
  const [phone, setPhone] = useState("");
  const [city, setCity] = useState("");
  const [deliveryTime, setDeliveryTime] = useState<string>("");
  const [deliveryMethod, setDeliveryMethod] = useState("");

  // Reset delivery method if the user changed the city and their
  // current selection is no longer available there. Mirrors the
  // behaviour in CheckoutPage so the two order flows stay
  // consistent — without this reset the user could fill out the
  // rest of the form only to see a server 400 (or a stale local
  // pick) on submit.
  useEffect(() => {
    if (deliveryMethod && isDeliveryBlocked(deliveryMethod, city)) {
      setDeliveryMethod("");
    }
  }, [city, deliveryMethod]);
  const [comment, setComment] = useState("");
  const [submitted, setSubmitted] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const [submitError, setSubmitError] = useState("");
  const [errors, setErrors] = useState<Record<string, string>>({});
  const [touched, setTouched] = useState<Record<string, boolean>>({});
  // Same persistence rule as CheckoutPage — see comment there. PR #56's 25s
  // timeout can abort a committed order, so the retry MUST reuse the key for
  // the backend to dedupe (priority #1: no double-counted orders).
  const idempotencyKeyRef = useRef<string | null>(null);

  const handlePhotos = (e: React.ChangeEvent<HTMLInputElement>) => {
    const files = Array.from(e.target.files || []);
    if (files.length > 0) {
      setPhotos((prev) => [...prev, ...files]);
      const previews = files.map((f) => URL.createObjectURL(f));
      setPhotoPreviews((prev) => [...prev, ...previews]);
      setErrors((prev) => ({ ...prev, photos: "" }));
      setTouched((prev) => ({ ...prev, photos: false }));
    }
  };

  const removePhoto = (index: number) => {
    setPhotos((prev) => prev.filter((_, i) => i !== index));
    setPhotoPreviews((prev) => prev.filter((_, i) => i !== index));
  };

  const validate = () => {
    const errs: Record<string, string> = {};
    if (photos.length === 0) errs.photos = "Прикрепите хотя бы одно фото";
    if (!height && !weight) errs.body = "Укажите рост или вес";
    if (height && Number(height) < 100)
      errs.body = "Рост не может быть меньше 100 см";
    if (height && Number(height) > 250)
      errs.body = "Рост не может быть больше 250 см";
    if (weight && Number(weight) < 30)
      errs.body = "Вес не может быть меньше 30 кг";
    if (weight && Number(weight) > 200)
      errs.body = "Вес не может быть больше 200 кг";
    if (!city.trim()) errs.city = "Укажите город";
    if (!phone.trim()) errs.phone = "Введите номер телефона";
    else if (!isValidPhone(phone))
      errs.phone = "Формат: +373 и 8 цифр";
    if (!deliveryMethod) errs.deliveryMethod = "Выберите способ доставки";
    setErrors(errs);
    return Object.keys(errs).length === 0;
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setTouched({ photos: true, body: true, phone: true, city: true, deliveryMethod: true });
    if (!validate()) return;
    if (!telegramLinked) {
      navigate("/profile?redirect=/custom-order", { replace: true });
      return;
    }

    setSubmitting(true);
    setSubmitError("");
    try {
      // Upload photos first, collect URLs
      const photoUrls: string[] = [];
      for (const file of photos) {
        const url = await uploadOrderPhoto(file);
        photoUrls.push(url);
      }

      const idempotencyKey =
        idempotencyKeyRef.current ?? (idempotencyKeyRef.current = crypto.randomUUID());
      await createOrder(
        {
          type: "individual",
          phone: phone.trim(),
          city: city.trim(),
          delivery_method: deliveryMethod,
          payment_method: "cod",
          total_minor: 0,
          height: height ? parseFloat(height) : undefined,
          weight: weight ? parseFloat(weight) : undefined,
          delivery_time: deliveryTime ? [deliveryTime] : [],
          comment,
          photos: photoUrls,
        },
        idempotencyKey,
      );
      setSubmitted(true);
      // Release the key — see comment on idempotencyKeyRef.
      idempotencyKeyRef.current = null;
    } catch (e: unknown) {
      setSubmitError(e instanceof Error ? e.message : "Ошибка при отправке");
    } finally {
      setSubmitting(false);
    }
  };

  const handleBlur = (field: string) => {
    setTouched((prev) => ({ ...prev, [field]: true }));
  };

  const canSubmit =
    photos.length > 0 && (height || weight) && city && deliveryMethod;

  const inputClass =
    "w-full rounded-xl bg-[var(--color-bg-card)] border border-[var(--color-border-custom)] px-4 py-3 text-base sm:text-sm text-[var(--color-text-primary)] outline-none focus:border-[#44944A] transition-colors placeholder:text-[var(--color-text-muted)]";

  return (
    <div className="px-6 py-24 lg:px-8">
      <Helmet>
        <title>{t("customOrder.title")} — MIORU</title>
        <meta
          name="description"
          content="Закажите одежду индивидуально через MIORU. Прикрепите фото, укажите параметры, и мы найдём нужную вещь."
        />
        <link rel="canonical" href="https://mioru.store/custom-order" />
      </Helmet>

      <div className="mx-auto max-w-2xl">
        <motion.h1
          initial={{ opacity: 0, y: 20 }}
          animate={{ opacity: 1, y: 0 }}
          className="text-4xl font-bold tracking-tighter text-[var(--color-text-primary)] sm:text-5xl"
        >
          {t("customOrder.title")}
        </motion.h1>

        <p className="mt-4 text-base text-[var(--color-text-muted)] max-w-2xl leading-relaxed">
          {t("customOrder.description")}
        </p>

        {!submitted ? (
          <motion.form
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ delay: 0.1 }}
            onSubmit={handleSubmit}
            className="mt-10 space-y-8"
          >
            {/* Manager block */}
            <Link
              to="/contacts"
              className="flex items-center justify-between gap-3 rounded-xl bg-[#44944A]/5 border border-[#44944A]/20 p-5 hover:bg-[#44944A]/10 hover:border-[#44944A]/40 transition-all cursor-pointer group"
            >
              <div>
                <p className="text-sm font-medium text-[var(--color-text-primary)]">
                  {t("customOrder.managerBlock")}
                </p>
                <p className="text-xs text-[var(--color-text-muted)] mt-1">
                  {t("customOrder.managerSubtitle")}
                </p>
              </div>
              <div className="w-9 h-9 rounded-full bg-[#44944A] flex items-center justify-center shrink-0 group-hover:scale-105 transition-transform">
                <span className="text-white text-lg leading-none">→</span>
              </div>
            </Link>

            {/* Photo */}
            <div>
              <label className="block text-sm font-medium text-[var(--color-text-primary)] mb-2">
                {t("customOrder.photo")}
              </label>

              {/* Previews */}
              {photoPreviews.length > 0 && (
                <div className="grid grid-cols-3 gap-2 mb-3">
                  {photoPreviews.map((src, i) => (
                    <div key={i} className="relative">
                      <img
                        src={src}
                        alt={`Preview ${i + 1}`}
                        className="w-full h-24 object-cover rounded-lg"
                      />
                      <button
                        type="button"
                        onClick={() => removePhoto(i)}
                        className="absolute -top-1.5 -right-1.5 w-5 h-5 bg-red-500 text-white rounded-full text-xs flex items-center justify-center hover:bg-red-600 transition-colors z-20"
                      >
                        ×
                      </button>
                    </div>
                  ))}
                </div>
              )}

              {/* Upload area */}
              <div className="relative">
                <input
                  type="file"
                  accept="image/*"
                  multiple
                  onChange={handlePhotos}
                  className="absolute inset-0 w-full h-full opacity-0 cursor-pointer z-10"
                />
                <div
                  className={`rounded-xl border-2 border-dashed p-6 text-center transition-colors ${
                    touched.photos && errors.photos
                      ? "border-red-500/50 hover:border-red-500"
                      : "border-[var(--color-border-custom)] hover:border-[#44944A]/50"
                  }`}
                >
                  <Upload className="h-6 w-6 text-[var(--color-text-muted)] mx-auto" />
                  <p className="text-sm text-[var(--color-text-secondary)] mt-1">
                    {t("customOrder.photoHint")}
                  </p>
                </div>
              </div>
              {touched.photos && errors.photos && (
                <p className="text-xs text-red-400 mt-1">{errors.photos}</p>
              )}
            </div>

            {/* Height + Weight */}
            <div className="grid gap-4 sm:grid-cols-2">
              <div>
                <label className="block text-sm font-medium text-[var(--color-text-primary)] mb-2">
                  {t("customOrder.height")}
                </label>
                <input
                  type="text"
                  inputMode="numeric"
                  value={height}
                  onKeyDown={(e) => {
                    if (
                      !/[0-9]/.test(e.key) &&
                      e.key !== "Backspace" &&
                      e.key !== "Tab" &&
                      !e.key.startsWith("Arrow")
                    ) {
                      e.preventDefault();
                    }
                  }}
                  onChange={(e) => {
                    const raw = e.target.value
                      .replace(/[^0-9]/g, "")
                      .slice(0, 3);
                    setHeight(raw);
                    if (raw && Number(raw) > 250) {
                      setHeight("250");
                    }
                  }}
                  onBlur={() => {
                    handleBlur("body");
                    if (height && Number(height) < 100) setHeight("100");
                    if (height && Number(height) > 250) setHeight("250");
                  }}
                  className={inputClass}
                  placeholder="175"
                  maxLength={3}
                />
              </div>
              <div>
                <label className="block text-sm font-medium text-[var(--color-text-primary)] mb-2">
                  {t("customOrder.weight")}
                </label>
                <input
                  type="text"
                  inputMode="numeric"
                  value={weight}
                  onKeyDown={(e) => {
                    if (
                      !/[0-9]/.test(e.key) &&
                      e.key !== "Backspace" &&
                      e.key !== "Tab" &&
                      !e.key.startsWith("Arrow")
                    ) {
                      e.preventDefault();
                    }
                  }}
                  onChange={(e) => {
                    const raw = e.target.value
                      .replace(/[^0-9]/g, "")
                      .slice(0, 3);
                    setWeight(raw);
                    if (raw && Number(raw) > 200) {
                      setWeight("200");
                    }
                  }}
                  onBlur={() => {
                    handleBlur("body");
                    if (weight && Number(weight) < 30) setWeight("30");
                    if (weight && Number(weight) > 200) setWeight("200");
                  }}
                  className={inputClass}
                  placeholder="70"
                  maxLength={3}
                />
              </div>
              {touched.body && errors.body && (
                <p className="text-xs text-red-400 mt-1">{errors.body}</p>
              )}
            </div>

            {/* Phone */}
            <div>
              <div className="flex items-center justify-between mb-2">
                <label className="block text-sm font-medium text-[var(--color-text-primary)]">
                  {t("checkout.phone")}
                </label>
                {user?.phone && user.phone !== phone && (
                  <button
                    type="button"
                    onClick={() => setPhone(user.phone)}
                    className="text-xs font-semibold uppercase tracking-wider text-[#44944A] hover:text-[var(--color-text-primary)] transition-colors"
                    data-testid="custom-order-use-my-phone"
                  >
                    {t("checkout.useMyPhone")}
                  </button>
                )}
              </div>
              <PhoneInput
                value={phone}
                onChange={(full) => {
                  setPhone(full);
                  if (touched.phone && errors.phone) {
                    setErrors((prev) => ({ ...prev, phone: "" }));
                  }
                }}
                placeholder={t("checkout.phonePlaceholder")}
                data-testid="custom-order-phone"
                className={`${inputClass} ${touched.phone && errors.phone ? "!border-red-500" : ""}`}
              />
              {touched.phone && errors.phone && (
                <p className="text-xs text-red-400 mt-1">{errors.phone}</p>
              )}
            </div>

            {/* City */}
            <div>
              <label className="block text-sm font-medium text-[var(--color-text-primary)] mb-2">
                {t("customOrder.city")}
              </label>
              <CityAutocomplete
                value={city}
                onChange={setCity}
                className={`${inputClass} ${touched.city && errors.city ? "!border-red-500" : ""}`}
                placeholder={t("customOrder.cityPlaceholder")}
              />
              {touched.city && errors.city && (
                <p className="text-xs text-red-400 mt-1">{errors.city}</p>
              )}
            </div>

            {/* Delivery time */}
            <div>
              <label className="block text-sm font-medium text-[var(--color-text-primary)] mb-2">
                {t("customOrder.deliveryTime")}
              </label>
              {/* KAN-59: delivery-time options are shown but NOT selectable —
                  the manager quotes the price per tariff. */}
              <div className="space-y-2">
                {deliveryTimeOptions.map((key) => (
                  <label
                    key={key}
                    aria-disabled="true"
                    className="flex items-center gap-3 rounded-xl border px-4 py-3 border-[var(--color-border-custom)] opacity-50 cursor-not-allowed transition-all"
                  >
                    <input
                      type="radio"
                      name="deliveryTime"
                      value={key}
                      checked={deliveryTime === key}
                      onChange={() => setDeliveryTime(key)}
                      disabled
                      className="accent-[#44944A]"
                    />
                    <span className="text-sm text-[var(--color-text-primary)]">
                      {t(`customOrder.deliveryTimeOptions.${key}`)}
                    </span>
                  </label>
                ))}
              </div>
              <p className="mt-3 text-xs text-[var(--color-text-muted)] leading-relaxed">
                {t("customOrder.deliveryTimeInfo")}
              </p>
            </div>

            {/* Delivery method */}
            <div>
              <label className="block text-sm font-medium text-[var(--color-text-primary)] mb-2">
                {t("customOrder.deliveryMethod")}
              </label>
              <div className="space-y-2">
                {deliveryMethods.map((method) => {
                  const disabled = isDeliveryBlocked(method.key, city);
                  return (
                  <label
                    key={method.key}
                    title={
                      disabled
                        ? t("customOrder.delivery.disabledForCity", {
                            method: t(
                              `customOrder.deliveryMethods.${method.key}`,
                            ),
                            city: city || "—",
                          })
                        : undefined
                    }
                    aria-disabled={disabled}
                    data-testid={`delivery-${method.key}`}
                    className={`flex items-start gap-3 rounded-xl border px-4 py-3 transition-all ${
                      disabled
                        ? "opacity-50 cursor-not-allowed border-[var(--color-border-custom)]"
                        : "cursor-pointer " +
                          (deliveryMethod === method.key
                            ? "border-[#44944A] bg-[#44944A]/10"
                            : "border-[var(--color-border-custom)] hover:border-[var(--color-text-muted)]")
                    }`}
                  >
                    <input
                      type="radio"
                      name="deliveryMethod"
                      value={method.key}
                      checked={deliveryMethod === method.key}
                      disabled={disabled}
                      onChange={(e) => {
                        // Hard guard: even if a stale radio somehow
                        // dispatches a change (e.g. devtools
                        // removed the disabled attribute), we don't
                        // let the form accept a blocked combination.
                        if (disabled) return;
                        setDeliveryMethod(e.target.value);
                        setTouched((prev) => ({
                          ...prev,
                          deliveryMethod: true,
                        }));
                      }}
                      className="mt-0.5 accent-[#44944A]"
                    />
                    <div className="flex items-center justify-between w-full">
                      <span className="text-sm text-[var(--color-text-primary)]">
                        {t(`customOrder.deliveryMethods.${method.key}`)}
                      </span>
                      <span
                        className={`text-xs font-medium px-2 py-0.5 rounded-full bg-[var(--color-bg-secondary)] ${method.priceColor} shrink-0 ml-3`}
                      >
                        {method.price}
                      </span>
                    </div>
                  </label>
                  );
                })}
              </div>
              {touched.deliveryMethod && errors.deliveryMethod && (
                <p className="text-xs text-red-400 mt-1">
                  {errors.deliveryMethod}
                </p>
              )}
            </div>

            {/* Comment */}
            <div>
              <label className="block text-sm font-medium text-[var(--color-text-primary)] mb-2">
                {t("customOrder.comment")}
              </label>
              <textarea
                value={comment}
                onChange={(e) => setComment(e.target.value)}
                rows={4}
                className={`${inputClass} resize-none`}
                placeholder={t("customOrder.commentPlaceholder")}
              />
            </div>

            {/* Submit */}
            {submitError && (
              <p data-testid="custom-order-error" className="text-sm text-red-400 text-center">{submitError}</p>
            )}
            {!telegramLinked && (
              <div
                data-testid="custom-order-telegram-required"
                className="rounded-xl border border-amber-400/30 bg-amber-400/10 p-4 text-sm text-amber-400"
              >
                {t("checkout.telegramRequired")}{" "}
                <button
                  type="button"
                  onClick={() => navigate("/profile?redirect=/custom-order")}
                  className="font-semibold underline underline-offset-2 hover:text-amber-300"
                >
                  {t("checkout.telegramRequiredCta")}
                </button>
              </div>
            )}
            <button
              type="submit"
              data-testid="custom-order-submit"
              disabled={!canSubmit || submitting || !telegramLinked}
              className="w-full flex items-center justify-center gap-2 rounded-xl bg-[#44944A] px-6 py-4 text-sm font-semibold text-black transition-all hover:shadow-[0_0_30px_rgba(68,148,74,0.3)] disabled:opacity-40 disabled:cursor-not-allowed disabled:hover:shadow-none"
            >
              <Send className="h-4 w-4" />
              {submitting ? "..." : t("customOrder.submit")}
            </button>

            <p className="text-xs text-[var(--color-text-muted)] text-center leading-relaxed">
              {t("customOrder.afterSubmit")}
            </p>
          </motion.form>
        ) : (
          <motion.div
            initial={{ opacity: 0, scale: 0.95 }}
            animate={{ opacity: 1, scale: 1 }}
            className="mt-10 rounded-2xl bg-[var(--color-bg-card)] border border-[var(--color-border-custom)] p-10 text-center"
          >
            <div className="w-16 h-16 mx-auto mb-6 rounded-full bg-[#44944A]/10 flex items-center justify-center">
              <Send className="h-8 w-8 text-[#44944A]" />
            </div>
            <h3 className="text-xl font-bold text-[var(--color-text-primary)]">
              Заявка отправлена!
            </h3>
            <p className="mt-3 text-sm text-[var(--color-text-secondary)] max-w-md mx-auto">
              {t("customOrder.afterSubmit")}
            </p>
            <Link
              to="/catalog"
              className="inline-flex items-center gap-2 mt-8 rounded-xl bg-[#44944A] px-6 py-3 text-sm font-semibold text-black transition-all hover:shadow-[0_0_30px_rgba(68,148,74,0.3)]"
            >
              Вернуться в каталог
            </Link>
          </motion.div>
        )}
      </div>
    </div>
  );
}
