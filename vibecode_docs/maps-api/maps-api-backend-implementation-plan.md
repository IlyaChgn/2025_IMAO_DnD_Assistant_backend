# Maps API Backend — План реализации (доработка)

> **Статус**: **GO** — все must-fix закрыты (2026-02-05). Реализация разрешена. См. L.3 (closed), M (PR Plan).
> Модуль `maps` уже существует в репозитории и содержит базовую реализацию CRUD.
> Данный план фокусируется на **выявленных расхождениях** между текущим кодом и OpenAPI-спецификацией (`maps-api-spec.yaml`) + implementation notes (`maps-api-implementation-notes.md`) и описывает конкретные правки.

---

## A) Анализ текущего репозитория

### A.1 Регистрация HTTP-роутов

- **Файл**: `internal/pkg/server/delivery/routers/router.go` — главная функция `NewRouter(...)` принимает все usecases и создаёт `mux.Router`.
- **Файл**: `internal/pkg/server/delivery/routers/maps.go` — функция `ServeMapsRouter()` регистрирует маршруты:
  ```
  /maps       GET   → ListMaps
  /maps       POST  → CreateMap
  /maps/{id}  GET   → GetMapByID
  /maps/{id}  PUT   → UpdateMap
  /maps/{id}  DELETE→ DeleteMap
  ```
- Все маршруты уже защищены `loginRequiredMiddleware` через `subrouter.Use(...)`.
- Базовый путь API: `/api` (задаётся в `NewRouter` через `router.PathPrefix("/api")`).

### A.2 Auth middleware и извлечение userId

- **Файл**: `internal/pkg/middleware/auth/login_required.go`
- Middleware `LoginRequiredMiddleware(uc, ctxUserKey)`:
  1. Достаёт cookie `session_id`
  2. Вызывает `uc.CheckAuth(ctx, session.Value)` → `(*models.User, bool)`
  3. Кладёт `*models.User` в контекст по ключу `ctxUserKey` (строка из `config.yaml`, поле `user_key`)
- **Извлечение в хендлерах**:
  ```go
  user := ctx.Value(h.ctxUserKey).(*models.User)
  userID := user.ID
  ```
- **Тип User**: `internal/models/user.go` — `type User struct { ID int; DisplayName string; ... }`

### A.3 Формат ошибок и JSON-ответов

**Общий (старый) формат** (`internal/pkg/server/delivery/responses/responses.go`):
- `SendOkResponse(w, body)` — код 200, `Content-Type: application/json`
- `SendErrResponse(w, code, status)` — `{ "status": "error message" }`

**Maps-специфичный формат** (уже реализован в `internal/pkg/maps/delivery/maps_handlers.go`):
- `sendMapsError(w, code, errCode, message, details)` → `{ "error": "...", "message": "...", "details": [...] }`
- `sendMapsOkResponse(w, code, body)`
- Модель: `models.MapsErrorResponse { Error, Message, Details []ValidationError }`

**App errors** (`internal/pkg/apperrors/maps.go`):
- `MapNotFoundError`, `MapPermissionDenied`, `InvalidMapNameError`, `InvalidSchemaVersion`, `InvalidDimensionsError`, `InvalidPlacementError`, `MapValidationError`, `InvalidUserIDError`

### A.4 Текущая структура модуля Maps

```
internal/pkg/maps/
├── interfaces.go                    # MapsRepository + MapsUsecases
├── mocks/mock_maps.go               # gomock
├── delivery/
│   ├── maps_handlers.go             # 5 хендлеров + sendMapsError/sendMapsOkResponse
│   └── maps_handlers_test.go        # Unit-тесты хендлеров (fake usecase)
├── usecases/
│   ├── maps.go                      # Бизнес-логика CRUD
│   ├── maps_test.go                 # Unit-тесты usecases (gomock)
│   └── validator.go                 # ValidateMapRequest + CategorizeValidationErrors
│   └── validator_test.go            # Табличные тесты валидатора
└── repository/
    ├── maps_queries.go              # SQL-константы
    └── maps_storage.go              # pgx-реализация MapsRepository
```

### A.5 Миграция БД

- **Файл**: `db/migrations/006_maps.up.sql` — таблица `maps` с JSONB `data`, индексы `(user_id)` и `(user_id, updated_at DESC)`, триггер `update_maps_updated_at` для автоматического обновления `updated_at`.
- Последняя миграция: `010_drop_vkid`.

---

## B) Расхождения с OpenAPI-спецификацией

### B.1 `MapUnitsPerTile` = 6, а по спеке = 8 — проверка кратности НЕ является обязательной

| Параметр | Текущее значение | Спецификация (mandatory) | Спецификация (optional) |
|----------|-----------------|--------------------------|------------------------|
| `MapUnitsPerTile` | 6 (обязательная проверка кратности) | `widthUnits > 0`, `heightUnits > 0` | Кратно 8 (MACRO_CELL_UNITS) |

**Файлы**: `internal/pkg/maps/usecases/validator.go:15`

**Проблема**: Сейчас валидатор **обязательно** проверяет кратность (причём числу 6, а не 8). Но implementation notes явно разделяют:
- **MUST validate**: `widthUnits` / `heightUnits` — "Positive integer" (т.е. `> 0`, без кратности).
- **Optional backend validation**: "widthUnits and heightUnits divisible by 8", "Placements x, y divisible by 8".

**Решение (принято)**: Убрать проверку кратности из обязательной валидации. Оставить только `> 0`. Удалить константу `MapUnitsPerTile`. Если в будущем потребуется опциональная проверка кратности 8 — добавить отдельную функцию `ValidateOptionalGridAlignment()`, которая вызывается по флагу/конфигу, но не блокирует сохранение.

**Обоснование**: Это единственный вариант, который (a) не ломает существующие данные в БД, (b) строго соответствует notes, (c) не требует миграции данных.

### B.2 Формат ответа ListMaps — массив vs объект

| Параметр | Текущий формат | Спецификация |
|----------|---------------|--------------|
| GET /maps response | `{ "maps": [...], "total": N }` | `[MapMetadata, ...]` (плоский массив) |

**Файлы**:
- `internal/models/maps.go` — `MapsList struct`
- `internal/pkg/maps/repository/maps_storage.go` — `ListMaps()` возвращает `*models.MapsList`
- `internal/pkg/maps/usecases/maps.go` — `ListMaps()`
- `internal/pkg/maps/delivery/maps_handlers.go` — `ListMaps()`
- `internal/pkg/maps/interfaces.go` — сигнатура `ListMaps`

**Что делать**: Изменить возвращаемый тип с `*models.MapsList` на `[]models.MapMetadata`. Убрать `CountMapsQuery` и дополнительный запрос `SELECT COUNT(*)`, т.к. спека не требует `total`. Если `total` понадобится в будущем, можно добавить header `X-Total-Count` или вернуть обёртку — но пока спека требует плоский массив.

> **RESOLVED (2026-02-05)**: Фронтенд НЕ вызывает `/api/maps` — Maps API является новой фичей. Ноль совпадений по grep `/api/maps` в `src/`. Следуем спеке: плоский массив, без `total`.

### B.3 `UpdateMapRequest.Name` — опциональное поле

| Параметр | Текущая реализация | Спецификация |
|----------|--------------------|--------------|
| `name` в UpdateMapRequest | Обязательное (валидируется `len >= 1`) | Опциональное (не в `required`) |
| `data` в UpdateMapRequest | Обязательное | Обязательное (в `required`) |

**Файлы**:
- `internal/models/maps.go:55-58` — `UpdateMapRequest { Name string; Data MapData }`
- `internal/pkg/maps/usecases/maps.go:69` — `ValidateMapRequest(req.Name, &req.Data)` — всегда валидирует name
- `internal/pkg/maps/usecases/validator.go:23-28` — проверка `len(name) < 1` → ошибка
- `internal/pkg/maps/repository/maps_queries.go:24-28` — `SET name = $3` (всегда перезаписывает)

**Что делать**:
1. Изменить `UpdateMapRequest.Name` на `*string` (pointer для отличия «не передали» от «пустая строка»):
   ```go
   type UpdateMapRequest struct {
       Name *string  `json:"name,omitempty"`
       Data MapData  `json:"data"`
   }
   ```
