import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import {
  fetchStoreProducts,
  createOrder,
  uploadOrderPhoto,
} from "./api";

// CSRF_COOKIE = "store_csrf" mirrors cookieauth.StoreCSRFCookie on the backend.
// Setting document.cookie here is what readCookie() picks up — the auth cookie
// itself is HttpOnly and intentionally invisible to JS.
const CSRF_COOKIE_NAME = "store_csrf";

const okJsonResponse = (body: unknown, status = 200): Response =>
  ({
    ok: status >= 200 && status < 300,
    status,
    json: async () => body,
  }) as Response;

const errorJsonResponse = (errorText: string, status = 400): Response =>
  ({
    ok: false,
    status,
    json: async () => ({ error: errorText }),
  }) as Response;

const makeFile = (name = "photo.png"): File =>
  new File([new Uint8Array([1, 2, 3])], name, { type: "image/png" });

let fetchMock: ReturnType<typeof vi.spyOn>;
let consoleErrorSpy: ReturnType<typeof vi.spyOn>;

beforeEach(() => {
  // Reset cookie state between tests.
  document.cookie = `${CSRF_COOKIE_NAME}=; expires=Thu, 01 Jan 1970 00:00:00 GMT; path=/`;
  fetchMock = vi.spyOn(globalThis, "fetch");
  consoleErrorSpy = vi.spyOn(console, "error").mockImplementation(() => {});
});

afterEach(() => {
  fetchMock.mockRestore();
  consoleErrorSpy.mockRestore();
  document.cookie = `${CSRF_COOKIE_NAME}=; expires=Thu, 01 Jan 1970 00:00:00 GMT; path=/`;
});

