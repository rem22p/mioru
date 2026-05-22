import { Routes, Route } from "react-router-dom";
import { lazy, Suspense, useState, useEffect } from "react";
import { useTranslation } from "react-i18next";
import Header from "@/components/layout/Header";
import Footer from "@/components/layout/Footer";
import WaveBackground from "@/components/layout/WaveBackground";
import { useScrollToTop } from "@/hooks/useScrollToTop";

const HomePage = lazy(() => import("@/routes/HomePage"));
const CatalogPage = lazy(() => import("@/routes/CatalogPage"));
const ProductPage = lazy(() => import("@/routes/ProductPage"));
const CartPage = lazy(() => import("@/routes/CartPage"));
const CheckoutPage = lazy(() => import("@/routes/CheckoutPage"));
const AvatarPage = lazy(() => import("@/routes/AvatarPage"));
const ProfilePage = lazy(() => import("@/routes/ProfilePage"));
const AdminPage = lazy(() => import("@/routes/AdminPage"));

function PageLoader() {
  return (
    <div className="flex items-center justify-center min-h-[60vh]">
      <div className="text-center">
        <div className="text-6xl mb-4 animate-pulse">👤</div>
        <p className="text-[var(--color-text-muted)] font-mono text-sm">Loading...</p>
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

  return (
    <>
      <WaveBackground />
      <div className="relative z-10 flex flex-col min-h-screen">
        <Header
          theme={theme}
          toggleTheme={toggleTheme}
          changeLanguage={changeLanguage}
        />
        <main className="flex-1">
          <ScrollToTop />
          <Suspense fallback={<PageLoader />}>
            <Routes>
              <Route path="/" element={<HomePage />} />
              <Route path="/catalog" element={<CatalogPage />} />
              <Route path="/product/:slug" element={<ProductPage />} />
              <Route path="/cart" element={<CartPage />} />
              <Route path="/checkout" element={<CheckoutPage />} />
              <Route path="/avatar" element={<AvatarPage />} />
              <Route path="/profile" element={<ProfilePage />} />
              <Route path="/admin" element={<AdminPage />} />
            </Routes>
          </Suspense>
        </main>
        <Footer />
      </div>
    </>
  );
}
