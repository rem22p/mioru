import { useState, FormEvent } from 'react';
import { Link, useNavigate } from 'react-router-dom';
import { motion } from 'framer-motion';
import { useAuthStore } from '@/stores/authStore';
import { login as apiLogin } from '@/lib/api';
import PasswordInput from '@/components/common/PasswordInput';
import { LogIn } from 'lucide-react';

export default function Login() {
  const [username, setUsername] = useState('');
  const [password, setPassword] = useState('');
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);
  const { login } = useAuthStore();
  const nav = useNavigate();

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault();
    setError('');
    if (!username.trim() || !password.trim()) {
      setError('Заполните все поля');
      return;
    }
    setLoading(true);
    try {
      const data = await apiLogin(username, password);
      login(data.access_token);
      nav('/');
    } catch (err) {
      const msg = err instanceof Error ? err.message : '';
      setError(msg.includes('401') ? 'Неверный логин или пароль' : 'Ошибка соединения');
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="min-h-screen flex items-center justify-center bg-[var(--color-bg-primary)] p-4">
      <motion.div
        initial={{ opacity: 0, y: 40 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ duration: 0.8, ease: [0.16, 1, 0.3, 1] }}
        className="w-full max-w-md"
      >
        <div className="rounded-2xl bg-[var(--color-bg-card)] border border-[var(--color-border-custom)] p-8">
          {/* Header */}
          <div className="text-center mb-8">
            <h1 className="text-3xl font-bold tracking-tighter text-[var(--color-text-primary)] mb-2">
              MIORU
            </h1>
            <p className="text-sm text-[var(--color-text-muted)]">
              Войдите в панель администратора
            </p>
          </div>

          <form onSubmit={handleSubmit} className="space-y-4">
            <div>
              <input
                type="text"
                placeholder="Никнейм"
                value={username}
                onChange={(e) => setUsername(e.target.value)}
                autoComplete="username"
                autoFocus
                className="w-full rounded-xl bg-[var(--color-bg-primary)] border border-[var(--color-border-custom)] px-4 py-2.5 text-sm text-[var(--color-text-primary)] outline-none focus:border-[#44944A] placeholder:text-[var(--color-text-muted)] transition-colors"
              />
            </div>

            <PasswordInput
              placeholder="Пароль"
              value={password}
              onChange={setPassword}
              autoComplete="current-password"
            />

            {error && (
              <p className="text-sm text-red-500 px-1">{error}</p>
            )}

            <button
              type="submit"
              disabled={loading}
              className="flex items-center justify-center gap-2 w-full rounded-xl bg-[#44944A] px-4 py-2.5 text-sm font-semibold text-black transition-all hover:shadow-[0_0_30px_rgba(68,148,74,0.3)] disabled:opacity-50 disabled:cursor-not-allowed"
            >
              <LogIn className="h-4 w-4" />
              {loading ? 'Вход...' : 'Войти'}
            </button>
          </form>

          <div className="mt-6 flex flex-col items-center gap-2">
            <p className="text-sm text-[var(--color-text-secondary)]">
              Нет аккаунта?{' '}
              <Link to="/register" className="text-[#44944A] hover:underline font-medium">
                Регистрация
              </Link>
            </p>
            <Link
              to="/forgot-password"
              className="text-sm text-[var(--color-text-muted)] hover:text-[#44944A] transition-colors"
            >
              Забыли пароль?
            </Link>
          </div>
        </div>
      </motion.div>
    </div>
  );
}
