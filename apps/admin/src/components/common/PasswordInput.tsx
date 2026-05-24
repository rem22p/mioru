import { useState } from 'react';
import { Eye, EyeOff } from 'lucide-react';

interface PasswordInputProps {
  value: string;
  onChange: (value: string) => void;
  placeholder?: string;
  autoComplete?: string;
  name?: string;
  className?: string;
}

export default function PasswordInput({ value, onChange, placeholder, autoComplete, name, className }: PasswordInputProps) {
  const [visible, setVisible] = useState(false);

  return (
    <div className="relative">
      <input
        type={visible ? 'text' : 'password'}
        name={name}
        value={value}
        onChange={(e) => onChange(e.target.value)}
        placeholder={placeholder}
        autoComplete={autoComplete}
        className={`w-full rounded-xl bg-[var(--color-bg-primary)] border border-[var(--color-border-custom)] px-4 py-2.5 pr-10 text-sm text-[var(--color-text-primary)] outline-none focus:border-[#44944A] placeholder:text-[var(--color-text-muted)] transition-colors ${className || ''}`}
      />
      <button
        type="button"
        className="absolute right-0 top-0 h-full px-3 text-[var(--color-text-muted)] hover:text-[var(--color-text-primary)] transition-colors"
        onClick={() => setVisible(!visible)}
        tabIndex={-1}
        aria-label={visible ? 'Скрыть пароль' : 'Показать пароль'}
      >
        {visible ? <EyeOff className="h-4 w-4" /> : <Eye className="h-4 w-4" />}
      </button>
    </div>
  );
}
