import { Helmet } from "react-helmet-async";
import { motion } from "framer-motion";
import { MessageCircle, Camera, Mail } from "lucide-react";

const contacts = [
  {
    icon: MessageCircle,
    label: "Telegram",
    value: "@miorumanager",
    href: "https://t.me/miorumanager",
    color: "text-blue-400",
    bg: "bg-blue-400/10",
  },
  {
    icon: Camera,
    label: "Instagram",
    value: "@mioru.store",
    href: "https://instagram.com/mioru.store",
    color: "text-pink-400",
    bg: "bg-pink-400/10",
  },
  {
    icon: Mail,
    label: "Email",
    value: "support@mioru.store",
    href: "mailto:support@mioru.store",
    color: "text-[#44944A]",
    bg: "bg-[#44944A]/10",
  },
];

export default function ContactsPage() {
  return (
    <div className="px-6 py-24 lg:px-8">
      <Helmet>
        <title>Контакты — MIORU</title>
        <meta
          name="description"
          content="Свяжитесь с MIORU через Telegram, Instagram или по почте."
        />
        <link rel="canonical" href="https://mioru.store/contacts" />
      </Helmet>

      <div className="mx-auto max-w-lg">
        <motion.h1
          initial={{ opacity: 0, y: 20 }}
          animate={{ opacity: 1, y: 0 }}
          className="text-4xl font-bold tracking-tighter text-[var(--color-text-primary)] sm:text-5xl mb-2"
        >
          Контакты
        </motion.h1>
        <motion.p
          initial={{ opacity: 0, y: 20 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ delay: 0.05 }}
          className="text-[var(--color-text-secondary)] mb-10"
        >
          Выберите удобный способ связи
        </motion.p>

        <motion.div
          initial={{ opacity: 0, y: 20 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ delay: 0.1 }}
          className="space-y-3"
        >
          {contacts.map((c) => (
            <a
              key={c.label}
              href={c.href}
              target="_blank"
              rel="noopener noreferrer"
              className="flex items-center gap-4 rounded-xl border border-[var(--color-border-custom)] p-4 hover:border-[#44944A]/50 transition-all hover:bg-[var(--color-bg-secondary)] group"
            >
              <div
                className={`w-10 h-10 rounded-full ${c.bg} flex items-center justify-center shrink-0`}
              >
                <c.icon className={`h-5 w-5 ${c.color}`} />
              </div>
              <div className="flex-1">
                <p className="text-sm font-medium text-[var(--color-text-primary)]">
                  {c.label}
                </p>
                <p className="text-sm text-[var(--color-text-secondary)]">
                  {c.value}
                </p>
              </div>
              <span className="text-xs text-[var(--color-text-muted)] group-hover:text-[#44944A] transition-colors">
                →
              </span>
            </a>
          ))}
        </motion.div>
      </div>
    </div>
  );
}
