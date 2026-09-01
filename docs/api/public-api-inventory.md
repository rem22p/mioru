# Public API inventory

Source of truth: `backend/api/cmd/server/main.go`. Columns: auth (cookie required),
CSRF (mutation gate), RL (rate-limited), success code, key error codes, and the
integration test that covers it. Test files live in `backend/api/internal/handler/`
(`integration_*_test.go`, package `handler_test` — real Postgres via the harness)
and `backend/api/cmd/server/main_test.go` (package `main` — infra helpers).

For exact request/response shapes and security checks per route, see the companion
`public-api-contract-reference.md`.

## Build-time-only routes

The following routes are **not** in the production binary. They live behind
the `//go:build e2e` build tag and are exercised by Playwright security specs
that need to drop the admin back to a known bcrypt hash. See
`docs/api/test-only-endpoints.md` for the full security model and verification
commands.

| Method+Path | Build | Auth | Success | Errors | E2E consumer |
|---|---|---|---|---|---|
| POST /api/_test/reset-admin | `e2e` tag | `X-E2E-Reset-Key` header (constant-time compare to `E2E_RESET_KEY` env) | 200 `{ok:true}` | 400 VALIDATION_FAILED, 403 FORBIDDEN, 500 INTERNAL, 503 TEST_RESET_DISABLED | `apps/admin/e2e/security.spec.ts` |

## Health
| Method+Path | Auth | CSRF | RL | Success | Errors | Integration test |
|---|---|---|---|---|---|---|
| GET /api/health | — | — | — | 200 `{status:ok}` | — | inline closure in `main()`, not extractable without a mux refactor (out of scope) |

## Store — public catalog (no auth)
| Method+Path | Auth | CSRF | RL | Success | Errors | Integration test |
|---|---|---|---|---|---|---|
| GET /api/products | — | — | — | 200 `{items,total,page,per_page}`. Filter arrays capped: 20 values each, ≤60 runes per brand / ≤40 per color / ≤32 per size — oversized → 400 VALIDATION_FAILED. **sort** ∈ `{popular, newest, created_at, price, -price, name, brand}` — empty or unknown value = `created_at DESC` (newest); `popular` uses `popularity_rank` (`popularity_rank_preorder` when `status=preorder`); `brand` sorts by the derived display name | — | `TestIntegrationListProducts` (+ store-level paginate/filter) |
| GET /api/products/facets | — | — | — | 200 `{brands,colors,sizes}` (KAN-14: collab brands listed individually) | 400 VALIDATION_FAILED | `TestIntegrationFacetsHappy`, `...DropsOwnSelection`, `...BadStatus`, `TestCollabBrandsFacetsSplit` |
| GET /api/products/{slug} | — | — | — | 200 product | 404 NOT_FOUND | `TestIntegrationGetProductBySlug`, `...NotFound` |
| GET /api/categories | — | — | — | 200 tree, nodes carry `products_count` (own + all descendants) | — | `TestIntegrationListCategories`, `TestIntegrationCategoryProductCounts` |

