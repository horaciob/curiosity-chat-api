---
name: curiosity-chat-api
description: "Patterns, conventions, and full reference for the Curiosity Chat API microservice. Use when building features, writing tests, adding endpoints, or integrating with this service. Covers Clean Architecture layers, WebSocket hub, JWT auth flow, message types, and the full endpoint contract."
metadata:
  author: horaciobranciforte
  version: 1.0.0
  domain: backend
  triggers: curiosity chat, chat api, websocket, real-time messaging, conversation, messages, poi share, jwt auth websocket, follow check
  role: specialist
  scope: implementation
  output-format: code
  related-skills: golang-pro, curiosity-api
---

# Curiosity Chat API

## Overview

Standalone Go microservice for real-time 1-on-1 chat between users who mutually follow each other in curiosity-api. Messages are persisted in PostgreSQL. Users can send text messages or share a POI (`type = "poi_share"`). WebSockets for real-time delivery, REST for HTTP fallback and history.

- **Module**: `github.com/horaciobranciforte/curiosity-chat-api`
- **Port**: `8081` (curiosity-api runs on `8080`)
- **DB**: `curiosity_chat` on port `5434` (test: `5435`)
- **Auth**: HS256 JWT with `sub` = userID — same secret as curiosity-api

## Core Conventions

### Test Naming: CamelCase Only
**MANDATORY**: all test function names use CamelCase without underscores.

```go
// CORRECT
func TestCreateConversationSuccess(t *testing.T) {}
func TestSendTextMessageEmptyContent(t *testing.T) {}
func TestGetMessagesNotParticipant(t *testing.T) {}

// WRONG — never use underscores
func TestCreateConversation_Success(t *testing.T) {}
```

### English Only
All code, comments, variable names, error messages, and documentation must be in English.

### IDs
Always use `uuid.New().String()` from `github.com/google/uuid`. Never hardcode IDs in tests.

## Architecture Layers

Dependencies flow inward: **Infrastructure → Adapter → UseCase → Domain**

### 1. Domain (`internal/domain/`)
- `entity/conversation.go` — `Conversation` with `HasParticipant`, `OtherUserID`; `NewConversation` normalizes `user1 < user2`
- `entity/message.go` — `Message` with `NewTextMessage`, `NewPOIShareMessage`; constants `MessageTypeText`, `MessageTypePOIShare`
- `errors/domain_errors.go` — sentinel errors (`ErrConversationNotFound`, `ErrNotParticipant`, `ErrUsersCannotChat`, etc.)

### 2. Use Cases (`internal/usecase/`)
One struct per use case, `Execute(ctx, ...args)` method. Interfaces defined in `interface.go` inside the same package.

```
usecase/conversation/
  interface.go          — Repository + FollowChecker interfaces
  create_conversation.go
  get_conversation.go
  list_conversations.go
  *_test.go

usecase/message/
  interface.go          — Repository + ConversationRepository interfaces
  send_message.go
  get_messages.go
  *_test.go
```

### 3. Adapter Layer (`internal/adapter/`)
- `repository/postgres/` — `ConversationRepository` (implements both conversation.Repository and message.ConversationRepository), `MessageRepository`
- `http/handler/` — `ConversationHandler`, `MessageHandler`, `WSHandler`, `HealthHandler`
- `http/middleware/auth.go` — `Authenticate` middleware + `UserIDFromContext`
- `http/response/` — `Success`, `Created`, `Collection`, `Error`, `JSON` helpers + DTO types
- `http/router/router.go` — chi router wiring + Swagger UI at `/swagger/`

### 4. Infrastructure (`internal/infrastructure/`)
- `auth/jwt.go` — `JWTService.Validate(token) (userID, error)` — HS256, extracts `sub`
- `config/config.go` — env-based config, `mustEnv("JWT_SECRET")` panics if missing
- `database/postgres.go` — `NewPostgresDB(dsn)` via sqlx
- `followclient/client.go` — `Client` calls curiosity-api `/api/v1/users/{id}/followers`; `NoopFollowChecker` for tests

