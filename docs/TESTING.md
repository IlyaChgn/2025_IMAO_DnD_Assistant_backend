# Testing Standards — D&D Assistant Backend

This document defines the team's testing conventions, mock standards, and available commands.
It is the single source of truth for how tests are written in this project.

---

## 1. Principles

### Test pyramid

```
              ┌──────────────┐
              │ Integration  │  build tag: //go:build integration
              ├──────────────┤
           ┌──┤  Delivery /  │  httptest.NewRecorder
           │  │  Contract    │  assert status codes + body
           │  ├──────────────┤
        ┌──┤  │  Usecase     │  main volume of tests
        │  │  │  Unit        │  mock repos + assert errors
        │  │  ├──────────────┤
     ┌──┤  │  │  Domain /    │  validators, pure funcs
     │  │  │  │  Pure Logic  │  no mocks needed
     └──┴──┴──┴──────────────┘
```

### Key rules

1. **Deterministic tests** — no `time.Sleep`, no reliance on `time.Now()` or `uuid.New()` in assertions. Use seams (interfaces) when determinism is needed.
2. **Error checking via `errors.Is()`** — usecase sentinel errors propagate through layers; always assert with `errors.Is`, not string comparison.
3. **Table-driven tests** — preferred for any method with 2+ scenarios. Use `t.Run(tt.name, ...)` + `t.Parallel()`.
4. **No log assertions** — logger is called but its output is never asserted.
5. **No exact UUID assertions** — assert `!= ""` or check format, not exact value.
6. **Context with logger** — `logger.FromContext(ctx)` returns a noop logger automatically; no special setup needed.
7. **Race detector** — `make test-race` catches data races before they become flaky tests.

---

## 2. Test structure

### Package strategy

| Test type          | Package                           | Why                                                      |
|--------------------|-----------------------------------|----------------------------------------------------------|
| Usecase unit       | `package usecases` (internal)     | Access to unexported struct for construction              |
| Delivery/handler   | `package delivery_test` (external)| Test only public API; simulate a real client              |
| Pure logic         | `package usecases` (internal)     | Functions are exported, works as-is                       |
| Integration        | `package *_test` (build-tagged)   | Isolated via `//go:build integration`                     |

### Naming conventions

- `*_test.go` — always next to the file under test
- `TestMethodName` or `TestMethodName_Scenario` — e.g. `TestGetEncountersList_NegativeStart`
- Short subtest names: `happy_path`, `repo_error`, `bad_json`, `no_permission`

### One function per method

- Usecase: one `TestGetCreaturesList`, one `TestSaveEncounter`, etc.
- Handler: one `TestGenerateDescription`, one `TestGetEncountersList`, etc.
- Cases go inside as table entries, not separate top-level functions.

---

## 3. Mock standards

### Usecase tests — gomock

- **Library:** `go.uber.org/mock` (gomock v0.6.0+)
- **Pattern:** table-driven with `setup func(...)` to configure mock expectations per case
- gomock auto-verifies no unexpected calls
- **Do NOT mock** infrastructure types (`pgxpool.Pool`, `redis.Client`, `*websocket.Conn`) at the usecase level

```go
tests := []struct {
    name    string
    setup   func(repo *mocks.MockEncounterRepository)
    wantErr error
}{
    {
        name: "repo error is propagated",
        setup: func(repo *mocks.MockEncounterRepository) {
            repo.EXPECT().GetEncounterByID(gomock.Any(), "id-1").Return(nil, repoErr)
        },
        wantErr: repoErr,
    },
}
```

### Delivery tests — hand-written fakes

- Simple struct with result/error fields implementing the usecase interface
- Sufficient for HTTP status code + error response mapping
- No gomock overhead needed at this layer

### Stateful in-memory fakes

Stateful hand-written fakes are acceptable for **storage-level** dependencies when:

1. The test verifies final state after a multi-step pipeline (e.g. `bestiary/llm fakeLLMStorage` stores jobs in a map, test checks status after async processing)
2. `gomock.DoAndReturn` would make the test harder to read without benefit
3. The fake implements a storage/repository interface, **not** an external service