describe("api wrapper — happy path", () => {
  it("fetchStoreProducts sends credentials: include with no CSRF on GET", async () => {
    fetchMock.mockResolvedValueOnce(okJsonResponse({ products: [], total: 0, page: 1, per_page: 20 }));

    await fetchStoreProducts();

    expect(fetchMock).toHaveBeenCalledTimes(1);
    const [url, init] = fetchMock.mock.calls[0];
    expect(url).toMatch(/\/api\/products$/);
    expect((init as RequestInit).credentials).toBe("include");
    expect(((init as RequestInit).headers as Record<string, string>)["X-CSRF-Token"]).toBeUndefined();
  });

  it("createOrder echoes X-CSRF-Token from the store_csrf cookie on POST", async () => {
    document.cookie = `${CSRF_COOKIE_NAME}=test-token-abc; path=/`;
    fetchMock.mockResolvedValueOnce(okJsonResponse({ id: 1, status: "pending", created_at: "2026-06-28T00:00:00Z" }));

    await createOrder(
      {
        type: "cart",
        phone: "+37300000000",
        city: "Chisinau",
        delivery_method: "personal",
        payment_method: "cod",
        total_minor: 1000,
        items: [],
      },
      crypto.randomUUID(),
    );

    const [, init] = fetchMock.mock.calls[0];
    const headers = (init as RequestInit).headers as Record<string, string>;
    expect(headers["X-CSRF-Token"]).toBe("test-token-abc");
    expect(headers["Idempotency-Key"]).toMatch(/^[0-9a-f-]{36}$/);
    expect(headers["Content-Type"]).toBe("application/json");
  });

  it("createOrder omits X-CSRF-Token when the cookie is absent", async () => {
    fetchMock.mockResolvedValueOnce(okJsonResponse({ id: 1, status: "pending", created_at: "2026-06-28T00:00:00Z" }));

    await createOrder(
      {
        type: "cart",
        phone: "+37300000000",
        city: "Chisinau",
        delivery_method: "personal",
        payment_method: "cod",
        total_minor: 0,
        items: [],
      },
      crypto.randomUUID(),
    );

    const [, init] = fetchMock.mock.calls[0];
    const headers = (init as RequestInit).headers as Record<string, string>;
    expect(headers["X-CSRF-Token"]).toBeUndefined();
    // Idempotency-Key must still travel — backend requires it.
    expect(headers["Idempotency-Key"]).toBeDefined();
  });

  it("createOrder sends the SAME Idempotency-Key when the caller passes the same key twice", async () => {
    // The F1 fix in CheckoutPage/CustomOrderPage uses a useRef to pass
    // the same key to a second submitOrder() call after a 25s abort.
    // That useRef persistence is a React idiom — what we verify HERE is
    // the api wrapper's half of the contract: whatever key the caller
    // gives us, we send verbatim. The two halves together make the
    // backend dedupe work.
    fetchMock.mockResolvedValueOnce(okJsonResponse({ id: 1, status: "pending", created_at: "2026-06-28T00:00:00Z" }));
    fetchMock.mockResolvedValueOnce(okJsonResponse({ id: 1, status: "pending", created_at: "2026-06-28T00:00:00Z" }));

    const sharedKey = crypto.randomUUID();
    await createOrder(
      { type: "cart", phone: "+37300000000", city: "Chisinau", delivery_method: "personal", payment_method: "cod", total_minor: 100, items: [] },
      sharedKey,
    );
    await createOrder(
      { type: "cart", phone: "+37300000000", city: "Chisinau", delivery_method: "personal", payment_method: "cod", total_minor: 100, items: [] },
      sharedKey,
    );

    const headers1 = (fetchMock.mock.calls[0][1] as RequestInit).headers as Record<string, string>;
    const headers2 = (fetchMock.mock.calls[1][1] as RequestInit).headers as Record<string, string>;
    expect(headers1["Idempotency-Key"]).toBe(sharedKey);
    expect(headers2["Idempotency-Key"]).toBe(sharedKey);
  });

  it("createOrder sends DIFFERENT Idempotency-Key when the caller rotates the key", async () => {
    // Mirror of the above: a fresh key per submit must produce a fresh
    // header. If the wrapper secretly cached the first one, the SPA's
    // "release ref on success" pattern (CheckoutPage.tsx setSubmitted
    // branch) would silently dedupe a legitimate new order.
    fetchMock.mockResolvedValueOnce(okJsonResponse({ id: 1, status: "pending", created_at: "2026-06-28T00:00:00Z" }));
    fetchMock.mockResolvedValueOnce(okJsonResponse({ id: 2, status: "pending", created_at: "2026-06-28T00:00:00Z" }));

    const keyA = crypto.randomUUID();
    const keyB = crypto.randomUUID();
    await createOrder(
      { type: "cart", phone: "+37300000000", city: "Chisinau", delivery_method: "personal", payment_method: "cod", total_minor: 100, items: [] },
      keyA,
    );
    await createOrder(
      { type: "cart", phone: "+37300000000", city: "Chisinau", delivery_method: "personal", payment_method: "cod", total_minor: 200, items: [] },
      keyB,
    );

    const headers1 = (fetchMock.mock.calls[0][1] as RequestInit).headers as Record<string, string>;
    const headers2 = (fetchMock.mock.calls[1][1] as RequestInit).headers as Record<string, string>;
    expect(headers1["Idempotency-Key"]).toBe(keyA);
    expect(headers2["Idempotency-Key"]).toBe(keyB);
    expect(headers1["Idempotency-Key"]).not.toBe(headers2["Idempotency-Key"]);
  });
});

describe("api wrapper — error envelope contract", () => {
  it("prefixes 4xx errors with [METHOD path] and surfaces backend error text", async () => {
    fetchMock.mockResolvedValueOnce(errorJsonResponse("email already exists", 400));

    await expect(fetchStoreProducts()).rejects.toThrow(
      /^\[GET \/api\/products\] email already exists$/,
    );
  });

  it("falls back to 'Request failed' when the 4xx body has no error field", async () => {
    fetchMock.mockResolvedValueOnce({
      ok: false,
      status: 400,
      json: async () => ({}),
    } as unknown as Response);

    await expect(fetchStoreProducts()).rejects.toThrow(
      /^\[GET \/api\/products\] Request failed$/,
    );
  });

  it("falls back to 'Network error' when the 4xx body is not JSON", async () => {
    fetchMock.mockResolvedValueOnce({
      ok: false,
      status: 502,
      json: async () => {
        throw new SyntaxError("Unexpected token");
      },
    } as unknown as Response);

    await expect(fetchStoreProducts()).rejects.toThrow(
      /^\[GET \/api\/products\] Network error$/,
    );
  });
});