2. В отдельной функции `ValidateUpdateMapRequest(name *string, data *MapData)`:
   - Если `name == nil` — пропустить (поле не передано, имя не меняется).
   - Если `name != nil && *name == ""` — **ошибка** (422 `INVALID_NAME`): спека требует `minLength: 1`.
   - Если `name != nil && len(*name) > 255` — **ошибка** (422 `INVALID_NAME`).
   - Если `name != nil` и длина ок — передать значение в SQL.
3. В SQL использовать `COALESCE`:
   ```sql
   UPDATE public.maps
   SET name = COALESCE($3, name), data = $4
   WHERE id = $1 AND user_id = $2
   RETURNING id, user_id, name, data, created_at, updated_at;
   ```
4. В `UpdateMap` репозитории: если `req.Name == nil`, передавать `nil` в параметр `$3` (pgx поддерживает `*string` напрямую).
5. Обновить тесты: кейсы `name=nil` (ок), `name=""` (422), `name="Valid"` (ок).

### B.4 Ownership check: 404 vs 403 (не различаются)

| Сценарий | Текущее поведение | Спецификация + Notes |
|----------|-------------------|----------------------|
| Карта не существует | `CheckPermission` → false → 403 FORBIDDEN | **404 NOT_FOUND** |
| Карта чужая | `CheckPermission` → false → 403 FORBIDDEN | **403 FORBIDDEN** |

**Текущая проблема**: `CheckPermission` делает `SELECT EXISTS(... WHERE id=$1 AND user_id=$2)`. Если карта не существует — тоже `false`. Usecase всегда возвращает `MapPermissionDenied` (403) для обоих случаев, хотя delivery-слой уже корректно маппит `MapNotFoundError → 404` и `MapPermissionDenied → 403` — просто usecase никогда не возвращает `MapNotFoundError` через CheckPermission.

**Что требуют spec и notes (однозначно)**:
- OpenAPI spec определяет **оба** ответа 403 и 404 для GET/PUT/DELETE `/maps/{id}`.
- Notes явно: _"Return 403 if not owner"_, _"Return 404 if map doesn't exist (prevents enumeration)"_.
- Псевдокод из notes: `if not map: raise NotFoundError; if map.user_id != user_id: raise ForbiddenError`.
- Тест-чеклист из notes: "Get map with wrong user → 403", "Get nonexistent map → 404", "Delete other user's map → 403", "Delete nonexistent map → 404".

**Решение (принято — Вариант Б)**: Различать 403 и 404. Заменить `CheckPermission` (SELECT EXISTS) на метод `CheckOwnership`, который:
1. Выполняет `SELECT user_id FROM maps WHERE id = $1`
2. Если строка не найдена → возвращает `MapNotFoundError`
3. Если `user_id != session.userId` → возвращает `MapPermissionDenied`
4. Если всё ок → возвращает `nil`

Это соответствует и спеке, и notes, и текущему delivery-коду (который уже маппит оба кода корректно).

> **Замечание о enumeration**: Да, 403 позволяет узнать, что карта с данным UUID существует. Для карт с UUID (128-bit) это практически не является вектором атаки. Notes выбирают этот подход осознанно.

### B.5 Прочие мелкие расхождения

| # | Текущее | Спецификация | Действие |
|---|---------|--------------|----------|
| 1 | `CategorizeValidationErrors` проверяет `err.Field[:15] == "data.placements"` | Хрупкая проверка по длине строки | Заменить на `strings.HasPrefix(err.Field, "data.placements")` |
| 2 | `layer` в `Placement` имеет тип `int` | Спека: `minimum: 0, default: 0` | Добавить валидацию `layer >= 0` (сейчас не валидируется) |
| 3 | Хендлер DeleteMap не обрабатывает `MapNotFoundError` отдельно | Спека: 404 для несуществующей карты | Delivery уже маппит `MapNotFoundError → 404` ✅ (ветка есть в текущем коде). С новым `CheckOwnership` будет работать корректно |

---

## C) База данных и миграции

### C.1 Текущая схема (уже создана)

Миграция `006_maps.up.sql` полностью соответствует рекомендациям из implementation notes:
- UUID PK с `gen_random_uuid()`
- `user_id BIGINT NOT NULL REFERENCES public.user(id)`
- `name VARCHAR(255) NOT NULL` с CHECK-ограничениями
- `data JSONB NOT NULL`
- `created_at TIMESTAMPTZ`, `updated_at TIMESTAMPTZ`
- Индексы: `maps_user_id_idx`, `maps_user_id_updated_at_idx`
- Триггер `maps_updated_at_trigger` для автообновления `updated_at`

### C.2 Необходимые миграции

Новых миграций **не требуется**. Схема БД полностью покрывает требования спецификации.

Индекс `idx_maps_data_schema ON maps((data->>'schemaVersion'))` из notes **не создан** в текущей миграции. Он нужен для будущей фильтрации карт по версии схемы.

**Опциональная миграция** (если решим добавить):
- **Файл**: `db/migrations/011_maps_schema_version_index.up.sql`
  ```sql
  CREATE INDEX IF NOT EXISTS maps_data_schema_version_idx
  ON public.maps ((data->>'schemaVersion'));
  ```
- **Файл**: `db/migrations/011_maps_schema_version_index.down.sql`
  ```sql
  DROP INDEX IF EXISTS maps_data_schema_version_idx;
  ```

### C.3 Подход к updated_at

Триггер `update_maps_updated_at()` уже создан — **явное обновление в коде не нужно**. Это правильный подход для данного проекта: encounter-модуль тоже не обновляет `updated_at` вручную (хотя у него нет такого триггера). Триггер гарантирует консистентность даже при прямых SQL-правках.

---

## D) Валидация и ошибки

### D.1 Обязательные правила валидации

| Поле | Правило | Слой | Текущий статус |
|------|---------|------|----------------|
| `name` (create) | 1 ≤ len ≤ 255 | usecases | ✅ Реализовано |
| `name` (update) | 1 ≤ len ≤ 255, **только если передан** | usecases | ❌ Всегда валидируется |
| `data.schemaVersion` | == 1 | usecases | ✅ Реализовано |
| `data.widthUnits` | > 0 | usecases | ✅ Реализовано |
| `data.heightUnits` | > 0 | usecases | ✅ Реализовано |
| `data.placements[*].id` | непустая строка | usecases | ✅ Реализовано |
| `data.placements[*].tileId` | непустая строка | usecases | ✅ Реализовано |
| `data.placements[*].x` | ≥ 0 | usecases | ✅ Реализовано |
| `data.placements[*].y` | ≥ 0 | usecases | ✅ Реализовано |
| `data.placements[*].rot` | ∈ [0, 1, 2, 3] | usecases | ✅ Реализовано |
| `data.placements[*].layer` | ≥ 0 | usecases | ❌ Не валидируется |

### D.2 Опциональные правила валидации (frontend-driven)

| Поле | Правило | Текущий статус | Рекомендация |
|------|---------|----------------|--------------|
| `widthUnits` | кратно 8, от 8 до 512 | ❌ Сейчас кратно 6 | Вынести в отдельную функцию или убрать |
| `heightUnits` | кратно 8, от 8 до 512 | ❌ Сейчас кратно 6 | Вынести в отдельную функцию или убрать |
| `placements[*].x` | кратно 8 | ❌ Сейчас кратно 6 | Вынести в отдельную функцию или убрать |
| `placements[*].y` | кратно 8 | ❌ Сейчас кратно 6 | Вынести в отдельную функцию или убрать |
| placements в пределах bounds | x + tileWidth ≤ widthUnits | ❌ Нет | Опционально |

### D.3 Разделение ответственности по слоям

| Слой | Что валидирует |
|------|---------------|
| **delivery** | Парсинг JSON (400 BAD_REQUEST), валидация UUID формата path-параметра (400), парсинг query-параметров start/size |
| **usecases** | Бизнес-правила: name длина, schemaVersion, dimensions, placements поля, ownership (403/404) |
| **repository** | DB constraints (CHECK, NOT NULL) — как fallback |

