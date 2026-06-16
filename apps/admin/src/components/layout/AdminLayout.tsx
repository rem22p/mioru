import { useEffect } from "react";
import { Routes, Route, Navigate, useNavigate } from "react-router-dom";
import { useAuthStore } from "@/stores/authStore";
import { useUiStore } from "@/stores/uiStore";
import Sidebar from "./Sidebar";
import Placeholder from "@/components/common/Placeholder";
import Products from "@/workspaces/Products";
import Orders from "@/workspaces/Orders";
import Users from "@/workspaces/Users";
import Customers from "@/workspaces/Customers";
import CustomerDetail from "@/workspaces/CustomerDetail";
import Profile from "@/workspaces/Profile";
import Settings from "@/workspaces/Settings";
import {
  Banknote,
  BarChart3,
  Bot,
  Flame,
  Lightbulb,
  Loader2,
} from "lucide-react";

export default function AdminLayout() {
  const { isAuthenticated, isLoading, user, fetchUser } = useAuthStore();
  const { sidebarCollapsed, toggleSidebar } = useUiStore();
  const nav = useNavigate();

  // Cookie-only auth: JS can't peek at the HttpOnly auth cookie, so on every
  // mount we ASK the API. fetchUser() resolves isAuthenticated to a concrete
  // bool — until then it's null and we render the neutral loader (otherwise
  // a reload of any /admin route would flash the login screen for one tick
  // before the /me round-trip completes).
  useEffect(() => {
    fetchUser();
  }, []);

  useEffect(() => {
    if (isAuthenticated === false) {
      nav("/login");
    }
  }, [isAuthenticated, nav]);

  if (isAuthenticated === null || isLoading) {
    return (
      <div className="flex h-screen items-center justify-center bg-[var(--color-bg-primary)]">
        <div className="text-center">
          <Loader2 className="h-8 w-8 text-[var(--color-accent)] animate-spin mx-auto mb-4" />
          <p className="text-sm text-[var(--color-text-muted)]">Загрузка...</p>
        </div>
      </div>
    );
  }

  if (!isAuthenticated) return null;

  return (
    <div className="flex h-screen overflow-hidden bg-[var(--color-bg-primary)]">
      <Sidebar collapsed={sidebarCollapsed} onToggle={toggleSidebar} />
      <main className="flex-1 overflow-y-auto">
        <Routes>
          <Route path="/" element={<Navigate to="/products" replace />} />
          <Route path="/products" element={<Products />} />
          <Route path="/orders" element={<Orders />} />
          <Route path="/users" element={user?.role === "super_admin" ? <Users /> : <Navigate to="/products" replace />} />
          <Route path="/customers" element={<Customers />} />
          <Route path="/customers/:id" element={<CustomerDetail />} />
          <Route path="/profile" element={<Profile />} />
          <Route path="/settings" element={<Settings />} />
          <Route
            path="/accounting"
            element={
              <Placeholder
                icon={Banknote}
                title="Бухгалтерия"
                desc="Модуль в разработке"
              />
            }
          />
          <Route
            path="/analytics"
            element={
              <Placeholder
                icon={BarChart3}
                title="Аналитика"
                desc="Модуль в разработке"
              />
            }
          />
          <Route
            path="/chatbot"
            element={
              <Placeholder
                icon={Bot}
                title="Чат-бот"
                desc="Модуль в разработке"
              />
            }
          />
          <Route
            path="/tinder"
            element={
              <Placeholder
                icon={Flame}
                title="Тиндер"
                desc="Модуль в разработке"
              />
            }
          />
          <Route
            path="/ideas"
            element={
              <Placeholder
                icon={Lightbulb}
                title="Идеи"
                desc="Модуль в разработке"
              />
            }
          />
          <Route path="*" element={<Navigate to="/products" replace />} />
        </Routes>
      </main>
    </div>
  );
}
