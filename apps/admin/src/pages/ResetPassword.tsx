import { useState, FormEvent } from 'react';
import { useParams, Link, useNavigate } from 'react-router-dom';
import { motion } from 'framer-motion';
import { resetPassword as apiResetPassword } from '@/lib/api';
import PasswordInput from '@/components/common/PasswordInput';
import { KeyRound, CheckCircle, ArrowLeft, AlertTriangle } from 'lucide-react';

export default function ResetPassword() {
  const { token } = useParams<{ token: string }>();
  const [password, setPassword] = useState('');
  const [confirm, setConfirm] = useState('');
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);
  const [done, setDone] = useState(false);
  const nav = useNavigate();

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault();
    setError('');
    if (!password.trim() || !confirm.trim()) {
      setError('Заполните все поля');
      return;
    }
    if (password !== confirm) {
      setError('Пароли не совпадают');
      return;
    }
    if (password.length < 8) {
      setError('Минимум 8 символов');
      return;
    }
    if (!token) {
      setError('Недействительная ссылка');
      return;
    }
    setLoading(true);
    try {
      await apiResetPassword(token, password);
      setDone(true);
      setTimeout(() => nav('/login'), 3000);
    } catch (err) {
      const msg = err instanceof Error ? err.message : '';
      setError(msg || 'Ошибка соединения');
    } finally {
      setLoading(false);
    }
  };

  if (done) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-[var(--color-bg-primary)] p-4">
        <motion.div
          initial={{ opacity: 0, y: 40 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: 0.8, ease: [0.16, 1, 0.3, 1] }}
          className="w-full max-w-md"
        >
          <div data-testid="reset-success" className="rounded-2xl bg-[var(--color-bg-card)] border border-[var(--color-border-custom)] p-8 text-center">
            <motion.div
              initial={{ scale: 0 }}
              animate={{ scale: 1 }}
              transition={{ delay: 0.2, type: 'spring', stiffness: 200 }}
              className="w-16 h-16 rounded-full bg-[#44944A]/10 flex items-center justify-center mx-auto mb-6"
            >
              <CheckCircle className="h-8 w-8 text-[#44944A]" />
            </motion.div>
            <h2 className="text-xl font-bold tracking-tighter text-[var(--color-text-primary)] mb-2">
              Пароль изменён
            </h2>
            <p className="text-sm text-[var(--color-text-secondary)]">
              Сейчас вы будете перенаправлены на страницу входа
            </p>
          </div>
        </motion.div>
      </div>
    );
  }

  if (!token) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-[var(--color-bg-primary)] p-4">
        <motion.div
          initial={{ opacity: 0, y: 40 }}
          animate={{ opacity: 1, y: 0 }}
          className="w-full max-w-md"
        >
          <div className="rounded-2xl bg-[var(--color-bg-card)] border border-[var(--color-border-custom)] p-8 text-center">
            <div className="w-16 h-16 rounded-full bg-red-500/10 flex items-center justify-center mx-auto mb-6">
              <AlertTriangle className="h-8 w-8 text-red-500" />
            </div>
            <h2 className="text-xl font-bold tracking-tighter text-[var(--color-text-primary)] mb-2">
              Недействительная ссылка
            </h2>
            <p className="text-sm text-[var(--color-text-secondary)] mb-6">
              Ссылка для сброса пароля недействительна или устарела
            </p>
            <Link
              to="/login"
              className="inline-flex items-center gap-2 text-sm text-[#44944A] hover:underline font-medium"
            >
              <ArrowLeft className="h-4 w-4" />
              Вернуться ко входу
            </Link>
          </div>
        </motion.div>
      </div>
    );
  }

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
              Новый пароль
            </h1>
            <p className="text-sm text-[var(--color-text-muted)]">
              Придумайте новый пароль для вашего аккаунта
            </p>
          </div>

          <form onSubmit={handleSubmit} className="space-y-4">
            <PasswordInput
              placeholder="Новый пароль"
              value={password}
              onChange={setPassword}
              autoComplete="new-password"
              testId="reset-password"
            />
            <PasswordInput
              placeholder="Повторите пароль"
              value={confirm}
              onChange={setConfirm}
              autoComplete="new-password"
              testId="reset-password-confirm"
            />

            {error && (
              <p data-testid="reset-error" className="text-sm text-red-500 px-1">{error}</p>
            )}

            <button
              type="submit"
              disabled={loading}
              data-testid="reset-submit"
              className="flex items-center justify-center gap-2 w-full rounded-xl bg-[#44944A] px-4 py-2.5 text-sm font-semibold text-black transition-all hover:shadow-[0_0_30px_rgba(68,148,74,0.3)] disabled:opacity-50 disabled:cursor-not-allowed"
            >
              <KeyRound className="h-4 w-4" />
              {loading ? 'Сохранение...' : 'Сменить пароль'}
            </button>
          </form>

          <div className="mt-6 text-center">
            <Link
              to="/login"
              className="inline-flex items-center gap-2 text-sm text-[var(--color-text-muted)] hover:text-[#44944A] transition-colors"
            >
              <ArrowLeft className="h-4 w-4" />
              Вернуться ко входу
            </Link>
          </div>
        </div>
      </motion.div>
    </div>
  );
}
