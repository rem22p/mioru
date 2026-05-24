import { useEffect } from "react";
import { Routes, Route, Navigate, useNavigate } from "react-router-dom";
import { useAuthStore } from "@/stores/authStore";
import { useUiStore } from "@/stores/uiStore";
import Sidebar from "./Sidebar";
import Placeholder from "@/components/common/Placeholder";
import Products from "@/workspaces/Products";
import Profile from "@/workspaces/Profile";
import Settings from "@/workspaces/Settings";
import {
  StickyNote,
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

  useEffect(() => {
    if (!isAuthenticated) {
      nav("/login");
      return;
    }
    fetchUser();
  }, []);

  if (!isAuthenticated) return null;

  if (isLoading) {
    return (
      <div className="flex h-screen items-center justify-center bg-[var(--color-bg-primary)]">
        <div className="text-center">
          <Loader2 className="h-8 w-8 text-[var(--color-accent)] animate-spin mx-auto mb-4" />
          <p className="text-sm text-[var(--color-text-muted)]">Загрузка...</p>
        </div>
      </div>
    );
  }

  return (
    <div className="flex h-screen overflow-hidden bg-[var(--color-bg-primary)]">
      <Sidebar collapsed={sidebarCollapsed} onToggle={toggleSidebar} />
      <main className="flex-1 overflow-y-auto">
        <Routes>
          <Route path="/" element={<Navigate to="/products" replace />} />
          <Route path="/products" element={<Products />} />
          <Route
            path="/notes"
            element={
              <Placeholder
                icon={StickyNote}
                title="Доска заметок"
                desc="Модуль в разработке"
              />
            }
          />
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