## Store — customer auth & profile
| Method+Path | Auth | CSRF | RL | Success | Errors | Integration test |
|---|---|---|---|---|---|---|
| POST /api/store/auth/register | — | — | ✓ | 201 customer (incl. `telegram`) + cookies | 400 VALIDATION_FAILED, 409 CONFLICT | `TestIntegrationCustomerRegisterHappy`, `...Duplicate`, `...Validation` |
| POST /api/store/auth/login | — | — | ✓ | 200 customer (incl. `telegram` binding) + cookies | 401 AUTH_INVALID | `TestIntegrationCustomerLoginHappyAndReuseCookie`, `...WrongPassword`, `...MissingUser`, `TestIntegrationCustomerLoginReportsTelegramBinding` |
| POST /api/store/auth/telegram | — | — | ✓ | 200/201 customer (`telegram.linked:true`) + cookies | 401 AUTH_INVALID, 503 (not configured) | `TestIntegrationTelegramLoginNotConfigured`, `...FakeHash`, `...NewCustomer`, `...ExistingLink` |
| POST /api/store/auth/logout | ✓ | ✓ | — | 200 | 403 CSRF_INVALID | `TestIntegrationCustomerLogoutCSRFGate` |
| GET /api/store/customers/me | ✓ | — | — | 200 customer (incl. `telegram` binding) | 401 AUTH_REQUIRED* | `TestIntegrationCustomerMeReturnsProfile`, `...MeRequiresAuth` |
| PUT /api/store/customers/me | ✓ | ✓ | — | 200 `{ok}` | 400 VALIDATION_FAILED (incl. `phone` not matching `^\+373\d{8}$`), 401 AUTH_INVALID/AUTH_REQUIRED, 403 | `TestIntegrationCustomerUpdateProfileHappy`, `...WrongPassword`, `...MissingPassword`, `...PhoneFormat` |
| PUT /api/store/customers/me/password | ✓ | ✓ | — | 200 `{ok}` | 400/401/403 | `TestIntegrationCustomerChangePasswordInvalidatesOldToken` (happy + epoch invalidation) |
| POST /api/store/customers/me/set-password | ✓ | ✓ | — | 200 `{ok}` | 400/401/403, 409 CONFLICT | `TestIntegrationCustomerSetPasswordRejectsPasswordedCustomer` |
| POST /api/store/customers/me/oauth | ✓ | ✓ | — | 200 `{ok}` | 400 VALIDATION_FAILED, 401 AUTH_INVALID, 403, 409 CONFLICT, 503 | `TestIntegrationLinkOAuthTelegramRejectsUnsigned` (hijack guard), `...NonTelegramHappy`, `...TelegramProfileDataIsServerBuilt` (spoof guard), `...OmittedProfileData`, `...MalformedProfileData`, `...TelegramNotConfigured` |
| GET /api/store/customers/me/orders | ✓ | — | — | 200 `{orders,total,page,per_page}` (individual orders carry `category`, `foot_length`, `photos`) | 401 | `TestIntegrationListOrdersIsolation` (paginate + cross-customer isolation), `TestIntegrationCustomerListOrdersIncludesFullDetails` (full-field contract incl. KAN-52 fields) |
| POST /api/store/orders | ✓ | ✓ | — | 201 order | 400 VALIDATION_FAILED (incl. missing/invalid `phone` — must match `^\+373\d{8}$`, KAN-52 individual `category` enum + `foot_length` bounds, stale product_id → `NOT_FOUND`, `cod + bus` for cart orders, `moldovaPost` for PMR cities), 403 TELEGRAM_REQUIRED (no Telegram binding), 409 IDEMPOTENCY_REPLAY, 409 INSUFFICIENT_STOCK | `TestIntegrationCreateOrderRequiresTelegram`, `TestIntegrationCreateOrder*` (happy/stock/replay/conflict/oversell/missing-key), `TestIntegrationCreateOrderRequiresPhone`, `TestIntegrationCreateOrderPersistsPhoneAndSyncsToProfile`, `TestIntegrationCreateOrderDoesNotResyncIdenticalPhone`, `TestIntegrationCreateOrderRejectsUnknownProduct`, `TestIntegrationCreateOrderRejectsCodWithBus`, `TestIntegrationCreateOrderAllowsCodWithBusForIndividual`, `TestIntegrationCreateOrderRejectsMoldovaPostInPNR`, `TestIntegrationCreateOrderCategory` |
| POST /api/store/orders/upload-photo | ✓ | ✓ | — | 200 `{url}` | 400, 403 | `TestIntegrationUploadOrderPhotoPNG`, `...RejectsNonPNG`, `...RequiresFile`, `...CSRFGate` |
| GET /api/store/customers/me/cart | ✓ | — | — | 200 cart | 401 | `TestIntegrationCartRoundTrip` |
| PUT /api/store/customers/me/cart | ✓ | ✓ | — | 200 | 400 NOT_FOUND (stale product_id), 401/403 | `TestIntegrationCartRoundTrip`, `TestIntegrationSaveCartCSRFGate`, `TestIntegrationSaveCartRejectsUnknownProduct` |
| GET /api/store/customers/me/favorites | ✓ | — | — | 200 | 401 | `TestIntegrationCustomerFavoritesRoundTrip` |
| PUT /api/store/customers/me/favorites | ✓ | ✓ | — | 200 | 400, 401/403 | `TestIntegrationCustomerFavoritesRoundTrip`, `...SaveFavoritesValidation` |

