# Deferred: httpOnly-cookie auth + CSRF (#30 item #8)

**Status:** designed, NOT implemented. Deferred from the 2026-05-26 autonomous
#30 security run for a supervised session.

**Why deferred:** this is a cross-cutting backend+frontend cutover for the
live store's login flow. The browser-runtime behaviour that decides whether
it works (cross-origin credentialed cookies, `SameSite`, dev-HTTP
`Secure`-cookie handling, CSRF rejection paths) cannot be validated by Go
unit tests or `npm run build` — only by clicking through the apps in a
browser against the running stack. The autonomous run had the explicit
escape "if it cannot be landed cleanly without breaking the apps, push the
green portion and STOP", and this is that escape.

The other 9 items of #30 shipped to `main`:
72615bb, f06cccb, 8745259, 1fd4111, f319e2d, 1a605b3, 9ac007d, 007d833, 651b254.

## Threat model (what the change buys us)

Today the JWT lives in `localStorage` on both apps. Any XSS that gets script
into either origin can exfiltrate the token. Moving to `HttpOnly` cookies
removes that exfiltration path. The cost is needing CSRF defence (because
cookies are ambient authority) — addressed below.

## Deployment context (relevant to design)

- Admin app: `VITE_API_URL=""` → calls `/api/*` on its **own origin**
  (nginx proxies to backend). Same-origin → cookies are trivially attached
  on `fetch`.
- Store app: `VITE_API_URL=https://api.mioru.store` → calls a **different
  origin** that shares the registrable domain `mioru.store`. By
  `SameSite` rules this is cross-origin but **same-site**, so
  `SameSite=Lax` cookies *are* attached on `fetch` to `api.mioru.store`
  provided the request is credentialed.
- Dev: store at `http://localhost:5173`, admin at `http://localhost:5174`,
  API at `http://localhost:8000`. All same-site (`localhost`). Browsers
  treat `localhost` as a secure context, so `Secure` cookies generally
  work over HTTP there, but the safe move is to gate `Secure` on
  `APP_ENV=production` (the env flag we added in #2).

## Backend design

1. **New cookies on login + customer register**
   - `auth_token` — the JWT. `HttpOnly; SameSite=Lax; Path=/;
     Secure (prod only); Max-Age=expiry`.
   - `csrf_token` — random 32-byte base64. NOT `HttpOnly` (the SPA must
     read it). `SameSite=Lax; Path=/; Secure (prod only); Max-Age=expiry`.
   - Use **separate** cookie names for admin (`auth_token` /
     `csrf_token`) and storefront (`store_auth` / `store_csrf`) so the
     two audiences don't collide when both apps run on the same eTLD+1.
2. **AuthMW / CustomerAuthMW** read the JWT from the cookie first, then
   fall back to `Authorization: Bearer` (transition window). The Bearer
   fallback can be removed once both frontends have switched.
3. **CSRF**: enforce on state-changing methods (`POST`/`PUT`/`PATCH`/
   `DELETE`) **only when the request authenticated via cookie** (Bearer
   requests skip CSRF — they have no ambient authority). Double-submit:
   compare `X-CSRF-Token` header against the `csrf_token` cookie value
   in constant time. Reject with 403 on mismatch. Login/register are
   exempt (no session yet).
4. **Logout endpoints** (`POST /api/auth/logout`, `POST /api/store/auth/logout`)
   set both cookies with `Max-Age=0; Expires=Thu, 01 Jan 1970…` to
   clear them.
5. **CORS** for the store's cross-origin path: `Access-Control-Allow-
   Credentials: true` (already reflect origin; verify and add the
   credentials header). Never `*` together with credentials.
6. **Keep** the existing JSON `access_token` in login responses for one
   transition release so existing clients aren't broken mid-deploy.

## Frontend design (both apps)

1. `src/lib/api.ts` — switch `fetch` to `credentials: 'include'`. Stop
   reading `localStorage.token`. On state-changing methods, read the CSRF
   cookie via `document.cookie` and add `X-CSRF-Token`.
2. `authStore` (admin) / customer store (store) — drop the in-memory
   token, drive auth state from a `GET /api/users/me` (or `/customers/me`)
   probe on app mount; cookies provide the credential.
3. `login` action: stop persisting `localStorage.token`. After a
   successful POST, hit `me` to populate state.
4. `logout` action: call the new logout endpoint, then clear in-memory
   state.
5. After self-password-change (item #7 already invalidates the current
   session), explicitly call logout + redirect to `/login` instead of
   relying on the next failing request.

## Rollout

1. Land the backend (additive — cookies set, Bearer still honoured, CSRF
   enforced only on cookie-auth). Green tests, no client change visible.
2. Land each frontend's switch. Verify in a browser: login, refresh, a
   mutation (CSRF), logout, password reset, cross-tab session, dev HTTP,
   prod HTTPS.
3. Remove the Bearer fallback and the JSON `access_token` from login
   responses once both frontends are out.

## Tests to write

- Backend (Go): cookie set on successful login (both audiences);
  AuthMW reads cookie when no Bearer; CSRF rejects mutation with missing
  / mismatching header; logout clears cookies; cookie has `HttpOnly`,
  `Secure` (prod), `SameSite=Lax`.
- Frontend (Vitest): `api()` sends `credentials:'include'`; CSRF header
  added on mutations only; no `localStorage.token` writes; logout calls
  the endpoint.
- Manual (cannot be automated here): login + mutation + logout on each
  app in dev and prod; cross-tab; SPA reload preserves auth.

## Risks captured

- `Access-Control-Allow-Credentials` misconfigured → store-app login
  appears to succeed but no subsequent call is authenticated.
- `Secure` cookie over dev HTTP → cookie silently dropped. Gate on
  `APP_ENV`.
- Forgetting to clear `localStorage.token` writes during the cutover →
  legacy XSS exfil path remains until the cookie-only path lands.
- CSRF enforcement turned on before the frontend sends the header →
  every mutation 403s. Mitigated by the "cookie-auth only" rule.
