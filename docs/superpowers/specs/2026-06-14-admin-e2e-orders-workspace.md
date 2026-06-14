# Дизайн: Admin E2E — orders-workspace

Дата: 2026-06-14
Ветка: `docs/admin-e2e-followups` (спека + план)
Issue: #39 (CLOSED by PR #46)
Зависимость: #42 (admin E2E foundation) — merged.
Статус: реализован в PR #46 (merged), этот документ обновлён post-facto.

## Проблема

`apps/admin/src/workspaces/Orders.tsx` — воркспейс заказов в админке. После #42 (Playwright foundation) появляется **рабочий** E2E-харнесс (login + product CRUD + upload + super-admin user-mgmt), но **orders** в нём не покрыт: список, пагинация, смена статуса, фильтры — всё вручную.

Без E2E-покрытия:
- Регрессии в админском orders-flow (status update ломает инвариант) ловятся только вручную.
- UI/UX баги (не отображается customer_email, битая пагинация) уезжают в main незамеченными.

## Цель (реализовано в PR #46)

1. **`apps/admin/e2e/orders.spec.ts`** — функциональный E2E:
   - список заказов (`GET /api/admin/orders`) — отображение строк, row count;
   - смена статуса (`PATCH /api/admin/orders/{id}/status`) — валидный переход; невалидный → ошибка в UI;
   - **status filter** (через URL query param `status` — UI control отложен, но backend filter уже работает);
   - **customer filter** (через URL query param `customerId`).
2. **`data-testid`** на строки заказов + workspace root: `orders-page`, `orders-row-{id}`.
3. **Без дублирования i18n-копии или Tailwind-классов** в селекторах.

## Границы (scope)

**В scope (реализовано в PR #46):**
- E2E для существующих orders-маршрутов.
- Self-skip при недоступном backend (per #42 convention).
- `data-testid` хуки на строки.
- Stub-based invalid-status test (no dependency on backend validation order).
- Self-skip на `admin can update order status` test когда нет orders в БД (orders come from customer flow).

**Вне scope (YAGNI):**
- Полное покрытие orders-REST API (покрыто integration-тестами в #35).
- Визуальная регрессия / a11y.
- **UI controls** для status/customer filter — backend фильтрует, UI просто меняет URL. Прямой input отложен per YAGNI. **Spec покрывает** URL-based filter — путь `?status=pending` и `?customerId=42` уже работают.

## Потенциальные баги для проверки (post-hoc)

- `customer_email` join может быть NULL если customer удалён — UI должен показывать «(deleted)». (Не покрыто E2E; integration tests #35.)
- Невалидный статус → UI показывает ошибку, **не 500**. **Покрыто** в PR #46 через `page.route()` stub.
- `total_minor` отображается в правильной валюте (MDL). (Не покрыто E2E; unit-тест #38.)
- Конкурентный status update: 2 админа → stale UI. Не E2E, manual check.

## Definition of Done (per #39)

- [x] `apps/admin/e2e/orders.spec.ts` зелёный.
- [x] `data-testid` проставлены.
- [x] PR #46 прошёл ревью и смержен.
- [x] Issue #39 закрыт через PR #46.

## Архитектура (наследует #42)

- Один shared login через `setup` project + `storageState` (per #42 convention; auth.spec делает fresh login сам).
- `workers: 1` (shared backend state).
- Self-skip через `test.skip(!backendUp, ...)` если backend не отвечает.
- `waitForResponse` на реальные API + `expect.toBeVisible()` — **без** `waitForTimeout`.

**Plan:** `docs/superpowers/plans/2026-06-14-admin-e2e-orders-workspace.md` (отдельный файл).