\* The storefront auth-gate `401` is currently emitted as `text/plain` without a machine `code` — see the #31 note below; `TestIntegrationCustomerAuthGateEnvelope` pins the correct contract and is deliberately failing until the fix lands.

## Admin — auth & profile
| Method+Path | Auth | CSRF | RL | Success | Errors | Integration test |
|---|---|---|---|---|---|---|
| POST /api/auth/register | ✓ super_admin | ✓ | — | 201 | 403 FORBIDDEN, 400 VALIDATION_FAILED (dup) | `TestIntegrationAdminRegisterRequiresSuperAdmin`, `...SuperAdminHappy`, `...Duplicate` |
| POST /api/auth/login | — | — | ✓ | 200 user + cookies | 401 AUTH_INVALID | `TestIntegrationAdminLoginHappyAndReuseCookie`, `...WrongPassword`, `...MissingUser` |
| POST /api/auth/forgot-password | — | — | ✓ | 200 (always, no enumeration) | 400 | `TestIntegrationAdminForgotPasswordAlways200`, `...BadEmail` |
| POST /api/auth/reset-password | — | — | ✓ | 200 `{ok}` | 400 | `TestIntegrationAdminResetPasswordInvalidToken`, `...Happy` (login with new pw) |
| POST /api/auth/logout | ✓ | ✓ | — | 200 | 403 CSRF_INVALID | `TestIntegrationAdminLogoutCSRFGate` |
| GET /api/users/me | ✓ | — | — | 200 user | 401 AUTH_REQUIRED | `TestIntegrationAdminMeReturnsProfile` |
| PUT /api/users/me/profile | ✓ | ✓ | — | 200 `{ok}` (no current_password) | 400, 401/403 | `TestIntegrationAdminUpdateProfileHappyNoPassword`, `...Validation` |
| PUT /api/users/me/password | ✓ | ✓ | — | 200 `{ok}` | 400, 401 AUTH_INVALID, 403 | `TestIntegrationAdminChangePasswordHappy`, `...WrongCurrent`, `...InvalidatesOldToken` |

