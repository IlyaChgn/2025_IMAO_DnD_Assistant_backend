Стратегия тестирования — D&D Assistant Backend                                                                                                                                
                                                                                                                                                                                  1. Карта системы (Investigation Results)

  Домены и usecase-методы
  Домен: encounter
  Usecase-методы: GetEncountersList, GetEncounterByID, SaveEncounter, UpdateEncounter, RemoveEncounter
  Зависимости (interfaces): EncounterRepository
  Сложность логики: Средняя: валидация start/size, проверка permissions, генерация UUID
  ────────────────────────────────────────
  Домен: maps
  Usecase-методы: CreateMap, GetMapByID, UpdateMap, DeleteMap, ListMaps
  Зависимости (interfaces): MapsRepository
  Сложность логики: Высокая: валидация (отдельный validator.go), permissions, JSON-маршалинг
  ────────────────────────────────────────
  Домен: bestiary
  Usecase-методы: GetCreaturesList, GetCreatureByEngName, GetUserCreaturesList, GetUserCreatureByEngName, AddGeneratedCreature, ParseCreatureFromImage,
    GenerateCreatureFromDescription
  Зависимости (interfaces): BestiaryRepository, BestiaryS3Manager, GeminiAPI
  Сложность логики: Высокая: владение, S3-загрузка, LLM-вызовы
  ────────────────────────────────────────
  Домен: bestiary/llm
  Usecase-методы: SubmitText, SubmitImage, GetJob + process (background)
  Зависимости (interfaces): LLMJobRepository, GeminiAPI, GeneratedCreatureProcessorUsecases
  Сложность логики: Высокая: async goroutine, multi-step pipeline
  ────────────────────────────────────────
  Домен: character
  Usecase-методы: GetCharactersList, AddCharacter, GetCharacterByMongoId
  Зависимости (interfaces): CharacterRepository
  Сложность логики: Средняя: валидация, permissions, file parsing
  ────────────────────────────────────────
  Домен: auth
  Usecase-методы: Login, Logout, CheckAuth, GetUserIDBySessionID
  Зависимости (interfaces): AuthRepository, VKApi, SessionManager
  Сложность логики: Высокая: multi-step OAuth, user upsert
  ────────────────────────────────────────
  Домен: maptiles
  Usecase-методы: GetCategories
  Зависимости (interfaces): MapTilesRepository
  Сложность логики: Низкая
  ────────────────────────────────────────
  Домен: table
  Usecase-методы: CreateSession, GetTableData, AddNewConnection
  Зависимости (interfaces): TableManager, EncounterRepository
  Сложность логики: Высокая: timers, goroutines, websocket
  ────────────────────────────────────────
  Домен: description
  Usecase-методы: GenerateDescription
  Зависимости (interfaces): gRPC client
  Сложность логики: Низкая (прокси)
  ────────────────────────────────────────
  Домен: statblockgenerator
  Usecase-методы: (пустые интерфейсы)
  Зависимости (interfaces): —
  Сложность логики: Нет логики
  Схема ошибок → HTTP-коды

  Usecase sentinel error          →  Delivery mapping          →  HTTP code
  ─────────────────────────────────────────────────────────────────────────
  StartPosSizeError               →  ErrSizeOrPosition         →  400
  InvalidInputError               →  ErrWrongEncounterName     →  400
  InvalidUserIDError              →  ErrInvalidID              →  400
  PermissionDeniedError           →  ErrForbidden              →  403
  MapPermissionDenied             →  FORBIDDEN                 →  403
  MapNotFoundError                →  NOT_FOUND                 →  404
  NoDocsErr                       →  (200 + nil body)          →  200
  ValidationErrorWrapper          →  INVALID_*                 →  422
  (all other / default)           →  ErrInternalServer         →  500

  Проблемы тестируемости (найденные)
  Проблема: logger.FromContext(ctx) — type assertion panic
  Где: Все usecases
  Влияние на тесты: Каждый тест должен создавать ctx с real logger
  Статус: ✅ Решено в PR1 — FromContext возвращает noop logger при отсутствии логгера в контексте
  ────────────────────────────────────────
  Проблема: var ctxKey string — глобальная мутабельная переменная в logger
  Где: logger/init.go:10
  Влияние на тесты: Параллельные тесты с разными ключами могут конфликтовать
  Статус: ✅ Решено в PR1 — заменено на type loggerCtxKey struct{}, LoggerConfig.Key deprecated
  ────────────────────────────────────────
  Проблема: uuid.NewString() напрямую
  Где: encounter/usecases:59, llm.go:26,45, auth/delivery:42
  Влияние на тесты: ID недетерминистичны, нельзя assert exact value
  ────────────────────────────────────────
  Проблема: time.Now() напрямую
  Где: auth/delivery/session.go:13, bestiary/repository/llm_storage.go, table/repository:45
  Влияние на тесты: Флэки при проверке timestamps
  ────────────────────────────────────────
  Проблема: go uc.process(ctx, id)
  Где: bestiary/usecases/llm.go:38,57
  Влияние на тесты: Race conditions в тестах
  ────────────────────────────────────────
  Проблема: *websocket.Conn в интерфейсе
  Где: table/interfaces.go
  Влияние на тесты: Невозможно подменить без реального websocket
  ────────────────────────────────────────
  Проблема: multipart.File в usecase
  Где: character/interfaces.go
  Влияние на тесты: Нужен реальный multipart reader
  ────────────────────────────────────────
  Проблема: Vendor mode, testify/require не vendored
  Где: vendor/
  Влияние на тесты: Только testify/assert доступен
  Существующие тесты

  - utils/merger/merger_test.go — external test package, table-driven, testify/assert
  - maps/usecases/validator_test.go — internal package, table-driven, stdlib testing
  - maptiles/usecases/maptiles_test.go — internal package, testify/assert, hand-written mock repo

  CI / Tooling

  - Нет CI (нет .github/workflows/)
  - Makefile в корне проекта: make test, make test-race, make test-cover (добавлен в PR2)
  - Нет golangci-lint конфига
  - Vendor mode (-mod=vendor) активен

  ---
  2. Тестовый стек

  Выбранный стек (с обоснованием)
  Инструмент: testing (stdlib)
  Решение: Обязателен
  Обоснование: Базис
  ────────────────────────────────────────
  Инструмент: testify/assert
  Решение: Использовать
  Обоснование: Уже vendored, уже используется в repo; более читаемые assertions чем if err != nil { t.Fatal }
  ────────────────────────────────────────
  Инструмент: testify/require
  Решение: Добавить в vendor
  Обоснование: assert не останавливает тест при критичном fail → cascade failures. require решает это. Одна команда: go mod vendor после go get
  ────────────────────────────────────────
  Инструмент: Мок-генерация
  Решение: Ручные fakes/stubs
  Обоснование: Интерфейсы в проекте небольшие (1–6 методов). Ручные стабы — минимум зависимостей, максимум понятности, нет кодогенерации. Генераторы (mockgen) имеют смысл при  
    10+ интерфейсах с 10+ методами — не наш случай
  ────────────────────────────────────────
  Инструмент: net/http/httptest
  Решение: Использовать (stdlib)
  Обоснование: Для handler-тестов — стандартный подход, gorilla/mux совместим
  ────────────────────────────────────────
  Инструмент: go test -race
  Решение: Обязателен
  Обоснование: Есть goroutines в LLM и table usecases
  ────────────────────────────────────────
  Инструмент: -coverprofile
  Решение: Рекомендован
  Обоснование: Метрика для отслеживания прогресса, но не KPI
  ────────────────────────────────────────
  Инструмент: testcontainers-go
  Решение: Отложить (этап 4)
  Обоснование: Нет CI, значит сначала unit-тесты; интеграционные — позже
  Что НЕ добавляем

  - gomock/mockgen — overhead для малых интерфейсов
  - ginkgo/gomega — BDD-стиль чужд проекту
  - golangci-lint — полезен, но вне скоупа тестовой стратегии

  ---
  3. Тестовая пирамида

                      ┌──────────────┐
                      │ Integration  │  ← 2–5 сценариев (этап 4)
                      │  (DB/HTTP)   │     build tag: //go:build integration
                      ├──────────────┤
                   ┌──┤  Delivery /  │  ← httptest.NewRecorder (этап 3)
                   │  │  Contract    │     assert status codes + body
                   │  ├──────────────┤
                ┌──┤  │  Usecase     │  ← основной объём (этап 2)
                │  │  │  Unit        │     fake repos + assert errors
                │  │  ├──────────────┤
             ┌──┤  │  │  Domain /    │  ← validators, pure funcs (этап 1–2)
             │  │  │  │  Pure Logic  │     no mocks needed
             └──┴──┴──┴──────────────┘

  Package strategy
  ┌────────────────────────┬─────────────────────────────────────────────────┬──────────────────────────────────────────────────────────────────────────────────────────┐       
  │       Тип теста        │                     Package                     │                                          Почему                                          │       
  ├────────────────────────┼─────────────────────────────────────────────────┼──────────────────────────────────────────────────────────────────────────────────────────┤       
  │ Usecase unit           │ package usecases (internal)                     │ Нужен доступ к unexported struct для конструирования; стаб реализует публичный интерфейс │       
  ├────────────────────────┼─────────────────────────────────────────────────┼──────────────────────────────────────────────────────────────────────────────────────────┤       
  │ Delivery/handler       │ package delivery_test (external)                │ Тестируем только публичный API хендлера; имитируем реального клиента                     │       
  ├────────────────────────┼─────────────────────────────────────────────────┼──────────────────────────────────────────────────────────────────────────────────────────┤       
  │ Pure logic (validator) │ package usecases (internal)                     │ Так уже сделано, функции exported — работает                                             │       
  ├────────────────────────┼─────────────────────────────────────────────────┼──────────────────────────────────────────────────────────────────────────────────────────┤       
  │ Integration            │ package integration_test (отдельная директория) │ Изолируем от unit-тестов через build tags                                                │       
  └────────────────────────┴─────────────────────────────────────────────────┴──────────────────────────────────────────────────────────────────────────────────────────┘       
  Naming conventions

  *_test.go               — всегда рядом с тестируемым файлом
  TestMethodName_Scenario  — e.g. TestGetEncountersList_NegativeStart
  Table-driven tests       — предпочтительны для валидации
  testdata/               — для фикстур JSON (если понадобится)

  Как избегать хрупких тестов

  1. Не проверять логи — logger вызывается, но его вывод не assert'ится
  2. Не проверять exact UUID — assert != "" или проверять формат
  3. Не полагаться на time.Now() — в тестах это пока некритично (timestamps в repo layer); если понадобится — inject Clock interface
  4. Context с logger — noop logger возвращается автоматически из FromContext, специальный setup не нужен
  5. -race flag — обнаруживает data races до того, как они станут flaky (make test-race, требует CGO_ENABLED=1)
  6. Handler error assertions — использовать testhelpers.DecodeErrorResponse(t, rr.Body), не дублировать decode + struct assert

  ---
  4. Принципы тестируемости — правила для команды

  Текущее состояние: что хорошо

  - Clean architecture соблюдена: интерфейсы вынесены в корень домена (interfaces.go)
  - Usecase зависят только от интерфейсов, а не от конкретных repo
  - Sentinel errors через errors.Is() — хорошо маппятся

  Правила

  1. DI на границах: конструкторы usecases принимают интерфейсы — уже соблюдается, продолжать
  2. Интерфейсы по потребителям: каждый usecase определяет свой набор зависимостей — уже сделано через interfaces.go
  3. Deterministic tests: пока uuid.NewString() и time.Now() используются в repo/delivery — это допустимо. При необходимости: type IDGenerator interface { New() string } и type
   Clock interface { Now() time.Time }
  4. Не смешивать transport и бизнес-логику: delivery слой уже чист — парсит request, вызывает usecase, маппит ошибку. Сохранять это
  5. Logger через context: принять как данность, создавать test context helper

  Минимальный рефакторинг (рекомендации, не блокеры)

  - encounter.SaveEncounter: uuid.NewString() вызывается напрямую — при тестировании нельзя проверить какой ID был передан в repo. Рекомендация: передавать ID generator через  
  конструктор. Не блокирует написание тестов (просто не assertим exact ID)
  - bestiary/llm.go: go uc.process(...) — при тестировании нужно ждать goroutine. Рекомендация: для тестов process можно вызывать синхронно или добавить sync.WaitGroup.        
  Отложить до этапа 2

  ---
  5. План внедрения

  Этап 1 — Foundation

  Задачи:
  - Добавить testify/require в vendor: go get github.com/stretchr/testify && go mod vendor
  - Создать shared test helper: internal/pkg/testhelpers/context.go с NewTestContext()
  - Создать шаблон fake repo (пример — encounter)
  - Добавить в root: Makefile с test, test-race, test-cover targets
  - Настроить go test -race ./... как default

  Этап 2 — Usecase Unit Tests (пилот)

  Пилотные домены: encounter и maps

  Почему именно они:
  - encounter — самый типичный CRUD, покрывает все паттерны (validation, permissions, repo delegation)
  - maps — добавляет валидатор (validator.go уже протестирован) + ValidationErrorWrapper
  - Вместе задают шаблон для всех остальных доменов

  Encounter usecase — матрица сценариев:
  ┌───────────────────┬─────────────────────────┬───────────────────────────────────────────┐
  │       Метод       │        Сценарий         │                  Assert                   │
  ├───────────────────┼─────────────────────────┼───────────────────────────────────────────┤
  │ GetEncountersList │ start < 0               │ errors.Is(err, StartPosSizeError)         │
  ├───────────────────┼─────────────────────────┼───────────────────────────────────────────┤
  │ GetEncountersList │ size <= 0               │ errors.Is(err, StartPosSizeError)         │
  ├───────────────────┼─────────────────────────┼───────────────────────────────────────────┤
  │ GetEncountersList │ happy path, no search   │ repo.GetEncountersList called             │
  ├───────────────────┼─────────────────────────┼───────────────────────────────────────────┤
  │ GetEncountersList │ happy path, with search │ repo.GetEncountersListWithSearch called   │
  ├───────────────────┼─────────────────────────┼───────────────────────────────────────────┤
  │ GetEncountersList │ repo error              │ error propagated                          │
  ├───────────────────┼─────────────────────────┼───────────────────────────────────────────┤
  │ GetEncounterByID  │ no permission           │ errors.Is(err, PermissionDeniedError)     │
  ├───────────────────┼─────────────────────────┼───────────────────────────────────────────┤
  │ GetEncounterByID  │ happy path              │ correct encounter returned                │
  ├───────────────────┼─────────────────────────┼───────────────────────────────────────────┤
  │ SaveEncounter     │ empty name              │ errors.Is(err, InvalidInputError)         │
  ├───────────────────┼─────────────────────────┼───────────────────────────────────────────┤
  │ SaveEncounter     │ name > 60 chars         │ errors.Is(err, InvalidInputError)         │
  ├───────────────────┼─────────────────────────┼───────────────────────────────────────────┤
  │ SaveEncounter     │ happy path              │ repo.SaveEncounter called with valid UUID │
  ├───────────────────┼─────────────────────────┼───────────────────────────────────────────┤
  │ UpdateEncounter   │ no permission           │ errors.Is(err, PermissionDeniedError)     │
  ├───────────────────┼─────────────────────────┼───────────────────────────────────────────┤
  │ UpdateEncounter   │ happy path              │ repo.UpdateEncounter called               │
  ├───────────────────┼─────────────────────────┼───────────────────────────────────────────┤
  │ RemoveEncounter   │ no permission           │ errors.Is(err, PermissionDeniedError)     │
  ├───────────────────┼─────────────────────────┼───────────────────────────────────────────┤
  │ RemoveEncounter   │ happy path              │ repo.RemoveEncounter called               │
  └───────────────────┴─────────────────────────┴───────────────────────────────────────────┘
  Maps usecase — матрица сценариев:
  ┌────────────┬──────────────────┬─────────────────────────────────┐
  │   Метод    │     Сценарий     │             Assert              │
  ├────────────┼──────────────────┼─────────────────────────────────┤
  │ ListMaps   │ start < 0        │ StartPosSizeError               │
  ├────────────┼──────────────────┼─────────────────────────────────┤
  │ ListMaps   │ size <= 0        │ StartPosSizeError               │
  ├────────────┼──────────────────┼─────────────────────────────────┤
  │ ListMaps   │ happy path       │ list returned                   │
  ├────────────┼──────────────────┼─────────────────────────────────┤
  │ CreateMap  │ validation fails │ ValidationErrorWrapper returned │
  ├────────────┼──────────────────┼─────────────────────────────────┤
  │ CreateMap  │ happy path       │ repo called with JSON bytes     │
  ├────────────┼──────────────────┼─────────────────────────────────┤
  │ GetMapByID │ no permission    │ MapPermissionDenied             │
  ├────────────┼──────────────────┼─────────────────────────────────┤
  │ GetMapByID │ happy path       │ map returned                    │
  ├────────────┼──────────────────┼─────────────────────────────────┤
  │ UpdateMap  │ no permission    │ MapPermissionDenied             │
  ├────────────┼──────────────────┼─────────────────────────────────┤
  │ UpdateMap  │ validation fails │ ValidationErrorWrapper          │
  ├────────────┼──────────────────┼─────────────────────────────────┤
  │ DeleteMap  │ no permission    │ MapPermissionDenied             │
  ├────────────┼──────────────────┼─────────────────────────────────┤
  │ DeleteMap  │ happy path       │ no error                        │
  └────────────┴──────────────────┴─────────────────────────────────┘
  Этап 3 — Delivery/Handler Tests

  Подход: httptest.NewRecorder + httptest.NewRequest, usecase подменяется fake-ом.

  Покрыть для encounter:
  ┌───────────────────┬───────────────────────────────────┬─────────────────┐
  │      Handler      │             Сценарий              │     Assert      │
  ├───────────────────┼───────────────────────────────────┼─────────────────┤
  │ GetEncountersList │ bad JSON body                     │ 400, ErrBadJSON │
  ├───────────────────┼───────────────────────────────────┼─────────────────┤
  │ GetEncountersList │ usecase returns StartPosSizeError │ 400             │
  ├───────────────────┼───────────────────────────────────┼─────────────────┤
  │ GetEncountersList │ usecase returns unknown error     │ 500             │
  ├───────────────────┼───────────────────────────────────┼─────────────────┤
  │ GetEncountersList │ happy path                        │ 200 + body      │
  ├───────────────────┼───────────────────────────────────┼─────────────────┤
  │ GetEncounterByID  │ missing/empty id var              │ 400             │
  ├───────────────────┼───────────────────────────────────┼─────────────────┤
  │ GetEncounterByID  │ PermissionDeniedError             │ 403             │
  ├───────────────────┼───────────────────────────────────┼─────────────────┤
  │ SaveEncounter     │ InvalidInputError                 │ 400             │
  └───────────────────┴───────────────────────────────────┴─────────────────┘
  Этап 4 — Integration Tests (позже)

  - Build tag: //go:build integration
  - Docker Compose для Postgres + MongoDB + Redis
  - 2–5 сценариев: auth login flow, encounter CRUD, bestiary list
  - testcontainers-go или docker compose up -d в test setup

  ---
  6. Матрица покрытия (домены × уровни × приоритет)
  ┌────────────────┬─────────────┬──────────────┬──────────┬─────────────┬────────────────────────────┐
  │     Домен      │ Domain/Pure │ Usecase Unit │ Delivery │ Integration │         Приоритет          │
  ├────────────────┼─────────────┼──────────────┼──────────┼─────────────┼────────────────────────────┤
  │ encounter      │      —      │   Есть ✅    │ Есть ✅  │     P4      │       Высший (пилот)       │
  ├────────────────┼─────────────┼──────────────┼──────────┼─────────────┼────────────────────────────┤
  │ maps           │   Есть ✅   │      P1      │    P2    │     P4      │       Высший (пилот)       │
  ├────────────────┼─────────────┼──────────────┼──────────┼─────────────┼────────────────────────────┤
  │ bestiary       │      —      │      P2      │    P3    │     P4      │          Высокий           │
  ├────────────────┼─────────────┼──────────────┼──────────┼─────────────┼────────────────────────────┤
  │ character      │      —      │      P2      │    P3    │     P4      │          Средний           │
  ├────────────────┼─────────────┼──────────────┼──────────┼─────────────┼────────────────────────────┤
  │ auth           │      —      │      P3      │    P3    │     P4      │          Средний           │
  ├────────────────┼─────────────┼──────────────┼──────────┼─────────────┼────────────────────────────┤
  │ maptiles       │      —      │   Есть ✅    │ Есть ✅  │      —      │           Низкий           │
  ├────────────────┼─────────────┼──────────────┼──────────┼─────────────┼────────────────────────────┤
  │ table          │      —      │      P3      │    —     │     P4      │ Низкий (websocket, сложно) │
  ├────────────────┼─────────────┼──────────────┼──────────┼─────────────┼────────────────────────────┤
  │ bestiary/llm   │      —      │      P3      │    P3    │     P4      │  Низкий (async goroutine)  │
  ├────────────────┼─────────────┼──────────────┼──────────┼─────────────┼────────────────────────────┤
  │ description    │      —      │      P3      │    P3    │      —      │    Низкий (gRPC proxy)     │
  ├────────────────┼─────────────┼──────────────┼──────────┼─────────────┼────────────────────────────┤
  │ utils/merger   │   Есть ✅   │      —       │    —     │      —      │           Готов            │
  ├────────────────┼─────────────┼──────────────┼──────────┼─────────────┼────────────────────────────┤
  │ maps/validator │   Есть ✅   │      —       │    —     │      —      │           Готов            │
  └────────────────┴─────────────┴──────────────┴──────────┴─────────────┴────────────────────────────┘
  ---
  7. Pilot PR Plan

  PR #1: "Test infrastructure + encounter usecase tests"

  Файлы:
  1. internal/pkg/testhelpers/context.go — shared NewTestContext() helper
  2. internal/pkg/encounter/usecases/encounter_test.go — полный набор usecase тестов (~14 сценариев)
  3. Makefile — test, test-race, test-cover targets

  Структура encounter_test.go:
  package usecases

  // fakeEncounterRepo — hand-written stub
  type fakeEncounterRepo struct {
      checkPermissionResult bool
      getListResult         *models.EncountersList
      getListErr            error
      getByIDResult         *models.Encounter
      // ...fields for capturing calls
      saveCalledWith        *models.SaveEncounterReq
  }

  func (f *fakeEncounterRepo) CheckPermission(...) bool { return f.checkPermissionResult }
  // ... implement all EncounterRepository methods

  func TestGetEncountersList_NegativeStart(t *testing.T) { ... }
  func TestGetEncountersList_ZeroSize(t *testing.T) { ... }
  func TestGetEncountersList_HappyPath_NoSearch(t *testing.T) { ... }
  // ... table-driven where groupable

  PR #2: "Maps usecase tests + delivery test template"

  1. internal/pkg/maps/usecases/maps_test.go — usecase тесты
  2. internal/pkg/encounter/delivery/encounter_handlers_test.go — handler тесты (шаблон для остальных)

  ---
  8. Backlog (чеклист с приоритетами)

  - ✅ P0 — Vendor testify/require (отложено — используем assert)
  - ✅ P0 — Создать internal/pkg/testhelpers/helpers.go (NewTestContext, MustJSON, DoRequest, DecodeJSON, DecodeErrorResponse)
  - ✅ P0 — Создать Makefile с test targets (PR2)
  - ✅ P0 — Исправить logger ctxKey (typed struct key + noop fallback, PR1)
  - ✅ P0 — Deprecated LoggerConfig.Key (PR2)
  - ✅ P1 — encounter/usecases/encounter_test.go (14 сценариев, PR1)
  - ✅ P1 — maptiles/usecases/maptiles_test.go (4 сценария table-driven, PR2)
  - P1 — maps/usecases/maps_test.go (11 сценариев)
  - ✅ P2 — encounter/delivery/encounter_handlers_test.go (2 сценария, PR1; обновлено PR2)
  - ✅ P2 — maptiles/delivery/maptiles_handlers_test.go (2 сценария, PR2)
  - P2 — maps/delivery/maps_handlers_test.go
  - P2 — bestiary/usecases/bestiary_test.go
  - P2 — character/usecases/character_test.go
  - P3 — auth/usecases/auth_test.go
  - P3 — bestiary/delivery/bestiary_handlers_test.go
  - P3 — bestiary/usecases/llm_test.go (requires synchronous process or WaitGroup)
  - P3 — table/usecases/table_test.go
  - P4 — Integration tests с //go:build integration
  - P4 — CI pipeline (GitHub Actions)

  ---
  9. Предложение по структуре файлов

  internal/
  ├── pkg/
  │   ├── testhelpers/
  │   │   └── helpers.go          ← NewTestContext(), MustJSON(), DoRequest(), DecodeJSON(), DecodeErrorResponse()
  │   ├── encounter/
  │   │   ├── usecases/
  │   │   │   ├── encounter.go
  │   │   │   └── encounter_test.go    ← usecase unit tests
  │   │   └── delivery/
  │   │       ├── encounter_handlers.go
  │   │       └── encounter_handlers_test.go  ← handler contract tests
  │   ├── maps/
  │   │   ├── usecases/
  │   │   │   ├── maps.go
  │   │   │   ├── maps_test.go
  │   │   │   ├── validator.go
  │   │   │   └── validator_test.go    ← уже есть
  │   │   └── delivery/
  │   │       ├── maps_handlers.go
  │   │       └── maps_handlers_test.go
  │   ...
