import { useEffect, useRef } from "react";
import { useAuthStore } from "@/stores/authStore";
import { fetchLinkOAuth, type TelegramAuthData } from "@/lib/api";

interface TelegramLoginButtonProps {
  botName: string;
  /** "login" (default) authenticates via /auth/telegram; "link" binds to the
   *  currently authenticated customer via /me/oauth. */
  mode?: "login" | "link";
  onSuccess?: () => void;
  onError?: (error: Error) => void;
}

// Telegram Login Widget integration — canonical script-tag pattern.
//
// The official widget reads its config from data-* attributes ON the
// <script> element itself (document.currentScript), then replaces that
// script with the login iframe. So we must attach the attributes to the
// script, not to a separate div created after load — the widget scans
// the DOM exactly once, at script evaluation time.
//
// On success it calls the global onTelegramAuth(user) with the signed
// payload; we forward it to the backend for cryptographic verification.
declare global {
  interface Window {
    TelegramLoginWidget?: {
      onAuth: (user: TelegramAuthData) => void;
    };
  }
}

export default function TelegramLoginButton({ botName, mode = "login", onSuccess, onError }: TelegramLoginButtonProps) {
  const telegramLogin = useAuthStore((s) => s.telegramLogin);
  const widgetRef = useRef<HTMLDivElement>(null);
  const callbackRef = useRef<TelegramLoginButtonProps>({ botName, mode, onSuccess, onError });
  callbackRef.current = { botName, mode, onSuccess, onError };

  useEffect(() => {
    const el = widgetRef.current;
    if (!el) return;
    if (!botName) return; // no bot configured — nothing to render

    // Expose the global callback BEFORE the widget script loads, so the
    // iframe's postMessage finds it.
    window.TelegramLoginWidget = {
      onAuth: (user: TelegramAuthData) => {
        const { mode: m, onSuccess: ok, onError: err } = callbackRef.current;
        const run = m === "link" ? fetchLinkOAuth(user).then(() => undefined) : telegramLogin(user);
        run.then(() => ok?.()).catch((e) => {
          err?.(e instanceof Error ? e : new Error(String(e)));
        });
      },
    };

    // If the widget script is already present (another instance mounted
    // it), don't double-inject — just make sure our callback is wired.
    const existing = document.querySelector<HTMLScriptElement>(
      'script[src*="telegram-widget.js"]'
    );
    if (existing) return;

    // Canonical pattern: the widget replaces THIS script with the iframe.
    const script = document.createElement("script");
    script.async = true;
    script.src = "https://telegram.org/js/telegram-widget.js?22";
    script.setAttribute("data-telegram-login", botName);
    script.setAttribute("data-size", "large");
    script.setAttribute("data-radius", "12");
    script.setAttribute("data-onauth", "TelegramLoginWidget.onAuth");
    script.setAttribute("data-request-access", "write");
    el.appendChild(script);
  }, [botName, telegramLogin]);

  return <div ref={widgetRef} className="flex justify-center" />;
}
