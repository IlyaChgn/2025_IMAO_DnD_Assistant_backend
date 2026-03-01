# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

D&D Assistant backend - a Go REST API and WebSocket server for Dungeons & Dragons encounter management. Uses VK ID OAuth for authentication.

## Build and Run Commands

```bash
# Run development server
go run cmd/app/main.go

# Run production server
go run cmd/app/main.go -prod

# Build
go build ./cmd/app/main.go

# Run tests
go test ./...

# Run single test
go test ./internal/pkg/utils/merger/... -v

# Database migrations
go run cmd/app/main.go -migrate latest     # Apply all
go run cmd/app/main.go -migrate 1          # Apply specific version

# Start dependencies (Postgres, Redis, Prometheus)
docker compose up -d

# With MongoDB and MinIO
docker compose -f docker-compose.yml -f docker-compose.mongo_and_minio.yml up -d
```

## Architecture

### Clean Architecture Pattern

Each feature module follows this structure:
```
feature/
├── interfaces.go       # Contract definitions (ports)
├── delivery/           # HTTP/gRPC handlers (adapters)
├── usecases/           # Business logic
├── repository/         # Data access
└── external/           # Third-party API clients
```

### Key Modules

- **auth** - VK ID OAuth, session management (Redis)
- **bestiary** - Creature management, AI generation via Gemini (MongoDB, MinIO)
- **character** - Character CRUD (MongoDB)
- **encounter** - Encounter management (PostgreSQL)
- **description** - Battle descriptions (gRPC client)
- **table** - Real-time WebSocket sessions
- **maptiles** - Map tile storage (MongoDB)

### Tech Stack

- **Router:** `gorilla/mux`
- **WebSockets:** `gorilla/websocket`
- **gRPC:** `google.golang.org/grpc`
- **PostgreSQL:** `jackc/pgx/v5` (connection pooling, no ORM)
- **MongoDB:** `go.mongodb.org/mongo-driver`
- **Redis:** `redis/go-redis/v9`
- **MinIO:** S3-compatible object storage for images
- **Logging:** `uber/zap`
- **Metrics:** `prometheus/client_golang`
- **Config:** `ilyakaznacheev/cleanenv` (YAML + env vars)

### Middleware Chain (applied in order)

1. Request logging
2. Request context enrichment
3. Panic recovery
4. Prometheus metrics
5. Auth verification (on protected routes)

### Request Handler Pattern

```go
func (h *Handler) Method(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()
    l := logger.FromContext(ctx)  // Extract logger from context

    // Decode request, call usecase
    result, err := h.usecases.DoSomething(ctx, data)

    // Error handling with custom app errors
    switch {
    case errors.Is(err, apperrors.SomeError):
        responses.SendErrorResponse(w, "ERROR_CODE", http.StatusBadRequest)
        return
    }

    responses.SendOkResponse(w, result)
}
```

### Session/Auth

- Sessions stored in Redis (30-day TTL)
- Session ID in `session_id` cookie
- Protected routes use `LoginRequiredMiddleware`
- User extracted from context via config key `cfg.CtxUserKey`

## API Routes

Base path: `/api`

- `/api/auth/*` - Authentication (login, logout, check)
- `/api/bestiary/*` - Creature CRUD and AI generation
- `/api/character/*` - Character management
- `/api/encounter/*` - Encounter management
- `/api/battle/description` - Battle descriptions (gRPC)
- `/api/maptiles/*` - Map tiles
- `/api/table/ws` - WebSocket endpoint
- `/api/llm/*` - AI generation jobs
- `/metrics` - Prometheus metrics

## Database Migrations

Location: `db/migrations/`

Format: `NNN_name.up.sql` and `NNN_name.down.sql`

Uses `golang-migrate/migrate/v4`.

## Configuration

- `internal/pkg/config/config.yaml` - Static config (server, endpoints, logger)
- `.env` / `prod.env` - Environment variables (database credentials, API keys)
- `cfg := config.ReadConfig(cfgPath)` loads both

## External Services

- **VK API** - OAuth authentication
- **Gemini AI** - Creature generation (accessed via SOCKS5 proxy)
- **Description Service** - gRPC at `localhost:50051`
- **Action Processor Service** - gRPC for creature actions

## Technical Documentation

Feature plans, API specs, and implementation logs are in `vibecode_docs/`. See [vibecode_docs/README.md](vibecode_docs/README.md) for index and [vibecode_docs/DOCS_STANDARDS.md](vibecode_docs/DOCS_STANDARDS.md) for document type conventions.
