# Public API contract reference

Source of truth for the integration-test cases. Verified against handler code on 2026-06-13.
Routes registered in `backend/api/cmd/server/main.go`. Error envelope: `{error, code}`.
`jsonError` derives the machine code from HTTP status (400→`VALIDATION_FAILED`, 401→`AUTH_INVALID`,
403→`FORBIDDEN`/`CSRF_INVALID`, 404→`NOT_FOUND`, 409→`CONFLICT`, 500→`INTERNAL`); `jsonErrorCode`
sets it explicitly.

## Storefront auth (no CSRF, rate-limited; bootstrap session)

- **POST /api/store/auth/register** — `customerH.Register`, JSON. Required: `email` (regex), `password` (8–72, letter+digit, not common), `first_name` (non-empty). Optional `last_name`, `phone` (if set: `^\+373\d{8}$` — `+373` plus exactly 8 digits, KAN-53; the pre-KAN-53 `^\+?\d{7,15}$` no longer validates). Success **201** + body `{id,email,first_name,last_name,phone,avatar_color,telegram}` (`telegram` is `{linked:false}` on a fresh account); sets `store_auth`(HttpOnly)+`store_csrf`. Errors: dup email→**409 CONFLICT**; bad field→**400 VALIDATION_FAILED**.
- **POST /api/store/auth/login** — `customerH.Login`, JSON `{email,password}`. `email`≤100,`password`≤72 (over→401 before lookup). Constant-time dummy hash on missing customer. Success **200** + profile (incl. `telegram` binding, read from `customer_oauth` — never hardcoded, the SPA gates checkout on it) + cookies. Wrong pw / missing user → **401 AUTH_INVALID**.
- **POST /api/store/auth/telegram** — `customerH.TelegramLogin`, JSON `{id>0,first_name,last_name?,username?,photo_url?,auth_date>0,hash}`. Security: `auth.VerifyTelegramAuth(data, botToken, 24h, now)` HMAC-SHA256. No bot token configured → **503 INTERNAL**. Bad/old signature → **401 AUTH_INVALID**. Existing oauth row → **200**; new → **201**; both set cookies and report `telegram:{linked:true,username,first_name}`.
- **POST /api/store/auth/logout** — `customerH.Logout`, behind `customerAuthMW`+`customerCSRF`. **200** `{ok:true}`, clears both cookies. No/bad token→401; CSRF mismatch→**403 CSRF_INVALID**.

## Storefront profile (cookie `store_auth`, CSRF on mutations)

