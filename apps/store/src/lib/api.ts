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

async function api<T>(path: string, options?: RequestInit): Promise<T> {
  const token = localStorage.getItem("token");
  const res = await fetch(`${API_URL}${path}`, {
    ...options,
    headers: {
      "Content-Type": "application/json",
      ...(token ? { Authorization: `Bearer ${token}` } : {}),
      ...options?.headers,
    },
  });
  if (!res.ok) {
    const body = await res.json().catch(() => ({ error: "Network error" }));
    throw new Error((body as ApiError).error || "Request failed");
  }
  return res.json();
}

/** API wrapper that reads customer_token (store customers, not admin users) */
async function customerApi<T>(path: string, options?: RequestInit): Promise<T> {
  const token = localStorage.getItem("customer_token");
  const res = await fetch(`${API_URL}${path}`, {
    ...options,
    headers: {
      "Content-Type": "application/json",
      ...(token ? { Authorization: `Bearer ${token}` } : {}),
      ...options?.headers,
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

export interface CustomerTokenResponse {
  access_token: string;
  token_type: string;
}

export interface CustomerProfile {
  id: number;
  email: string;
  first_name: string;
  last_name: string;
  phone: string;
  avatar_color: string;
}

export const fetchStoreRegister = (data: CustomerRegisterData) =>
  api<CustomerTokenResponse>("/api/store/auth/register", {
    method: "POST",
    body: JSON.stringify(data),
  });

export const fetchStoreLogin = (data: CustomerLoginData) =>
  api<CustomerTokenResponse>("/api/store/auth/login", {
    method: "POST",
    body: JSON.stringify(data),
  });

export const fetchStoreCustomerMe = () =>
  customerApi<CustomerProfile>("/api/store/customers/me");

export const fetchStoreCustomerUpdate = (data: Record<string, string>) =>
  customerApi<{ ok: true }>("/api/store/customers/me", {
    method: "PUT",
    body: JSON.stringify(data),
  });

export const fetchStoreCustomerChangePassword = (data: {
  current_password: string;
  new_password: string;
}) =>
  customerApi<{ ok: true }>("/api/store/customers/me/password", {
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
