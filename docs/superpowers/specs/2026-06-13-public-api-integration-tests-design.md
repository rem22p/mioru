# Дизайн: анализ публичного API + интеграционные тесты по базовым кейсам

Дата: 2026-06-13
Ветка: `feat/api-integration-tests`
Статус: предложен, ожидает ревью

## Проблема

У бэкенда есть тесты на двух уровнях, но между ними — дыра:

- `internal/store/*_test.go` — бьют реальный Postgres, но **только слой store**.
- `internal/handler/*_test.go` — `httptest.NewRecorder` + **фейки** (`fakeUserStore`, `fakeCustomerStore`, `oversellFakeStore`, `raceLoserFakeStore` и т.п.), `package handler` (white-box), по одному хендлеру, **без** реальной цепочки middleware, без роутинга, без реальной БД. Покрывают **логику хендлера**: выбор ветки идемпотентности (replay 201 vs 409), маппинг sentinel-ошибок (`ErrInsufficientStock`→409 `INSUFFICIENT_STOCK`, `ErrIdempotencyHashMismatch`→409 `IDEMPOTENCY_REPLAY`), валидацию individual-заказов — но всё на **симулированном** store.
- `cmd/server/main_test.go` — проверяет только конфиг сервера (таймауты, CSP, CORS), но **не** связанные маршруты.

**Пробел:** ни один тест не гоняет хендлер через реальную цепочку middleware (`AuthMW`/`CustomerAuthMW` → `RequireAdmin` → `CSRF`) против реального тестового Postgres с полным циклом запрос/ответ (куки, CSRF, заголовки). Поэтому HTTP-контракт публичного API (коды ответов, конверты ошибок с `code`, парсинг `Idempotency-Key`, поведение auth/CSRF-гейтов end-to-end) нигде не закреплён регрессионно.

## Цель

1. **Инвентарь публичного API** — единый документ-таблица по всем маршрутам из `main.go`, служащий чеклистом покрытия и точкой ревью scope.
2. **Набор интеграционных тестов по базовым кейсам** на handler-уровне с реальным Postgres: happy-path по всем публичным эндпоинтам + ключевые auth/CSRF-гейты + финансовые инварианты заказов на HTTP-уровне.

## Границы (scope)

**В scope:**
- Happy-path для каждого публичного маршрута (корректные коды/тело).
- Auth/CSRF-гейты: `401` без сессии, `403` без/с битым CSRF, `403` customer-токеном на admin-маршруте, super-admin гейт.
- Финансовые инварианты заказов **на HTTP-уровне**: списание стока, идемпотентность `CreateOrder` (replay → тот же заказ без двойного списания; conflict → `409 IDEMPOTENCY_REPLAY`; отсутствие `Idempotency-Key` → `400 VALIDATION_FAILED`), цены берутся с сервера, изоляция `ListOrders`.

**Вне scope (YAGNI для «базовых кейсов»):**
- Исчерпывающие ветки валидации каждого поля (частично уже на store-уровне).
- Полный перебор пагинации/фасетов (store-уровень покрывает: `TestListProductsPagination`, `TestListProductsFilters`).
- Рефактор `main()` для извлечения боевого `mux` — не требуется при выбранном handler-уровне.
- E2E через `httptest.NewServer` с реальным cookie-jar.
- Дублирование store-инвариантов, уже покрытых (`TestCreateOrderOversellFails`, `...IdempotencyReplay`, `...HashMismatch`, `...DecrementsStock`, `...RecalculatesPrice`).

## Архитектура решения

### Компонент 1 — общий reset-харнесс (cycle-free, DRY)

Reset-логика «чистый store + TRUNCATE» сейчас живёт в `internal/store/harness_test.go` (`resetTables`), использует приватное `s.pool` и не экспортируется — из пакета `handler` её не достать.

**Ограничение Go (важно):** все store-тесты — `package store` (white-box, обращаются к `s.pool`). Если бы `testStore(t)` делегировал в `storetest.Fresh(t)`, получился бы цикл импорта `store (тест) → storetest → store`. Поэтому делегирование из store-тестов в `storetest` **невозможно**.

