import { Routes, Route } from "react-router-dom";
import { lazy, Suspense, useState, useEffect } from "react";
import { useTranslation } from "react-i18next";
import Header from "@/components/layout/Header";
import Footer from "@/components/layout/Footer";
import WaveBackground from "@/components/layout/WaveBackground";
import { useScrollToTop } from "@/hooks/useScrollToTop";
import { useAuthStore } from "@/stores/authStore";

const HomePage = lazy(() => import("@/routes/HomePage"));
const CatalogPage = lazy(() => import("@/routes/CatalogPage"));
const ProductPage = lazy(() => import("@/routes/ProductPage"));
const CartPage = lazy(() => import("@/routes/CartPage"));
const CheckoutPage = lazy(() => import("@/routes/CheckoutPage"));
const AvatarPage = lazy(() => import("@/routes/AvatarPage"));
const ProfilePage = lazy(() => import("@/routes/ProfilePage"));
const CustomOrderPage = lazy(() => import("@/routes/CustomOrderPage"));
const ContactsPage = lazy(() => import("@/routes/ContactsPage"));
const FavoritesPage = lazy(() => import("@/routes/FavoritesPage"));

function PageLoader() {
  const { t } = useTranslation();
  return (
    <div className="flex items-center justify-center min-h-[60vh]">
      <div className="text-center">
        <p className="text-[var(--color-text-muted)] font-mono text-sm">
          {t("common.loading")}
        </p>
      </div>
    </div>
  );
}

function ScrollToTop() {
  useScrollToTop();
  return null;
}

export default function App() {
  const { i18n } = useTranslation();
  const [theme, setTheme] = useState<"dark" | "light">(
    () => (localStorage.getItem("theme") as "dark" | "light") || "dark",
  );

  const toggleTheme = () => {
    const next = theme === "dark" ? "light" : "dark";
    setTheme(next);
    localStorage.setItem("theme", next);
  };

  // Apply theme class to <html> so CSS variables cascade everywhere
  useEffect(() => {
    const root = document.documentElement;
    if (theme === "light") {
      root.classList.add("light");
    } else {
      root.classList.remove("light");
    }
  }, [theme]);

  const changeLanguage = (lng: string) => {
    i18n.changeLanguage(lng);
  };

  // Check auth on mount
  const fetchMe = useAuthStore((s) => s.fetchMe);
  useEffect(() => { fetchMe(); }, [fetchMe]);

  return (
    <>
      <WaveBackground />
      <div className="relative z-10 flex flex-col min-h-screen">
        <Header
          theme={theme}
          toggleTheme={toggleTheme}
          changeLanguage={changeLanguage}
        />
        <main className="flex-1 flex flex-col">
          <ScrollToTop />
          <Suspense fallback={<PageLoader />}>
            <Routes>
              <Route path="/" element={<HomePage />} />
              <Route path="/catalog" element={<CatalogPage />} />
              <Route path="/catalog/:categorySlug" element={<CatalogPage />} />
              <Route path="/product/:slug" element={<ProductPage />} />
              <Route path="/cart" element={<CartPage />} />
              <Route path="/checkout" element={<CheckoutPage />} />
              <Route path="/avatar" element={<AvatarPage />} />
              <Route path="/profile" element={<ProfilePage />} />
              <Route path="/custom-order" element={<CustomOrderPage />} />
              <Route path="/contacts" element={<ContactsPage />} />
              <Route path="/favorites" element={<FavoritesPage />} />
            </Routes>
          </Suspense>
        </main>
        <Footer />
      </div>
    </>
  );
}
