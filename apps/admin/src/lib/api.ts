import type {
  User,
  Product,
  Category,
  ProductFilter,
  ProductsResponse,
} from "@/types";

const API_URL = import.meta.env.VITE_API_URL || "https://api.mioru.store";

export function getImageUrl(path: string): string {
  if (!path) return "";
  if (path.startsWith("http")) return path;
  return `${API_URL}${path}`;
}

interface ApiError {
  error: string;
}

// CSRF_COOKIE matches cookieauth.AdminCSRFCookie on the backend.
const CSRF_COOKIE = "csrf_token";

// readCookie returns the value of the named cookie or null. Used to pick up
// the CSRF token the backend set on login — the auth cookie itself is
// HttpOnly and intentionally invisible to JS.
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

// Methods that the backend's CSRF middleware refuses without a token. Keep
// in sync with middleware/csrf.go (anything not in the safe-list).
const MUTATING_METHODS = new Set(["POST", "PUT", "PATCH", "DELETE"]);

async function api<T>(path: string, options?: RequestInit): Promise<T> {
  const headers: Record<string, string> = {};

  if (!(options?.body instanceof FormData)) {
    headers["Content-Type"] = "application/json";
  }

  // Cookie-only auth: AuthMW reads the JWT from an HttpOnly cookie. We must
  // opt in to sending cookies with cross-origin requests (admin SPA ↔
  // api.mioru.store) — fetch defaults to "same-origin".
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
  if (res.status === 204) return null as T;
  return res.json();
}

// ── Auth ──

// Login no longer returns a token: the backend sets HttpOnly auth + readable
// CSRF cookies on the response. The body carries a small public profile so
// the UI can render immediately without a follow-up /me round-trip.
export const login = (username: string, password: string) =>
  api<{
    id: number;
    username: string;
    email: string;
    display_name: string;
    role: string;
  }>("/api/auth/login", {
    method: "POST",
    body: JSON.stringify({ username, password }),
  });

// Registration is invite-only (an existing admin creates the account); it
// returns the new user's summary, not a session token.
export const register = (body: {
  first_name: string;
  last_name: string;
  email: string;
  username: string;
  password: string;
}) =>
  api<{ username: string; email: string; display_name: string; role: string }>(
    "/api/auth/register",
    {
      method: "POST",
      body: JSON.stringify(body),
    },
  );

export const logout = () =>
  api<{ ok: true }>("/api/auth/logout", { method: "POST" });

export const forgotPassword = (email: string) =>
  api<{ message: string }>("/api/auth/forgot-password", {
    method: "POST",
    body: JSON.stringify({ email }),
  });

export const resetPassword = (token: string, password: string) =>
  api<{ message: string }>("/api/auth/reset-password", {
    method: "POST",
    body: JSON.stringify({ token, password }),
  });

export const me = () => api<User>("/api/users/me");

export const updateUser = (body: Record<string, string>) =>
  api<User>("/api/users/me/profile", {
    method: "PUT",
    body: JSON.stringify(body),
  });

export const changePassword = (
  current_password: string,
  new_password: string,
) =>
  api<{ message: string }>("/api/users/me/password", {
    method: "PUT",
    body: JSON.stringify({ current_password, new_password }),
  });

// ── Products ──

export const fetchProducts = (params: Partial<ProductFilter>) => {
  const searchParams = new URLSearchParams();
  Object.entries(params).forEach(([key, value]) => {
    if (value !== undefined && value !== null && value !== "") {
      searchParams.set(key, String(value));
    }
  });
  return api<ProductsResponse>(
    `/api/admin/products?${searchParams.toString()}`,
  );
};

export const fetchProduct = (slug: string) =>
  api<Product>(`/api/admin/products/${slug}`);

export const createProduct = (data: FormData) =>
  api<Product>("/api/admin/products", {
    method: "POST",
    body: data,
  });

export const updateProduct = (slug: string, data: FormData) =>
  api<Product>(`/api/admin/products/${slug}`, {
    method: "PUT",
    body: data,
  });

export const deleteProduct = (slug: string) =>
  api<void>(`/api/admin/products/${slug}`, { method: "DELETE" });

export const fetchCategories = () =>
  api<Category[]>("/api/admin/categories?flat=1");

export const uploadImage = (file: File) => {
  const fd = new FormData();
  fd.append("file", file);
  return api<{ url: string }>("/api/admin/upload", {
    method: "POST",
    body: fd,
  });
};

// ── Admin orders ──

export interface AdminOrder {
  id: number;
  customer_id: number;
  customer_email: string;
  customer_first_name: string;
  type: string;
  total_minor: number;
  status: string;
  city: string;
  delivery_method: string;
  payment_method: string;
  street: string;
  house: string;
  apartment: string;
  comment: string;
  height?: number;
  weight?: number;
  delivery_time?: string[];
  photos?: string[];
  items?: AdminOrderItem[];
  created_at: string;
}

export interface AdminOrderItem {
  id: number;
  product_id: number;
  product_name: string;
  size_label: string;
  quantity: number;
  price_minor: number;
}

export interface AdminOrdersResponse {
  orders: AdminOrder[];
  total: number;
  page: number;
  per_page: number;
}

export const fetchAdminOrders = (page = 1, perPage = 20, status?: string, type?: string) => {
  const params = new URLSearchParams();
  params.set("page", String(page));
  params.set("per_page", String(perPage));
  if (status) params.set("status", status);
  if (type) params.set("type", type);
  return api<AdminOrdersResponse>("/api/admin/orders?" + params.toString());
};

export const updateOrderStatus = (orderId: number, status: string) =>
  api<{ ok: true }>(`/api/admin/orders/${orderId}/status`, {
    method: "PATCH",
    body: JSON.stringify({ status }),
  });

export { api, API_URL };
