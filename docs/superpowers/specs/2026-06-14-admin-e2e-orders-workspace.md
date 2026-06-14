# Дизайн: Admin E2E — orders-workspace

Дата: 2026-06-14
Ветка: `docs/admin-e2e-followups` (спека + план)
Issue: #39
Зависимость: #42 (admin E2E foundation) должен быть вмёржен
Статус: предложен, ожидает ревью

## Проблема

`apps/admin/src/workspaces/Orders.tsx` — воркспейс заказов в админке. После вмёрженного #42 (Playwright foundation) появляется **рабочий** E2E-харнесс (login + product CRUD + upload + super-admin user-mgmt), но **orders** в нём пока **не покрыт**: список, пагинация, смена статуса, фильтры — всё вручную.

Без E2E-покрытия:
- Регрессии в админском orders-flow (status update ломает инвариант) ловятся только вручную.
- UI/UX баги (не отображается customer_email, битая пагинация) уезжают в main незамеченными.

## Цель

1. **Спека + план** — design как у #32/#37 (ПР с интеграционными тестами), чтобы было легко взять issue без перепроектирования.
2. **`apps/admin/e2e/orders.spec.ts`** — функциональный E2E:
   - список заказов (`GET /api/admin/orders`) — отображение строк, join `customer_email`, пагинация;
   - смена статуса (`PATCH /api/admin/orders/{id}/status`) — валидный переход отражается в UI; невалидный статус → ошибка.
3. **`data-testid`** на строки заказов и контрол смены статуса (per CLAUDE.md «Stable selectors»).
4. **Без дублирования i18n-копии или Tailwind-классов** в селекторах.

## Границы (scope)

**В scope:**
- E2E для существующих/предстоящих orders-маршрутов.
- Self-skip при недоступном backend на `:8000` (per #42 convention).
- `data-testid` хуки на строки, фильтры, кнопки смены статуса, пагинацию.
- Mutation-evidence: spec сначала с `t.Skip`, потом unskip после фикса (если баг найден).

**Вне scope (YAGNI):**
- Полная покрытие orders-REST API (это **не** orders-E2E задача; покрыто integration-тестами в #35).
- Визуальная регрессия / a11y — отдельный spec.
- Фильтры по статусу/дате/сумме, если их ещё нет в UI (в #39 упомянуто «если есть» — реализация отложена, но **testid зарезервированы**).

## Потенциальные баги для проверки

- `customer_email` join может быть NULL если customer удалён — UI должен показывать «(deleted)» или грациозно скрывать. E2E проверит **happy path** (создаём customer → видим email).
- Невалидный статус (`PATCH {id}/status {status: "bogus"}`) — UI должен показать ошибку, **не 500** (per integration tests в #35 этот код уже `400 VALIDATION_FAILED`).
- `total_minor` отображается как `9.99 L` (MDL) — **не** как `9.99 ₽`. Если UI хардкодит символ, regression-тест поймает (per #38 `formatPrice` fix для стора).
- Конкурентный status update: 2 админа меняют статус одного заказа → один получает stale UI. Это **не** баг для E2E, но data-testid должен быть стабильным под оба сценария.

## Definition of Done (per #39)

- `apps/admin/e2e/orders.spec.ts` зелёный против живого backend.
- `data-testid` проставлены и не дублируют i18n-копию.
- Спека + план в `docs/superpowers/{specs,plans}/` ревьюнуты.

## Архитектура (наследует #42)

- Один shared login через `setup` project + `storageState` (per #42 convention; auth.spec делает fresh login сам).
- `workers: 1` (shared backend state) + unique order ID per run + cleanup.
- Self-skip через `test.skip(!backendUp, ...)` если `/api/health` не отвечает.
- `waitForResponse` на реальные API + `expect.toBeVisible()` — **без** `waitForTimeout`.

**Spec:** `docs/superpowers/plans/2026-06-14-admin-e2e-orders-workspace.md` (отдельный файл).