Cycle-free решение с одним источником правды для списка таблиц:

1. **`internal/store/reset_testdata.go`** (не-`_test` файл, `package store`) — экспортированный метод-носитель канонического TRUNCATE:

   ```
   // ResetTestData truncates all data tables (keeps seeded categories).
   // TEST-ONLY: requires a disposable DB; never call against production.
   func (s *PostgresStore) ResetTestData(ctx context.Context) error
   ```

   Имеет доступ к `s.pool` (тот же пакет). Список таблиц живёт **только здесь** — единственный источник правды.

2. **`internal/store/harness_test.go`** — `resetTables` переписывается на вызов `s.ResetTestData(ctx)` (white-box тесты пакета `store` остаются рабочими, без цикла).

3. **`internal/storetest/storetest.go`** (`package storetest`, импортирует `store`) — helper для внешних пакетов:

   ```
   func Fresh(t testing.TB) *store.PostgresStore
   ```
   - `t.Skip()` если `TEST_DATABASE_URL` пуст.
   - `store.NewPostgresStore(ctx, url)` (миграции + сид категорий) → `s.ResetTestData(ctx)` → `t.Cleanup(s.Close)`.
   - Используется тестами пакета `handler` (нет цикла: `handler_test → storetest → store`).

**Тредофф:** `ResetTestData` — экспортированный деструктивный метод на production-типе. Это осознанный компромисс: in-package white-box store-тесты + внешний `storetest` оба должны звать reset, а ограничение цикла импорта Go не даёт иначе сохранить один источник правды. Митигируется явным test-only doc-комментом и тем, что метод не может сработать случайно (нужен явный вызов).

### Компонент 2 — интеграционный харнесс (`package handler_test`)

Файл `internal/handler/integration_harness_test.go`:

- `newCustomerSession(t, st)` / `newAdminSession(t, st, role)` — создают customer/user в БД, минтят реальный JWT через `auth`, возвращают структуру с куками (`store_auth`+`store_csrf` / `auth_token`+`csrf_token`).
- `wrapCustomer(h)` / `wrapAdmin(h)` / `wrapSuperAdmin(h)` — оборачивают хендлер в **ту же** цепочку middleware, что собирает `main.go` (`CustomerAuthMW`+`CSRF`, `AuthMW`+`RequireAdmin`+`CSRF`, `AuthMW`+`RequireSuperAdmin`+`CSRF`). Зависимости middleware (`SecretKey`, `PasswordChangedAt`-лукапы, `getRole`) берутся из реального store.
- `do(t, handler, method, path, opts)` — request-helper: ставит куки, `X-CSRF-Token`, `Idempotency-Key`, JSON-body; гоняет через `httptest.NewRecorder`; возвращает `(status, headers, body)`.

Это даёт реальное прохождение auth/CSRF-гейтов по каждому маршруту **без** извлечения боевого `mux` из `main()`: композим те же middleware вокруг хендлера в тесте.

### Компонент 3 — тест-файлы по доменам

| Файл | Покрытие |
|---|---|
| `integration_storefront_test.go` | `GET /api/products`, `/facets`, `/{slug}`, `/categories`; customer register/login/me/logout; профиль; cart/favorites round-trip |
| `integration_orders_test.go` | `CreateOrder` happy-path через **реальный store**; **end-to-end инварианты** (fake-юниты их не доказывают): реальное списание стока в БД (проверка после заказа); idempotency replay — тот же ключ+тело дважды через реальный хендлер+БД → оба `201`, **тот же `order.id`**, сток списан **однократно**; conflict (тот же ключ, другое тело) → `409 IDEMPOTENCY_REPLAY`; oversell (qty > stock) → `409 INSUFFICIENT_STOCK`; missing `Idempotency-Key` → `400 VALIDATION_FAILED`; цены пересчитаны сервером; `ListOrders` пагинация+изоляция; admin `PATCH /orders/{id}/status` |
| `integration_admin_test.go` | admin product CRUD happy-path (с канонизацией `status`: legacy `pre_order`→`preorder`, `none`→`out_of_stock` из `product_form`, миграция `011_product_status_check.sql`); гейты: `401` без куки, `403` customer-токеном на admin, `403` без/с битым CSRF; users — super-admin гейт |