- **GET /api/store/customers/me** — `customerH.Me`. **200** `{id,email,first_name,last_name,phone,avatar_color,telegram}`, where `telegram` is `{linked:bool,username?,first_name?}` derived from the customer's `customer_oauth` rows (`profile_data` is best-effort — a missing username is fine).
- **PUT /api/store/customers/me** — `customerH.UpdateProfile`, JSON map. Requires `current_password` (wrong→**401 AUTH_INVALID**, missing→**401 AUTH_REQUIRED**). Updatable: `first_name`≤100,`last_name`≤100,`phone`≤30 **and `^\+373\d{8}$` when non-empty** (empty clears it),`avatar_color`≤20. Unknown key / empty map / too long / bad phone format → **400 VALIDATION_FAILED**. Success **200** `{ok:true}`.
- **PUT /api/store/customers/me/password** — `customerH.ChangePassword`, JSON `{current_password,new_password}`. Wrong current→**401**. `new_password` 8–72, must differ. **Bumps `password_changed_at`** → invalidates prior tokens. Success **200** `{ok:true}`.
- **POST /api/store/customers/me/set-password** — `customerH.SetPassword`. Only OAuth-only customers (`hashed_password=''`); else **409 CONFLICT**. `{new_password}` 8–72 strong. Bumps `password_changed_at`. **200** `{ok:true}`.
- **POST /api/store/customers/me/oauth** — `customerH.LinkOAuth`, JSON. `provider` (≤50, required), `profile_data`≤2000, optional (omitted → stored as `{}`; malformed JSON → **400 VALIDATION_FAILED**). For `provider="telegram"`: requires `id>0,first_name,auth_date>0,hash` and runs full `VerifyTelegramAuth` (unsigned/unowned → **401 AUTH_INVALID**) — prevents identity hijack; no bot token configured → **503** (payload shape is checked first, so a malformed body is still **400**); `profile_data` is **built server-side** from the signed `first_name/last_name/username/photo_url` and the client-supplied blob is ignored — the signature covers those fields, so the stored handle cannot be spoofed (`TelegramLogin` builds it the same way). For other providers: `oauth_id` (≤100, required), `profile_data` taken as given. Identity already bound to another customer → **409 CONFLICT**. Success **200** `{ok:true}`.
- **GET /api/store/customers/me/favorites** — `customerH.GetFavorites`. **200** `{product_ids:[...]}` (never null).
- **PUT /api/store/customers/me/favorites** — `customerH.SaveFavorites`, `{product_ids:[...]}`. ≤200 entries, each >0; else **400**. **200** `{ok:true}`.
- **GET /api/store/customers/me/orders** — `customerH.ListOrders`, paginated `{orders,total,page,per_page}`. Each order: `id,order_code,type,status,total_minor,phone,city,delivery_method,payment_method,street,house,apartment,comment,created_at` + `items[]` (batched product join: `product_id,product_name,product_slug,size_label,quantity,price_minor`, `image_url` when the product has one). **KAN-52:** on `type=individual` orders the response also carries `category` (`clothing|shoes|accessories`), `foot_length` (cm, nullable), `photos` (`/uploads/…` paths); cart orders omit these fields (stored as NULL). Admin-only fields (`customer_id`,`customer_email`,`customer_first_name`) are scrubbed. **401** without a session.
- **POST /api/store/orders** — `customerH.CreateOrder`, behind `customerAuthMW`+`storeCSRF`, `Idempotency-Key` required. **Gate:** the customer must hold a `telegram` row in `customer_oauth`; otherwise **403 TELEGRAM_REQUIRED** before the body is read (the storefront disables the confirm button, but the API is the enforcing side). Then validation (`phone` required and matching `^\+373\d{8}$`, PMR delivery rules), per-size stock decrement and the idempotency record inside one transaction. **KAN-52:** `type=individual` must carry `category` ∈ `{clothing,shoes,accessories}`; `shoes` requires `foot_length` (10–40 cm) and rejects height/weight, `accessories` rejects all measurements, `clothing` keeps optional height/weight. `category`/`foot_length` on any other order type → **400 VALIDATION_FAILED**. Success **201** order.
- **POST /api/store/orders/upload-photo** — `customerH.UploadOrderPhoto`, multipart field `file`, cap 10 MiB. PNG-only (`allowedImageExts={.png}` + sniff `image/png`). Success **200** `{url:"/uploads/...png"}`. >10MiB / no file / non-png / bad sniff → **400**.

## Admin auth (no CSRF, rate-limited)

- **POST /api/auth/login** — `authH.Login`, `{username,password}`. `username`≤100,`password`≤72 (over→401). Dummy hash on missing user. **200** `{id,username,email,display_name,role}` + `auth_token`(HttpOnly)+`csrf_token`. Wrong/missing → **401 AUTH_INVALID**.
- **POST /api/auth/register** — `authH.Register`, behind `authMW`+`RequireSuperAdmin`+`adminCSRF` (**super_admin only**). `{first_name,last_name,email,username(≥2,[a-zA-Z0-9_]),password(8–72 strong)}`. Role hardcoded `"admin"`. Success **201** (no cookie). Not super→**403 FORBIDDEN**; dup username/email→**400 VALIDATION_FAILED**.
- **POST /api/auth/forgot-password** — `authH.ForgotPassword`, `{email}`. **Always 200** `{message:...}` (no user enumeration). Token stored as SHA-256, TTL 1h. Bad email format→**400**.
- **POST /api/auth/reset-password** — `authH.ResetPassword`, `{token,password}`. Atomic single-use+expiry `DELETE ... RETURNING username`. Invalid/expired token / bad password→**400**. Success **200** `{ok:true}`; bumps `password_changed_at`.
- **POST /api/auth/logout** — `authH.Logout`, `authMW`+`adminCSRF`. **200** `{ok:true}`, clears cookies.

## Admin profile (cookie `auth_token`, CSRF on mutations)

- **GET /api/users/me** — `authH.Me`. **200** `{id,username,first_name,last_name,email,display_name,avatar_color,role}`.
- **PUT /api/users/me/profile** — `authH.UpdateProfile`, JSON map. Updatable: `display_name`(1–100),`avatar_color`≤20,`first_name`≤100,`last_name`≤100. **No current_password.** Unknown/empty→**400**. **200** `{ok:true}`.
- **PUT /api/users/me/password** — `authH.ChangePassword`, `{current_password,new_password}`. Wrong→401; new 8–72, must differ; bumps `password_changed_at`. **200** `{ok:true}`.

## Admin resources (cookie `auth_token`, CSRF on mutations)