### 5. WebSocket (`internal/ws/hub.go`)
- `Hub` — `map[userID]map[*Client]bool`, channels for register/unregister/broadcast
- `Client` — `UserID string`, `Conn *websocket.Conn`, `Send chan []byte`, `WritePump()`
- `Hub.BroadcastTo(userID, OutgoingMessage)` — delivers to all connections of a user
- Auth constants: `MessageTypeAuth = "auth"`, `AuthDeadline = 10 * time.Second`

## Error Handling
Use `internal/pkg/apperror/` typed errors:

| Constructor | Type | HTTP |
|---|---|---|
| `apperror.Validation(msg, err)` | VALIDATION | 400 |
| `apperror.BadRequest(msg, err)` | BAD_REQUEST | 400 |
| `apperror.NotFound(msg, err)` | NOT_FOUND | 404 |
| `apperror.Forbidden(msg, err)` | FORBIDDEN | 403 |
| `apperror.Conflict(msg, err)` | CONFLICT | 409 |
| `apperror.Internal(msg, err)` | INTERNAL | 500 |

Handlers call `handleUseCaseError(w, err)` which maps `*apperror.Error` to HTTP responses.

## Testing Patterns

### Unit Tests (Use Cases)
```go
func TestSendTextMessageSuccess(t *testing.T) {
    msgRepo := new(mocks.MessageRepositoryMock)
    convRepo := new(mocks.ConversationRepositoryMock)
    uc := NewSendMessage(msgRepo, convRepo)

    ctx := context.Background()
    convID := uuid.New().String()
    senderID := uuid.New().String()

    conv := entity.NewConversation(senderID, uuid.New().String())
    conv.ID = convID
    convRepo.On("GetByID", ctx, convID).Return(conv, nil)
    msgRepo.On("Create", ctx, mock.AnythingOfType("*entity.Message")).Return(nil)
    convRepo.On("UpdateLastMessageAt", ctx, convID, mock.AnythingOfType("time.Time")).Return(nil)

    msg, err := uc.Execute(ctx, convID, senderID, SendMessageInput{Type: "text", Content: "Hello!"})
    require.NoError(t, err)
    assert.Equal(t, "text", msg.Type)
    msgRepo.AssertExpectations(t)
    convRepo.AssertExpectations(t)
}
```

Mocks are hand-written in `test/mocks/` using `testify/mock`.

> Load `references/testing-patterns.md` for full mock and assertion patterns.

## Message Types

| `type` | Required field | Forbidden field | DB constraint |
|---|---|---|---|
| `text` | `content` (non-empty) | `poi_id` | `content IS NOT NULL AND poi_id IS NULL` |
| `poi_share` | `poi_id` (UUID) | `content` | `poi_id IS NOT NULL AND content IS NULL` |

Enforced at use case layer AND by PostgreSQL CHECK constraint.

## Conversation Uniqueness

Pair (A, B) is stored with `user1_id < user2_id` (lexicographic). `NewConversation(a, b)` normalizes order. DB enforces with `UNIQUE(user1_id, user2_id)` + `CHECK(user1_id < user2_id)`. `POST /api/v1/conversations` always returns the same row for the same pair.

## WebSocket Auth Protocol

Token goes in the **first application frame**, never in the URL.

```
Client → ws://localhost:8081/api/v1/ws   (no token in URL)
Client → {"type":"auth","token":"<jwt>"}  (within 10s)
Server ← {"type":"auth_ok","user_id":"<uuid>"}
         — or close(1008) on failure/timeout
Client → {"type":"text","conversation_id":"<uuid>","content":"Hello!"}
Server → broadcast OutgoingMessage to sender + other participant
```

## API Reference

> Load `references/api-endpoints.md` for full endpoint contract with request/response examples.

### Endpoints

| Method | Path | Auth | Description |
|---|---|---|---|
| `GET` | `/api/v1/health` | No | Health check |
| `GET` | `/api/v1/ws` | First WS frame | WebSocket |
| `POST` | `/api/v1/conversations` | Bearer | Create or get conversation |
| `GET` | `/api/v1/conversations` | Bearer | List my conversations |
| `GET` | `/api/v1/conversations/{id}` | Bearer | Get conversation |
| `GET` | `/api/v1/conversations/{id}/messages` | Bearer | Message history |
| `POST` | `/api/v1/conversations/{id}/messages` | Bearer | Send message (HTTP fallback) |
| `GET` | `/swagger/` | No | Swagger UI |
