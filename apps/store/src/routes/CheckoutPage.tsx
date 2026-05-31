import { useState, useEffect } from "react";
import { motion, AnimatePresence } from "framer-motion";
import { useNavigate } from "react-router-dom";
import { useCartStore } from "@/stores/cartStore";
import { useAuthStore } from "@/stores/authStore";
import { CreditCard, Truck, Check, ChevronRight, Package } from "lucide-react";
import { useTranslation } from "react-i18next";
import { Helmet } from "react-helmet-async";

export default function CheckoutPage() {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const isAuthenticated = useAuthStore((s) => s.isAuthenticated);
  const [currentStep, setCurrentStep] = useState(1);
  const items = useCartStore((state) => state.items);
  const totalPrice = useCartStore((state) => state.totalPrice());

  // Guard — redirect to auth if not logged in
  useEffect(() => {
    if (!isAuthenticated) {
      navigate("/profile?redirect=/checkout", { replace: true });
    }
  }, [isAuthenticated, navigate]);

  const [formData, setFormData] = useState({
    name: "",
    phone: "",
    email: "",
    city: "",
    street: "",
    house: "",
    apartment: "",
    paymentMethod: "card" as "card" | "cod" | "sbp",
  });

  const steps = [
    { id: 1, label: t("checkout.steps.contacts"), icon: Truck },
    { id: 2, label: t("checkout.steps.address"), icon: Package },
    { id: 3, label: t("checkout.steps.payment"), icon: CreditCard },
    { id: 4, label: t("checkout.steps.confirmation"), icon: Check },
  ];

  const updateField = (field: string, value: string) => {
    setFormData((prev) => ({ ...prev, [field]: value }));
  };

  const canProceed = () => {
    switch (currentStep) {
      case 1:
        return formData.name && formData.phone && formData.email;
      case 2:
        return formData.city && formData.street && formData.house;
      case 3:
        return formData.paymentMethod;
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
            {[
              {
                label: t("checkout.fields.name"),
                field: "name",
                placeholder: "Иван Иванов",
                type: "text",
              },
              {
                label: t("checkout.fields.phone"),
                field: "phone",
                placeholder: "+7 (999) 000-00-00",
                type: "tel",
              },
              {
                label: t("checkout.fields.email"),
                field: "email",
                placeholder: "ivan@example.com",
                type: "email",
              },
            ].map((input) => (
              <div key={input.field}>
                <label className="block text-sm font-medium text-[var(--color-text-primary)] mb-2">
                  {input.label}
                </label>
                <input
                  type={input.type}
                  value={
                    formData[input.field as keyof typeof formData] as string
                  }
                  onChange={(e) => updateField(input.field, e.target.value)}
                  className={inputBaseClass}
                  placeholder={input.placeholder}
                />
              </div>
            ))}
          </div>
        );
      case 2:
        return (
          <div className="space-y-4">
            <div>
              <label className="block text-sm font-medium text-[var(--color-text-primary)] mb-2">
                {t("checkout.fields.city")}
              </label>
              <input
                type="text"
                value={formData.city}
                onChange={(e) => updateField("city", e.target.value)}
                className={inputBaseClass}
                placeholder="Москва"
              />
            </div>
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
          </div>
        );
      case 3:
        return (
          <div className="space-y-3">
            {[
              {
                id: "card",
                label: t("checkout.payment.card"),
                desc: t("checkout.payment.cardDesc"),
              },
              {
                id: "sbp",
                label: t("checkout.payment.sbp"),
                desc: t("checkout.payment.sbpDesc"),
              },
              {
                id: "cod",
                label: t("checkout.payment.cod"),
                desc: t("checkout.payment.codDesc"),
              },
            ].map((method) => (
              <button
                key={method.id}
                onClick={() => updateField("paymentMethod", method.id)}
                className={`w-full rounded-xl border p-5 text-left transition-all relative ${
                  formData.paymentMethod === method.id
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
            ))}
          </div>
        );
      case 4:
        return (
          <div className="space-y-6">
            <div className="rounded-2xl bg-[var(--color-bg-card)] border border-[var(--color-border-custom)] p-6">
              <h3 className="text-xs font-semibold uppercase tracking-[0.2em] text-[var(--color-text-muted)] mb-4">
                {t("checkout.orderSummary")}
              </h3>
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
                      {(item.product.price * item.quantity).toLocaleString(
                        "ru-RU",
                      )}{" "}
                      ₽
                    </span>
                  </div>
                ))}
                <div className="border-t border-[var(--color-border-custom)] pt-3 flex justify-between">
                  <span className="font-semibold text-[var(--color-text-primary)]">
                    {t("checkout.total")}
                  </span>
                  <span className="font-bold text-[#44944A]">
                    {totalPrice.toLocaleString("ru-RU")} ₽
                  </span>
                </div>
              </div>
            </div>
            <button
              onClick={() => alert("Заказ оформлен! (Mock)")}
              className="w-full rounded-xl bg-[#44944A] px-6 py-4 text-sm font-semibold text-black transition-all hover:shadow-[0_0_30px_rgba(192,254,57,0.3)]"
            >
              {t("checkout.confirm")}
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
          {currentStep < 4 && (
            <button
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
