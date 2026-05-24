import { useState } from 'react';
import { useAuthStore } from '@/stores/authStore';
import { updateUser, changePassword as apiChangePassword } from '@/lib/api';
import { AVATAR_COLORS } from '@/lib/constants';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import PasswordInput from '@/components/common/PasswordInput';
import { User, Check, AlertTriangle, Shield } from 'lucide-react';

export default function Profile() {
  const { user, fetchUser } = useAuthStore();
  const [dn, setDn] = useState(user?.display_name || '');
  const [ac, setAc] = useState(user?.avatar_color || '#f85149');
  const [pwCur, setPwCur] = useState('');
  const [pwNew, setPwNew] = useState('');
  const [pwConfirm, setPwConfirm] = useState('');
  const [pwMode, setPwMode] = useState(false);
  const [alerts, setAlerts] = useState<{ type: 'ok' | 'err' | 'warn'; text: string }[]>([]);

  const addAlert = (type: 'ok' | 'err' | 'warn', text: string) => {
    setAlerts(prev => [...prev, { type, text }]);
    setTimeout(() => setAlerts(prev => prev.slice(1)), 4000);
  };

  const validatePw = (pw: string): string[] => {
    const errs: string[] = [];
    if (pw.length < 8) errs.push('минимум 8 символов');
    if (pw.length > 72) errs.push('максимум 72 символа');
    if (!(/[a-zA-Z]/.test(pw) && /[0-9]/.test(pw))) errs.push('нужны буквы и цифры');
    return errs;
  };

  const saveProfile = async () => {
    setAlerts([]);
    try {
      await updateUser({ display_name: dn, avatar_color: ac });
      await fetchUser();
      addAlert('ok', 'Профиль сохранён');
    } catch {
      addAlert('err', 'Ошибка сохранения профиля');
    }
  };

  const changePassword = async () => {
    setAlerts([]);
    if (!pwCur.trim()) { addAlert('err', 'Введите текущий пароль'); return; }
    if (!pwNew.trim()) { addAlert('err', 'Введите новый пароль'); return; }
    if (!pwConfirm.trim()) { addAlert('err', 'Повторите новый пароль'); return; }
    if (pwNew !== pwConfirm) { addAlert('err', 'Пароли не совпадают'); return; }
    if (pwNew === pwCur) { addAlert('warn', 'Новый пароль должен отличаться от текущего'); return; }

    const errs = validatePw(pwNew);
    if (errs.length) { addAlert('err', errs.join(', ')); return; }

    try {
      await apiChangePassword(pwCur, pwNew);
      setPwCur(''); setPwNew(''); setPwConfirm(''); setPwMode(false);
      addAlert('ok', 'Пароль изменён');
    } catch (e: unknown) {
      const m = e instanceof Error ? e.message : '';
      addAlert('err', m.includes('401') ? 'Неверный текущий пароль' : 'Ошибка смены пароля');
    }
  };

  if (!user) return null;

  const initial = (user.display_name || user.username)[0]?.toUpperCase() || '?';

  return (
    <div className="max-w-2xl mx-auto p-6 space-y-8">
      <div className="flex items-center gap-4">
        <div
          className="w-20 h-20 rounded-full flex items-center justify-center text-2xl font-bold text-white"
          style={{ backgroundColor: ac }}
        >
          {initial}
        </div>
        <div>
          <h2 className="text-xl font-semibold">{user.display_name || user.username}</h2>
          <p className="text-[var(--color-text-secondary)]">@{user.username} · {user.email}</p>
        </div>
      </div>

      <div className="space-y-4">
        <div className="space-y-2">
          <Label>Отображаемое имя</Label>
          <Input value={dn} onChange={e => setDn(e.target.value)} placeholder="Как вас зовут?" />
        </div>

        <div className="space-y-2">
          <Label>Цвет аватарки</Label>
          <div className="flex gap-2">
            {AVATAR_COLORS.map(c => (
              <button
                key={c}
                className={`w-8 h-8 rounded-full border-2 transition-all ${c === ac ? 'border-white scale-110' : 'border-transparent'}`}
                style={{ backgroundColor: c }}
                onClick={() => setAc(c)}
              />
            ))}
          </div>
        </div>

        <Button onClick={saveProfile}>
          <User className="h-4 w-4 mr-2" />
          Сохранить профиль
        </Button>
      </div>

      <div className="space-y-4">
        <Label>Смена пароля</Label>
        {!pwMode ? (
          <Button variant="outline" onClick={() => { setPwMode(true); setAlerts([]); }}>
            <Shield className="h-4 w-4 mr-2" />
            Изменить пароль
          </Button>
        ) : (
          <div className="space-y-3 p-4 border border-[var(--color-border-custom)] rounded-lg">
            <PasswordInput
              name="current_password"
              placeholder="Текущий пароль"
              value={pwCur}
              onChange={setPwCur}
              autoComplete="current-password"
            />
            <PasswordInput
              name="new_password"
              placeholder="Новый пароль"
              value={pwNew}
              onChange={setPwNew}
              autoComplete="new-password"
            />
            <PasswordInput
              name="confirm_password"
              placeholder="Повторите новый пароль"
              value={pwConfirm}
              onChange={setPwConfirm}
              autoComplete="new-password"
            />
            <div className="flex gap-2">
              <Button onClick={changePassword}>Сменить пароль</Button>
              <Button variant="ghost" onClick={() => { setPwMode(false); setPwCur(''); setPwNew(''); setPwConfirm(''); }}>
                Отмена
              </Button>
            </div>
          </div>
        )}
      </div>

      {alerts.length > 0 && (
        <div className="space-y-2">
          {alerts.map((a, i) => (
            <div
              key={i}
              className={`flex items-center gap-2 p-3 rounded-md text-sm ${
                a.type === 'ok' ? 'bg-green-500/10 text-green-500' :
                a.type === 'err' ? 'bg-red-500/10 text-red-500' :
                'bg-yellow-500/10 text-yellow-500'
              }`}
            >
              {a.type === 'ok' ? <Check className="h-4 w-4" /> :
               a.type === 'err' ? <AlertTriangle className="h-4 w-4" /> :
               <AlertTriangle className="h-4 w-4" />}
              {a.text}
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