### Компонент 4 — документ-инвентарь

`docs/api/public-api-inventory.md` — таблица по всем маршрутам из `main.go`:

`method+path | auth | csrf | rate-limit | вход | success-код/тело | error-коды | тест-кейсы (новые / уже на store-уровне)`

Пишется **первым** в фазе реализации — служит чеклистом, по которому верифицируется полнота тестов.

## Поток данных (типичный тест)

```
storetest.Fresh(t) → чистый PostgresStore (реальная test-БД)
  → newCustomerSession(t, st) → создан customer + куки
    → do(t, wrapCustomer(customerH.CreateOrder), "POST", "/api/store/orders",
          {cookies, csrf, idempotencyKey, body})
      → реальная цепочка: CustomerAuthMW → CSRF → CreateOrder → store.CreateOrder → БД
        → assert: 201, тело заказа, сток в БД уменьшен
```

## Обработка ошибок (что пиннят тесты)

- Конверт `{error, code}` с правильным машинным `code` на 4xx (`VALIDATION_FAILED`, `AUTH_REQUIRED`/`AUTH_INVALID`, `CSRF_INVALID`, `FORBIDDEN`, `IDEMPOTENCY_REPLAY`, `INSUFFICIENT_STOCK`, `NOT_FOUND`). Примечание: `INSUFFICIENT_STOCK` используется хендлером `CreateOrder`, но **отсутствует** в зарезервированном списке кодов CLAUDE.md — инвентарь это фиксирует как кандидат на сверку (issue, если решим расхождением).
- 5xx → ровно `{"error":"Internal error","code":"INTERNAL"}` без утечки `err.Error()`.

## Тестирование

- Запуск: docker test-PG (`:55433`, `mioru_test`) + `TEST_DATABASE_URL`, `go test ./... -race`. Никогда не боевая БД.
- Без `time.Sleep`; время — через инъекцию clock там, где уже поддержано (store-уровень).
- Все новые тесты `t.Skip()` при пустом `TEST_DATABASE_URL` (как store-тесты).

## Риски

- **Рефактор `harness_test.go`** может задеть существующие store-тесты — верифицируем зелёный прогон store-пакета до и после.
- **Telegram fire-and-forget в `CreateOrder`** стартует горутину; в тестах `tgNotifier` инжектим `nil` (хендлер это уже проверяет) → без сетевых вызовов.

## Найденные баги → GitHub issue

Интеграционные тесты могут вскрыть реальные баги в продакшен-коде (а не в самих тестах). Процесс:

- **Баг теста** (неверная ассерта/сетап) — чиним на месте.
- **Реальный баг продакшен-кода** (неверный код ответа, утечка `err.Error()`, дырка в гейте, нарушение инварианта) — **не чиним молча в этой ветке**. Заводим GitHub issue (`gh issue create`) с: маршрутом, ожидаемым vs фактическим поведением, ссылкой на тест, severity. Тест помечается `t.Skip("blocked by #N: <кратко>")` или фиксирует фактическое (баговое) поведение с комментарием `// FIXME(#N)`, чтобы не блокировать остальную сетку. Issue линкуется в PR.

Решение «скип или зафиксировать факт» принимается по severity: гейт безопасности / финансовый инвариант — скип + issue (не закрепляем дыру как «ожидаемое»); косметика контракта — фиксируем факт + `FIXME(#N)`.

## Критерии готовности

1. `docs/api/public-api-inventory.md` покрывает все маршруты из `main.go`.
2. Каждая строка инвентаря с пометкой «новый кейс» имеет соответствующий интеграционный тест.
3. `go test ./... -race` зелёный при поднятой test-БД; store-пакет не регрессировал.
4. Тесты падают, если убрать auth/CSRF-гейт или сломать идемпотентность (проверяется при написании — red→green).
