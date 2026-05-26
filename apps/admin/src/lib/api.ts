import type {
  User,
  Product,
  Category,
  ProductFilter,
  ProductsResponse,
} from "@/types";

const API_URL = import.meta.env.VITE_API_URL || "";

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
  const headers: Record<string, string> = {};

  if (!(options?.body instanceof FormData)) {
    headers["Content-Type"] = "application/json";
  }

  if (token) {
    headers["Authorization"] = `Bearer ${token}`;
  }

  const res = await fetch(`${API_URL}${path}`, {
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

export const login = (username: string, password: string) =>
  api<{ access_token: string }>("/api/auth/login", {
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

export { api, API_URL };