### D.4 Таблица «сценарий → HTTP статус → error code»

| Сценарий | HTTP | Error Code | Текущий статус |
|----------|------|------------|----------------|
| Невалидный JSON | 400 | `BAD_REQUEST` | ✅ |
| UUID path-параметра невалиден | 400 | `BAD_REQUEST` | ✅ |
| Невалидное name (пустое, >255) | 422 | `INVALID_NAME`¹ | ❌ Сейчас 422 + `BAD_REQUEST` |
| Нет аутентификации | 401 | `UNAUTHORIZED` | ✅ (middleware) |
| Чужая карта | 403 | `FORBIDDEN` | ❌ Сейчас всегда 403 (не различает с not found) |
| Карта не найдена | 404 | `NOT_FOUND` | ❌ Сейчас 403 (через CheckPermission) |
| schemaVersion != 1 | 422 | `INVALID_SCHEMA_VERSION` | ✅ |
| widthUnits/heightUnits ≤ 0 | 422 | `INVALID_DIMENSIONS` | ✅ |
| rot вне [0..3] | 422 | `INVALID_PLACEMENT` | ✅ |
| x или y < 0 | 422 | `INVALID_PLACEMENT` | ✅ |
| id/tileId пустые | 422 | `INVALID_PLACEMENT` | ✅ |
| layer < 0 | 422 | `INVALID_PLACEMENT` | ❌ Нет валидации |
| Внутренняя ошибка | 500 | `INTERNAL_ERROR` | ✅ |

¹ **Новый код `INVALID_NAME`**: Notes привязывают `BAD_REQUEST` строго к HTTP 400 (синтаксические ошибки). Имя, не проходящее `minLength/maxLength` — семантическая ошибка (JSON валиден, поле присутствует, значение некорректно), поэтому HTTP 422 + собственный код. Добавить `INVALID_NAME` в `CategorizeValidationErrors` для `field == "name"` и в `apperrors/maps.go`.

### D.5 Формат error response

Текущий формат `MapsErrorResponse` **уже соответствует спеке**:
```json
{
  "error": "VALIDATION_ERROR",
  "message": "Human-readable message",
  "details": [
    { "field": "data.schemaVersion", "message": "Must be 1" }
  ]
}
```

Совпадает с `ErrorResponse` из OpenAPI spec. Никаких изменений не нужно.

---

## E) Интеграция с существующими компонентами

### E.1 Логирование

Уже реализовано корректно:
- `l := logger.FromContext(ctx)` в каждом хендлере и usecase
- `l.DeliveryError(...)`, `l.UsecasesWarn(...)`, `l.RepoError(...)` с контекстными данными

### E.2 Метрики

Уже реализовано:
- `dbcall.DBCall` / `dbcall.ErrOnlyDBCall` оборачивают все запросы в метрики через `mymetrics.DBMetrics`
- HTTP-метрики применяются через middleware в `NewRouter`

### E.3 Совместимость с maptiles

`tileId` в `Placement` — просто строка-ссылка на тайл из MongoDB коллекции `map_tiles`. Никаких JOIN'ов нет, referential integrity не проверяется. Это корректно по спеке: _"tileId строка — только хранение ссылки, без join'ов"_.

### E.4 CORS

Настраивается глобально в `internal/pkg/server/app.go` через `handlers.CORS(...)`. Maps-эндпоинты автоматически покрыты.

---

## F) Test & Quality Strategy

> **Принцип**: ≤ 12 проверок суммарно. Каждая привязана к конкретному риску/контракту из плана. Предпочтение: маленький стабильный набор вместо большого flaky.
>
> **Почему 12, а не 25+**: Предыдущая версия раздела перечисляла ~23 теста (13 delivery + 4 usecase + 4 validator + интеграционные), многие дублировали проверки между слоями (один и тот же error → один HTTP-код, но проверяется на 2 уровнях). Сокращение до 12 покрывает все **уникальные риски** из разделов B, D, H, I без дублирования. Delivery-тесты оставлены только для контрактов, которые не покрываются usecase-уровнем (формат тела 204, `[]` vs `null`, JSON parse error → 400).

### F.1 Minimal Test Set (12 проверок)

| # | Название | Уровень | Что проверяет (риск → раздел) | Фикстуры | Команда |
|---|---------|---------|------------------------------|----------|---------|
| 1 | `TestValidateMapRequest_NoDivisibilityCheck` | unit | Убрана кратность 6/8 → **B.1, D.2**. `widthUnits=13, heightUnits=7` (>0, не кратны) → 0 ошибок | `validator_test.go`: `MapData{SchemaVersion:1, WidthUnits:13, HeightUnits:7, Placements:[]{valid}}` | `go test -mod=vendor -run NoDivisibility ./internal/pkg/maps/usecases/...` |
| 2 | `TestValidateMapRequest_LayerNegative` | unit | Новая валидация `layer >= 0` → **B.5#2, D.1**. `layer=-1` → error field `data.placements[0].layer` | `validator_test.go`: один Placement с `Layer:-1`, остальное валидно | `go test -mod=vendor -run LayerNeg ./internal/pkg/maps/usecases/...` |
| 3 | `TestCategorize_NameField_ReturnsINVALID_NAME` | unit | Новый error code → **D.4, H.сводная**. `field=="name"` → `"INVALID_NAME"` | `validator_test.go`: `[]ValidationError{{Field:"name", Message:"too short"}}` | `go test -mod=vendor -run Categorize_Name ./internal/pkg/maps/usecases/...` |
| 4 | `TestValidateUpdateMapRequest_NameNil_OK` | unit | name optional → **B.3, H.PUT**. `name=nil` → 0 ошибок | `validator_test.go`: `ValidateUpdateMapRequest(nil, &validData)` | `go test -mod=vendor -run UpdateMap.*NameNil ./internal/pkg/maps/usecases/...` |
| 5 | `TestValidateUpdateMapRequest_NameEmpty_Error` | unit | name="" → 422 → **B.3, D.4**. Пустая строка → error field `name` | `validator_test.go`: `empty := ""; ValidateUpdateMapRequest(&empty, &validData)` | `go test -mod=vendor -run UpdateMap.*NameEmpty ./internal/pkg/maps/usecases/...` |
| 6 | `TestGetMapByID_Ownership_NotFound_404` | unit | CheckOwnership → `MapNotFoundError` → **B.4, H.GET** | `maps_test.go` (gomock): mock `CheckOwnership` returns `MapNotFoundError` | `go test -mod=vendor -run Ownership_NotFound ./internal/pkg/maps/usecases/...` |
| 7 | `TestGetMapByID_Ownership_WrongUser_403` | unit | CheckOwnership → `MapPermissionDenied` → **B.4, H.GET** | `maps_test.go` (gomock): mock `CheckOwnership` returns `MapPermissionDenied` | `go test -mod=vendor -run Ownership_WrongUser ./internal/pkg/maps/usecases/...` |
| 8 | `TestListMaps_EmptySlice_NotNull` | unit | `[]` vs `null` → **B.2, H.GET /maps, K.3** | `maps_handlers_test.go`: mock usecase returns `[]models.MapMetadata{}` (not nil). Assert body == `[]` (2 bytes), Content-Type == `application/json` | `go test -mod=vendor -run EmptySlice ./internal/pkg/maps/delivery/...` |
| 9 | `TestDeleteMap_204_NoBody_NoContentType` | unit | 204 contract → **H.DELETE** | `maps_handlers_test.go`: mock usecase → nil error. Assert: code==204, body=="", header `Content-Type` absent | `go test -mod=vendor -run 204_NoBody ./internal/pkg/maps/delivery/...` |
| 10 | `TestCreateMap_BadJSON_400` | unit | JSON parse → 400 BAD_REQUEST → **D.4, H.POST** | `maps_handlers_test.go`: body `{invalid`. Assert: code==400, `"error":"BAD_REQUEST"` | `go test -mod=vendor -run BadJSON ./internal/pkg/maps/delivery/...` |
| 11 | `TestUpdateMap_NameOmitted_PreservesOldName` | unit | COALESCE contract → **B.3, H.PUT** | `maps_test.go` (gomock): `UpdateMapRequest{Name:nil, Data:valid}`. Mock repo verifies arg `name==nil`, returns MapFull with old name | `go test -mod=vendor -run NameOmitted ./internal/pkg/maps/usecases/...` |
| 12 | `make verify` + build | quality gate | Моки актуальны, gofmt, go vet, все тесты → **J.4, J.5** | — | `make verify && go build -mod=vendor ./cmd/app/main.go` |

