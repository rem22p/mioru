import { useState, useEffect } from "react";
import { motion, AnimatePresence } from "framer-motion";
import { useNavigate } from "react-router-dom";
import { useCartStore } from "@/stores/cartStore";
import { useAuthStore } from "@/stores/authStore";
import { createOrder } from "@/lib/api";
import { useCurrencyStore } from "@/stores/currencyStore";
import { formatPrice } from "@/lib/currency";
import { CreditCard, Check, ChevronRight, Package } from "lucide-react";
import { useTranslation } from "react-i18next";
import { isDeliveryBlocked as isMethodBlocked } from "@/lib/deliveryRules";
import { isValidPhone } from "@/lib/phoneValidation";
import { Helmet } from "@dr.pogodin/react-helmet";
import CityAutocomplete from "@/components/ui/CityAutocomplete";

const deliveryMethods = [
  { key: "personal", priceFree: true, priceColor: "text-[#44944A]" },
  { key: "address", priceLabel: "checkout.delivery.price25", priceColor: "text-[var(--color-text-secondary)]" },
  { key: "bus", priceLabel: "checkout.delivery.priceUpTo50", priceColor: "text-[var(--color-text-secondary)]" },
  { key: "express", priceLabel: "checkout.delivery.priceUpTo50", priceColor: "text-[var(--color-text-secondary)]" },
  { key: "moldovaPost", priceLabel: "checkout.delivery.priceUpTo50", priceColor: "text-[var(--color-text-secondary)]" },
];


// Delivery methods blocked in PMR cities
// Delivery-method city rules live in lib/deliveryRules.ts so both
// CheckoutPage (cart) and CustomOrderPage (individual) share a
// single source of truth — the rule comment block used to live here
// and the same logic got copy-pasted with subtle drift between the
// two pages. Any change now goes through isDeliveryBlocked().

