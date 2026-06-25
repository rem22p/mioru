import { useState, useRef, useEffect } from 'react';
import { Link, useNavigate, useLocation } from 'react-router-dom';
import {
  User,
  Settings,
  LogOut,
  ChevronLeft,
  ChevronRight,
  Package,
  ShoppingBag,
  UsersRound,
  UserCircle,
  Send,
  Banknote,
  BarChart3,
  Bot,
  Flame,
  Lightbulb,
} from "lucide-react";
import { motion, AnimatePresence } from "framer-motion";
import { useAuthStore } from "@/stores/authStore";
import { WORKSPACES } from "@/lib/constants";
import { cn } from "@/lib/utils";

const ICON_MAP: Record<string, React.ComponentType<{ className?: string }>> = {
  Package,
  ShoppingBag,
  UsersRound,
  UserCircle,
  Send,
  Banknote,
  BarChart3,
  Bot,
  Flame,
  Lightbulb,
};

interface SidebarProps {
  collapsed: boolean;
  onToggle: () => void;
}

export default function Sidebar({ collapsed, onToggle }: SidebarProps) {
  const { user, logout } = useAuthStore();
  const [menuOpen, setMenuOpen] = useState(false);
  const menuRef = useRef<HTMLDivElement>(null);
  const nav = useNavigate();
  const loc = useLocation();

  const currentWs = loc.pathname === '/' ? 'products' : loc.pathname.slice(1);

  // Hide "users" workspace from non-super_admin
  const visibleWorkspaces = WORKSPACES.filter(
    (ws) => ws.id !== 'users' || user?.role === 'super_admin'
  );

  useEffect(() => {
    const handler = (e: MouseEvent) => {
      if (menuRef.current && !menuRef.current.contains(e.target as Node)) setMenuOpen(false);
    };
    document.addEventListener('mousedown', handler);
    return () => document.removeEventListener('mousedown', handler);
  }, []);

  const handleLogout = async () => {
    await logout();
    nav('/login');
  };

  const initial = (user?.display_name || user?.username || '?')[0]?.toUpperCase() || '?';
  const displayName =
    user?.display_name ||
    `${user?.first_name || ''} ${user?.last_name || ''}`.trim() ||
    user?.username ||
    'User';

  return (
    <motion.aside
      animate={{ width: collapsed ? 64 : 224 }}
      transition={{ duration: 0.3, ease: [0.16, 1, 0.3, 1] }}
      className="flex flex-col h-screen bg-[var(--color-bg-card)] border-r border-[var(--color-border-custom)] overflow-hidden"
    >
      {/* Logo */}
      <div className="flex items-center justify-between h-14 px-4 border-b border-[var(--color-border-custom)] shrink-0">
        <AnimatePresence mode="wait">
          {!collapsed && (
            <motion.span
              key="logo"
              initial={{ opacity: 0, x: -10 }}
              animate={{ opacity: 1, x: 0 }}
              exit={{ opacity: 0, x: -10 }}
              transition={{ duration: 0.2 }}
              className="text-xl font-bold tracking-tighter text-[var(--color-text-primary)]"
            >
              MIORU
            </motion.span>
          )}
        </AnimatePresence>
        <button
          onClick={onToggle}
          className={cn(
            'flex items-center justify-center h-8 w-8 rounded-lg text-[var(--color-text-muted)] hover:text-[var(--color-text-primary)] hover:bg-[var(--color-bg-secondary)] transition-colors',
            collapsed && 'mx-auto',
          )}
        >
          {collapsed ? <ChevronRight className="h-4 w-4" /> : <ChevronLeft className="h-4 w-4" />}
        </button>
      </div>

      {/* Workspaces */}
      <nav className="flex-1 overflow-y-auto py-3 px-2">
        {!collapsed && (
          <div className="px-3 py-1.5 text-[11px] font-semibold text-[var(--color-text-muted)] uppercase tracking-widest">
            Пространства
          </div>
        )}
        {visibleWorkspaces.map((ws, i) => {
          const Icon = ICON_MAP[ws.icon];
          const isActive = currentWs === ws.id;
          return (
            <motion.button
              key={ws.id}
              disabled={!ws.active}
              onClick={() => nav(`/${ws.id}`)}
              whileHover={ws.active ? { x: 4 } : {}}
              whileTap={ws.active ? { scale: 0.97 } : {}}
              className={cn(
                'w-full flex items-center gap-3 px-3 py-2.5 my-0.5 text-sm rounded-xl transition-all',
                isActive &&
                  'bg-[#44944A]/10 text-[#44944A] font-medium border-l-[3px] border-l-[#44944A]',
                !isActive && ws.active && 'text-[var(--color-text-secondary)] hover:text-[var(--color-text-primary)] hover:bg-[var(--color-bg-secondary)]',
                !ws.active && 'opacity-30 cursor-not-allowed',
                collapsed && 'justify-center px-2 border-l-0',
              )}
              title={collapsed ? ws.label : undefined}
            >
              {Icon && <Icon className="h-5 w-5 flex-shrink-0" />}
              {!collapsed && <span className="truncate">{ws.label}</span>}
            </motion.button>
          );
        })}
      </nav>

      {/* User section */}
      <div className="border-t border-[var(--color-border-custom)] p-2 shrink-0" ref={menuRef}>
        <div className="relative">
          <button
            onClick={() => setMenuOpen(!menuOpen)}
            className={cn(
              'w-full flex items-center gap-3 rounded-xl p-2 hover:bg-[var(--color-bg-secondary)] transition-colors',
              collapsed && 'justify-center',
            )}
          >
            <div
              className="h-8 w-8 rounded-full flex items-center justify-center text-sm font-bold text-white flex-shrink-0"
              style={{ backgroundColor: user?.avatar_color || '#f85149' }}
            >
              {initial}
            </div>
            <AnimatePresence mode="wait">
              {!collapsed && (
                <motion.div
                  key="user-info"
                  initial={{ opacity: 0 }}
                  animate={{ opacity: 1 }}
                  exit={{ opacity: 0 }}
                  className="flex-1 text-left overflow-hidden"
                >
                  <div className="text-sm font-medium truncate">{displayName}</div>
                  <div className="text-xs text-[var(--color-text-muted)] truncate">
                    @{user?.username}
                  </div>
                </motion.div>
              )}
            </AnimatePresence>
          </button>

          <AnimatePresence>
            {menuOpen && (
              <motion.div
                initial={{ opacity: 0, y: 8, scale: 0.96 }}
                animate={{ opacity: 1, y: 0, scale: 1 }}
                exit={{ opacity: 0, y: 8, scale: 0.96 }}
                transition={{ duration: 0.2 }}
                className={cn(
                  'absolute left-0 mb-2 w-48 bg-[var(--color-bg-card)] border border-[var(--color-border-custom)] rounded-xl shadow-xl py-1 z-50 overflow-hidden',
                  collapsed ? 'bottom-full' : 'bottom-full',
                )}
              >
                <Link
                  to="/profile"
                  onClick={() => setMenuOpen(false)}
                  className="flex items-center gap-3 px-4 py-2.5 text-sm text-[var(--color-text-secondary)] hover:text-[var(--color-text-primary)] hover:bg-[var(--color-bg-secondary)] transition-colors"
                >
                  <User className="h-4 w-4" />
                  Профиль
                </Link>
                <Link
                  to="/settings"
                  onClick={() => setMenuOpen(false)}
                  className="flex items-center gap-3 px-4 py-2.5 text-sm text-[var(--color-text-secondary)] hover:text-[var(--color-text-primary)] hover:bg-[var(--color-bg-secondary)] transition-colors"
                >
                  <Settings className="h-4 w-4" />
                  Настройки
                </Link>
                <div className="h-px bg-[var(--color-border-custom)] my-1" />
                <button
                  onClick={handleLogout}
                  className="w-full flex items-center gap-3 px-4 py-2.5 text-sm text-red-500 hover:bg-red-500/10 transition-colors"
                >
                  <LogOut className="h-4 w-4" />
                  Выход
                </button>
              </motion.div>
            )}
          </AnimatePresence>
        </div>
      </div>
    </motion.aside>
  );
}
