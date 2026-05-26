# План реализации #8: httpOnly-cookie аутентификация + CSRF

**Статус:** активный план. Работаем по review-workflow — пункт за пунктом
через апрув, по умолчанию ничего не коммитится без явного «да».

История: см. `docs/2026-05-26-deferred-cookie-auth.md` — оригинальный
дизайн на английском, по которому #8 был отложен из автономного прогона.
Этот файл — рабочий, на русском, с чек-листом прогресса.

## Цель

Убрать JWT из `localStorage` обоих SPA (источник XSS-эксфильтрации) и
держать его в `HttpOnly`-куки. Защитить state-changing запросы от CSRF
через double-submit-cookie. Поэтапно, обратимо, без слома логина в проде.

## Контекст развёртывания (важно для дизайна)

- **Админка** (`apps/admin`): `VITE_API_URL=""` → ходит на `/api/*`
  **своего origin** (nginx проксирует в бэкенд). Same-origin → cookies
  крепятся к `fetch` тривиально.
- **Сторфронт** (`apps/store`): `VITE_API_URL=https://api.mioru.store` →
  ходит на **другой origin** того же registrable domain `mioru.store`. По
  правилам SameSite это **same-site** cross-origin → `SameSite=Lax`-куки
  *прикрепляются* к `fetch` с `credentials:'include'`. Нужен
  `Access-Control-Allow-Credentials: true` в CORS.
