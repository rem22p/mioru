import { useState, FormEvent } from 'react';
import { Link } from 'react-router-dom';
import { motion } from 'framer-motion';
import { forgotPassword as apiForgotPassword } from '@/lib/api';
import { Mail, CheckCircle, ArrowLeft } from 'lucide-react';

export default function ForgotPassword() {
  const [email, setEmail] = useState('');
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);
  const [sent, setSent] = useState(false);

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault();
    setError('');
    if (!email.trim()) {
      setError('Введите email');
      return;
    }
    setLoading(true);
    try {
      await apiForgotPassword(email);
      setSent(true);
    } catch (err) {
      const msg = err instanceof Error ? err.message : '';
      setError(msg || 'Ошибка соединения');
    } finally {
      setLoading(false);
    }
  };

  if (sent) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-[var(--color-bg-primary)] p-4">
        <motion.div
          initial={{ opacity: 0, y: 40 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: 0.8, ease: [0.16, 1, 0.3, 1] }}
          className="w-full max-w-md"
        >
          <div className="rounded-2xl bg-[var(--color-bg-card)] border border-[var(--color-border-custom)] p-8 text-center">
            <motion.div
              initial={{ scale: 0 }}
              animate={{ scale: 1 }}
              transition={{ delay: 0.2, type: 'spring', stiffness: 200 }}
              className="w-16 h-16 rounded-full bg-[#44944A]/10 flex items-center justify-center mx-auto mb-6"
            >
              <CheckCircle className="h-8 w-8 text-[#44944A]" />
            </motion.div>
            <h2 className="text-xl font-bold tracking-tighter text-[var(--color-text-primary)] mb-2">
              Письмо отправлено
            </h2>
            <p className="text-sm text-[var(--color-text-secondary)] mb-6">
              Проверьте почту и перейдите по ссылке для сброса пароля
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
              MIORU
            </h1>
            <p className="text-sm text-[var(--color-text-muted)]">
              Введите email, и мы отправим ссылку для сброса пароля
            </p>
          </div>

          <form onSubmit={handleSubmit} className="space-y-4">
            <input
              type="email"
              placeholder="Email"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              autoComplete="email"
              autoFocus
              className="w-full rounded-xl bg-[var(--color-bg-primary)] border border-[var(--color-border-custom)] px-4 py-2.5 text-sm text-[var(--color-text-primary)] outline-none focus:border-[#44944A] placeholder:text-[var(--color-text-muted)] transition-colors"
            />

            {error && (
              <p className="text-sm text-red-500 px-1">{error}</p>
            )}

            <button
              type="submit"
              disabled={loading}
              className="flex items-center justify-center gap-2 w-full rounded-xl bg-[#44944A] px-4 py-2.5 text-sm font-semibold text-black transition-all hover:shadow-[0_0_30px_rgba(68,148,74,0.3)] disabled:opacity-50 disabled:cursor-not-allowed"
            >
              <Mail className="h-4 w-4" />
              {loading ? 'Отправка...' : 'Отправить ссылку'}
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