## Admin — resources (admin role, DB-checked)
| Method+Path | Auth | CSRF | RL | Success | Errors | Integration test |
|---|---|---|---|---|---|---|
| GET /api/admin/users | ✓ super_admin | — | — | 200 array | 401/403 FORBIDDEN | `TestIntegrationAdminListUsersHappy`, `TestIntegrationNonSuperAdminCannotListUsers` |
| DELETE /api/admin/users/{username} | ✓ super_admin | ✓ | — | 204 | 400 VALIDATION_FAILED (self), 401/403, 404 NOT_FOUND | `TestIntegrationAdminDeleteUserHappy`, `...Self`, `...NotFound` |
| GET /api/admin/categories | ✓ admin | — | — | 200 tree/flat | 401/403 | `TestIntegrationAdminCategoriesTree`, `...Flat` |
| GET /api/admin/products | ✓ admin | — | — | 200 `{products,total,...}` | 401/403 | `TestIntegrationAdminListProducts` |
| POST /api/admin/products | ✓ admin | ✓ | — | 201 `{id,product}` (KAN-14: multipart `brands[]`) | 400/401/403 | `TestIntegrationAdminCreateAndGetProduct`, `...CreateProductCSRFGate`, `TestParseProductFromFormBrands` |
| GET /api/admin/products/{slug} | ✓ admin | — | — | 200 | 404 | `TestIntegrationAdminCreateAndGetProduct` |
| PUT /api/admin/products/{slug} | ✓ admin | ✓ | — | 200 product | 400/401/403, 404 NOT_FOUND, 409 CONFLICT | `TestIntegrationAdminUpdateProductHappy`, `...NotFound`, `...DuplicateSlug` |
| DELETE /api/admin/products/{slug} | ✓ admin | ✓ | — | 200 `{ok}` | 401/403, 404 NOT_FOUND | `TestIntegrationAdminDeleteProductHappy`, `...NotFound` |
| GET /api/admin/orders | ✓ admin | — | — | 200 `{orders,total,...}` (joined customer_email) | 401/403 | `TestIntegrationAdminListAllOrdersJoinsCustomer` |
| PATCH /api/admin/orders/{id}/status | ✓ admin | ✓ | — | 200 `{ok}` | 400 VALIDATION_FAILED, 401/403 | `TestIntegrationAdminUpdateOrderStatus` (valid), `...UpdateStatusInvalid`, `...NonNumericId` |
| GET /api/admin/customers | ✓ admin | — | — | 200 `{customers,total,page,per_page}` (rolled-up order stats, Telegram link) | 401/403 | `TestIntegrationAdminListCustomers` |
| GET /api/admin/customers/{id} | ✓ admin | — | — | 200 full customer + order list (no `hashed_password`) | 401/403, 404 NOT_FOUND | `TestIntegrationAdminGetCustomerDetail`, `TestIntegrationAdminCustomersRequiresAuth` |
| POST /api/admin/upload | ✓ admin | ✓ | — | 200 `{url}` | 400 | `TestIntegrationAdminUploadPNG`, `...RejectsNonPNG` |

## Cross-cutting gates & infra
| Concern | Integration test |
|---|---|
| Admin route rejects no session (401) | `TestIntegrationAdminRouteRejectsNoSession` |
| Admin route rejects customer token / wrong `typ` (401) | `TestIntegrationAdminRouteRejectsCustomerToken` |
| CORS reflects only allowlisted origins (+ credentials, Vary) | `TestCORSReflectsOnlyAllowlistedOrigin` (package main) |
| CORS preflight short-circuits to 204 | `TestCORSPreflightReturns204` (package main) |
| `/uploads/` nosniff + locked-down CSP | `TestUploadsSecurityHeaders` (package main) |
| Server timeouts (Slowloris guard) | `TestNewServerHasTimeouts` (package main) |
| Security headers / CSP no unsafe-inline | `TestSecurityHeadersCSPNoUnsafeInline` (package main) |

## Known contract notes
- `INSUFFICIENT_STOCK` is returned by `CreateOrder` but was NOT in the CLAUDE.md reserved code list. Resolved: added to the reserved codes in CLAUDE.md (documentation gap; the code is the intended distinct signal the storefront branches on).
- **#31 (resolved in #34):** Storefront `CustomerAuthMW` (`internal/middleware/customer_auth.go`) and the rate limiter (`internal/middleware/ratelimit.go`) previously emitted `http.Error` (`text/plain`, no machine `code`), violating the CLAUDE.md "JSON envelope with `code` from middleware too" rule. Both now emit the JSON envelope via `jsonerr.ErrorCode`, mirroring the admin path (`auth.go`, `csrf.go`, `require_super_admin.go`) — `AUTH_REQUIRED` (no/empty cookie), `AUTH_INVALID` (bad/expired token, revoked session), `INTERNAL` (epoch lookup failure), `RATE_LIMITED` + `Retry-After: 60` (limit exceeded). The regression test `TestIntegrationCustomerAuthGateEnvelope` is unskipped; `TestIntegrationCustomerAuthGateBadTokenEnvelope` and `TestRateLimitEnvelope`/`TestRateLimitUnderLimitPasses`/`TestRateLimitFailOpenOnStoreError` were added as additional pins.
- The GitHub Actions workflow (`.github/workflows/backend-ci.yml`, issue **#33**) is not part of this branch — the agent's PAT lacks `workflow` scope, so it must be added via SSH by a maintainer.
