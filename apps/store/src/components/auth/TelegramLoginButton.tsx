import { useEffect, useRef, useCallback } from "react";
import { useTranslation } from "react-i18next";
import { fetchTelegramLogin, type TelegramAuthData } from "@/lib/api";
import { useAuthStore } from "@/stores/authStore";

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

export default function TelegramLoginButton({
  botName,
  onSuccess,
  onError,
}: TelegramLoginButtonProps) {
  const { t } = useTranslation();
  const setUser = useAuthStore((s) => s.setUser);
  const containerRef = useRef<HTMLDivElement>(null);
  const widgetLoaded = useRef(false);

  const handleAuth = useCallback(
    async (user: TelegramAuthData) => {
      try {
        const profile = await fetchTelegramLogin(user);
        setUser({
          id: String(profile.id),
          name: [profile.first_name, profile.last_name]
            .filter(Boolean)
            .join(" "),
          email: profile.email || "",
          avatarParams: {
            gender: "male",
            height: 170,
            weight: 70,
            fatPercentage: 20,
            musclePercentage: 40,
          },
          xpBalance: 0,
          vipLevel: 0,
        });
        onSuccess?.();
      } catch (err) {
        onError?.(err instanceof Error ? err : new Error("Telegram login failed"));
      }
    },
    [setUser, onSuccess, onError],
  );

  useEffect(() => {
    // Register callback before script loads.
    window.onTelegramAuth = handleAuth;

    // Avoid double-loading the widget script.
    if (widgetLoaded.current) return;
    widgetLoaded.current = true;

    const script = document.createElement("script");
    script.src = "https://telegram.org/js/telegram-widget.js?22";
    script.async = true;
    script.setAttribute("data-telegram-login", botName);
    script.setAttribute("data-size", "large");
    script.setAttribute("data-onauth", "onTelegramAuth(user)");
    script.setAttribute("data-request-access", "write");

    if (containerRef.current) {
      containerRef.current.appendChild(script);
    }

    return () => {
      window.onTelegramAuth = undefined;
    };
  }, [botName, handleAuth]);

  return (
    <div className="flex flex-col items-center gap-3">
      <p className="text-xs text-[var(--color-text-muted)]">
        {t("auth.orLoginWith")}
      </p>
      <div ref={containerRef} className="flex justify-center" />
    </div>
  );
}