- **Dev**: store `http://localhost:5173`, admin `http://localhost:5174`,
  API `http://localhost:8000`. Все same-site (`localhost`). `Secure` гейтим
  по `APP_ENV=production` (флаг введён в #2), иначе в dev по HTTP куки не
  поставятся.

## Дизайн-решения (требуют апрува до старта)

1. **Раздельные имена куки** для админки и сторфронта (`auth_token`/
   `csrf_token` vs `store_auth`/`store_csrf`) — оба апа могут жить под
   одним eTLD+1, и без разделения они затирают друг друга.
2. **`SameSite=Lax`** для всех куки — работает для same-site cross-origin
   fetch (наш случай), даёт baseline-CSRF (третьи стороны вообще не
   присылают куки). Strict отрезал бы навигационные переходы из писем —
   избыточно.
3. **CSRF — double-submit cookie**: при логине сервер ставит
   `csrf_token` (НЕ `HttpOnly`, JS читает); клиент эхом шлёт его в
   `X-CSRF-Token` на mutation-запросах; сервер сравнивает
   header == cookie в constant-time. Stateless, не требует серверного
   хранилища.
4. **Cookie-only, без Bearer-fallback** (по решению пользователя
   2026-05-26): средневременная поддержка Bearer не вводится. На бэке
   middleware читает токен ТОЛЬКО из куки; на mutation-запросах CSRF
   обязателен. JSON-ответы login/register БОЛЬШЕ НЕ содержат
   `access_token`. Следствие: backend cookie-cutover и оба фронт-
   cutover-а должны попадать в `main` ОДНОЙ согласованной сменой —
   между ними логин не работает. Поэтому работаем на feature-branch
   (`feat/cookie-auth`), валидируем браузером, мержим в `main` одним
   мерж-коммитом.
5. **Logout-эндпоинты**: новые `POST /api/auth/logout` и `POST
   /api/store/auth/logout` — затирают обе куки `Max-Age=0`.
6. **Self-password-change**: после успешной смены пароля фронт явно
   зовёт logout и редиректит на /login (бэк уже инвалидирует через `iat`
   в #7, фронт просто перестаёт ждать первый неудачный запрос).

## Порядок шагов (работа на ветке `feat/cookie-auth`, не в `main`)

Каждый шаг — отдельный коммит на ветке через апрув. В `main` мержим
ОДНИМ согласованным набором после успешной браузер-валидации обоих
аппов. Промежуточные состояния ветки могут быть нерабочими в браузере
(например, после Шага 1 фронты сломаны до Шага 4/5) — это ожидаемо,
именно потому ветка.

- [x] **Шаг 1. Бэк: cookie + CSRF + logout (cookie-only, ломающий).** — `ab6218a`
  - Новый `internal/cookieauth` — helpers `SetAuthCookie`,
    `ClearAuthCookie`, `GenCSRFToken`, `SetCSRFCookie`. Все флаги
    (`HttpOnly`/`SameSite=Lax`/`Path=/`/`Max-Age`/`Secure`) выставляются
    по `cfg.IsProduction`.
  - `internal/middleware/auth.go` + `customer_auth.go`: токен читаем
    ТОЛЬКО из куки (`auth_token` / `store_auth`). Без Bearer. Header
    `Authorization` игнорируется. (Тесты на Bearer удаляются.)
  - `internal/middleware/csrf.go` — middleware double-submit:
    пропускает `GET`/`HEAD`/`OPTIONS`; на mutation-запросах
    обязательно сверяет `X-CSRF-Token` header с одноимённой кукой через
    `crypto/subtle.ConstantTimeCompare`; на несовпадении → 403. Bearer
    больше не существует — спецслучая для него нет.
  - Хендлеры:
    - admin `Login` — ставит обе куки, JSON-ответ БЕЗ `access_token`
      (возвращает только `{"username":…, "role":…}`).
    - customer `Login` + customer `Register` — то же (своя пара куки).
    - admin `Register` — токен не выдаётся (это уже сделано в #4) —
      кука НЕ ставится (приглашение, не сессия).
  - Новые `POST /api/auth/logout` + `POST /api/store/auth/logout` под
    своими AuthMW — затирают обе куки.
  - `main.go` — подцепить CSRF-middleware ко всем mutation-маршрутам
    под AuthMW/CustomerAuthMW. Login/forgot/reset/store-register
    исключены (там сессии ещё нет). `Access-Control-Allow-Credentials:
    true` в CORS, проверить что credentialed-ответ не использует `*`.
  - Тесты (Go): cookie выставлена на логине; AuthMW читает из куки;
    Authorization header игнорируется; CSRF без header → 403, с
    совпадающим → 200; logout затирает куки; флаги корректны по
    `APP_ENV`.
  - **Коммит:** `feat(auth): HttpOnly-cookie auth + CSRF (cookie-only)`

- [x] **Шаг 2. Админка: переезд на куки.** — `002a063` (часть)
  - `apps/admin/src/lib/api.ts` — `credentials:'include'`, читать
    `csrf_token` из `document.cookie`, слать `X-CSRF-Token` на
    `POST`/`PUT`/`PATCH`/`DELETE`. Перестать читать/писать
    `localStorage.token`. Убрать поле `access_token` из типов.
  - `authStore` — драйвить auth-state от `GET /api/users/me` на mount,
    не хранить токен; в стейте только `user` (или `null`).
  - `Login.tsx`: после успешного POST → вызвать `getMe()` → set user
    → redirect. Никакого `localStorage.setItem`.
  - `Logout`-кнопка/действие: `POST /api/auth/logout` → clear user
    state → redirect `/login`.
  - После self-password-change: вызвать logout + redirect.
  - Тесты Vitest: `api()` шлёт `credentials:'include'`; CSRF-header
    только на мутациях; нет записи в `localStorage.token`. `npm run
    build`, `npm test` зелёные.
  - **Коммит:** `feat(admin): switch auth to HttpOnly cookies + CSRF`

- [x] **Шаг 3. Сторфронт: переезд на куки.** — `002a063` (часть)
  - То же, что для админки. Cross-origin особенности:
    - `credentials:'include'` обязателен (без него кука не шлётся);
    - cookie `Domain` не ставим (host-only `api.mioru.store`);
    - `SameSite=Lax` достаточно (same-site по eTLD+1).
  - `cartStore`/`authStore` (или как оно у стора называется) — убрать
    `localStorage.token`.
  - Тесты Vitest + `npm run build` + `npm test`.
  - **Коммит:** `feat(store): switch auth to HttpOnly cookies + CSRF`

- [ ] **Шаг 4. Браузер-валидация (вместе, до мержа).**
  - Поднять backend локально (`go run ./cmd/server`) + оба фронта
    (`npm run dev`).
  - Прокликать в обоих браузерах: login → mutation → logout; password
    reset; self password change; cross-tab; dev HTTP cookie ставится.
  - Если что-то не работает — фиксим на ветке, повторяем.

- [ ] **Шаг 5. Документация + мерж.**
  - Обновить `CLAUDE.md`:
    - «Auth» — переписать под cookie + CSRF (раздельные имена куки,
      SameSite=Lax, double-submit, logout-эндпоинты).
    - «Security posture» — упомянуть, что JWT больше НЕ хранится в
      `localStorage`, и описать CSRF-инвариант.
    - API-контракт — добавить logout, упомянуть требование
      `X-CSRF-Token` на мутациях.
  - Удалить `docs/2026-05-26-deferred-cookie-auth.md` (исторический
    deferred-док) и этот файл (`docs/cookie-auth-plan.md`) — план
    больше не активен после мержа.
  - `git merge --no-ff feat/cookie-auth` в `main`, push.

## Тесты, которые останутся «ручными»

В автоматизированных тестах нельзя проверить:
- что браузер реально шлёт куку на cross-origin fetch с
  `credentials:'include'` + `SameSite=Lax` (api.mioru.store ↔
  store.mioru.store);
- что `Secure`-кука корректно ставится по HTTPS в проде и НЕ ставится
  по dev-HTTP вне `localhost`-исключения;
- что после logout кука действительно стёрта.

Эти три прокликиваем вручную вместе на шагах 4 и 5.

## Риски и митигации

- `Access-Control-Allow-Credentials` забыт → стор-логин «успешен», но
  кука не шлётся → бесконечный 401. Митигация: тест на CORS-заголовки.
- `Secure` через dev HTTP без localhost-исключения → кука молча
  отброшена. Митигация: гейт по `APP_ENV=production`.
- Промежуточные коммиты на ветке после Шага 1 ломают логин в обоих
  аппах — это нормально, ветка не идёт в `main` до Шага 5.
- Деплой при не-готовых фронтах = моментальный логин-аут всех. Митигация:
  мерж в `main` только после браузер-валидации Шага 4 и одним мерж-
  коммитом, чтобы выкат был атомарным.
- Запись в `localStorage.token` где-то осталась → XSS-эксфильтрация
  не закрыта. Митигация: grep на `localStorage` в обоих апп ах + тест
  Vitest, фейлящийся на любую запись.

## Прогресс

- 2026-05-26 — Шаги 1–3 выполнены автономно на ветке `feat/cookie-auth`
  (`ab6218a` бэк, `002a063` оба SPA). Все автотесты зелёные: backend
  `go test ./... -race` (включая новые cookieauth/csrf/auth/handler-
  тесты), admin Vitest 29/29 + `tsc` + `vite build`, store Vitest 20/20
  на тронутых модулях + `tsc` + `vite build`. Ветка запушена в origin.
  `cartStore.test.ts` падает 8/9 — это **до-существующая поломка на
  `main`** (verified stash + re-run), вне моего изменения; почистим
  отдельной задачей.
- **Текущий шаг — 4: браузер-валидация.** Прогон оставлен пользователю.
  Что прокликать: см. список в Шаге 4 (login → mutation → logout в обоих
  SPA, password-reset, self-password-change, dev cookie ставится по
  HTTP). Backend поднимать через `cd backend/api && export
  PATH="$HOME/go-sdk/go/bin:$PATH" && export GOFLAGS=-mod=mod && go run
  ./cmd/server`. Если найдётся косяк — правлю на ветке и зову на
  повторный прогон.
- **Шаг 5 — мерж в `main`** — за пользователем (политика workflow:
  пуш/мерж по явному разрешению).
