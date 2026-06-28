import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import { login, me, updateUser } from "./api";

// CSRF_COOKIE = "csrf_token" mirrors cookieauth.AdminCSRFCookie on the
// backend. Setting document.cookie here is what readCookie() picks up —
// the auth cookie itself is HttpOnly and intentionally invisible to JS.
const CSRF_COOKIE_NAME = "csrf_token";

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

let fetchMock: ReturnType<typeof vi.spyOn>;
let consoleErrorSpy: ReturnType<typeof vi.spyOn>;

beforeEach(() => {
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
  it("me() sends credentials: include with no CSRF on GET", async () => {
    fetchMock.mockResolvedValueOnce(
      okJsonResponse({ id: 1, username: "admin", email: "a@x", display_name: "Admin", role: "admin" }),
    );

    await me();

    expect(fetchMock).toHaveBeenCalledTimes(1);
    const [url, init] = fetchMock.mock.calls[0];
    expect(url).toMatch(/\/api\/users\/me$/);
    expect((init as RequestInit).credentials).toBe("include");
    expect(((init as RequestInit).headers as Record<string, string>)["X-CSRF-Token"]).toBeUndefined();
  });

  it("login() echoes X-CSRF-Token from csrf_token cookie on POST", async () => {
    document.cookie = `${CSRF_COOKIE_NAME}=admin-csrf-abc; path=/`;
    fetchMock.mockResolvedValueOnce(
      okJsonResponse({ id: 1, username: "admin", email: "a@x", display_name: "Admin", role: "admin" }),
    );

    await login("admin", "x");

    const [, init] = fetchMock.mock.calls[0];
    const headers = (init as RequestInit).headers as Record<string, string>;
    expect(headers["X-CSRF-Token"]).toBe("admin-csrf-abc");
    expect(headers["Content-Type"]).toBe("application/json");
  });

  it("updateUser() omits X-CSRF-Token when the cookie is absent (still POST)", async () => {
    fetchMock.mockResolvedValueOnce(
      okJsonResponse({ id: 1, username: "admin", email: "a@x", display_name: "Admin", role: "admin" }),
    );

    await updateUser({ display_name: "Renamed" });

    const [, init] = fetchMock.mock.calls[0];
    const headers = (init as RequestInit).headers as Record<string, string>;
    expect(headers["X-CSRF-Token"]).toBeUndefined();
  });
});

describe("api wrapper — 204 No Content", () => {
  it("returns null on 204 (admin-specific path the store wrapper lacks)", async () => {
    fetchMock.mockResolvedValueOnce({
      ok: true,
      status: 204,
      json: async () => {
        throw new SyntaxError("no body");
      },
    } as unknown as Response);

    // logout() is the canonical 204 caller.
    const result = await (await import("./api")).logout();
    expect(result).toBeNull();
  });
});

describe("api wrapper — error envelope contract", () => {
  it("prefixes 4xx errors with [METHOD path] and surfaces backend error text", async () => {
    fetchMock.mockResolvedValueOnce(errorJsonResponse("username already taken", 400));

    await expect(login("admin", "x")).rejects.toThrow(
      /^\[POST \/api\/auth\/login\] username already taken$/,
    );
  });

  it("falls back to 'Request failed' when the 4xx body has no error field", async () => {
    fetchMock.mockResolvedValueOnce({
      ok: false,
      status: 400,
      json: async () => ({}),
    } as unknown as Response);

    await expect(login("admin", "x")).rejects.toThrow(
      /^\[POST \/api\/auth\/login\] Request failed$/,
    );
  });
});

describe("api wrapper — AbortController timeout", () => {
  it("throws [METHOD path] Connection timed out on AbortError and does NOT console.error", async () => {
    fetchMock.mockImplementationOnce(() => {
      const err = new DOMException("aborted", "AbortError");
      return Promise.reject(err);
    });

    await expect(me()).rejects.toThrow(/^\[GET \/api\/users\/me\] Connection timed out/);
    // F6: AbortError must NOT also be logged via console.error — otherwise
    // every legitimate timeout spams the dev console.
    expect(consoleErrorSpy).not.toHaveBeenCalled();
  });
});

describe("api wrapper — non-Abort network failures", () => {
  it("prefixes generic TypeError with [METHOD path] and logs a PII-free diagnostic", async () => {
    // Simulates the real "Failed to fetch" you get on a network drop.
    fetchMock.mockRejectedValueOnce(new TypeError("Failed to fetch"));

    await expect(me()).rejects.toThrow(/^\[GET \/api\/users\/me\] Failed to fetch$/);

    // F6: the diagnostic payload MUST NOT echo the original err.message —
    // backend 4xx text can include user input ("email … already exists").
    expect(consoleErrorSpy).toHaveBeenCalledTimes(1);
    const [tag, payload] = consoleErrorSpy.mock.calls[0];
    expect(tag).toBe("[mioru-admin] API request failed");
    expect(payload).toEqual({
      path: "/api/users/me",
      method: "GET",
      errorType: "TypeError",
    });
    expect(payload).not.toHaveProperty("error");
    // And the original PII-bearing message must not leak into the log.
    const serialized = JSON.stringify(payload);
    expect(serialized).not.toMatch(/Failed to fetch/);
  });
});

describe("api wrapper — timeout signal is attached (F5 evidence)", () => {
  it("passes an AbortSignal to fetch (the 25s timeout is wired)", async () => {
    fetchMock.mockResolvedValueOnce(
      okJsonResponse({ id: 1, username: "admin", email: "a@x", display_name: "Admin", role: "admin" }),
    );

    await me();

    const [, init] = fetchMock.mock.calls[0];
    const signal = (init as RequestInit).signal;
    expect(signal).toBeInstanceOf(AbortSignal);
    expect((signal as AbortSignal).aborted).toBe(false);
  });
});