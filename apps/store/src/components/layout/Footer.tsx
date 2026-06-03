import { Link } from "react-router-dom";
import { useTranslation } from "react-i18next";

function useTheme(): "dark" | "light" {
  if (typeof window === "undefined") return "dark";
  return (localStorage.getItem("theme") as "dark" | "light") || "dark";
}

export default function Footer() {
  const { t } = useTranslation();
  const isLight = useTheme() === "light";

  const navLinks = [
    { labelKey: "nav.inStock", href: "/catalog" },
    { labelKey: "nav.customOrder", href: "/custom-order" },
    { labelKey: "nav.avatar", href: "/avatar" },
    { labelKey: "nav.cart", href: "/cart" },
  ];

  return (
    <footer className="relative border-t border-[var(--color-border-custom)] bg-[var(--color-bg-primary)] pb-[env(safe-area-inset-bottom)]">
      <div className="mx-auto max-w-7xl px-6 py-16 lg:px-8">
        <div className="grid gap-12 sm:grid-cols-2 lg:grid-cols-3">
          <div className="sm:col-span-2 lg:col-span-1">
            <Link to="/" className="inline-block">
              <img src={isLight ? "/favicon-black.ico" : "/favicon.ico"} alt="MIORU" className="h-10 w-10" />
            </Link>
            <p className="mt-4 text-sm leading-relaxed text-[var(--color-text-muted)]">
              {t("footer.description")}
            </p>
          </div>

          <div>
            <h3 className="text-xs font-semibold uppercase tracking-[0.2em] text-[var(--color-text-muted)]">
              {t("footer.navigation")}
            </h3>
            <ul className="mt-6 space-y-1">
              {navLinks.map((item) => (
                <li key={item.labelKey}>
                  <Link
                    to={item.href}
                    className="inline-block py-2 text-sm text-[var(--color-text-secondary)] transition-colors hover:text-[#44944A]"
                  >
                    {t(item.labelKey)}
                  </Link>
                </li>
              ))}
            </ul>
          </div>

          <div>
            <h3 className="text-xs font-semibold uppercase tracking-[0.2em] text-[var(--color-text-muted)]">
              {t("footer.contacts")}
            </h3>
            <ul className="mt-6 space-y-3">
              <li>
                <a
                  href="https://t.me/miorumanager"
                  target="_blank"
                  rel="noopener noreferrer"
                  className="text-sm text-[var(--color-text-secondary)] transition-colors hover:text-[#44944A]"
                >
                  Telegram: @miorumanager
                </a>
              </li>
              <li>
                <a
                  href="https://instagram.com/mioru.store"
                  target="_blank"
                  rel="noopener noreferrer"
                  className="text-sm text-[var(--color-text-secondary)] transition-colors hover:text-[#44944A]"
                >
                  Instagram: @mioru.store
                </a>
              </li>
              <li className="text-sm text-[var(--color-text-secondary)]">
                support@mioru.store
              </li>
            </ul>
          </div>
        </div>

        <div className="mt-16 flex flex-col items-center justify-between gap-4 border-t border-[var(--color-border-custom)] pt-8 sm:flex-row">
          <p className="text-xs text-[var(--color-text-muted)]">
            {t("footer.copyright")}
          </p>
          <p className="text-xs text-[var(--color-text-muted)]">
            {t("footer.rights")}
          </p>
        </div>
      </div>
    </footer>
  );
}
