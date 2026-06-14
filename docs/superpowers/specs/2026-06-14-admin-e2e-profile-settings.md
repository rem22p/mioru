# Дизайн: Admin E2E — profile & settings

Дата: 2026-06-14
Ветка: `docs/admin-e2e-followups` (спека + план)
Issue: #40
Зависимости: #42 (admin E2E foundation) + #49 (security isolation) merged.
Статус: обновлено после ревью Максима (от 14:20), reflects actual implementation on main.

## Проблема

`apps/admin/src/workspaces/Profile.tsx` и `Settings.tsx` — воркспейсы личных настроек админа. После #42 появляется рабочий E2E-харнесс, но **profile/settings** пока **не покрыты**:

- **Profile:** редактирование профиля, ошибки в форме смены пароля.
- **Settings:** тема (light/dark через класс `light` на `<html>`) и масштаб UI (`--ui`).

Без E2E:
- Смена пароля может перестать инвалидировать старую сессию (per CLAUDE.md security hardening: `iat < password_changed_at` reject в middleware) — регрессия пройдёт мимо unit-тестов. **Покрыто отдельным security.spec.ts** (см. Архитектура).
- Settings state может перестать персиститься (только localStorage, не покрыто).

## Цель

1. **`apps/admin/e2e/profile.spec.ts`** (authenticated project) — E2E для profile:
   - редактирование профиля (`PUT /api/users/me/profile`) — успех + round-trip через reload;
   - несовпадение new/confirm пароля → UI error, **без** network call;
   - неверный current password → backend 4xx + UI error, **старая сессия жива**.
2. **`apps/admin/e2e/settings.spec.ts`** (authenticated project) — E2E для settings:
   - тема (light/dark) — `<html class="light">` toggle + persist в `localStorage`;
   - масштаб UI (`--ui` CSS variable) — range slider + persist.
3. **`apps/admin/e2e/security.spec.ts`** (security project, отдельный) — критичный:
   - смена пароля → старый токен → 401 `AUTH_INVALID`. **Покрыто в #49** (security isolation).
4. **`data-testid`** на поля профиля, кнопки сохранения, контролы темы/масштаба (per CLAUDE.md).
5. **Без** `waitForTimeout` (anti-flake per CLAUDE.md).

## Границы (scope)

**В scope:**
- Happy-path для profile edit + reload round-trip.
- Mismatched confirm-password (UI error, no network).
- Wrong current password (backend 4xx + UI error, session still alive).
- Theme toggle (light/dark) + localStorage persist + reload.
- UI scale slider + `--ui` CSS variable + localStorage persist + reload.
- Self-skip при недоступном backend (per #42).
- `data-testid` хуки (landed in PR #49).

**Вне scope (YAGNI):**
- Negative tests для всех полей profile (email format и т.д. — это для **integration tests** в #35, не E2E).
- **Locale settings** — `Settings.tsx` НЕ реализует per-user locale (YAGNI; i18n — глобальный). Соответственно spec/plan не покрывает `settings-locale-*`.
- Визуальная регрессия / a11y.
- Multi-admin / RBAC для settings (settings всегда self).
- Сессия-инвалидация — отдельный spec в security project (см. ниже).

## Потенциальные баги для проверки

- **Session invalidation при смене пароля.** Это **критичный** security-инвариант. Per CLAUDE.md: middleware rejects tokens with `iat < password_changed_at`. E2E для этого живёт в **`security.spec.ts`** (security project): (1) сменить пароль, (2) попытаться использовать старый токен → ожидать 401 `AUTH_INVALID`. **Изолирован в #49**, потому что мутация shared admin state ломает iat < changed_at для всех последующих `login()` вызовов в authenticated project. Если баг — filed as issue, security test skipped (per #32 convention).
- **`PUT /api/users/me/profile`** без `Content-Type` или с битым JSON → 400 envelope с `VALIDATION_FAILED` code. UI должен показать ошибку, не 500.
- **Settings persist race** — `localStorage.setItem` async? Sync? В `themeStore.ts:applyTheme()` и `uiStore.ts:applyScale()` запись **синхронная**, проблем нет. Тем не менее, для надёжности используем `expect.poll(() => localStorage.getItem(...))` или reload round-trip.
- **Theme flash** — при перезагрузке страницы тема должна примениться **до** первого paint. Это не в scope (vitest + RTL), но data-testid должен стабильно работать в обоих состояниях.

## Definition of Done (per #40)

- `apps/admin/e2e/profile.spec.ts` (3 tests) + `settings.spec.ts` (3 tests) зелёные в authenticated project.
- `apps/admin/e2e/security.spec.ts` (1 test) зелёный в security project.
- Спека + план ревьюнуты.
- `data-testid` проставлены (landed in #49).

## Архитектура (наследует #42 + #49)

- **Authenticated project:** shared login через `setup` project + `storageState` (reused, no auth in this spec). `workers: 1` (serial) + cleanup via API. `waitForResponse` + `expect.toBeVisible()` — без `waitForTimeout`.
- **Security project:** `dependencies: ["setup"]`, calls `POST /api/_test/reset-admin` (dev-only, build-tag `e2e`, X-E2E-Reset-Key auth per #49) before each test to drop admin back to known state. Excluded from authenticated project via top-level `testIgnore`.

**Spec:** `docs/superpowers/plans/2026-06-14-admin-e2e-profile-settings.md` (отдельный файл).

---

## Test contract (закреплено в PR #49)

| Spec | Testid | Файл |
|---|---|---|
| `profile-display-name` | input | Profile.tsx |
| `profile-avatar-color-{value}` | color button | Profile.tsx |
| `profile-save` | submit | Profile.tsx |
| `profile-password-current` | password input | Profile.tsx |
| `profile-password-new` | password input | Profile.tsx |
| `profile-password-confirm` | password input | Profile.tsx |
| `profile-password-submit` | submit | Profile.tsx |
| `profile-alert-success` | banner | Profile.tsx |
| `profile-alert-error` | banner | Profile.tsx |
| `settings-page` | container | Settings.tsx |
| `settings-theme-light` | button | Settings.tsx |
| `settings-theme-dark` | button | Settings.tsx |
| `settings-scale-slider` | range input | Settings.tsx |

## localStorage contract (закреплено в #49 + stores на main)

| Key | Type | Range / values | Set by | Read by |
|---|---|---|---|---|
| `ui_theme` | string | `'dark'` (default) / `'light'` | `themeStore.ts:applyTheme()` | `themeStore.ts:getInitialTheme()` |
| `ui_scale` | number (as string) | 11–18, default 13 | `uiStore.ts:applyScale()` | `uiStore.ts:getInitialScale()` |

## CSS variables (закреплено в #49 + stores на main)

| Variable | Value | Set by |
|---|---|---|
| `--ui` | `<n>px` where n is ui_scale | `uiStore.ts:applyScale()` |
| `<html>.classList` | contains `light` iff theme === 'light' | `themeStore.ts:applyTheme()` |
