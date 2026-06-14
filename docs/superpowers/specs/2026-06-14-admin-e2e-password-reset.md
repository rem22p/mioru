# Дизайн: Admin E2E — forgot/reset-password

Дата: 2026-06-14
Ветка: `docs/admin-e2e-followups` (спека + план)
Issue: #41
Зависимости: #42 (admin E2E foundation) + #49 (test endpoint hardening pattern).
Статус: обновлено после ревью Максима (от 14:20) — Approach A отклонён (нарушает CLAUDE.md "не логировать plaintext reset-токены"), предложен новый подход через dev-only test endpoint, который *генерирует* токен и возвращает raw (по паттерну #49 `ResetAdminForTest`).

## Проблема

`apps/admin/src/pages/ForgotPassword.tsx` и `ResetPassword.tsx` — страницы восстановления пароля. После #42 появляется рабочий E2E-харнесс, но forgot/reset **не покрыты**. Это:
- **Forgot** — всегда возвращает 200 (без энумерации пользователей, по security-конвенции).
- **Reset** — принимает токен из URL (`/reset/:token`), валидирует, меняет пароль.

**Особенность:** реальный reset-токен живёт только в письме (`RESEND_API_KEY` пуст в dev/CI → письмо лишь логируется), в БД хранится **только SHA-256 хеш** (per security test `TestResetTokenHashedAtRest`). В E2E **нет способа** получить raw-токен — это блокер.

## Цель

1. **Зафиксировать способ получения токена в тестовой среде** — dev-only test endpoint, который *генерирует* и возвращает raw (см. «Архитектура»).
2. **`apps/admin/e2e/password-reset.spec.ts`** (authenticated project) — E2E:
   - Forgot: отправка email → 200 (без энумерации).
   - Reset: невалидный токен → ошибка в UI; валидный токен → пароль меняется, логин новым паролем работает.
3. **`data-testid`** на forgot/reset поля, submit, error/success.
4. **Без** `waitForTimeout`.

## Границы (scope)

**В scope:**
- Forgot happy path (200, без энумерации, identical body для known/unknown email).
- Reset с валидным токеном (полученным через dev-only test endpoint).
- Reset с невалидным токеном.
- `data-testid` хуки.

**Вне scope (YAGNI):**
- Срок жизни токена / expiry handling (integration tests в #35).
- Multi-channel email delivery.
- Rate-limit на forgot/reset (integration tests).
- Admin self-reset (per #40 spec, password-change flow lives in `security.spec.ts`).
- Чтение raw токена из логов / прямой store access (rejected — см. Архитектура).
- Side-column для raw токенов (rejected — нарушает security contract `TestResetTokenHashedAtRest`).

## Потенциальные баги для проверки

- **Token enumeration в forgot** — `POST /api/auth/forgot-password` для **существующего** email vs **несуществующего** email должен возвращать **одинаковый** response (status + body). Это критично для privacy. E2E проверит status + body (timing-атаки — не в scope).
- **Reset token reuse** — после успешного reset тот же токен → ошибка. Если API возвращает 200 на reuse — баг (token становится постоянным). Integration tests в #35 покрывают; e2e проверим indirectly через invalid token test.
- **Reset с пустым паролём / коротким** — UI должен валидировать (per password policy) **до** отправки.
- **Logout after reset** — после смены пароля текущая сессия должна стать невалидной (т.к. reset пишет `password_changed_at = NOW()`). **Уже покрыто** в `security.spec.ts` (#49), дублировать здесь не нужно.

## Архитектура: способ получения токена

3 варианта рассмотрены, **финальный выбор**:

### ✅ Рекомендуется: dev-only test endpoint, который *генерирует* токен

Добавить `POST /api/_test/create-reset-token` (dev-only):

| Барьер | Механизм | Зачем |
|---|---|---|
| **Compile-time** | `//go:build e2e` build-tag на файле handler | Production binary физически не содержит handler/route |
| **Runtime gate** | `!cfg.IsProduction() && E2E_RESET_KEY != ""` в `test_routes_e2e.go` | Fail-safe если build-tag забыт |
| **Auth gate** | `X-E2E-Reset-Key` header constant-time-compared to `E2E_RESET_KEY` env (via `subtle.ConstantTimeCompare`) | Каждый запрос — аутентифицирован |
| **Fail-closed** | Empty `E2E_RESET_KEY` env → 503 `TEST_RESET_DISABLED` (не fail-open) | Misconfiguration ≠ open door |
| **No raw in DB** | Вызывает `store.CreateResetToken(username, rawToken)` — тот же путь, что production forgot-password flow | SHA-256 hash only in `password_reset_tokens` |
| **No raw in logs** | Никаких `log.Printf("token: %s", ...)` | CLAUDE.md "не логировать plaintext" |

Handler returns:
```json
{ "token": "<raw-token-32-bytes-base64>" }
```

**Почему этот вариант:**
- Чисто, явно, **не** зависит от чтения stdout/logs.
- Trivially mockable в Playwright через `request.post()`.
- Production-safe по построению (4 независимых барьера).
- **Не нарушает** security contract `TestResetTokenHashedAtRest` — token *генерируется* и хешируется тем же путём, что production.
- Повторяет **тот же паттерн**, что уже landed в #49 (`ResetAdminForTest`) — consistency wins.

### ❌ Альтернатива 1 (отклонена): читать токен из логов backend

Парсить stdout/файл логов в test-харнессе. **Отклонено:**
- Требует, чтобы store логировал raw токен — **нарушает** CLAUDE.md "не логировать plaintext reset-токены".
- Лог-format flake (зависит от text-форматирования, уровней логирования, structured-vs-plain).
- Требует доступа к логам backend из Playwright — сложная композиция.

### ❌ Альтернатива 2 (отклонена): side-column для raw токенов

Добавить `raw_token` колонку в `password_reset_tokens`, заполняемую только при `TEST_KEEP_RAW_TOKENS=1`. **Отклонено:**
- Расширяет production schema ради тестов.
- Всё ещё требует, чтобы кто-то помнил про env var при локальной разработке.
- Ломает mental model: production таблица знает о test-only колонке.

**Spec:** `docs/superpowers/plans/2026-06-14-admin-e2e-password-reset.md` (отдельный файл).