- **GET /api/admin/users** — `authH.ListUsers`, super-admin only.
- **DELETE /api/admin/users/{username}** — `authH.DeleteUser`, super-admin. Caller==target→**400 VALIDATION_FAILED**; not found→**404 NOT_FOUND**; success **204** (no body). No last-admin guard.
- **GET /api/admin/categories** — `productH.Categories`, admin. `?flat=1`→flat list w/ `parent_id`; else tree. **200** `[]model.Category`.
- **GET /api/admin/products** — `productH.List`, admin. `category_id,search,brand,sort,page,per_page`. **200** `{products:[...],total,page,per_page}` (key `products`).
- **POST /api/admin/products** — `productH.Create`, admin, **multipart**. Required `slug,name,category_id>0`. Status normalized (`in_stock|preorder|out_of_stock`). **Brands (KAN-14):** multipart `brands[]` array — each ≤60 chars, max 5, trimmed/deduped; the legacy single `brand` field is accepted as a one-element fallback. Response product carries `brand` (display name `A x B`, derived as `array_to_string(brands,' x ')`) + `brands` (array). Success **201** `{id,product}`.
- **GET /api/admin/products/{slug}** — `productH.Get`, admin. **200** product; missing→404.
- **PUT /api/admin/products/{slug}** — `productH.Update`, admin, multipart (cap 32MiB). Required `slug,name,category_id`. Brands: same `brands[]` contract as Create (KAN-14). Not found→**404 NOT_FOUND**; dup slug→**409 CONFLICT**; `existing_images[]`≤20 each ≤500 & prefix `/uploads/`|`https://`. Success **200** full product.
- **DELETE /api/admin/products/{slug}** — `productH.Delete`, admin. Success **200** `{ok:true}` (not 204); missing→**404**.
- **GET /api/admin/orders** — `adminOrderH.ListAll`, admin. `page,per_page(≤100),status?,type?`. **200** `{orders:[...],total,page,per_page}`; each order carries joined `customer_email`,`customer_first_name`; items batch-loaded with `product_name`. Cross-customer (all orders).
- **PATCH /api/admin/orders/{id}/status** — `adminOrderH.UpdateStatus`, admin. `{status}` ∈ `{pending,processing,shipped,delivered,cancelled}`. Non-numeric id / empty / not-in-enum → **400 VALIDATION_FAILED**. Success **200** `{ok:true}`.
- **POST /api/admin/upload** — `productH.Upload`, admin, multipart field `file`, cap 10MiB. **PNG-only** (sniff `image/png`; SVG rejected; error message misleadingly lists more types). Success **200** `{url}`. No file / non-png / bad sniff → **400** (no 413 — multipart-parse failure is 400).

## Store public / infra

- **GET /api/categories** — `storeH.ListCategories`. **200** category **tree** (`id,parent_id,name,slug,criteria,sort_order,cover_image?,children[]`). KAN-55: each node also carries `products_count` — products in the category **and all its descendants** (recursive), powering the catalog chip badges.
- **GET /api/products/facets** — `storeH.ListFacets`. Same params as `/api/products`. **Drops** `brand/brands/colors/sizes` before facet query (selecting a facet does not hide its siblings); keeps `status`. **200** `{brands,colors,sizes}`. KAN-14: the brands facet lists each collaboration brand individually (`unnest(brands)`), never the joined `A x B` string. Bad status→**400**.
- **GET /api/health** — inline, no mw. **200** `{status:"ok"}`.
- **GET /uploads/** — `uploadsSecurity`+FileServer. Sets `X-Content-Type-Options: nosniff` and `Content-Security-Policy: default-src 'none'; sandbox`. Path traversal blocked by stdlib `http.FileServer`/`path.Clean`.
- **OPTIONS /api/** — inline preflight. Allowlisted `Origin`→`Access-Control-Allow-Origin: <origin>` + `Allow-Credentials: true` + `Vary: Origin`; non-allowlisted→no ACAO. Always `Allow-Methods: GET,POST,PUT,PATCH,DELETE,OPTIONS`, `Allow-Headers: Content-Type, X-CSRF-Token, Idempotency-Key`. **204**.

## Harness note

`integration_harness_test.go` mints sessions directly (`customerSession`/`userSession` insert a row + `auth.CreateToken`), it does NOT capture `Set-Cookie`. Login/register flow tests need a small helper to read `rr.Result().Cookies()` and build a `*session` from `store_auth`/`store_csrf` (or `auth_token`/`csrf_token`). Telegram/OAuth happy-paths need a `customerH` built with a known bot token + a valid HMAC computed via the same `auth.VerifyTelegramAuth` algorithm.
