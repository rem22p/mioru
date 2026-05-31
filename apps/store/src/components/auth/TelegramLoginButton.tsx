import { useTranslation } from "react-i18next";

interface TelegramLoginButtonProps {
  botName: string;
  onSuccess?: () => void;
  onError?: (error: Error) => void;
}

// Telegram login widget placeholder.
// The backend Telegram OAuth flow (issue #1) is functional but the
// widget integration is deferred. When re-enabled, import
// fetchTelegramLogin from @/lib/api and wire the callback.
export default function TelegramLoginButton(_props: TelegramLoginButtonProps) {
  const { t } = useTranslation();

  return (
    <div className="flex flex-col items-center gap-3">
      <p className="text-xs text-[var(--color-text-muted)]">
        {t("auth.orLoginWith")}
      </p>
      <div className="flex justify-center">
        <span className="text-xs text-[var(--color-text-muted)]">
          {t("auth.telegramUnavailable")}
        </span>
      </div>
    </div>
  );
}
