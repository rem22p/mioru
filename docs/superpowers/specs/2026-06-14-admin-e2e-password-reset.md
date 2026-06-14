# Дизайн: Admin E2E — forgot/reset-password

Дата: 2026-06-14
Ветка: `docs/admin-e2e-followups` (спека + план)
Issue: #41
Зависимость: #42 (admin E2E foundation) должен быть вмёржен
Статус: предложен, ожидает ревью

## Проблема

`apps/admin/src/pages/ForgotPassword.tsx` и `ResetPassword.tsx` — страницы восстановления пароля. После #42 появляется работающий E2E-харнесс, но forgot/reset **не покрыты**. Это:

- **Forgot** — всегда возвращает 200 (без энумерации пользователей, по security-конвенции).
- **Reset** — принимает токен из URL (`/reset/:token`), валидирует, меняет пароль.

**Особенность:** реальный reset-токен живёт только в письме (`RESEND_API_KEY` пуст в dev/CI → письмо лишь логируется), в БД хранится **только SHA-256 хеш**. В E2E **нет способа** получить raw-токен — это блокер.

## Цель

1. **Зафиксировать способ получения токена в тестовой среде** (см. «Архитектура» ниже).
2. **`apps/admin/e2e/password-reset.spec.ts`** — E2E:
   - Forgot: отправка email → 200 (без энумерации).
   - Reset: невалидный токен → ошибка в UI; валидный токен → пароль меняется, логин новым паролем работает.
3. **`data-testid`** на forgot/reset поля, submit, error/success.
4. **Без** `waitForTimeout`.

## Границы (scope)

**В scope:**
- Forgot happy path (200, без энумерации).
- Reset с валидным токеном (полученным через test-helper).
- Reset с невалидным токеном.
- `data-testid` хуки.

**Вне scope (YAGNI):**
- Срок жизни токена / expiry handling (integration tests в #35).
- Multi-channel email delivery (per test, мы получаем токен из логов).
- Rate-limit на forgot/reset (integration tests).

## Потенциальные баги для проверки

- **Token enumeration в forgot** — `POST /api/auth/forgot-password` для **существующего** email vs **несуществующего** email должен возвращать **одинаковый** response (status + body). Это критично для privacy. E2E не проверит timing-атак, но проверит status + body.
- **Reset token reuse** — после успешного reset тот же токен → ошибка. Если API возвращает 200 на reuse — баг (token становится постоянным).
- **Reset с пустым паролём / коротким** — UI должен валидировать (per password policy) **до** отправки.
- **Logout after reset** — после смены пароля текущая сессия должна стать невалидной. Это **может** не покрываться (зависит от того, меняет ли reset token `password_changed_at` для admin). Если нет — это feature, не bug. Spec покрывает только customer-side reset, **не** admin self-reset.

## Архитектура: способ получения токена

3 варианта (per #41 issue), выбор фиксируем:

### ✅ Рекомендуется: тестовый endpoint под `APP_ENV != production`

Добавить `POST /api/test/reset-token` (или `GET /api/admin/auth/reset-token/:email`):

- Включён **только** при `APP_ENV=development` или `APP_ENV=test` — **выключен** в production.
- Возвращает raw-токен для данного email (или 404 если нет).
- Защищён require super-admin (или require спец-роли) в dev-режиме.

**Почему этот вариант:**
- Чисто, явно, **не** зависит от чтения stdout/logs.
- Trivially mockable в Playwright через `request.post()`.
- Production-safe по построению (gated by env).

### Альтернатива: читать токен из логов backend

Парсить stdout/файл логов в test-харнессе. Менее надёжно (флейки на log-format), требует доступа к логам.

### Альтернатива: прямой доступ к store

Вызвать `store.GetResetTokenHash(email)`, конвертировать в raw. Хрупко, нарушает encapsulation.

**Spec:** `docs/superpowers/plans/2026-06-14-admin-e2e-password-reset.md` (отдельный файл).
