import type { Product, Category } from "@/types";

const API_URL = import.meta.env.VITE_API_URL || "https://api.mioru.store";

export function getImageUrl(path: string): string {
  if (!path) return "";
  if (path.startsWith("http")) return path;
  return `${API_URL}${path}`;
}

interface ApiError {
  error: string;
}

// CSRF_COOKIE matches cookieauth.StoreCSRFCookie on the backend.
const CSRF_COOKIE = "store_csrf";

// readCookie returns the value of the named cookie or null. Used to pick up
// the CSRF token the backend set on customer login — the auth cookie itself
// is HttpOnly and intentionally invisible to JS.
function readCookie(name: string): string | null {
  const prefix = `${name}=`;
  for (const raw of document.cookie.split(";")) {
    const trimmed = raw.trim();
    if (trimmed.startsWith(prefix)) {
      return decodeURIComponent(trimmed.slice(prefix.length));
    }
  }
  return null;
}

// Methods the backend's CSRF middleware refuses without a token. Keep in
// sync with middleware/csrf.go (anything not in the safe-list).
const MUTATING_METHODS = new Set(["POST", "PUT", "PATCH", "DELETE"]);

// api drives both unauthenticated public catalog requests and authenticated
// customer requests. Cookie-only auth: AuthMW reads the JWT from an HttpOnly
// cookie; the SPA opts in to sending cookies cross-origin (store.mioru.store
// ↔ api.mioru.store) via credentials: include. Mutations also echo the CSRF
// cookie back in the X-CSRF-Token header.
async function api<T>(path: string, options?: RequestInit): Promise<T> {
  const headers: Record<string, string> = {
    "Content-Type": "application/json",
  };

  const method = (options?.method || "GET").toUpperCase();
  if (MUTATING_METHODS.has(method)) {
    const csrf = readCookie(CSRF_COOKIE);
    if (csrf) {
      headers["X-CSRF-Token"] = csrf;
    }
  }

  const res = await fetch(`${API_URL}${path}`, {
    credentials: "include",
    ...options,
    headers: {
      ...headers,
      ...((options?.headers as Record<string, string>) || {}),
    },
  });
  if (!res.ok) {
    const body = await res.json().catch(() => ({ error: "Network error" }));
    throw new Error((body as ApiError).error || "Request failed");
  }
  return res.json();
}

// ── Public catalog ──

export const fetchStoreProducts = (params: Record<string, string>) =>
  api<{ products: Product[]; total: number }>(
    "/api/products?" + new URLSearchParams(params),
  );

export const fetchStoreProduct = (slug: string) =>
  api<Product>("/api/products/" + slug);

export const fetchStoreCategories = () => api<Category[]>("/api/categories");

// ── Store customer auth ──

export interface CustomerRegisterData {
  email: string;
  password: string;
  first_name: string;
  last_name: string;
  phone?: string;
}

export interface CustomerLoginData {
  email: string;
  password: string;
}

// CustomerProfile is the public profile shape the backend returns from
// register / login (cookies carry the session — no token in the JSON body).
export interface CustomerProfile {
  id: number;
  email: string;
  first_name: string;
  last_name: string;
  phone: string;
  avatar_color: string;
}

export const fetchStoreRegister = (data: CustomerRegisterData) =>
  api<CustomerProfile>("/api/store/auth/register", {
    method: "POST",
    body: JSON.stringify(data),
  });

export const fetchStoreLogin = (data: CustomerLoginData) =>
  api<CustomerProfile>("/api/store/auth/login", {
    method: "POST",
    body: JSON.stringify(data),
  });

export const fetchStoreLogout = () =>
  api<{ ok: true }>("/api/store/auth/logout", { method: "POST" });

export const fetchStoreCustomerMe = () =>
  api<CustomerProfile>("/api/store/customers/me");

export const fetchStoreCustomerUpdate = (data: Record<string, string>) =>
  api<{ ok: true }>("/api/store/customers/me", {
    method: "PUT",
    body: JSON.stringify(data),
  });

export const fetchStoreCustomerChangePassword = (data: {
  current_password: string;
  new_password: string;
}) =>
  api<{ ok: true }>("/api/store/customers/me/password", {
    method: "PUT",
    body: JSON.stringify(data),
  });

/** Map a category slug or name to an emoji icon */
export function getCategoryEmoji(slugOrName: string): string {
  const s = slugOrName.toLowerCase();
  if (s.includes("sneaker") || s.includes("кроссовк") || s.includes("adidași"))
    return "👟";
  if (
    s.includes("slide") ||
    s.includes("тапк") ||
    s.includes("slapi") ||
    s.includes("papuci")
  )
    return "🩴";
  if (
    s.includes("tshirt") ||
    s.includes("tee") ||
    s.includes("футболк") ||
    s.includes("tricou")
  )
    return "👕";
  if (
    s.includes("short") ||
    s.includes("шорт") ||
    s.includes("pantaloni scurti")
  )
    return "🩳";
  if (s.includes("bracelet") || s.includes("браслет") || s.includes("brățară"))
    return "⛓️";
  return "👤";
}

export { api, API_URL };
