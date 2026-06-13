# Public API inventory

Source of truth: `backend/api/cmd/server/main.go`. Columns: auth (cookie required),
CSRF (mutation gate), RL (rate-limited), success code, key error codes, and the
integration test that covers it (or "store-level" / "fake-unit" where covered
elsewhere).

## Health
| Method+Path | Auth | CSRF | RL | Success | Errors | Integration test |
|---|---|---|---|---|---|---|
| GET /api/health | — | — | — | 200 `{status:ok}` | — | (trivial, skip) |

## Store — public catalog (no auth)
| Method+Path | Auth | CSRF | RL | Success | Errors | Integration test |
|---|---|---|---|---|---|---|
| GET /api/products | — | — | — | 200 `{items,total,page,per_page}` | — | storefront: list+paginate |
| GET /api/products/facets | — | — | — | 200 `{brands,colors,sizes}` | — | storefront: facets |
| GET /api/products/{slug} | — | — | — | 200 product | 404 NOT_FOUND | storefront: get + 404 |
| GET /api/categories | — | — | — | 200 tree | — | storefront: categories |

## Store — customer auth & profile
| Method+Path | Auth | CSRF | RL | Success | Errors | Integration test |
|---|---|---|---|---|---|---|
| POST /api/store/auth/register | — | — | ✓ | 200 customer | 400 VALIDATION_FAILED, 409 CONFLICT | storefront: register happy |
| POST /api/store/auth/login | — | — | ✓ | 200 customer | 401 AUTH_INVALID | storefront: login happy + bad creds |
| POST /api/store/auth/telegram | — | — | ✓ | 200 customer | 401 AUTH_INVALID | (telegram sig — store-level) |
| POST /api/store/auth/logout | ✓ | ✓ | — | 204/200 | 403 CSRF_INVALID | storefront: logout + CSRF gate |
| GET /api/store/customers/me | ✓ | — | — | 200 customer | 401 AUTH_REQUIRED | storefront: me + 401 |
| PUT /api/store/customers/me | ✓ | ✓ | — | 200 customer | 401/403 | storefront: profile update |
| PUT /api/store/customers/me/password | ✓ | ✓ | — | 200 | 400/401/403 | (covered by fake-unit; gate only) |
| POST /api/store/customers/me/set-password | ✓ | ✓ | — | 200 | 400/401/403 | (gate only) |
| POST /api/store/customers/me/oauth | ✓ | ✓ | — | 200 | 401/403/409 | (oauth verify — store-level) |
| GET /api/store/customers/me/orders | ✓ | — | — | 200 `{items,total,...}` | 401 | orders: ListOrders paginate+isolation |
| POST /api/store/orders | ✓ | ✓ | — | 201 order | 400 VALIDATION_FAILED, 409 IDEMPOTENCY_REPLAY, 409 INSUFFICIENT_STOCK | orders: full suite |
| POST /api/store/orders/upload-photo | ✓ | ✓ | — | 200 `{url}` | 400 | (multipart — out of base scope) |
| GET /api/store/customers/me/cart | ✓ | — | — | 200 cart | 401 | storefront: cart round-trip |
| PUT /api/store/customers/me/cart | ✓ | ✓ | — | 200 | 401/403 | storefront: cart round-trip |
| GET /api/store/customers/me/favorites | ✓ | — | — | 200 | 401 | storefront: favorites round-trip |
| PUT /api/store/customers/me/favorites | ✓ | ✓ | — | 200 | 401/403 | storefront: favorites round-trip |

## Admin — auth & profile
| Method+Path | Auth | CSRF | RL | Success | Errors | Integration test |
|---|---|---|---|---|---|---|
| POST /api/auth/register | ✓ super_admin | ✓ | — | 200 | 401/403 FORBIDDEN | admin: super-admin gate |
| POST /api/auth/login | — | — | ✓ | 200 user | 401 AUTH_INVALID | admin: login happy + bad creds |
| POST /api/auth/forgot-password | — | — | ✓ | 200 | — | (email — out of base scope) |
| POST /api/auth/reset-password | — | — | ✓ | 200 | 400 | (store-level) |
| POST /api/auth/logout | ✓ | ✓ | — | 204/200 | 403 CSRF_INVALID | admin: logout gate |
| GET /api/users/me | ✓ | — | — | 200 user | 401 AUTH_REQUIRED | admin: me + 401 |
| PUT /api/users/me/profile | ✓ | ✓ | — | 200 | 401/403 | (gate only) |
| PUT /api/users/me/password | ✓ | ✓ | — | 200 | 400/401/403 | (gate only) |

## Admin — resources (admin role, DB-checked)
| Method+Path | Auth | CSRF | RL | Success | Errors | Integration test |
|---|---|---|---|---|---|---|
| GET /api/admin/users | ✓ super_admin | — | — | 200 | 401/403 FORBIDDEN | admin: super-admin gate |
| DELETE /api/admin/users/{username} | ✓ super_admin | ✓ | — | 200 | 401/403 | admin: super-admin gate |
| GET /api/admin/categories | ✓ admin | — | — | 200 tree | 401/403 | admin: 403 customer-token |
| GET /api/admin/products | ✓ admin | — | — | 200 list | 401/403 | admin: CRUD + gates |
| POST /api/admin/products | ✓ admin | ✓ | — | 201/200 | 400/401/403 | admin: create (status canon) |
| GET /api/admin/products/{slug} | ✓ admin | — | — | 200 | 404 | admin: CRUD |
| PUT /api/admin/products/{slug} | ✓ admin | ✓ | — | 200 | 400/401/403/404 | admin: update |
| DELETE /api/admin/products/{slug} | ✓ admin | ✓ | — | 200 | 401/403/404 | admin: delete |
| GET /api/admin/orders | ✓ admin | — | — | 200 | 401/403 | orders: admin list |
| PATCH /api/admin/orders/{id}/status | ✓ admin | ✓ | — | 200 | 400/401/403/404 | orders: admin status update |
| POST /api/admin/upload | ✓ admin | ✓ | — | 200 `{url}` | 400 | (multipart — out of base scope) |

## Known contract notes (resolved)
- `INSUFFICIENT_STOCK` is returned by `CreateOrder` but was NOT in the CLAUDE.md reserved code list. Resolved: added to the reserved codes in CLAUDE.md (documentation gap, the code is the intended distinct signal the storefront branches on).
- Storefront `CustomerAuthMW` (`internal/middleware/customer_auth.go`, 5 sites) and the rate limiter (`internal/middleware/ratelimit.go`) emit `http.Error` (`text/plain`, no machine `code`), violating the CLAUDE.md "JSON envelope with `code` from middleware too" rule. The **admin** path (`auth.go`, `csrf.go`, `require_super_admin.go`) already complies via `jsonerr.ErrorCode`. Filed as **#31**; a skipped regression test (`TestIntegrationCustomerAuthGateEnvelope`) pins the correct contract and unskips when #31 lands. Not fixed in this branch.
