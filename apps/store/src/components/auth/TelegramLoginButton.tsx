import { useTranslation } from "react-i18next";
import { useAuthStore } from "@/stores/authStore";
import type { TelegramAuthData } from "@/lib/api";

// Extend window with Telegram widget callback.
declare global {
  interface Window {
    onTelegramAuth?: (user: TelegramAuthData) => void;
  }
}

interface TelegramLoginButtonProps {
  botName: string;
  onSuccess?: () => void;
  onError?: (error: Error) => void;
}

// This component is no longer used — kept for reference.
// The store-based auth flow now uses AuthSection on /profile page.
export default function TelegramLoginButton({
  botName: _botName,
  onSuccess: _onSuccess,
  onError: _onError,
}: TelegramLoginButtonProps) {
  const { t } = useTranslation();

  return (
    <div className="flex flex-col items-center gap-3">
      <p className="text-xs text-[var(--color-text-muted)]">
        {t("auth.orLoginWith")}
      </p>
      <div className="flex justify-center">
        <span className="text-xs text-[var(--color-text-muted)]">
          Telegram login is currently unavailable
        </span>
      </div>
    </div>
  );
}