> **Интеграционные тесты (repository)**: N/A. В проекте есть `make test-integration` (`Makefile:17`), но нет готовых PostgreSQL-фикстур для maps. SQL-корректность верифицируется unit-тестами usecase-слоя (gomock) + ручным smoke (F.4). При появлении тестовой БД-инфраструктуры — добавить один CRUD-цикл как первый интеграционный кейс.
>
> **E2E автоматизированные**: N/A — нет тестового окружения с auth. Заменяется ручным API smoke (F.4).

### F.2 Anti-Flake Measures

| Мера | Как обеспечивается |
|------|--------------------|
| **Изоляция данных** | Все unit-тесты используют gomock (usecases) или `httptest.ResponseRecorder` (delivery). Нет общего состояния, нет БД, нет сети |
| **Нет зависимости от времени** | Тесты не проверяют значения `updatedAt`/`createdAt`. Timestamp-поля в моках — фиксированные (`time.Date(2025,1,1,0,0,0,0,time.UTC)`) |
| **Нет гонок** | `go test -race` доступен через `make test-race` (Makefile:10). Рекомендуется запускать перед merge |
| **Нет sleep/ожиданий** | Все тесты синхронные, без goroutine, каналов, таймеров |
| **UUID детерминированность** | В тестах — фиксированные строки (`"550e8400-e29b-41d4-a716-446655440000"`), не `uuid.New()` |
| **Retry** | N/A — чистые unit-тесты не требуют retry |

### F.3 CI Quality Gates

| # | Gate | Команда | Ожидание | Источник |
|---|------|---------|----------|----------|
| 1 | gofmt | `gofmt -l ./internal/ ./cmd/ ./db/` | Пустой вывод | `Makefile:27` |
| 2 | Mocks regen | `GOFLAGS=-mod=vendor go generate -run mockgen ./internal/...` | `git diff --exit-code` → 0 (нет diff) | `Makefile:4`, **J.5** |
| 3 | go vet | `go vet -mod=vendor ./...` | Exit 0 | `Makefile:31` |
| 4 | Unit tests | `go test -mod=vendor ./...` | 0 failures | `Makefile:7`, **J.4** |
| 5 | Build | `go build -mod=vendor ./cmd/app/main.go` | Exit 0 | `CLAUDE.md` |

**Одной командой**: `make verify && go build -mod=vendor ./cmd/app/main.go`

> **golangci-lint**: N/A — в проекте нет `.golangci.yml` в корне. Используются `gofmt` + `go vet` (Makefile target `verify`).

### F.4 How to Verify Manually (10 шагов)

Предусловие: сервер запущен (`go run cmd/app/main.go`), БД+Redis подняты (`docker compose up -d`), есть валидная сессия.

```bash
SESSION="session_id=<valid_cookie_value>"
```

| # | Действие | Команда | Ожидание |
|---|----------|---------|----------|
| 1 | POST create | `curl -s -w "\n%{http_code}" -X POST http://localhost:8080/api/maps -H "Cookie: $SESSION" -H "Content-Type: application/json" -d '{"name":"Test","data":{"schemaVersion":1,"widthUnits":13,"heightUnits":7,"placements":[]}}'` | **201** + JSON с `id`, `name:"Test"`. Запомнить `<uuid>` |
| 2 | GET list (не пусто) | `curl -s -w "\n%{http_code}" "http://localhost:8080/api/maps?start=0&size=10" -H "Cookie: $SESSION"` | **200** + JSON-массив `[{"id":"...",...}]` (не объект `{"maps":[...]}`) |
| 3 | GET by id | `curl -s -w "\n%{http_code}" http://localhost:8080/api/maps/<uuid> -H "Cookie: $SESSION"` | **200** + MapFull с `data` |
| 4 | GET invalid UUID | `curl -s -w "\n%{http_code}" http://localhost:8080/api/maps/not-a-uuid -H "Cookie: $SESSION"` | **400** + `"error":"BAD_REQUEST"` |
| 5 | GET nonexistent | `curl -s -w "\n%{http_code}" http://localhost:8080/api/maps/00000000-0000-0000-0000-000000000000 -H "Cookie: $SESSION"` | **404** + `"error":"NOT_FOUND"` |
| 6 | PUT без name | `curl -s -w "\n%{http_code}" -X PUT http://localhost:8080/api/maps/<uuid> -H "Cookie: $SESSION" -H "Content-Type: application/json" -d '{"data":{"schemaVersion":1,"widthUnits":64,"heightUnits":64,"placements":[]}}'` | **200** + `name:"Test"` (сохранилось) |
| 7 | PUT name="" | `curl -s -w "\n%{http_code}" -X PUT http://localhost:8080/api/maps/<uuid> -H "Cookie: $SESSION" -H "Content-Type: application/json" -d '{"name":"","data":{"schemaVersion":1,"widthUnits":64,"heightUnits":64,"placements":[]}}'` | **422** + `"error":"INVALID_NAME"` |
| 8 | DELETE | `curl -s -w "\n%{http_code}" -X DELETE http://localhost:8080/api/maps/<uuid> -H "Cookie: $SESSION"` | **204** + пустое тело |
| 9 | DELETE повторный | `curl -s -w "\n%{http_code}" -X DELETE http://localhost:8080/api/maps/<uuid> -H "Cookie: $SESSION"` | **404** + `"error":"NOT_FOUND"` |
| 10 | GET list (пусто) | `curl -s -w "\n%{http_code}" "http://localhost:8080/api/maps?start=0&size=10" -H "Cookie: $SESSION"` | **200** + `[]` (пустой массив, не `null`) |

---

## G) План работ по шагам

### Этап 1: Исправить ownership-check (Вариант Б — различать 403/404)

- [ ] **`internal/pkg/maps/interfaces.go`** — заменить `CheckPermission(ctx, id, userID) bool` на `CheckOwnership(ctx, id, userID) error` (возвращает `nil` | `MapNotFoundError` | `MapPermissionDenied`)
- [ ] **`internal/pkg/maps/repository/maps_queries.go`** — заменить `CheckMapPermissionQuery` на:
  ```sql
  SELECT user_id FROM public.maps WHERE id = $1;
  ```
- [ ] **`internal/pkg/maps/repository/maps_storage.go`** — реализовать `CheckOwnership`:
  - `pgx.ErrNoRows` → `apperrors.MapNotFoundError`
  - `user_id != userID` → `apperrors.MapPermissionDenied`
  - совпадение → `nil`
- [ ] **`internal/pkg/maps/usecases/maps.go`** — в `GetMapByID`, `UpdateMap`, `DeleteMap`:
  - Заменить `CheckPermission` на `CheckOwnership`
  - Пробрасывать ошибку напрямую (`MapNotFoundError` или `MapPermissionDenied`)
- [ ] **`internal/pkg/maps/delivery/maps_handlers.go`** — delivery уже корректно маппит оба кода, изменения не требуются:
  - `MapPermissionDenied` → 403 FORBIDDEN ✅
  - `MapNotFoundError` → 404 NOT_FOUND ✅
- [ ] Обновить моки: `go generate ./internal/pkg/maps/...`
- [ ] Обновить тесты в `usecases/maps_test.go`: заменить `CheckPermission` на `CheckOwnership`, добавить кейс «карта не найдена → MapNotFoundError»

### Этап 2: Исправить формат ListMaps (массив вместо объекта)

- [ ] **`internal/models/maps.go`** — удалить `MapsList struct` (или оставить если нужен в другом месте)
- [ ] **`internal/pkg/maps/interfaces.go`** — изменить `ListMaps` на `ListMaps(...) ([]models.MapMetadata, error)`
- [ ] **`internal/pkg/maps/repository/maps_queries.go`** — удалить `CountMapsQuery`
- [ ] **`internal/pkg/maps/repository/maps_storage.go`** — `ListMaps()`:
  - Убрать запрос COUNT
  - Возвращать `[]models.MapMetadata` вместо `*models.MapsList`
  - **Важно**: инициализировать слайс через `make([]models.MapMetadata, 0)`, чтобы JSON-сериализация давала `[]`, а не `null`
