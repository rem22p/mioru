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
          {/* Logo */}
          <Link to="/" className="md:absolute md:left-6 lg:left-8">
            <motion.img
              src="/favicon.ico"
              alt="MIORU"
              className="h-10 w-10"
              whileHover={{ scale: 1.1 }}
            />
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

            {/* Mobile menu button */}
            <button
              className="h-11 w-11 flex items-center justify-center md:hidden transition-colors rounded-lg text-[var(--color-text-primary)]"
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
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
            exit={{ opacity: 0 }}
            transition={{ duration: 0.3 }}
            className={`fixed inset-0 z-40 md:hidden ${
              isLight ? "bg-white/98" : "bg-[var(--color-bg-primary)]/98"
            } backdrop-blur-xl`}
          >
            <nav className="flex h-full flex-col items-center justify-center gap-8">
              {mobileLinks.map((link, i) => (
                <motion.div
                  key={link.href}
                  initial={{ opacity: 0, y: 20 }}
                  animate={{ opacity: 1, y: 0 }}
                  transition={{ delay: i * 0.1 }}
                >
                  <Link
                    to={link.href}
                    className={`text-3xl font-bold block py-2 px-4 transition-colors ${
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
        )}
      </AnimatePresence>
    </>
  );
}