export default function CheckoutPage() {
  const { t } = useTranslation();
  const currency = useCurrencyStore((s) => s.currency);
  const navigate = useNavigate();
  const isAuthenticated = useAuthStore((s) => s.isAuthenticated);
  const user = useAuthStore((s) => s.user);
  const [currentStep, setCurrentStep] = useState(1);
  const items = useCartStore((state) => state.items);
  const totalPrice = useCartStore((state) => state.totalPrice());
  const clearCart = useCartStore((state) => state.clearCart);

  useEffect(() => {
    if (!isAuthenticated) {
      navigate("/profile?redirect=/checkout", { replace: true });
    }
  }, [isAuthenticated, navigate]);

  const [submitted, setSubmitted] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const [orderError, setOrderError] = useState("");

  const submitOrder = async () => {
    setSubmitting(true);
    setOrderError("");
    try {
      const idempotencyKey = crypto.randomUUID();
      const itemsData = items.map((item) => ({
        product_id: item.product.id,
        size_label: item.size,
        quantity: item.quantity,
        price_minor: item.product.price * 100, // MDL → bani
      }));
      await createOrder(
        {
          type: "cart",
          // phone + city are trimmed here because the client-side
          // gate (isValidPhone / normaliseCity) trims before
          // validation, but the backend phoneRE does NOT trim
          // (^\+?\d{7,15}$). Raw "  +373...  " from a paste/IME
          // passes the gate but would 400 on the server — see
          // PR #51 round-3 review (mmx003) for the full discussion.
          // CustomOrderPage.tsx:140 applies the same trim for parity.
          city: formData.city.trim(),
          phone: formData.phone.trim(),
          delivery_method: formData.deliveryMethod,
          payment_method: formData.paymentMethod,
          street: formData.street || undefined,
          house: formData.house || undefined,
          apartment: formData.apartment || undefined,
          total_minor: totalPrice * 100,
          items: itemsData,
        },
        idempotencyKey,
      );
      clearCart();
      setSubmitted(true);
    } catch (e: unknown) {
      setOrderError(e instanceof Error ? e.message : "Ошибка при создании заказа");
    } finally {
      setSubmitting(false);
    }
  };

  const [formData, setFormData] = useState({
    phone: "",
    city: "",
    street: "",
    house: "",
    apartment: "",
    deliveryMethod: "" as string,
    paymentMethod: "card" as "card" | "cod",
  });

  const steps = [
    { id: 1, label: t("checkout.steps.address"), icon: Package },
    { id: 2, label: t("checkout.steps.payment"), icon: CreditCard },
    { id: 3, label: t("checkout.steps.confirmation"), icon: Check },
  ];

  // isPaymentBlocked: "Оплата при получении" (cod) is physically
  // impossible for the bus (маршрутка) delivery method — there's no
  // courier to hand the money to, so a frontend user could only be
  // confused if the option stayed enabled. Card payment is always
  // available regardless of delivery method.
  //
  // We return the same shape as isMethodBlocked so the JSX below
  // can apply the same disabled style, tooltip and useEffect-driven
  // auto-reset without duplicating the visual logic.
  const isPaymentBlocked = (
    paymentId: "card" | "cod",
    deliveryMethod: string,
  ): boolean => {
    if (paymentId === "card") return false;
    if (paymentId === "cod") return deliveryMethod === "bus";
    return false;
  };

  const updateField = (field: string, value: string) => {
    setFormData((prev) => ({ ...prev, [field]: value }));
  };

  // Reset delivery method if blocked by city change
  useEffect(() => {
    if (formData.deliveryMethod && isMethodBlocked(formData.deliveryMethod, formData.city)) {
      updateField("deliveryMethod", "");
    }
  }, [formData.city]);

  // Reset payment method if it became blocked by a delivery method
  // change (e.g. user picked "cod" for personal delivery, then
  // switched the city to one that only allows "bus" — at that
  // moment "cod" is no longer valid and must fall back to "card").
  // Without this reset the user would proceed to step 3 with a
  // payment choice the server would then 400 on.
  useEffect(() => {
    if (
      formData.deliveryMethod &&
      formData.paymentMethod &&
      isPaymentBlocked(formData.paymentMethod as "card" | "cod", formData.deliveryMethod)
    ) {
      updateField("paymentMethod", "card");
    }
  }, [formData.deliveryMethod]);

  const canProceed = () => {
    switch (currentStep) {
      case 1:
        if (!formData.phone || !formData.city || !formData.deliveryMethod) return false;
        // Phone must look like +<digits> (7-15 digits, optional leading +).
        // Mirror of backend `phoneRE`; see apps/store/src/lib/phoneValidation.ts.
        // The backend re-validates, but we block submit early so the user
        // doesn't bounce on the API round-trip.
        if (!isValidPhone(formData.phone)) return false;
        if (formData.deliveryMethod === "address") {
          return formData.street !== "" && formData.house !== "";
        }
        return true;
      case 2:
        return true;
      default:
        return true;
    }
  };

  const inputBaseClass =
    "w-full rounded-xl bg-[var(--color-bg-card)] border border-[var(--color-border-custom)] px-4 py-3 text-base sm:text-sm text-[var(--color-text-primary)] outline-none focus:border-[#44944A] transition-colors";

  const renderStepContent = () => {
    switch (currentStep) {
      case 1:
        return (
          <div className="space-y-4">
            <div>
              <div className="flex items-center justify-between mb-2">
                <label className="block text-sm font-medium text-[var(--color-text-primary)]">
                  {t("checkout.phone")}
                </label>
                {isAuthenticated && user?.phone && user.phone !== formData.phone && (
                  <button
                    type="button"
                    onClick={() => updateField("phone", user.phone)}
                    className="text-xs font-semibold uppercase tracking-wider text-[#44944A] hover:text-[var(--color-text-primary)] transition-colors"
                    data-testid="checkout-use-my-phone"
                  >
                    {t("checkout.useMyPhone")}
                  </button>
                )}
              </div>
              <input
                type="tel"
                inputMode="tel"
                autoComplete="tel"
                value={formData.phone}
                onChange={(e) => updateField("phone", e.target.value)}
                placeholder={t("checkout.phonePlaceholder")}
                data-testid="checkout-phone"
                className={inputBaseClass}
              />
            </div>
            <div>
              <label className="block text-sm font-medium text-[var(--color-text-primary)] mb-2">
                {t("checkout.fields.city")}
              </label>
              <CityAutocomplete
                value={formData.city}
                onChange={(v) => updateField("city", v)}
                className={inputBaseClass}
                testId="checkout-city"
              />
            </div>

            <div>
              <label className="block text-sm font-medium text-[var(--color-text-primary)] mb-2">
                {t("checkout.delivery.title")}
              </label>
              <div className="space-y-2">
                {deliveryMethods.map((method) => {
                  const disabled = isMethodBlocked(method.key, formData.city);
                  return (
                  <label
                    key={method.key}
                    className={`flex items-start gap-3 rounded-xl border px-4 py-3 transition-all ${
                      disabled
                        ? "opacity-30 cursor-not-allowed"
                        : "cursor-pointer " + (formData.deliveryMethod === method.key
                          ? "border-[#44944A] bg-[#44944A]/10"
                          : "border-[var(--color-border-custom)] hover:border-[var(--color-text-muted)]")
                    }`}
                  >
                    <input
                      type="radio"
                      data-testid={`checkout-delivery-${method.key}`}
                      name="deliveryMethod"
                      value={method.key}
                      checked={formData.deliveryMethod === method.key}
                      onChange={(e) => updateField("deliveryMethod", e.target.value)}
                      disabled={disabled}
                      className="mt-0.5 accent-[#44944A]"
                    />
                    <div className="flex items-center justify-between w-full">
                      <span className="text-sm text-[var(--color-text-primary)]">
                        {t(`checkout.delivery.${method.key}`)}
                      </span>
                      <span className={`text-xs font-medium px-2 py-0.5 rounded-full bg-[var(--color-bg-secondary)] ${method.priceColor} shrink-0`}>
                        {method.priceFree ? t("checkout.delivery.free") : t(method.priceLabel!)}
                      </span>
                    </div>
                  </label>
                  );
                })}
              </div>
            </div>

            {formData.deliveryMethod === "address" && (
              <>
                <div>
                  <label className="block text-sm font-medium text-[var(--color-text-primary)] mb-2">
                    {t("checkout.fields.street")}
                  </label>
                  <input
                    type="text"
                    value={formData.street}
                    onChange={(e) => updateField("street", e.target.value)}
                    className={inputBaseClass}
                    placeholder="Тверская"
                  />
                </div>
                <div className="grid grid-cols-2 gap-4">
                  <div>
                    <label className="block text-sm font-medium text-[var(--color-text-primary)] mb-2">
                      {t("checkout.fields.house")}
                    </label>
                    <input
                      type="text"
                      inputMode="numeric"
                      value={formData.house}
                      onChange={(e) => updateField("house", e.target.value)}
                      className={inputBaseClass}
                      placeholder="12"
                    />
                  </div>
                  <div>
                    <label className="block text-sm font-medium text-[var(--color-text-primary)] mb-2">
                      {t("checkout.fields.apartment")}
                    </label>
                    <input
                      type="text"
                      inputMode="numeric"
                      value={formData.apartment}
                      onChange={(e) => updateField("apartment", e.target.value)}
                      className={inputBaseClass}
                      placeholder="45"
                    />
                  </div>
                </div>
              </>
            )}
          </div>
        );
      case 2:
        return (
          <div className="space-y-3">
            {[
              {
                id: "card",
                label: t("checkout.payment.card"),
                desc: t("checkout.payment.cardDesc"),
              },
              {
                id: "cod",
                label: t("checkout.payment.cod"),
                desc: t("checkout.payment.codDesc"),
              },
            ].map((method) => {
              const disabled = isPaymentBlocked(
                method.id as "card" | "cod",
                formData.deliveryMethod,
              );
              return (
              <button
                key={method.id}
                onClick={() => {
                  // Hard guard: even if a stale radio somehow
                  // dispatches a click (e.g. devtools removed the
                  // disabled attribute), we don't let the form
                  // accept a blocked combination. The server still
                  // re-validates, but the client guard means a
                  // confused user never sees a 400 they could have
                  // been prevented from triggering.
                  if (disabled) return;
                  updateField("paymentMethod", method.id);
                }}
                disabled={disabled}
                aria-disabled={disabled}
                title={
                  disabled
                    ? t("checkout.payment.codDisabled", {
                        method: t(
                          `checkout.delivery.${formData.deliveryMethod}`,
                        ),
                      })
                    : undefined
                }
                data-testid={`payment-${method.id}`}
                className={`w-full rounded-xl border p-5 text-left transition-all relative ${
                  disabled
                    ? "opacity-50 cursor-not-allowed border-[var(--color-border-custom)] bg-[var(--color-bg-card)]"
                    : formData.paymentMethod === method.id
                      ? "border-[#44944A] bg-[#44944A]/10"
                      : "border-[var(--color-border-custom)] bg-[var(--color-bg-card)] hover:border-[var(--color-text-muted)]"
                }`}
              >
                <div className="flex items-center gap-3">
                  <div
                    className={`w-5 h-5 rounded-full border-2 flex items-center justify-center shrink-0 ${
                      formData.paymentMethod === method.id
                        ? "border-[#44944A]"
                        : "border-[var(--color-text-muted)]"
                    }`}
                  >
                    {formData.paymentMethod === method.id && (
                      <div className="w-2.5 h-2.5 rounded-full bg-[#44944A]" />
                    )}
                  </div>
                  <div>
                    <div className="font-medium text-[var(--color-text-primary)]">{method.label}</div>
                    <div className="text-xs text-[var(--color-text-muted)] mt-1">
                      {method.desc}
                    </div>
                  </div>
                </div>
              </button>
              );
            })}
          </div>
        );
      case 3:
        if (submitted) {
          return (
            <motion.div
              data-testid="checkout-success"
              initial={{ opacity: 0, scale: 0.95 }}
              animate={{ opacity: 1, scale: 1 }}
              className="rounded-2xl bg-[var(--color-bg-card)] border border-[var(--color-border-custom)] p-8 text-center"
            >
              <div className="w-16 h-16 mx-auto mb-6 rounded-full bg-[#44944A]/10 flex items-center justify-center">
                <Check className="h-8 w-8 text-[#44944A]" />
              </div>
              <h3 className="text-xl font-bold text-[var(--color-text-primary)]">
                {t("checkout.confirmation.title")}
              </h3>
              <p className="mt-3 text-sm text-[var(--color-text-secondary)] max-w-md mx-auto leading-relaxed">
                {t("checkout.confirmation.message")}
              </p>
            </motion.div>
          );
        }
        return (
          <div className="space-y-6">
            <div className="rounded-2xl bg-[var(--color-bg-card)] border border-[var(--color-border-custom)] p-6">
              <h3 className="text-xs font-semibold uppercase tracking-[0.2em] text-[var(--color-text-muted)] mb-4">
                {t("checkout.orderSummary")}
              </h3>

              {/* City, delivery, payment info */}
              <div className="space-y-2 mb-4 pb-4 border-b border-[var(--color-border-custom)]">
                <div className="flex justify-between text-sm">
                  <span className="text-[var(--color-text-muted)]">{t("checkout.summary.city")}</span>
                  <span className="text-[var(--color-text-primary)]">{formData.city}</span>
                </div>
                <div className="flex justify-between text-sm">
                  <span className="text-[var(--color-text-muted)]">{t("checkout.summary.delivery")}</span>
                  <span className="text-[var(--color-text-primary)]">{formData.deliveryMethod ? t(`checkout.delivery.${formData.deliveryMethod}`) : ""}</span>
                </div>
                {formData.deliveryMethod === "address" && (
                  <div className="flex justify-between text-sm">
                    <span className="text-[var(--color-text-muted)]">{t("checkout.summary.address")}</span>
                    <span className="text-[var(--color-text-primary)]">{formData.street}, {formData.house}{formData.apartment ? `, ${t("checkout.summary.apartment")} ${formData.apartment}` : ""}</span>
                  </div>
                )}
                <div className="flex justify-between text-sm">
                  <span className="text-[var(--color-text-muted)]">{t("checkout.summary.payment")}</span>
                  <span className="text-[var(--color-text-primary)]">{t(`checkout.payment.${formData.paymentMethod}`)}</span>
                </div>
              </div>

              {/* Order items */}
              <div className="space-y-3">
                {items.map((item) => (
                  <div
                    key={`${item.product.id}-${item.size}`}
                    className="flex justify-between text-sm"
                  >
                    <span className="text-[var(--color-text-secondary)]">
                      {item.product.name} × {item.quantity} ({item.size})
                    </span>
                    <span className="text-[var(--color-text-primary)]">
                      {formatPrice(item.product.price * item.quantity, currency)}
                    </span>
                  </div>
                ))}
                <div className="border-t border-[var(--color-border-custom)] pt-3 flex justify-between">
                  <span className="font-semibold text-[var(--color-text-primary)]">
                    {t("checkout.total")}
                  </span>
                  <span className="font-bold text-[#44944A]">
                    {formatPrice(totalPrice, currency)}
                  </span>
                </div>
              </div>
            </div>
            {orderError && (
              <p data-testid="checkout-error" className="text-sm text-red-400 text-center">{orderError}</p>
            )}
            <button
              data-testid="checkout-confirm"
              onClick={submitOrder}
              disabled={submitting}
              className="w-full rounded-xl bg-[#44944A] px-6 py-4 text-sm font-semibold text-black transition-all hover:shadow-[0_0_30px_rgba(192,254,57,0.3)] disabled:opacity-50 disabled:cursor-not-allowed"
            >
              {submitting ? "..." : t("checkout.confirm")}
            </button>
          </div>
        );
    }
  };

  return (
    <div className="px-6 py-24 lg:px-8">
      <Helmet>
        <title>Оформление заказа — MIORU</title>
        <meta
          name="description"
          content="Оформите заказ в MIORU. Быстрая доставка, безопасная оплата картой, СБП или при получении."
        />
        <meta property="og:title" content="Оформление заказа — MIORU" />
        <link rel="canonical" href="https://mioru.store/checkout" />
      </Helmet>
      <div className="mx-auto max-w-2xl">
        <motion.h1
          initial={{ opacity: 0, y: 20 }}
          animate={{ opacity: 1, y: 0 }}
          className="text-4xl font-bold tracking-tighter text-[var(--color-text-primary)] sm:text-5xl"
        >
          {t("checkout.title")}
        </motion.h1>

        {/* Stepper */}
        <div className="mt-12 flex flex-wrap items-center justify-between gap-2">
          {steps.map((step, index) => (
            <div key={step.id} className="flex items-center">
              <div
                className={`flex h-10 w-10 items-center justify-center rounded-full border transition-all ${
                  currentStep >= step.id
                    ? "border-[#44944A] bg-[#44944A] text-black"
                    : "border-[var(--color-border-custom)] bg-[var(--color-bg-card)] text-[var(--color-text-muted)]"
                }`}
              >
                <step.icon className="h-4 w-4" />
              </div>
              {index < steps.length - 1 && (
                <ChevronRight className="mx-2 h-4 w-4 text-[var(--color-border-light)] hidden sm:block" />
              )}
            </div>
          ))}
        </div>

        <AnimatePresence mode="wait">
          <motion.div
            key={currentStep}
            initial={{ opacity: 0, x: 20 }}
            animate={{ opacity: 1, x: 0 }}
            exit={{ opacity: 0, x: -20 }}
            transition={{ duration: 0.3 }}
            className="mt-10"
          >
            {renderStepContent()}
          </motion.div>
        </AnimatePresence>

        <div className="mt-10 flex justify-between">
          {currentStep > 1 && (
            <button
              onClick={() => setCurrentStep(currentStep - 1)}
              className="rounded-xl border border-[var(--color-border-custom)] px-6 py-3 text-sm text-[var(--color-text-primary)] transition-all hover:bg-[var(--color-bg-card)]"
            >
              {t("checkout.back")}
            </button>
          )}
          {currentStep < 3 && (
            <button
              data-testid="checkout-next"
              onClick={() => setCurrentStep(currentStep + 1)}
              disabled={!canProceed()}
              className="ml-auto rounded-xl bg-[#44944A] px-6 py-3 text-sm font-semibold text-black transition-all hover:shadow-[0_0_30px_rgba(192,254,57,0.3)] disabled:opacity-50 disabled:cursor-not-allowed"
            >
              {t("checkout.next")}
            </button>
          )}
        </div>
      </div>
    </div>
  );
}