- [ ] **`internal/pkg/maps/usecases/maps.go`** — `ListMaps()`: обновить сигнатуру
- [ ] **`internal/pkg/maps/delivery/maps_handlers.go`** — `ListMaps()`: `sendMapsOkResponse(w, 200, list)` где `list` — `[]MapMetadata`
- [ ] Обновить моки и тесты

### Этап 3: Сделать `name` опциональным в UpdateMapRequest

- [ ] **`internal/models/maps.go`** — изменить:
  ```go
  type UpdateMapRequest struct {
      Name *string  `json:"name,omitempty"`
      Data MapData  `json:"data"`
  }
  ```
- [ ] **`internal/pkg/maps/usecases/validator.go`** — создать отдельную функцию `ValidateUpdateMapRequest(name *string, data *MapData)`:
  - Если `name != nil` — валидировать длину
  - Если `name == nil` — пропустить
  - Остальная валидация (data) — как раньше
- [ ] **`internal/pkg/maps/usecases/maps.go`** — `UpdateMap()`: вызывать `ValidateUpdateMapRequest` вместо `ValidateMapRequest`
- [ ] **`internal/pkg/maps/repository/maps_queries.go`** — изменить `UpdateMapQuery`:
  ```sql
  UPDATE public.maps
  SET name = COALESCE($3, name), data = $4
  WHERE id = $1 AND user_id = $2
  RETURNING id, user_id, name, data, created_at, updated_at;
  ```
- [ ] **`internal/pkg/maps/repository/maps_storage.go`** — `UpdateMap()`: передавать `name` как `*string` (nil для SQL NULL → COALESCE сохранит старое)
- [ ] **`internal/pkg/maps/interfaces.go`** — обновить сигнатуру `UpdateMap` в `MapsRepository` если аргумент `name` меняется с `string` на `*string`
- [ ] Обновить тесты: добавить кейс «обновление без name»

### Этап 4: Исправить валидатор (убрать кратность, добавить `layer`, `INVALID_NAME`)

- [ ] **`internal/pkg/maps/usecases/validator.go`**:
  - Удалить константу `MapUnitsPerTile = 6` и все проверки `% MapUnitsPerTile != 0` из `ValidateMapRequest` и `validatePlacement`
  - Оставить только обязательные проверки: `widthUnits > 0`, `heightUnits > 0`, `x >= 0`, `y >= 0`
  - Добавить проверку `layer >= 0` в `validatePlacement()` (field: `data.placements[N].layer`)
  - Исправить `CategorizeValidationErrors`:
    - Заменить `err.Field[:15] == "data.placements"` на `strings.HasPrefix(err.Field, "data.placements")`
    - Добавить `case "name": return "INVALID_NAME"` (новый error code, см. D.4)
- [ ] **`internal/pkg/apperrors/maps.go`** — добавить `InvalidNameError` (для полноты, хотя основная обработка через `ValidationErrorWrapper`)
- [ ] **`internal/pkg/maps/usecases/validator_test.go`**:
  - Удалить все тесты на кратность 6 (`"width not multiple of 6"`, `"x not multiple of 6"` и т.п.)
  - Обновить валидные значения в тестах на произвольные положительные числа (не обязательно кратные 8)
  - Добавить тест `layer < 0 → ошибка` (field `data.placements[0].layer`)
  - Добавить тест `layer = 0 → ок`
  - Добавить тест `CategorizeValidationErrors` для `field == "name"` → `"INVALID_NAME"`

### Этап 5: Добавить тесты из Minimal Test Set (F.1)

- [ ] **`internal/pkg/maps/usecases/validator_test.go`** — тесты #1–5 из F.1:
  - `TestValidateMapRequest_NoDivisibilityCheck` (удалить старые тесты кратности 6)
  - `TestValidateMapRequest_LayerNegative`
  - `TestCategorize_NameField_ReturnsINVALID_NAME`
  - `TestValidateUpdateMapRequest_NameNil_OK`
  - `TestValidateUpdateMapRequest_NameEmpty_Error`
- [ ] **`internal/pkg/maps/usecases/maps_test.go`** — тесты #6, #7, #11 из F.1:
  - `TestGetMapByID_Ownership_NotFound_404`
  - `TestGetMapByID_Ownership_WrongUser_403`
  - `TestUpdateMap_NameOmitted_PreservesOldName`
- [ ] **`internal/pkg/maps/delivery/maps_handlers_test.go`** — тесты #8, #9, #10 из F.1:
  - `TestListMaps_EmptySlice_NotNull`
  - `TestDeleteMap_204_NoBody_NoContentType`
  - `TestCreateMap_BadJSON_400`

### Этап 6: Перегенерировать моки и финальная проверка

- [ ] `go generate ./internal/pkg/maps/...`
- [ ] `go test ./internal/pkg/maps/...`
- [ ] `go test ./...` — все тесты проекта
- [ ] `go build ./cmd/app/main.go` — проверка компиляции
- [ ] Ручной e2e через curl (см. раздел F.4)

---

## H) Контрактная матрица эндпоинтов

Ниже зафиксированы внешние контракты каждого эндпоинта. Любое расхождение реализации с этой таблицей — баг.

### GET /api/maps (listMyMaps)

| Аспект | Контракт |
|--------|----------|
| **Вход** | Query: `start` (int ≥ 0, default 0), `size` (int 1..100, default 20). Невалидные значения → молча заменяются дефолтами (текущее поведение) |
| **Выход** | 200 + `application/json` + **JSON-массив** `[MapMetadata, ...]`. При 0 карт — пустой массив `[]`, **не** `null` |
| **Ошибки** | 401 `UNAUTHORIZED` (middleware) |
| **Инварианты** | Возвращает только карты текущего `userId`. Сортировка `updated_at DESC`. Данные `data` НЕ включаются |

> **Важно**: Go nil-слайс сериализуется в `null`. В репозитории инициализировать слайс через `make([]models.MapMetadata, 0)` или проверять перед возвратом.

### POST /api/maps (createMap)

| Аспект | Контракт |
|--------|----------|
| **Вход** | Body JSON: `CreateMapRequest { name (string, 1-255, required), data (MapData, required) }`. Max body: 10 MB |
| **Выход** | **201 Created** + `application/json` + `MapFull` (id, userId, name, createdAt, updatedAt, data) |
| **Ошибки** | 400 `BAD_REQUEST` (невалидный JSON), 401 `UNAUTHORIZED`, 422 `INVALID_NAME` / `INVALID_SCHEMA_VERSION` / `INVALID_DIMENSIONS` / `INVALID_PLACEMENT` |
| **Инварианты** | `userId` берётся из сессии, не из тела. `id` генерируется сервером (UUID v4). `createdAt == updatedAt` при создании |

### GET /api/maps/{id} (getMapById)

| Аспект | Контракт |
|--------|----------|
| **Вход** | Path: `id` (UUID формат) |
| **Выход** | 200 + `application/json` + `MapFull` |
| **Ошибки** | 400 `BAD_REQUEST` (невалидный UUID), 401 `UNAUTHORIZED`, 403 `FORBIDDEN` (чужая карта), 404 `NOT_FOUND` (не существует) |
| **Инварианты** | Ownership: сначала проверить существование → 404, затем владение → 403 |

### PUT /api/maps/{id} (updateMap)

| Аспект | Контракт |
|--------|----------|
| **Вход** | Path: `id` (UUID). Body JSON: `UpdateMapRequest { name (*string, 1-255, optional), data (MapData, required) }`. Max body: 10 MB |
| **Выход** | **200 OK** + `application/json` + `MapFull` (обновлённая) |
| **Ошибки** | 400 `BAD_REQUEST` (невалидный JSON/UUID), 401 `UNAUTHORIZED`, 403 `FORBIDDEN`, 404 `NOT_FOUND`, 422 `INVALID_*` |
| **Инварианты** | Если `name` отсутствует в JSON — имя не меняется (COALESCE). Если `name: ""` — 422. `updatedAt` обновляется триггером. Ownership проверяется ДО валидации тела |

