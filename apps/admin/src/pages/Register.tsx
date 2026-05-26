import { useState, FormEvent } from 'react';
import { Link, useNavigate } from 'react-router-dom';
import { motion } from 'framer-motion';
import { register as apiRegister } from '@/lib/api';
import PasswordInput from '@/components/common/PasswordInput';
import { UserPlus } from 'lucide-react';

export default function Register() {
  const [firstName, setFirstName] = useState('');
  const [lastName, setLastName] = useState('');
  const [email, setEmail] = useState('');
  const [username, setUsername] = useState('');
  const [password, setPassword] = useState('');
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);
  const nav = useNavigate();

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault();
    setError('');
    if (!firstName.trim() || !lastName.trim() || !email.trim() || !username.trim() || !password.trim()) {
      setError('Заполните все поля');
      return;
    }
    if (password.length < 8) {
      setError('Минимум 8 символов');
      return;
    }
    setLoading(true);
    try {
      // Invite-only: an existing admin creates the account. The call returns the
      // new user's summary (no token), so we stay logged in as the current admin
      // and return to the dashboard rather than logging in as the new account.
      await apiRegister({
        first_name: firstName,
        last_name: lastName,
        email,
        username,
        password,
      });
      nav('/');
    } catch (err) {
      const msg = err instanceof Error ? err.message : '';
      setError(msg || 'Ошибка соединения');
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
              Создайте аккаунт администратора
            </p>
          </div>

          <form onSubmit={handleSubmit} className="space-y-4">
            <div className="grid grid-cols-2 gap-3">
              <input
                type="text"
                placeholder="Имя"
                value={firstName}
                onChange={(e) => setFirstName(e.target.value)}
                autoComplete="given-name"
                className="w-full rounded-xl bg-[var(--color-bg-primary)] border border-[var(--color-border-custom)] px-4 py-2.5 text-sm text-[var(--color-text-primary)] outline-none focus:border-[#44944A] placeholder:text-[var(--color-text-muted)] transition-colors"
              />
              <input
                type="text"
                placeholder="Фамилия"
                value={lastName}
                onChange={(e) => setLastName(e.target.value)}
                autoComplete="family-name"
                className="w-full rounded-xl bg-[var(--color-bg-primary)] border border-[var(--color-border-custom)] px-4 py-2.5 text-sm text-[var(--color-text-primary)] outline-none focus:border-[#44944A] placeholder:text-[var(--color-text-muted)] transition-colors"
              />
            </div>

            <input
              type="email"
              placeholder="Email"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              autoComplete="email"
              className="w-full rounded-xl bg-[var(--color-bg-primary)] border border-[var(--color-border-custom)] px-4 py-2.5 text-sm text-[var(--color-text-primary)] outline-none focus:border-[#44944A] placeholder:text-[var(--color-text-muted)] transition-colors"
            />

            <input
              type="text"
              placeholder="Никнейм"
              value={username}
              onChange={(e) => setUsername(e.target.value)}
              autoComplete="username"
              className="w-full rounded-xl bg-[var(--color-bg-primary)] border border-[var(--color-border-custom)] px-4 py-2.5 text-sm text-[var(--color-text-primary)] outline-none focus:border-[#44944A] placeholder:text-[var(--color-text-muted)] transition-colors"
            />

            <PasswordInput
              placeholder="Пароль"
              value={password}
              onChange={setPassword}
              autoComplete="new-password"
            />

            {error && (
              <p className="text-sm text-red-500 px-1">{error}</p>
            )}

            <button
              type="submit"
              disabled={loading}
              className="flex items-center justify-center gap-2 w-full rounded-xl bg-[#44944A] px-4 py-2.5 text-sm font-semibold text-black transition-all hover:shadow-[0_0_30px_rgba(68,148,74,0.3)] disabled:opacity-50 disabled:cursor-not-allowed"
            >
              <UserPlus className="h-4 w-4" />
              {loading ? 'Регистрация...' : 'Зарегистрироваться'}
            </button>
          </form>

          <p className="mt-6 text-sm text-[var(--color-text-secondary)] text-center">
            Уже есть аккаунт?{' '}
            <Link to="/login" className="text-[#44944A] hover:underline font-medium">
              Войти
            </Link>
          </p>
        </div>
      </motion.div>
    </div>
  );
}
