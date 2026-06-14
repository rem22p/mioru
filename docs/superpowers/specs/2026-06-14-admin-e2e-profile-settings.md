# Дизайн: Admin E2E — profile & settings

Дата: 2026-06-14
Ветка: `docs/admin-e2e-followups` (спека + план)
Issue: #40
Зависимость: #42 (admin E2E foundation) должен быть вмёржен
Статус: предложен, ожидает ревью

## Проблема

`apps/admin/src/workspaces/Profile.tsx` и `Settings.tsx` — воркспейсы личных настроек админа. После вмёрженного #42 появляется рабочий E2E-харнесс, но **profile/settings** пока **не покрыты**:

- **Profile:** редактирование профиля, смена пароля (с инвалидацией старой сессии — критичный security-инвариант).
- **Settings:** тема (light/dark через класс `light` на `<html>`), масштаб UI (`--ui`), локаль — персистентность в `localStorage` и применение.

Без E2E:
- Смена пароля может перестать инвалидировать старую сессию (per CLAUDE.md security hardening: `iat < password_changed_at` reject в middleware) — регрессия пройдёт мимо unit-тестов.
- Settings state может перестать персиститься (только localStorage, не покрыто).

## Цель

1. **`apps/admin/e2e/profile.spec.ts`** — E2E для profile:
   - редактирование профиля (`PUT /api/users/me/profile`) — успех;
   - смена пароля (`PUT /api/users/me/password`) — успех + **инвалидация старой сессии** (старый токен → 401 `AUTH_INVALID`).
2. **`apps/admin/e2e/settings.spec.ts`** — E2E для settings (если в #40 упомянуто):
   - тема (light/dark) — `<html class="light">` toggle + persist в `localStorage`;
   - масштаб UI (`--ui` CSS variable) — селект + persist;
   - локаль — селект + persist + i18n-применение (через `<html lang="...">` или эквивалент).
3. **`data-testid`** на поля профиля, кнопки сохранения, контролы темы/масштаба (per CLAUDE.md).
4. **Без** `waitForTimeout` (anti-flake per CLAUDE.md).

## Границы (scope)

**В scope:**
- Happy-path для profile edit + password change (с проверкой session invalidation).
- Happy-path для settings (тема/масштаб/локаль) с проверкой `localStorage` persist.
- Self-skip при недоступном backend (per #42).
- `data-testid` хуки.

**Вне scope (YAGNI):**
- Negative tests для всех полей profile (email format и т.д. — это для **integration tests** в #35, не E2E).
- Визуальная регрессия / a11y.
- Multi-admin / RBAC для settings (settings всегда self).

## Потенциальные баги для проверки

- **Session invalidation при смене пароля.** Это **критичный** security-инвариант. Per CLAUDE.md: middleware rejects tokens with `iat < password_changed_at`. E2E должен: (1) сменить пароль, (2) попытаться использовать старый токен → ожидать 401 `AUTH_INVALID`. Если баг — filed как issue, spec временно `t.Skip` (per #32 convention).
- **`PUT /api/users/me/profile`** без `Content-Type` или с битым JSON → 400 envelope с `VALIDATION_FAILED` code. UI должен показать ошибку, не 500.
- **Settings persist race** — `localStorage.setItem` async? Sync? Если sync, проблем нет. Если async + медленный render, тест может прочитать до persist. Использовать `waitForResponse` на любой I/O, или `expect.poll(() => localStorage.getItem(...))`.
- **Theme flash** — при перезагрузке страницы тема должна примениться **до** первого paint. Это не в scope (vitest + RTL), но data-testid должен стабильно работать в обоих состояниях.

## Definition of Done (per #40)

- `apps/admin/e2e/profile.spec.ts` (+ `settings.spec.ts` при необходимости) зелёные против живого backend.
- Спека + план ревьюнуты.
- `data-testid` проставлены.

## Архитектура (наследует #42)

- Shared login через `setup` project + `storageState` (reused, no auth in this spec).
- `workers: 1` (serial) + cleanup via API.
- Self-skip при backend down.
- `waitForResponse` + `expect.toBeVisible()` — без `waitForTimeout`.

**Spec:** `docs/superpowers/plans/2026-06-14-admin-e2e-profile-settings.md` (отдельный файл).