### DELETE /api/maps/{id} (deleteMap)

| Аспект | Контракт |
|--------|----------|
| **Вход** | Path: `id` (UUID) |
| **Выход** | **204 No Content** — пустое тело, **без** заголовка `Content-Type` |
| **Ошибки** | 400 `BAD_REQUEST` (невалидный UUID), 401 `UNAUTHORIZED`, 403 `FORBIDDEN`, 404 `NOT_FOUND` |
| **Инварианты** | Удаление идемпотентно по эффекту (повторный DELETE → 404). Ownership проверяется перед удалением |

> **204 No Content**: текущий код корректно вызывает `w.WriteHeader(http.StatusNoContent)` без записи тела. Убедиться, что `sendMapsOkResponse` **не вызывается** для DELETE (он ставит `Content-Type: application/json`).

### Сводная таблица error codes

| Error Code | HTTP | Когда | Определён в |
|------------|------|-------|-------------|
| `BAD_REQUEST` | 400 | Невалидный JSON, невалидный UUID, отсутствие тела | notes |
| `UNAUTHORIZED` | 401 | Нет session cookie / JWT | notes |
| `FORBIDDEN` | 403 | Карта принадлежит другому пользователю | notes |
| `NOT_FOUND` | 404 | Карта с данным id не существует | notes |
| `INVALID_NAME` | 422 | `name` пустое или >255 символов | **новый** (план) |
| `INVALID_SCHEMA_VERSION` | 422 | `data.schemaVersion != 1` | notes |
| `INVALID_DIMENSIONS` | 422 | `widthUnits` или `heightUnits` ≤ 0 | notes |
| `INVALID_PLACEMENT` | 422 | rot ∉ [0..3], x/y < 0, id/tileId пустые, layer < 0 | notes |
| `INTERNAL_ERROR` | 500 | Непредвиденная ошибка | convention |

---

## I) Compatibility & Rollout

### I.a) Breaking changes (обратно-несовместимые изменения)

| # | Изменение | Затронутые клиенты | Серьёзность | Mitigation |
|---|-----------|-------------------|-------------|------------|
| 1 | **ListMaps**: ответ `{"maps":[...],"total":N}` → `[...]` | Фронтенд, парсящий `response.maps` или `response.total` | **P0 → ✅ SAFE** | Фронтенд не вызывает `/api/maps` (новая фича, 0 клиентского кода). Координация не требуется |
| 2 | **Ownership 403→404**: несуществующие карты раньше давали 403, теперь 404 | Клиенты, обрабатывающие 403 как «не найдено» | **P0 → ✅ SAFE** | Фронтенд не вызывает `/api/maps/{id}` (новая фича). Координация не требуется |
| 3 | **Новый error code `INVALID_NAME`**: раньше name-ошибка возвращала `BAD_REQUEST`, теперь `INVALID_NAME` | Клиенты, парсящие `error` field в 422-ответах | **P1** | Добавить `INVALID_NAME` в документацию. Клиенты, не парсящие error code, не затронуты |
| 4 | **UpdateMapRequest.name опциональное**: раньше name required, теперь optional | Нет (relaxation) | **P1 safe** | Клиенты, отправляющие name, продолжат работать. Новое поведение: можно не передавать name |
| 5 | **Удаление проверки кратности**: раньше width/height/x/y MUST быть кратны 6, теперь — только > 0 / ≥ 0 | Нет (relaxation) | **P1 safe** | Ранее отклоняемые запросы теперь принимаются. Корректно по spec |
| 6 | **Новая валидация `layer >= 0`**: раньше отрицательный layer принимался, теперь 422 | Клиенты, отправляющие `layer < 0` | **P1** | Маловероятно: layer < 0 не имеет смысла. Проверить, есть ли такие данные в БД (см. I.f) |

### I.b) Миграции и данные

- **Новых SQL-миграций не требуется** для основных изменений (B.1–B.4). Схема БД не меняется.
- **Опциональная миграция** `011_maps_schema_version_index` (раздел C.2) — только индекс, без DDL-изменений таблицы. Безопасна, но не обязательна для rollout.
- **Существующие данные в `data` JSONB**: могут содержать значения, кратные 6 (старая валидация). После снятия проверки кратности они остаются валидными — relaxation не ломает данные.
- **Нет data migration**: ни одно изменение не требует модификации существующих строк в таблице `maps`.

### I.c) Reversibility (откат)

| Изменение | Откат | Сложность |
|-----------|-------|-----------|
| ListMaps формат | Вернуть `MapsList` struct и COUNT-запрос | Низкая (git revert) |
| CheckOwnership | Вернуть `CheckPermission` | Низкая (git revert) |
| name optional | Вернуть `Name string` | Низкая (git revert) |
| Удаление кратности | Вернуть `MapUnitsPerTile` | Низкая, но данные, сохранённые с некратными значениями, станут «невалидными» ⚠️ |
| layer >= 0 | Убрать проверку | Низкая |
| INVALID_NAME код | Вернуть BAD_REQUEST | Низкая |

**Общий вывод**: Все изменения — application-level (Go код). Нет DDL-миграций, нет data transforms. Git revert покрывает любой откат, кроме edge case с данными, сохранёнными после снятия проверки кратности.

### I.d) Partial rollout (постепенный выкат)

Рекомендуемый порядок деплоя по этапам (раздел G):

1. **Этап 4** (валидатор) — можно деплоить первым: relaxation кратности и добавление layer ≥ 0 — минимальный risk, не затрагивает API-контракт верхнего уровня.
2. **Этап 1** (ownership) — деплоить вторым: улучшает поведение, но меняет коды ответов. Требует awareness фронтенда.
3. **Этап 3** (name optional) — деплоить третьим: relaxation, безопасен.
4. **Этап 2** (ListMaps формат) — деплоить **последним**: самый breaking change. Требует координации с фронтендом.
5. **Этапы 5–6** (тесты, моки) — параллельно с любым этапом.

> **Можно ли деплоить всё за один релиз?** Да, если фронтенд обновляется одновременно. Если нет — использовать partial rollout по порядку выше.

### I.e) Error policy (политика ошибок)

- **Принцип**: Новые ошибки (расширение пространства кодов) — допустимы, если клиенты обрабатывают неизвестные коды gracefully (fallback на HTTP status).
- **`INVALID_NAME`**: новый код, но в том же HTTP 422 пространстве. Клиенты, смотрящие только на HTTP status, не затронуты.
- **Ужесточение `layer >= 0`**: запросы с `layer < 0`, ранее принимавшиеся, теперь получат 422. Это единственное ужесточение. Риск минимален (layer < 0 бессмысленен).
- **Ослабление кратности**: запросы с некратными значениями, ранее отклонявшиеся, теперь принимаются. Безопасное направление.

### I.f) Existing data impact (влияние на существующие данные)

Проверить перед деплоем:

```sql
-- 1. Есть ли карты с layer < 0 в placements? (станут «невалидными» при re-save)
SELECT id, name FROM maps
WHERE EXISTS (
    SELECT 1 FROM jsonb_array_elements(data->'placements') p
    WHERE (p->>'layer')::int < 0
);

-- 2. Есть ли карты с некратными width/height? (были отклонены старой валидацией — маловероятно, но проверить)
SELECT id, name, data->>'widthUnits' AS w, data->>'heightUnits' AS h FROM maps
WHERE (data->>'widthUnits')::int % 6 != 0
   OR (data->>'heightUnits')::int % 6 != 0;

-- 3. Сколько всего карт? (понять масштаб)
SELECT COUNT(*) FROM maps;
```

**Ожидание**: Запрос 1 вернёт 0 строк (layer < 0 бессмысленен). Запрос 2 вернёт 0 строк (старая валидация не пропускала). Запрос 3 — для контекста.

### I.g) Idempotency (идемпотентность)