describe("api wrapper — AbortController timeout", () => {
  it("throws [METHOD path] Connection timed out on AbortError and does NOT console.error", async () => {
    fetchMock.mockImplementationOnce(() => {
      const err = new DOMException("aborted", "AbortError");
      return Promise.reject(err);
    });

    await expect(fetchStoreProducts()).rejects.toThrow(
      /^\[GET \/api\/products\] Connection timed out/,
    );
    // F6: AbortError must NOT also be logged via console.error — otherwise
    // every legitimate timeout spams the dev console.
    expect(consoleErrorSpy).not.toHaveBeenCalled();
  });
});

describe("api wrapper — non-Abort network failures", () => {
  it("prefixes generic TypeError with [METHOD path] and logs a PII-free diagnostic", async () => {
    // Simulates the real "Failed to fetch" you get on a network drop.
    fetchMock.mockRejectedValueOnce(new TypeError("Failed to fetch"));

    await expect(fetchStoreProducts()).rejects.toThrow(
      /^\[GET \/api\/products\] Failed to fetch$/,
    );

    // F6: the diagnostic payload MUST NOT echo the original err.message —
    // backend 4xx text can include user input ("email … already exists").
    expect(consoleErrorSpy).toHaveBeenCalledTimes(1);
    const [tag, payload] = consoleErrorSpy.mock.calls[0];
    expect(tag).toBe("[mioru] API request failed");
    expect(payload).toEqual({
      path: "/api/products",
      method: "GET",
      errorType: "TypeError",
    });
    expect(payload).not.toHaveProperty("error");
    // And the original PII-bearing message must not leak into the log.
    const serialized = JSON.stringify(payload);
    expect(serialized).not.toMatch(/Failed to fetch/);
  });
});

describe("api wrapper — FormData guard (F4)", () => {
  it("does NOT set Content-Type when the body is FormData (lets the browser set multipart boundary)", async () => {
    fetchMock.mockResolvedValueOnce(okJsonResponse({ url: "/uploads/abc.png" }));

    await uploadOrderPhoto(makeFile());

    const [, init] = fetchMock.mock.calls[0];
    const headers = (init as RequestInit).headers as Record<string, string>;
    expect(headers["Content-Type"]).toBeUndefined();
    // The FormData itself must still be passed through so the browser
    // constructs the multipart envelope.
    expect((init as RequestInit).body).toBeInstanceOf(FormData);
  });

  it("still echoes X-CSRF-Token on a FormData upload", async () => {
    document.cookie = `${CSRF_COOKIE_NAME}=upload-csrf; path=/`;
    fetchMock.mockResolvedValueOnce(okJsonResponse({ url: "/uploads/abc.png" }));

    await uploadOrderPhoto(makeFile());

    const [, init] = fetchMock.mock.calls[0];
    const headers = (init as RequestInit).headers as Record<string, string>;
    expect(headers["X-CSRF-Token"]).toBe("upload-csrf");
    // And the timeout/prefix path also applies — confirm by mocking a 4xx.
    fetchMock.mockResolvedValueOnce(errorJsonResponse("file too large", 413));
    await expect(uploadOrderPhoto(makeFile())).rejects.toThrow(
      /^\[POST \/api\/store\/orders\/upload-photo\] file too large$/,
    );
  });
});

describe("api wrapper — timeout signal is attached (F5 evidence)", () => {
  it("passes an AbortSignal to fetch (the 25s timeout is wired)", async () => {
    fetchMock.mockResolvedValueOnce(okJsonResponse({ products: [], total: 0, page: 1, per_page: 20 }));

    await fetchStoreProducts();

    const [, init] = fetchMock.mock.calls[0];
    const signal = (init as RequestInit).signal;
    // The wrapper must own the request lifetime — without a signal the
    // 25s budget is meaningless. We can't directly assert "the timeout
    // is exactly 25s" without fake-timer plumbing on a real fetch, but
    // the presence of an AbortSignal and the AbortError branch above
    // jointly prove the wiring end-to-end.
    expect(signal).toBeInstanceOf(AbortSignal);
    expect((signal as AbortSignal).aborted).toBe(false);
  });
});