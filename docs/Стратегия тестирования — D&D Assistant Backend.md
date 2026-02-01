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

  - GitHub Actions CI (добавлено в PR4):
    - `.github/workflows/ci.yml` — push/PR: `make test` (unit) + `make test-race` (race detector) + `go vet`
    - `.github/workflows/integration.yml` — manual (workflow_dispatch): Postgres 16 + Redis 7 services, `make test-integration`
    - Race detector обязателен на Linux (CGO_ENABLED=1 по умолчанию), невозможен на Windows без gcc
    - Vendor mode: `cache: false` в setup-go, все команды через `-mod=vendor`
  - Makefile в корне проекта: make test, make test-race, make test-cover, make test-integration, make mocks, make verify (добавлен в PR2, дополнен PR3)
  - Нет golangci-lint конфига
  - Vendor mode (-mod=vendor) активен

  Локальная проверка перед коммитом: `make verify`

  Команда `make verify` последовательно выполняет:
  1. `gofmt` — проверяет форматирование (без изменения файлов, только проверка)
  2. `go vet` — статический анализ
  3. `go test` — запуск всех unit-тестов
  4. Консистентность моков — запускает `make mocks`, затем `git diff --exit-code` чтобы убедиться, что сгенерированные моки соответствуют текущим интерфейсам

  Если mock-файлы устарели (интерфейс изменился, а `make mocks` не был выполнен), verify упадёт с ошибкой.

  Правило: при изменении interfaces.go любого домена — обязательно выполнить `make mocks` и закоммитить обновлённые mock-файлы. `make verify` ловит нарушения этого правила автоматически.

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
  Решение: gomock (go.uber.org/mock) для usecase-зависимостей + ручные fakes для delivery
  Обоснование: gomock генерирует строгие моки с проверкой вызовов (unexpected call = fail). Usecase-тесты используют сгенерированные моки для repo/clients интерфейсов. Delivery-тесты используют ручные fakes для usecase-интерфейсов (проще, достаточно для HTTP-маппинга). Генерация: `make mocks` или `go generate ./internal/pkg/auth/...`
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
  - ✅ bestiary/actions_processor: protobuf types (structpb.Struct, AsMap) не должны попадать в usecase слой. Решено: введён ActionProcessorGateway интерфейс (возвращает map[string]interface{}), конвертация structpb.Struct → map вынесена в delivery adapter (action_processor_adapter.go). Usecase теперь тестируется без protobuf зависимостей
  - ✅ bestiary/llm: async goroutine и uuid.New() делали тесты недетерминированными. Решено: AsyncRunner интерфейс (prod: go fn(), test: fn()), IDGenerator интерфейс (prod: uuid.New(), test: фиксированный ID). Usecase тестируется синхронно без sleep/таймеров. Prod реализации в infra.go
  - ✅ table: time.AfterFunc и utils.RandString делали тесты недетерминированными, *websocket.Conn в интерфейсе блокировал transport-тесты. Решено: SessionIDGenerator интерфейс (prod: RandString, test: фиксированный ID), TimerFactory + SessionTimer интерфейсы (prod: time.AfterFunc, test: fake без реальных таймеров). Usecase-логика (permission check, session creation, timer registration) тестируется синхронно. Websocket transport (repository/session.go, repository/participants.go) остаётся без unit-тестов — тонкий адаптер, покрывается интеграционно. Prod реализации в usecases/infra.go

  Стандарт команды: тесты и моки (PR12)

  Handler contract tests (delivery):
  1. Одна тест-функция на handler/endpoint (TestLogin, TestLogout, TestCheckAuth)
  2. Table-driven: []struct с name, request setup, fake, wantStatus, wantErrCode
  3. Субтесты через t.Run(tt.name, ...) + t.Parallel()
  4. Assertions: status code + testhelpers.DecodeErrorResponse (без полного сравнения JSON body)
  5. Ручные fakes для usecase-интерфейсов (простые struct с полями-результатами)
  6. Короткие имена кейсов: bad_json, vk_api_error, happy_path, no_cookie

  Usecase unit tests:
  1. Table-driven с setup func(...) для конфигурации mock expectations
  2. Сгенерированные моки (gomock) для интерфейсов зависимостей (repo, clients, session manager)
  3. gomock проверяет отсутствие unexpected calls автоматически
  4. НЕ мокать инфраструктурные типы (pgxpool, redis client, *websocket.Conn) на уровне usecase

  Генерация моков:
  1. Добавить `//go:generate mockgen -source=interfaces.go -destination=mocks/mock_<domain>.go -package=mocks` в interfaces.go домена
  2. Запуск: `make mocks` или `go generate ./internal/pkg/<domain>/...`
  3. Сгенерированные файлы коммитятся в repo (vendor mode)
  4. Пакет: `go.uber.org/mock` (v0.6.0+), CLI: `mockgen`
  5. При изменении interfaces.go → `make mocks` → коммит обновлённых моков
  6. `make verify` автоматически проверяет консистентность моков перед коммитом

  Статус миграции usecase-тестов на gomock:
  ┌────────────────────────┬──────────┬─────────────────────────────────────────────────────┐
  │         Домен          │  Статус  │                     Примечание                       │
  ├────────────────────────┼──────────┼─────────────────────────────────────────────────────┤
  │ auth                   │  ✅ gomock │ Пилот (PR12)                                       │
  ├────────────────────────┼──────────┼─────────────────────────────────────────────────────┤
  │ encounter              │  ✅ gomock │ MockEncounterRepository                             │
  ├────────────────────────┼──────────┼─────────────────────────────────────────────────────┤
  │ maps                   │  ✅ gomock │ MockMapsRepository                                  │
  ├────────────────────────┼──────────┼─────────────────────────────────────────────────────┤
  │ character              │  ✅ gomock │ MockCharacterRepository + hand-written fakeFile     │
  ├────────────────────────┼──────────┼─────────────────────────────────────────────────────┤
  │ maptiles               │  ✅ gomock │ MockMapTilesRepository                              │
  ├────────────────────────┼──────────┼─────────────────────────────────────────────────────┤
  │ bestiary               │  ✅ gomock │ MockBestiaryRepository, MockBestiaryS3Manager,      │
  │                        │           │ MockGeminiAPI                                       │
  ├────────────────────────┼──────────┼─────────────────────────────────────────────────────┤
  │ bestiary/actions_proc  │  ✅ gomock │ MockActionProcessorGateway                          │
  ├────────────────────────┼──────────┼─────────────────────────────────────────────────────┤
  │ bestiary/gen_creature  │  ✅ gomock │ MockActionProcessorUsecases                         │
  ├────────────────────────┼──────────┼─────────────────────────────────────────────────────┤
  │ bestiary/llm           │  ✅ fake  │ Stateful fakeLLMStorage (in-memory storage fake);   │
  │                        │           │ допустимо по правилу stateful storage fakes (см.    │
  │                        │           │ ниже). Остальные зависимости через gomock            │
  ├────────────────────────┼──────────┼─────────────────────────────────────────────────────┤
  │ table                  │  ✅ gomock │ MockTableManager, MockSessionIDGenerator,           │
  │                        │           │ MockTimerFactory, MockSessionTimer +                │
  │                        │           │ cross-domain encmocks.MockEncounterRepository       │
  ├────────────────────────┼──────────┼─────────────────────────────────────────────────────┤
  │ description            │  ✅ gomock │ DescriptionGateway seam (protobuf→Go types),       │
  │                        │           │ MockDescriptionGateway                              │
  └────────────────────────┴──────────┴─────────────────────────────────────────────────────┘

  Правило: stateful in-memory fakes
  Stateful hand-written fakes допустимы для storage-уровня зависимостей, когда:
  1. Тест верифицирует конечное состояние хранилища после multi-step pipeline (пример: bestiary/llm fakeLLMStorage хранит jobs в map, тест проверяет статус после async processing)
  2. gomock DoAndReturn усложняет тест без выигрыша в читаемости
  3. Фейк реализует storage-интерфейс (repository), а не внешний сервис/клиент

  Внешние сервисы и клиенты (gRPC, HTTP API, S3) всегда мокаются через gomock.
  Stdlib-интерфейсы (multipart.File) могут использовать hand-written fakes (пример: character/fakeFile).

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
  Этап 4 — Integration Tests

  Инфраструктура (добавлена в PR3):
  - Build tag: `//go:build integration` + `// +build integration` (файлы: `*_integration_test.go`)
  - Обычный `go test ./...` НЕ запускает integration tests
  - `make test-integration` — запуск с тегом integration
  - `make integration-up` — поднимает PostgreSQL + Redis через docker compose
  - `make integration-down` — останавливает контейнеры

  Конфигурация:
  - Env var `TEST_POSTGRES_DSN` — DSN для тестовой базы (пример: `postgres://user:pass@localhost:5432/testdb`)
  - Тест пропускается с `t.Skip()` если env не задан
  - Тест создаёт схему, seedит данные, чистит за собой в t.Cleanup

  Принципы:
  1. Один smoke test на адаптер — не разрастаться
  2. Детерминированный: создаёт свои данные, не зависит от существующих
  3. Изолированный: cleanup в defer/t.Cleanup
  4. NoopDBMetrics из testhelpers для обхода Prometheus в тестах
  5. Не добавлять testcontainers без необходимости — используем существующий docker-compose

  Текущее покрытие:
  - ✅ encounter/repository: SaveEncounter + GetEncounterByID (PostgreSQL smoke test)
  - ✅ table/delivery: ServeWS websocket upgrade + mux routing smoke test (in-process httptest.NewServer + gorilla/websocket.Dial, PR11; не требует внешней инфраструктуры; проверяет upgrade, извлечение session ID из URL vars, передачу user из context). Глубокие WS-сценарии (broadcast, multi-participant, reconnect) остаются вне scope — покрываются ручным/E2E тестированием

  ---
  6. Матрица покрытия (домены × уровни × приоритет)
  ┌────────────────┬─────────────┬──────────────┬──────────┬─────────────┬────────────────────────────┐
  │     Домен      │ Domain/Pure │ Usecase Unit │ Delivery │ Integration │         Приоритет          │
  ├────────────────┼─────────────┼──────────────┼──────────┼─────────────┼────────────────────────────┤
  │ encounter      │      —      │   Есть ✅    │ Есть ✅  │  Есть ✅    │       Высший (пилот)       │
  ├────────────────┼─────────────┼──────────────┼──────────┼─────────────┼────────────────────────────┤
  │ maps           │   Есть ✅   │   Есть ✅    │ Есть ✅  │     P4      │       Высший (пилот)       │
  ├────────────────┼─────────────┼──────────────┼──────────┼─────────────┼────────────────────────────┤
  │ bestiary       │      —      │   Есть ✅    │ Есть ✅  │     P4      │          Высокий           │
  ├────────────────┼─────────────┼──────────────┼──────────┼─────────────┼────────────────────────────┤
  │ character      │      —      │   Есть ✅    │ Есть ✅  │     P4      │          Средний           │
  ├────────────────┼─────────────┼──────────────┼──────────┼─────────────┼────────────────────────────┤
  │ auth           │      —      │   Есть ✅    │ Есть ✅  │     P4      │          Средний           │
  ├────────────────┼─────────────┼──────────────┼──────────┼─────────────┼────────────────────────────┤
  │ maptiles       │      —      │   Есть ✅    │ Есть ✅  │      —      │           Низкий           │
  ├────────────────┼─────────────┼──────────────┼──────────┼─────────────┼────────────────────────────┤
  │ table          │      —      │   Есть ✅    │ Есть ✅  │  Есть ✅    │ Покрыт (seam: TimerFactory)│
  ├────────────────┼─────────────┼──────────────┼──────────┼─────────────┼────────────────────────────┤
  │ bestiary/llm   │      —      │   Есть ✅    │    P3    │     P4      │  Покрыт (seam: AsyncRunner)│
  ├────────────────┼─────────────┼──────────────┼──────────┼─────────────┼────────────────────────────┤
  │ description    │      —      │   Есть ✅    │ Есть ✅  │      —      │    Низкий (gRPC proxy)     │
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
  - ✅ P1 — maps/usecases/maps_test.go (20 сценариев, 5 методов, PR4)
  - ✅ P2 — encounter/delivery/encounter_handlers_test.go (2 сценария, PR1; обновлено PR2)
  - ✅ P2 — maptiles/delivery/maptiles_handlers_test.go (2 сценария, PR2)
  - ✅ P2 — maps/delivery/maps_handlers_test.go (4 сценария, PR4)
  - ✅ P2 — bestiary/usecases/bestiary_test.go (19 read-сценариев + 6 processor, PR5+PR6; write/LLM методы пропущены — S3, ObjectID, GeminiAPI)
  - ✅ P2 — character/usecases/character_test.go (13 сценариев, PR3)
  - ✅ P2 — character/delivery/character_handlers_test.go (2 сценария, PR3)
  - ✅ P3 — auth/usecases/auth_test.go (15 сценариев, 4 метода + bugfix nil deref, PR5; рефакторинг PR12: gomock вместо ручных fakes)
  - ✅ P3 — auth/delivery/auth_handlers_test.go (4 сценария, PR5; рефакторинг PR12: table-driven, 9 субтестов в 3 функциях)
  - ✅ P3 — bestiary/delivery/bestiary_handlers_test.go (12 сценариев, PR5+PR6)
  - ✅ P3 — description/usecases/description_test.go (2 сценария, PR5)
  - ✅ P3 — description/delivery/description_handlers_test.go (2 сценария, PR5)
  - ✅ P3 — bestiary/usecases/llm_test.go (12 сценариев, PR8; seams: AsyncRunner (sync fake в тестах), IDGenerator (фиксированный ID), все зависимости через интерфейсы)
  - ✅ P3 — table/usecases/table_test.go (8 сценариев, PR9; seams: SessionIDGenerator (фиксированный ID), TimerFactory (fake timer), бизнес-логика тестируется синхронно, websocket transport skipped)
  - ✅ P3 — table/delivery/table_handlers_test.go (7 сценариев, PR9+PR10; CreateSession: BadJSON→400, PermissionDenied→400, ScanError→400, GenericError→500; GetTableData: MissingID→400, UsecaseError→400, HappyPath→200)
  - ✅ P4 — table/delivery/servews_integration_test.go (1 smoke, PR11; websocket upgrade через httptest.NewServer + gorilla/websocket.Dial, mux routing с {id} param, user injection через middleware; запуск: `make test-integration` или `go test -tags=integration`)
  - ✅ P3 — bestiary/usecases/actions_processor_test.go (10 сценариев, PR7; seam: ActionProcessorGateway интерфейс, protobuf→map конвертация вынесена в delivery adapter)
  - SKIP — statblockgenerator (пустой stub, нет методов)
  - ✅ P4 — Integration test scaffold: build tag, Makefile targets, NoopDBMetrics (PR3)
  - ✅ P4 — encounter/repository integration smoke test: Save + GetByID (PR3)
  - ✅ P4 — CI pipeline: ci.yml (unit + race + vet) + integration.yml (manual, Postgres + Redis) (PR4)

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
  │   │   ├── repository/
  │   │   │   └── encounter_integration_test.go  ← //go:build integration
  │   │   └── delivery/
  │   │       ├── encounter_handlers.go
  │   │       └── encounter_handlers_test.go  ← handler contract tests
  │   ├── character/
  │   │   ├── usecases/
  │   │   │   ├── character.go
  │   │   │   └── character_test.go    ← usecase unit tests
  │   │   └── delivery/
  │   │       ├── character_handlers.go
  │   │       └── character_handlers_test.go  ← handler contract tests
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