| Операция | Идемпотентность | Комментарий |
|----------|-----------------|-------------|
| GET /maps | ✅ Идемпотентный | Read-only |
| GET /maps/{id} | ✅ Идемпотентный | Read-only |
| POST /maps | ❌ Не идемпотентный | Каждый вызов создаёт новую карту. По spec это корректно. Клиент должен дедуплицировать на своей стороне |
| PUT /maps/{id} | ✅ Идемпотентный | Повторный PUT с тем же телом → тот же результат (кроме `updatedAt`, обновляемого триггером) |
| DELETE /maps/{id} | ⚠️ По эффекту идемпотентный, по HTTP-коду — нет | Первый DELETE → 204, повторный → 404. Это стандартное REST-поведение и соответствует spec |

---

## J) GO/NO-GO критерии по совместимости

Перед деплоем каждый пункт должен быть **GO**:

| # | Критерий | GO условие | Способ проверки |
|---|----------|------------|-----------------|
| 1 | **Фронтенд готов к ListMaps как массиву** | ✅ **GO** — фронтенд не вызывает `/api/maps` (новая фича) | Статический анализ `2025_IMAO_DnD_Assistant_frontend/src/` (2026-02-05) |
| 2 | **Фронтенд обрабатывает 404 для несуществующих карт** | ✅ **GO** — фронтенд не имеет клиентского кода для maps | Статический анализ фронтенда (2026-02-05) |
| 3 | **Нет карт с `layer < 0` в БД** | SQL-запрос из I.f возвращает 0 строк | Выполнить запрос на prod-БД |
| 4 | **Все тесты проходят** | `go test ./...` — 0 failures | CI pipeline |
| 5 | **Моки актуальны** | `go generate ./internal/pkg/maps/...` не создаёт diff | CI pipeline |
| 6 | **Миграции применимы** | `go run cmd/app/main.go -migrate latest` — без ошибок | Проверка на staging |
| 7 | **Ручной e2e пройден** | Все 10 шагов из F.4 возвращают ожидаемые коды | Manual QA |

**Вердикт (2026-02-05): GO** — пункты J.1, J.2 закрыты (фронтенд не вызывает Maps API — новая фича). Остальные пункты — execution-time, проверяются в ходе реализации.

**NO-GO** (блокеры деплоя, если возникнут):
- SQL-запрос I.f.1 возвращает > 0 строк → **NO-GO** для Этапа 4 (layer validation) до ручной проверки данных
- Любой тест падает → **NO-GO**

---

## K) Риски и вопросы

### Принятые решения (из этого плана)

| # | Решение | Обоснование |
|---|---------|-------------|
| 1 | **Ownership: Вариант Б (403/404 раздельно)** | Spec и notes однозначно требуют оба кода. Enumeration через UUID практически невозможен |
| 2 | **Кратность: убрать из обязательной валидации** | Notes помечают divisible-by-8 как optional. Не ломает существующие данные |
| 3 | **Новый error code `INVALID_NAME`** | BAD_REQUEST привязан к HTTP 400 по notes. Невалидное имя — семантическая ошибка → 422 |
| 4 | **ListMaps → плоский массив** | Spec требует `type: array`. Breaking change, но координируется через GO/NO-GO (раздел J) |

### Оставшиеся вопросы (требуют решения перед реализацией)

| # | Вопрос / Риск | Рекомендация |
|---|---------------|--------------|
| 1 | ~~**ListMaps: нужен ли `total` на фронтенде?**~~ | ✅ **RESOLVED** (2026-02-05): Фронтенд не вызывает `/api/maps`. Maps API — новая фича без клиентского кода. Следуем спеке: плоский массив |
| 2 | **Каскадное удаление**: `REFERENCES public.user(id)` без `ON DELETE CASCADE` — orphaned maps при удалении пользователя | Добавить миграцию, если удаление пользователей планируется. Иначе — не трогать |
| 3 | **ListMaps: nil slice → `null`** — Go сериализует `nil []MapMetadata` как `null`, а не `[]`. Нужно гарантировать `[]` | Инициализировать `make([]models.MapMetadata, 0)` в репозитории |
| 4 | **Данные с layer < 0** — если такие есть в БД, новая валидация не даст re-save этих карт без исправления layer | Выполнить SQL из I.f перед деплоем. **GO/NO-GO #3** |

---

## L) Acceptance Criteria & Go/No-Go Freeze

> **Цель раздела**: зафиксировать проверяемые критерии готовности. После выполнения всех этапов (G) реализация проверяется по этому списку. Итерации по архитектуре/дизайну плана завершены — этот раздел фиксирует финальные ожидания.

### L.1 Acceptance Criteria

Каждый пункт — однозначно проверяемый. Реализация считается завершённой, только если все 8 выполнены.

| # | Критерий | Вход / действие | Ожидаемый результат | Ссылка |
|---|----------|-----------------|---------------------|--------|
| AC-1 | **ListMaps возвращает JSON-массив** | `GET /api/maps?start=0&size=10` (авторизован, есть карты) | HTTP 200, тело — `[{...}, ...]` (плоский массив). При 0 карт — `[]`, не `null`, не `{"maps":[...]}` | B.2, H.GET /maps |
| AC-2 | **Ownership: not found → 404** | `GET /api/maps/<несуществующий-uuid>` | HTTP 404, `"error":"NOT_FOUND"` | B.4, H.GET |
| AC-3 | **Ownership: чужая → 403** | `GET /api/maps/<uuid-чужой-карты>` | HTTP 403, `"error":"FORBIDDEN"` | B.4, H.GET |
| AC-4 | **UpdateMap: name опционален** | `PUT /api/maps/<uuid>` с телом `{"data":{...}}` (без `name`) | HTTP 200, `name` в ответе = прежнее значение (не пустое, не изменилось) | B.3, H.PUT |
| AC-5 | **UpdateMap: name="" → 422** | `PUT /api/maps/<uuid>` с телом `{"name":"","data":{...}}` | HTTP 422, `"error":"INVALID_NAME"` | B.3, D.4, H.PUT |
| AC-6 | **Кратность не обязательна** | `POST /api/maps` с `widthUnits:13, heightUnits:7` (>0, не кратные ни 6, ни 8) | HTTP 201, карта создана | B.1, D.2 |
| AC-7 | **DELETE: 204 без тела** | `DELETE /api/maps/<uuid>` (своя карта) | HTTP 204, тело пустое, заголовок `Content-Type` отсутствует. Повторный DELETE → 404 | H.DELETE |
| AC-8 | **Ошибки: 400 для синтаксиса, 422 для семантики** | (a) `POST` с `{invalid-json` → 400 `BAD_REQUEST`; (b) `POST` с `schemaVersion:2` → 422 `INVALID_SCHEMA_VERSION`; (c) `POST` с `name:""` → 422 `INVALID_NAME` | Коды и error-body соответствуют H.сводная | D.4, H.сводная |

### L.2 Definition of Done

- [ ] Все 6 этапов из раздела G выполнены (чекбоксы в G отмечены)
- [ ] Все 12 тестов из F.1 написаны и проходят
- [ ] `make verify && go build -mod=vendor ./cmd/app/main.go` — exit 0 (F.3)
- [ ] Все 8 acceptance criteria (L.1) подтверждены — unit-тестами (F.1) или ручным smoke (F.4)
- [ ] GO/NO-GO критерии из J (пункты 1–7) — все GO

### L.3 Must-fix before GO

~~Блокеры, которые должны быть закрыты до начала реализации.~~ **Все закрыты (2026-02-05).**

| # | Блокер | Статус | Доказательство |
|---|--------|--------|----------------|
| MF-1 | **Фронтенд: ListMaps формат** | ✅ **CLOSED** | Статический анализ `2025_IMAO_DnD_Assistant_frontend/src/`: `grep -r "/api/maps" src/` → 0 совпадений. Нет файлов `*maps*.api.ts`. Нет роутов map editor в `AppRouter.tsx`. Maps API — полностью новая фича, фронтенд будет писаться под новый контракт (плоский массив). |
| MF-2 | **Фронтенд: ownership 404** | ✅ **CLOSED** | Тот же анализ: фронтенд не вызывает `/api/maps/{id}`, нет обработки 403/404 для maps. Код, завязанный на 403 для «не найдено» — отсутствует. Изменение безопасно. |

