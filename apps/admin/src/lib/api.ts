import type {
  User,
  Product,
  Category,
  ProductFilter,
  ProductsResponse,
} from "@/types";

// Dev/prod API URL.
// Vite 8.x bug: import.meta.env.DEV is stable false in dev mode, so the
// intended `??`-style fallback on DEV cannot be used. We branch on MODE
// instead, which is the documented workaround. In dev, VITE_API_URL=""
// falls through to an empty string → fetch("/api/...") is same-origin and
// hits the Vite proxy. In production, VITE_API_URL must be set explicitly
// at build time; we fall back to api.mioru.store if absent (build-time
// resolution — no runtime env var read on the server).
const API_URL =
  import.meta.env.MODE === "production"
    ? import.meta.env.VITE_API_URL || "https://api.mioru.store"
    : import.meta.env.VITE_API_URL || "";

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

// FETCH_TIMEOUT_MS mirrors apps/store/src/lib/api.ts — bounds every API
// call so a slow/lossy connection fails fast instead of hanging the SPA.
const FETCH_TIMEOUT_MS = 25_000;

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

  const controller = new AbortController();
  const timeout = setTimeout(() => controller.abort(), FETCH_TIMEOUT_MS);

  try {
    const res = await fetch(`${API_URL}${path}`, {
      credentials: "include",
      ...options,
      // Timeout signal must win over any caller-supplied signal so the
      // 25s budget is never silently disabled by a composable wrapper
      // (e.g. a future React useEffect cleanup-cancel). Merge via
      // AbortSignal.any so both still cancel the request.
      signal: options?.signal
        ? AbortSignal.any([controller.signal, options.signal])
        : controller.signal,
      headers: {
        ...headers,
        ...((options?.headers as Record<string, string>) || {}),
      },
    });

    if (!res.ok) {
      const body = await res.json().catch(() => ({ error: "Network error" }));
      // Throw without the [METHOD path] prefix — the catch below adds
      // it once. Prefixing here would produce "[GET /x] [GET /x] …".
      throw new Error((body as ApiError).error || "Request failed");
    }
    if (res.status === 204) return null as T;
    return res.json();
  } catch (err) {
    if (err instanceof DOMException && err.name === "AbortError") {
      throw new Error(
        `[${method} ${path}] Connection timed out — please check your internet and try again`,
      );
    }
    // Surface the diagnostic shape in the console so affected users can
    // report the exact cause (DNS failure, reset, CORS, etc.) — the user-
    // visible message is intentionally generic. We deliberately omit
    // err.message because backend 4xx envelopes can echo user input
    // ("email … already exists", "phone … invalid format"), which is PII
    // in any support screen-share.
    console.error("[mioru-admin] API request failed", {
      path,
      method: options?.method || "GET",
      errorType: err instanceof Error ? err.name : typeof err,
    });
    const msg = err instanceof Error ? err.message : String(err);
    throw new Error(`[${method} ${path}] ${msg}`);
  } finally {
    clearTimeout(timeout);
  }
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
  order_code: string;
  customer_id: number;
  customer_email: string;
  customer_first_name: string;
  /** Contact phone the customer typed at checkout. Always present
   *  (>= migration 012) — even for guest/anonymous checkouts the
   *  order row stores it. Managers click the chip to copy it to
   *  the clipboard for outbound calls. */
  phone: string;
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
  measurements?: Record<string, number | string>;
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

// Admin Customers workspace: a paginated list of every customer
// with rolled-up order stats and the Telegram link, if any. The
// shape mirrors the backend AdminCustomerRow in
// store/admin_customers.go — the Go json tags are snake_case so we
// match them here verbatim, no manual key remapping.
export interface AdminCustomer {
  id: number;
  email: string;
  first_name: string;
  last_name: string;
  phone: string;
  avatar_color: string;
  created_at: string;
  orders_count: number;
  total_spent_minor: number;
  last_order_at: string;
  telegram_linked: boolean;
  telegram_username: string;
  telegram_chat_id: string;
}

export interface AdminCustomersResponse {
  customers: AdminCustomer[];
  total: number;
  page: number;
  per_page: number;
}

export interface AdminCustomerOrderSummary {
  id: number;
  order_code: string;
  type: string;
  total_minor: number;
  status: string;
  created_at: string;
  items_count: number;
}

export interface AdminCustomerDetail {
  id: number;
  email: string;
  first_name: string;
  last_name: string;
  phone: string;
  avatar_color: string;
  created_at: string;
  updated_at: string;
  // password_changed_at is scrubbed server-side (json:"-") — do
  // not add it back. The field is PII-adjacent metadata and should
  // never reach the admin frontend. PR #52 review LOW #8 / I1.
  orders_count: number;
  total_spent_minor: number;
  first_order_at: string;
  last_order_at: string;
  telegram_linked: boolean;
  telegram_username: string;
  telegram_chat_id: string;
  orders: AdminCustomerOrderSummary[];
}

export const fetchAdminCustomers = (
  page = 1,
  perPage = 20,
  search?: string,
) => {
  const params = new URLSearchParams();
  params.set("page", String(page));
  params.set("per_page", String(perPage));
  if (search) params.set("search", search);
  return api<AdminCustomersResponse>(
    "/api/admin/customers?" + params.toString(),
  );
};

export const fetchAdminCustomerDetail = (id: number) =>
  api<AdminCustomerDetail>(`/api/admin/customers/${id}`);

export { api, API_URL };

// ─── Telegram admin workspace ────────────────────────────────────────────
//
// Types and fetchers for the /telegram route. The backend
// answers three endpoints:
//
//	GET  /api/admin/telegram/diagnose
//	GET  /api/admin/telegram/messages?page=&per_page=&status=
//	POST /api/admin/telegram/test
//
// `TelegramMessageRow` is the full row from the
// `telegram_messages` log table — text is the *exact* payload
// the bot tried to send (MarkdownV2), so the manager can
// inspect it when Telegram returned 400 "can't parse
// entities".
export interface TelegramMessageRow {
  id: number;
  order_id?: number;
  chat_id: string;
  text: string;
  parse_mode: string;
  status: "pending" | "sent" | "failed";
  http_status?: number;
  error?: string;
  telegram_message_id?: number;
  duration_ms?: number;
  sent_at: string;
}

export interface TelegramDiagnose {
  bot_token_set: boolean;
  bot_username: string;
  manager_chat_count: number;
  last_24h_total: number;
  last_24h_sent: number;
  last_24h_failed: number;
  last_send_at?: string;
  last_send_error?: string;
  last_send_status: "ok" | "failed" | "never" | "unknown";
}

export interface TelegramMessagesResponse {
  messages: TelegramMessageRow[];
  total: number;
  page: number;
  per_page: number;
}

export interface TelegramTestResult {
  chat_id: string;
  ok: boolean;
  status: "sent" | "failed";
  http_status?: number;
  error?: string;
  duration_ms: number;
}

export const fetchTelegramDiagnose = () =>
  api<TelegramDiagnose>("/api/admin/telegram/diagnose");

export const fetchTelegramMessages = (
  page = 1,
  perPage = 20,
  status?: string,
) => {
  const params = new URLSearchParams({ page: String(page), per_page: String(perPage) });
  if (status && status !== "all") params.set("status", status);
  return api<TelegramMessagesResponse>(`/api/admin/telegram/messages?${params}`);
};

export const sendTelegramTest = () =>
  api<{ results: TelegramTestResult[] }>("/api/admin/telegram/test", { method: "POST" });
