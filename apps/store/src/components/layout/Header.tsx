import { useState, useEffect } from "react";
import { Link } from "react-router-dom";
import { useTranslation } from "react-i18next";
import { Menu, X, Sun, Moon, Globe, User, ShoppingBag, Heart } from "lucide-react";
import { useCartStore } from "@/stores/cartStore";
import { motion, AnimatePresence } from "framer-motion";

const desktopLinks = [
  { href: "/catalog", labelKey: "nav.inStock" },
  { href: "/custom-order", labelKey: "nav.customOrder" },
  { href: "/avatar", labelKey: "nav.avatar" },
];

const mobileLinks = [
  { href: "/catalog", labelKey: "nav.inStock" },
  { href: "/custom-order", labelKey: "nav.customOrder" },
  { href: "/avatar", labelKey: "nav.avatar" },
  { href: "/cart", labelKey: "nav.cart" },
  { href: "/favorites", labelKey: "nav.favorites" },
];

const languages = [
  { code: "ru", label: "RU" },
  { code: "en", label: "EN" },
  { code: "ro", label: "RO" },
];

interface HeaderProps {
  theme: "dark" | "light";
  toggleTheme: () => void;
  changeLanguage: (lng: string) => void;
}

export default function Header({
  theme,
  toggleTheme,
  changeLanguage,
}: HeaderProps) {
  const { t, i18n } = useTranslation();
  const [scrolled, setScrolled] = useState(false);
  const [menuOpen, setMenuOpen] = useState(false);
  const [langOpen, setLangOpen] = useState(false);
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
        <div className="relative mx-auto flex h-20 max-w-7xl items-center justify-between md:justify-end px-6 lg:px-8">
          {/* Mobile menu button — left side */}
          <button
            className="h-11 w-11 flex items-center justify-center md:hidden transition-colors rounded-lg order-first"
            onClick={() => setMenuOpen(!menuOpen)}
            aria-label={menuOpen ? t("common.close") : "Menu"}
          >
            {menuOpen ? (
              <X className="h-6 w-6 text-[var(--color-text-primary)]" />
            ) : (
              <Menu className="h-6 w-6 text-[var(--color-text-primary)]" />
            )}
          </button>

          {/* Logo */}
          <Link to="/" className="md:absolute md:left-6 lg:left-8">
            <motion.span
              className={`text-2xl font-bold tracking-tighter ${isLight ? "text-gray-900" : "text-[var(--color-text-primary)]"}`}
              whileHover={{ scale: 1.05 }}
            >
              MIORU
            </motion.span>
          </Link>

          {/* Desktop Nav — centered */}
          <nav className="hidden md:block absolute left-1/2 top-1/2 -translate-x-1/2 -translate-y-1/2">
            <div className="flex items-center gap-10">
              {desktopLinks.map((link) => (
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
            </div>
          </nav>

          {/* Right-side icons */}
          <div className="flex items-center gap-1">
            {/* Favorites */}
            <Link
              to="/favorites"
              className="h-11 w-11 hidden md:flex items-center justify-center transition-colors rounded-lg text-[var(--color-text-secondary)] hover:text-[var(--color-text-primary)]"
              aria-label={t("nav.favorites")}
            >
              <Heart className="h-5 w-5" />
            </Link>

            {/* Cart */}
            <Link
              to="/cart"
              className="h-11 w-11 hidden md:flex items-center justify-center transition-colors rounded-lg relative text-[var(--color-text-secondary)] hover:text-[var(--color-text-primary)]"
              aria-label={t("nav.cart")}
            >
              <ShoppingBag className="h-5 w-5" />
              {totalItems > 0 && (
                <span className="absolute -top-0.5 -right-0.5 flex h-4 w-4 items-center justify-center rounded-full bg-[#44944A] text-[10px] font-bold text-black">
                  {totalItems > 9 ? "9+" : totalItems}
                </span>
              )}
            </Link>

            {/* Language Switcher */}
            <div className="relative">
              <button
                onClick={() => setLangOpen(!langOpen)}
                className={`flex items-center gap-1 h-11 px-2 text-sm font-medium transition-colors rounded-lg ${
                  isLight
                    ? "text-gray-500 hover:text-gray-900"
                    : "text-[var(--color-text-secondary)] hover:text-[var(--color-text-primary)]"
                }`}
                aria-label="Change language"
              >
                <Globe className="h-4 w-4" />
                <span className="hidden sm:inline">
                  {i18n.language?.toUpperCase()}
                </span>
              </button>
              {langOpen && (
                <div
                  className={`absolute right-0 top-full mt-2 rounded-xl border p-1 shadow-lg z-50 ${
                    isLight
                      ? "bg-white border-gray-200"
                      : "bg-[var(--color-bg-card)] border-[var(--color-border-custom)]"
                  }`}
                >
                  {languages.map((lang) => (
                    <button
                      key={lang.code}
                      onClick={() => {
                        changeLanguage(lang.code);
                        setLangOpen(false);
                      }}
                      className={`block w-full text-left px-4 py-2 text-sm rounded-lg transition-colors ${
                        i18n.language === lang.code
                          ? "text-[#44944A] bg-[#44944A]/10"
                          : isLight
                            ? "text-gray-600 hover:text-gray-900 hover:bg-gray-100"
                            : "text-[var(--color-text-secondary)] hover:text-[var(--color-text-primary)] hover:bg-[var(--color-border-custom)]"
                      }`}
                    >
                      {lang.label}
                    </button>
                  ))}
                </div>
              )}
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
          </div>
        </div>
      </header>

      {/* Mobile slide-out menu */}
      <AnimatePresence>
        {menuOpen && (
          <>
            {/* Backdrop */}
            <motion.div
              initial={{ opacity: 0 }}
              animate={{ opacity: 0.5 }}
              exit={{ opacity: 0 }}
              className="fixed inset-0 z-40 bg-black md:hidden"
              onClick={() => setMenuOpen(false)}
            />
            {/* Slide-out panel */}
            <motion.div
              initial={{ x: "-100%" }}
              animate={{ x: 0 }}
              exit={{ x: "-100%" }}
              transition={{ type: "spring", damping: 25, stiffness: 200 }}
              className={`fixed top-0 left-0 bottom-0 z-50 w-72 md:hidden ${
                isLight ? "bg-white" : "bg-[var(--color-bg-primary)]"
              } pt-[env(safe-area-inset-top)]`}
            >
              <div className="flex items-center justify-between px-6 h-20">
                <Link to="/" onClick={() => setMenuOpen(false)}>
                  <span className={`text-2xl font-bold tracking-tighter ${isLight ? "text-gray-900" : "text-[var(--color-text-primary)]"}`}>
                    MIORU
                  </span>
                </Link>
                <button
                  onClick={() => setMenuOpen(false)}
                  className="h-11 w-11 flex items-center justify-center"
                >
                  <X className={`h-6 w-6 ${isLight ? "text-gray-900" : "text-[var(--color-text-primary)]"}`} />
                </button>
              </div>
              <nav className="flex flex-col px-6 pt-4 gap-1">
                {mobileLinks.map((link, i) => (
                  <motion.div
                    key={link.href}
                    initial={{ opacity: 0, x: -20 }}
                    animate={{ opacity: 1, x: 0 }}
                    transition={{ delay: i * 0.08 }}
                  >
                    <Link
                      to={link.href}
                      className={`block py-3 text-lg font-bold tracking-wider transition-colors ${
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
            </motion.div>
          </>
        )}
      </AnimatePresence>
    </>
  );
}