> **Итог**: Оба блокера закрыты. Maps API — новая фича без существующего клиентского кода. План переведён в GO.

### L.4 Follow-ups (после GO, не блокируют реализацию)

| # | Задача | Приоритет | Ссылка |
|---|--------|-----------|--------|
| FU-1 | Выполнить SQL-запросы из I.f на prod перед деплоем Этапа 4 (layer validation) | Перед деплоем | I.f, J.3 |
| FU-2 | Опциональная миграция `011_maps_schema_version_index` — решить, нужен ли индекс `(data->>'schemaVersion')` | Низкий | C.2 |
| FU-3 | Каскадное удаление: добавить `ON DELETE CASCADE` на FK `maps.user_id`, если удаление пользователей станет фичей | По необходимости | K.2 |
| FU-4 | Интеграционные тесты для repository (maps CRUD cycle) при появлении тестовой БД-инфраструктуры | Низкий | F.1 (примечание) |

### L.5 How to Verify

```bash
# 1. Quality gates (CI)
make verify && go build -mod=vendor ./cmd/app/main.go

# 2. Unit-тесты maps-модуля (все 12 из F.1 должны пройти)
go test -mod=vendor -v ./internal/pkg/maps/...

# 3. Ручной smoke (сервер + БД запущены, есть session cookie)
#    Пройти 10 шагов из F.4 — ключевые проверки:
#    POST widthUnits=13 → 201            (AC-6)
#    GET list           → 200 + [...]    (AC-1)
#    GET nonexistent    → 404            (AC-2)
#    PUT без name       → 200, имя то же (AC-4)
#    PUT name=""        → 422            (AC-5)
#    DELETE             → 204, пустое    (AC-7)
#    DELETE повторный   → 404            (AC-7)

# 4. Pre-deploy: SQL-проверка на prod (FU-1)
#    Выполнить 3 запроса из I.f — все должны вернуть 0 строк
```

---

## M) PR Plan (Implementation Sequence)

> **Принцип**: Каждый PR — атомарный, деплой-безопасный, ревьюируемый за одну сессию. Порядок следует зависимостям из G и рискам из I.d.

### PR-1: `maps: remove divisibility validation, add layer check`

**Этап G**: 4 (валидатор)
**Слои**: usecases
**Deploy**: ✅ Safe to merge, ✅ Safe to deploy (relaxation + minor tightening)

- [ ] Удалить константу `MapUnitsPerTile` и все проверки `% MapUnitsPerTile` из `validator.go`
- [ ] Добавить проверку `layer >= 0` в `validatePlacement()`
- [ ] Исправить `CategorizeValidationErrors`: `strings.HasPrefix` вместо `[:15]`
- [ ] Добавить `case "name": return "INVALID_NAME"` в `CategorizeValidationErrors`
- [ ] Обновить `validator_test.go`: удалить тесты кратности, добавить `layer<0`, `INVALID_NAME`
- [ ] `make verify` — exit 0

### PR-2: `maps: make UpdateMap name optional`

**Этап G**: 3 (name optional)
**Слои**: models, usecases, repository
**Deploy**: ✅ Safe to merge, ✅ Safe to deploy (relaxation)

- [ ] `models/maps.go`: `UpdateMapRequest.Name` → `*string` с `json:"name,omitempty"`
- [ ] `validator.go`: создать `ValidateUpdateMapRequest(name *string, data *MapData)`
- [ ] `maps.go` (usecases): вызывать `ValidateUpdateMapRequest` в `UpdateMap`
- [ ] `maps_queries.go`: UPDATE с `COALESCE($3, name)`
- [ ] `maps_storage.go`: передавать `*string` в `$3`
- [ ] `interfaces.go`: обновить сигнатуру `UpdateMap` в `MapsRepository` если изменилась
- [ ] Тесты: `name=nil` → OK, `name=""` → 422, `name="Valid"` → OK
- [ ] `make verify` — exit 0

### PR-3: `maps: distinguish 404/403 with CheckOwnership`

**Этап G**: 1 (ownership)
**Слои**: interfaces, repository, usecases
**Deploy**: ✅ Safe to merge, ✅ Safe to deploy (фронтенд не вызывает maps API)

- [ ] `interfaces.go`: `CheckPermission` → `CheckOwnership(ctx, id, userID) error`
- [ ] `maps_queries.go`: `SELECT user_id FROM public.maps WHERE id = $1`
- [ ] `maps_storage.go`: реализовать `CheckOwnership` (NoRows→404, wrong user→403, ok→nil)
- [ ] `maps.go` (usecases): заменить `CheckPermission` → `CheckOwnership` в Get/Update/Delete
- [ ] Перегенерировать моки: `go generate ./internal/pkg/maps/...`
- [ ] Тесты: not found → `MapNotFoundError`, wrong user → `MapPermissionDenied`
- [ ] `make verify` — exit 0

### PR-4: `maps: return flat array from ListMaps`

**Этап G**: 2 (ListMaps формат)
**Слои**: models, interfaces, repository, usecases, delivery
**Deploy**: ✅ Safe to merge, ✅ Safe to deploy (фронтенд не вызывает maps API)

- [ ] `models/maps.go`: удалить `MapsList` struct (или пометить deprecated)
- [ ] `interfaces.go`: `ListMaps` → `([]models.MapMetadata, error)`
- [ ] `maps_queries.go`: удалить `CountMapsQuery`
- [ ] `maps_storage.go`: возвращать `[]models.MapMetadata`, инициализация через `make([]T, 0)`
- [ ] `maps.go` (usecases): обновить сигнатуру
- [ ] `maps_handlers.go` (delivery): `sendMapsOkResponse(w, 200, list)`
- [ ] Перегенерировать моки
- [ ] Тест: пустой список → `[]` (не `null`)
- [ ] `make verify` — exit 0

### PR-5: `maps: add contract tests (F.1 test set)`

**Этап G**: 5 (тесты)
**Слои**: usecases (tests), delivery (tests)
**Deploy**: ✅ Safe to merge, ✅ Safe to deploy (тесты only)
**Зависимости**: после PR-1..PR-4

- [ ] `validator_test.go`: тесты #1–5 из F.1
- [ ] `maps_test.go`: тесты #6, #7, #11 из F.1
- [ ] `maps_handlers_test.go`: тесты #8, #9, #10 из F.1
- [ ] Все 12 тестов из F.1 проходят
- [ ] `make verify` — exit 0

### PR-6: `maps: final verification and mock regen`

**Этап G**: 6 (финальная проверка)
**Слои**: mocks, CI
**Deploy**: ✅ Safe to merge, ✅ Safe to deploy
**Зависимости**: после PR-5

- [ ] `go generate ./internal/pkg/maps/...` — моки актуальны, `git diff --exit-code`
- [ ] `go test -mod=vendor ./...` — все тесты проекта проходят
- [ ] `go build -mod=vendor ./cmd/app/main.go` — компиляция ок
- [ ] `make test-race` — нет гонок
- [ ] Acceptance criteria L.1 (AC-1..AC-8) подтверждены unit-тестами

### PR-7 (post-merge): `maps: manual smoke verification`

**Не PR** — ручная проверка после деплоя всех PR.
**Deploy**: ⏳ Hold deploy до прохождения

- [ ] Развернуть сервер + БД (`docker compose up -d && go run cmd/app/main.go`)
- [ ] Выполнить SQL-проверки из I.f (FU-1)
- [ ] Пройти 10 шагов F.4 (ручной smoke)
- [ ] Все 8 acceptance criteria (L.1) подтверждены
- [ ] GO/NO-GO критерии J.3–J.7 — все GO
- [ ] **DEPLOY**

### Граф зависимостей PR

```
PR-1 (validator) ──┐
PR-2 (name opt)  ──┼──→ PR-5 (tests) ──→ PR-6 (final) ──→ PR-7 (smoke)
PR-3 (ownership) ──┤
PR-4 (ListMaps)  ──┘
```

PR-1..PR-4 могут идти **параллельно** (независимые слои, минимальные конфликты). PR-5 ждёт все четыре. PR-6 — финальная верификация. PR-7 — ручной smoke после merge.
