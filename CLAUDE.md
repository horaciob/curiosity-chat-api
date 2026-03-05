# curiosity-chat-api — Agent Instructions

## Project

Real-time 1-on-1 chat microservice for the Curiosity platform.
- **Module**: `github.com/horaciobranciforte/curiosity-chat-api`
- **Port**: `8081`
- **DB**: PostgreSQL 15, database `curiosity_chat`, port `5434` (test: `5435`)
- **Related service**: `curiosity-user-api` on port `8084` (provides JWT validation and follow graph)

## Mandatory Conventions

### Language
All code, comments, variable names, error messages, SQL, and documentation must be in **English**.

### Test naming
CamelCase only — **never use underscores** in test function names.
```go
// CORRECT
func TestCreateConversationSuccess(t *testing.T) {}
func TestSendTextMessageEmptyContent(t *testing.T) {}

// WRONG
func TestCreateConversation_Success(t *testing.T) {}
```

### IDs
Always `uuid.New().String()`. Never hardcode UUIDs in tests.

## Architecture

Clean Architecture — dependencies flow inward only:
```
cmd/api/main.go
  └── internal/adapter/http/         (handlers, middleware, router, response DTOs)
        └── internal/usecase/        (business logic, repository interfaces)
              └── internal/domain/   (entities, domain errors — zero dependencies)
  └── internal/adapter/repository/postgres/  (SQL implementations)
  └── internal/infrastructure/       (auth, config, database, followclient)
  └── internal/ws/                   (WebSocket hub)
  └── internal/pkg/apperror/         (typed errors → HTTP codes)
```

Key rules:
- Use case constructors: `New<Name>(deps...) *<Name>` with single `Execute(ctx, ...) (..., error)`
- Repository interfaces defined in `internal/usecase/<domain>/interface.go`
- Handlers inject concrete use case structs, not interfaces
- `handleUseCaseError(w, err)` maps `*apperror.Error` to HTTP responses

## Error Handling

Use `internal/pkg/apperror/`:
- `apperror.Validation(msg, err)` → 400
- `apperror.NotFound(msg, err)` → 404
- `apperror.Forbidden(msg, err)` → 403
- `apperror.Internal(msg, err)` → 500

## WebSocket Security

JWT token goes in the **first WebSocket frame**, never in the URL.
```
ws://localhost:8081/api/v1/ws           (connect — no token in URL)
→ {"type":"auth","token":"<jwt>"}       (first frame, within 10s)
← {"type":"auth_ok","user_id":"<uuid>"} (server ack)
```

## Common Commands

```bash
make run           # run server (generates docs first)
make test          # go test -v ./...
make docs          # swag init → regenerate swagger
make migrate       # run migrations on prod + test DBs
make db-setup      # docker compose up + migrate
make install-tools # install migrate + swag CLIs
```

## Skills Available

- `/golang-pro` — Go concurrency, interfaces, generics, testing patterns
- `/curiosity-chat-api` — This project's patterns, architecture, and API reference
