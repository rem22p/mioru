import { useState, useEffect } from "react";
import { Link } from "react-router-dom";
import { useTranslation } from "react-i18next";
import { Menu, X, Sun, Moon, User, ShoppingBag } from "lucide-react";
import { useCartStore } from "@/stores/cartStore";
import { LanguageToggle, CurrencyToggle } from "./PreferenceToggles";
import { motion, AnimatePresence } from "framer-motion";

// Desktop nav and the burger-menu links are the same set (KAN-56 removed the
// avatar/cart divergence). One source of truth — desktop/mobile differ only by
// the responsive classes on their <nav> wrappers, not by the link set.
const navLinks = [
  { href: "/catalog", labelKey: "nav.inStock" },
  { href: "/custom-order", labelKey: "nav.customOrder" },
  { href: "/favorites", labelKey: "nav.favorites" },
];

interface HeaderProps {
  theme: "dark" | "light";
  toggleTheme: () => void;
}

export default function Header({
  theme,
  toggleTheme,
}: HeaderProps) {
  const { t } = useTranslation();
  const [scrolled, setScrolled] = useState(false);
  const [menuOpen, setMenuOpen] = useState(false);
  const totalItems = useCartStore((state) => state.totalItems());

  useEffect(() => {
    const handleScroll = () => setScrolled(window.scrollY > 50);
    window.addEventListener("scroll", handleScroll, { passive: true });
    return () => window.removeEventListener("scroll", handleScroll);
  }, []);

  useEffect(() => {
    if (menuOpen) {
      document.body.style.overflow = "hidden";
    } else {
      document.body.style.overflow = "";
    }
    return () => {
      document.body.style.overflow = "";
    };
  }, [menuOpen]);

  const isLight = theme === "light";

  return (
    <>
      <header
        className={`fixed top-0 left-0 right-0 z-50 transition-all duration-500 pt-[env(safe-area-inset-top)] ${
          scrolled
            ? isLight
              ? "bg-white/90 backdrop-blur-md border-b border-gray-200"
              : "bg-[var(--color-bg-primary)]/90 backdrop-blur-md border-b border-[var(--color-border-custom)]"
            : "bg-transparent"
        }`}
      >
        <div className="mx-auto flex h-20 max-w-7xl items-center justify-between px-6 lg:px-8">
          {/* Logo — shrink-0 so it never collapses when the row is tight */}
          <Link to="/" className="shrink-0">
            <motion.img
              src={isLight ? "/favicon-black.ico" : "/favicon.ico"}
              alt="MIORU"
              className="h-10 w-10"
              whileHover={{ scale: 1.1 }}
            />
          </Link>

          {/* Desktop Nav — in-flow (not absolute-centered). Dead-centering the
              nav is incompatible with the wide preference-toggle cluster the
              manager asked for (KAN-56): at ≤1280 a centered nav overlaps the
              cluster. In-flow justify-between makes overlap impossible; the nav
              sits left-of-center. Shown only at lg+ so the 768–1023 band is
              served by the burger menu (pills live at the bottom of that
              overlay), avoiding the logo-collapse/profile-off-screen regression. */}
          <nav className="hidden lg:flex items-center gap-8 lg:gap-10">
            {navLinks.map((link) => (
              <Link
                key={link.href}
                to={link.href}
                className={`group relative text-sm font-bold tracking-wider transition-colors ${
                  isLight
                    ? "text-gray-500 hover:text-gray-900"
                    : "text-[var(--color-text-secondary)] hover:text-[var(--color-text-primary)]"
                }`}
              >
                {t(link.labelKey)}
                <span className="absolute -bottom-1 left-0 h-px w-0 bg-[#44944A] transition-all duration-300 group-hover:w-full" />
              </Link>
            ))}
          </nav>

          {/* Right-side icons — shrink-0 so the cluster never collapses */}
          <div className="flex items-center gap-1 shrink-0">
            {/* Cart — always visible (KAN-56: replaces the globe on mobile) */}
            <Link
              to="/cart"
              className="h-11 w-11 flex items-center justify-center transition-colors rounded-lg relative text-[var(--color-text-secondary)] hover:text-[var(--color-text-primary)]"
              aria-label={t("nav.cart")}
            >
              <ShoppingBag className="h-5 w-5" />
              {totalItems > 0 && (
                <span className="absolute -top-0.5 -right-0.5 flex h-4 w-4 items-center justify-center rounded-full bg-[#44944A] text-[10px] font-bold text-black">
                  {totalItems > 9 ? "9+" : totalItems}
                </span>
              )}
            </Link>

            {/* Language & currency pill toggles — desktop inline (lg+) */}
            <div className="hidden lg:flex items-center gap-2 px-2">
              <LanguageToggle />
              <CurrencyToggle />
            </div>

            {/* Theme Toggle */}
            <button
              onClick={toggleTheme}
              className={`h-11 w-11 flex items-center justify-center transition-colors rounded-lg ${
                isLight
                  ? "text-gray-500 hover:text-gray-900"
                  : "text-[var(--color-text-secondary)] hover:text-[var(--color-text-primary)]"
              }`}
              aria-label={t("theme.toggle")}
            >
              {theme === "dark" ? (
                <Sun className="h-5 w-5" />
              ) : (
                <Moon className="h-5 w-5" />
              )}
            </button>

            {/* Profile */}
            <Link
              to="/profile"
              className={`h-11 w-11 flex items-center justify-center transition-colors rounded-lg ${
                isLight
                  ? "text-gray-500 hover:text-gray-900"
                  : "text-[var(--color-text-secondary)] hover:text-[var(--color-text-primary)]"
              }`}
            >
              <User className="h-5 w-5" />
            </Link>

            {/* Mobile menu button — shown below lg (768–1023 uses the burger) */}
            <button
              className="h-11 w-11 flex items-center justify-center lg:hidden transition-colors rounded-lg text-[var(--color-text-primary)]"
              onClick={() => setMenuOpen(!menuOpen)}
              aria-label={menuOpen ? t("common.close") : "Menu"}
            >
              {menuOpen ? (
                <X className="h-6 w-6" />
              ) : (
                <Menu className="h-6 w-6" />
              )}
            </button>
          </div>
        </div>
      </header>

      {/* Mobile menu — fullscreen overlay */}
      <AnimatePresence>
        {menuOpen && (
          <motion.div
            data-testid="mobile-menu"
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
            exit={{ opacity: 0 }}
            transition={{ duration: 0.3 }}
            className={`fixed inset-0 z-40 lg:hidden ${
              isLight ? "bg-white/98" : "bg-[var(--color-bg-primary)]/98"
            } backdrop-blur-xl`}
          >
            <nav className="flex h-full flex-col items-center justify-center gap-8">
              {navLinks.map((link, i) => (
                <motion.div
                  key={link.href}
                  initial={{ opacity: 0, y: 20 }}
                  animate={{ opacity: 1, y: 0 }}
                  transition={{ delay: i * 0.1 }}
                >
                  <Link
                    to={link.href}
                    className={`text-3xl font-bold text-center block py-2 px-4 transition-colors ${
                      isLight
                        ? "text-gray-900 hover:text-[#44944A]"
                        : "text-[var(--color-text-primary)] hover:text-[#44944A]"
                    }`}
                    onClick={() => setMenuOpen(false)}
                  >
                    {t(link.labelKey)}
                  </Link>
                </motion.div>
              ))}
            </nav>

            {/* Language & currency toggles pinned to the bottom (KAN-56) */}
            <div className="absolute bottom-0 left-0 right-0 flex items-center justify-center gap-3 pb-[max(2.5rem,env(safe-area-inset-bottom))]">
              <LanguageToggle />
              <CurrencyToggle />
            </div>
          </motion.div>
        )}
      </AnimatePresence>
    </>
  );
}
