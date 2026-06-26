import type { Product, Category } from "@/types";

// Mirror apps/admin/src/lib/api.ts: branch on production explicitly so a
// custom MODE (staging/preview) resolves the same way in both SPAs. In
// production VITE_API_URL must be set at build time; we fall back to
// api.mioru.store if absent. Outside production, "" → same-origin Vite proxy.
const API_URL =
  import.meta.env.MODE === "production"
    ? import.meta.env.VITE_API_URL || "https://api.mioru.store"
    : import.meta.env.VITE_API_URL || "";

export function getImageUrl(path: string): string {
  if (!path) return "";
  if (path.startsWith("http")) return path;
  return `${API_URL}${path}`;
}

// getThumbUrl returns the thumbnail URL for an image path.
// Thumbnails follow the naming convention: /uploads/xxx.png → /uploads/thumb_xxx.png
export function getThumbUrl(path: string): string {
  if (!path) return "";
  const base = getImageUrl(path);
  // Replace last path segment: xxx.png → thumb_xxx.png
  const i = base.lastIndexOf("/");
  if (i < 0) return base;
  return base.slice(0, i + 1) + "thumb_" + base.slice(i + 1);
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

// CatalogQuery mirrors the backend's storefront filter contract. Multi-value
// fields (brand/color/size) come through as repeated query keys; scalar
// fields (sort, page, per_page, category_id, search, price_min, price_max)
// stay as strings so the caller doesn't have to remember which is which.
export type CatalogQuery = Partial<{
  category_id: string | string[];
  search: string;
  sort: string;
  page: string;
  per_page: string;
  price_min: string;
  price_max: string;
  brand: string[];
  color: string[];
  size: string[];
  status: string; // "in_stock" | "preorder" | "out_of_stock"
}>;

export interface ProductsPage {
  products: Product[];
  total: number;
  page: number;
  per_page: number;
}

export interface ProductFacets {
  brands: string[];
  colors: string[];
  sizes: string[];
}

function buildCatalogParams(query: CatalogQuery): string {
  const params = new URLSearchParams();
  for (const [key, value] of Object.entries(query)) {
    if (value == null) continue;
    if (Array.isArray(value)) {
      for (const v of value) {
        if (v) params.append(key, v);
      }
    } else if (value !== "") {
      params.append(key, value);
    }
  }
  return params.toString();
}

export const fetchStoreProducts = (query: CatalogQuery = {}) => {
  const qs = buildCatalogParams(query);
  return api<ProductsPage>("/api/products" + (qs ? "?" + qs : ""));
};

export const fetchStoreProductFacets = (query: CatalogQuery = {}) => {
  // Facets follow the scope of the active catalog view: category, search,
  // price, AND the in-stock / preorder status toggle. Without `status`,
  // chip counts on the "Preorder" tab would mirror the "In stock" tab —
  // a real UX bug. The backend's ListFacets already keeps
  // `filter.Status` and totals products per status, so this just stops
  // the client from stripping it.
  const scope: CatalogQuery = {
    category_id: query.category_id,
    search: query.search,
    price_min: query.price_min,
    price_max: query.price_max,
    status: query.status,
  };
  const qs = buildCatalogParams(scope);
  return api<ProductFacets>("/api/products/facets" + (qs ? "?" + qs : ""));
};

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

// ── Orders ──

// Reviewer finding #3 (P0): the backend now returns a much
// wider payload — model.Order + items with product_slug and
// image_url, plus phone at the order level. Pre-fix the TS
// interface silently dropped every extra field, so the
// storefront's `OrderCard` in ProfilePage rendered orders
// without a phone chip, without product slugs for the
// "view product" link, without product images. Adding the
// fields here is a no-op at runtime (TS extras are already
// discarded by JSON.parse), but it makes the type contract
// honest so future regression tests can pin the wider
// payload.
export interface StoreOrderItem {
  id: number;
  product_id: number;
  product_name: string;
  product_slug: string;
  size_label: string;
  quantity: number;
  price_minor: number;
  image_url?: string;
}

export interface StoreOrder {
  id: number;
  total_minor: number;
  status: string;
  type: string;
  city: string;
  phone: string;
  delivery_method: string;
  payment_method: string;
  street: string;
  house: string;
  apartment: string;
  comment: string;
  height?: number;
  weight?: number;
  delivery_time?: string[];
  items?: StoreOrderItem[];
  created_at: string;
}

export const fetchStoreCustomerOrders = () =>
  api<{ orders: StoreOrder[]; total: number; page: number; per_page: number }>("/api/store/customers/me/orders");

// ── Orders (checkout) ──

export interface CreateOrderItem {
  product_id: number;
  size_label: string;
  quantity: number;
  price_minor: number;
}

export interface CreateOrderData {
  type: string;
  /** Contact phone for the order. Required on every checkout. */
  phone: string;
  city: string;
  delivery_method: string;
  payment_method: string;
  street?: string;
  house?: string;
  apartment?: string;
  comment?: string;
  total_minor: number;
  height?: number;
  weight?: number;
  delivery_time?: string[];
  photos?: string[];
  items?: CreateOrderItem[];
}

export const createOrder = (data: CreateOrderData, idempotencyKey: string) =>
  api<{ id: number; status: string; created_at: string }>("/api/store/orders", {
    method: "POST",
    body: JSON.stringify(data),
    headers: { "Idempotency-Key": idempotencyKey },
  });

/** Upload a photo for an individual order. Returns the file URL path (e.g. /uploads/xxx.png). */
export const uploadOrderPhoto = async (file: File): Promise<string> => {
  const formData = new FormData();
  formData.append("file", file);

  const method = "POST";
  const headers: Record<string, string> = {};
  const csrf = readCookie(CSRF_COOKIE);
  if (csrf) headers["X-CSRF-Token"] = csrf;

  const res = await fetch(`${API_URL}/api/store/orders/upload-photo`, {
    method,
    credentials: "include",
    body: formData,
    headers,
  });
  if (!res.ok) {
    const body = await res.json().catch(() => ({ error: "Upload failed" }));
    throw new Error((body as ApiError).error || "Upload failed");
  }
  const data = await res.json();
  return data.url;
};

// ── OAuth ──

export interface TelegramAuthData {
  id: number;
  first_name: string;
  last_name?: string;
  username?: string;
  photo_url?: string;
  auth_date: number;
  hash: string;
}

export const fetchTelegramLogin = (data: TelegramAuthData) =>
  api<CustomerProfile>("/api/store/auth/telegram", {
    method: "POST",
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

// ── Cart & Favorites sync ──

export interface CartSyncItem {
  product_id: number;
  size_label: string;
  quantity: number;
  height_cm?: number;
  weight_kg?: number;
}

export const fetchCustomerCart = () =>
  api<{ items: CartSyncItem[] }>("/api/store/customers/me/cart");

export const saveCustomerCart = (items: CartSyncItem[]) =>
  api<{ ok: true }>("/api/store/customers/me/cart", {
    method: "PUT",
    body: JSON.stringify({ items }),
  });

export const fetchCustomerFavorites = () =>
  api<{ product_ids: number[] }>("/api/store/customers/me/favorites");

export const saveCustomerFavorites = (product_ids: number[]) =>
  api<{ ok: true }>("/api/store/customers/me/favorites", {
    method: "PUT",
    body: JSON.stringify({ product_ids }),
  });

export { api, API_URL };