**External services/clients** (gRPC, HTTP API, S3) are always mocked via gomock.

**Stdlib interfaces** (`multipart.File`) may use hand-written fakes when appropriate.

---

## 4. Mock generation

Each domain's `interfaces.go` has a `//go:generate` directive:

```go
//go:generate mockgen -source=interfaces.go -destination=mocks/mock_<domain>.go -package=mocks
```

**Generated mocks are NOT committed to git.** They are gitignored (`internal/pkg/**/mocks/`) and regenerated automatically by `make test`, `make test-race`, and `make verify`.

### Workflow

1. Edit `interfaces.go` (add/change/remove a method)
2. Run `make test` or `make verify` — mocks are regenerated automatically
3. Alternatively, run `make mocks` directly to regenerate without running tests

### On a fresh clone

```bash
make test    # generates mocks, then runs tests — works out of the box
```

No manual mock generation step needed. The `test`, `test-race`, `test-cover`, and `verify` targets all depend on `mocks`.

---

## 5. Commands

### Unit tests

```bash
make test           # generates mocks, then: go test -mod=vendor ./...
make test-race      # generates mocks, then: go test -race -mod=vendor ./...
make test-cover     # generates mocks, then: go test -coverprofile + go tool cover
```

All test targets depend on `mocks` — no manual generation step needed.

### Mock generation (standalone)

```bash
make mocks                                        # GOFLAGS=-mod=vendor go generate -run mockgen ./internal/...
go generate -run mockgen ./internal/pkg/auth/...  # single domain
```

### Pre-commit verification

```bash
make verify
```

Runs sequentially:
1. `gofmt` — checks formatting (hard failure)
2. `make mocks` — regenerates mocks
3. `go vet` — static analysis
4. `go test` — full unit test suite

### Integration tests

```bash
make integration-up     # docker compose up -d postgres redis
make test-integration   # go test -mod=vendor -tags=integration ./...
make integration-down   # docker compose down
```

Integration tests use build tag `//go:build integration` and are never run by `make test`.

Requires `TEST_POSTGRES_DSN` env var. Tests skip with `t.Skip()` if not set.

### CI pipeline

CI (`.github/workflows/ci.yml`) runs on every push/PR:

**Unit Tests** job:

1. `go install go.uber.org/mock/mockgen@v0.6.0` — installs mockgen binary
2. `make verify` — gofmt + mocks + go vet + go test

**Race Detector** job:

1. `go install go.uber.org/mock/mockgen@v0.6.0` — installs mockgen binary
2. `make test-race` — go test -race

CI installs mockgen, then runs make targets; these targets generate mocks automatically before running checks (via the `mocks` dependency in the Makefile).

**Prerequisite:** `mockgen` must be in `$PATH`. Locally, install once with `go install go.uber.org/mock/mockgen@v0.6.0`. In CI, this is handled automatically.

**Network dependency:** `go install mockgen@v0.6.0` requires network access (downloads the binary). This is available in GitHub Actions. Mock generation itself (`go generate -run mockgen`) does NOT need network — it reads local source files.

---

## 6. Error mapping reference

| Usecase sentinel error   | HTTP code | Delivery constant       |
|--------------------------|-----------|-------------------------|
| `StartPosSizeError`      | 400       | `ErrSizeOrPosition`     |
| `InvalidInputError`      | 400       | `ErrWrongEncounterName` |
| `InvalidUserIDError`     | 400       | `ErrInvalidID`          |
| `PermissionDeniedError`  | 403       | `ErrForbidden`          |
| `MapPermissionDenied`    | 403       | `FORBIDDEN`             |
| `MapNotFoundError`       | 404       | `NOT_FOUND`             |
| `NoDocsErr`              | 200       | (nil body)              |
| `ValidationErrorWrapper` | 422       | `INVALID_*`             |
| (default)                | 500       | `ErrInternalServer`     |
