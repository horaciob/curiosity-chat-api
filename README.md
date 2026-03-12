# curiosity-chat-api

Real-time 1-on-1 chat microservice for the Curiosity platform. Users who mutually follow each other can exchange text messages and share Points of Interest (POIs) via WebSockets, with full PostgreSQL persistence.

## Overview

- **Language**: Go
- **Architecture**: Clean Architecture (entity → use case → repository → HTTP adapter)
- **Transport**: REST (JSON:API) + WebSockets (gorilla/websocket)
- **Database**: PostgreSQL 15
- **Port**: `8081` (curiosity-api runs on `8080`)
- **Auth**: JWT — same token issued by `curiosity-api` (shared `JWT_SECRET`)

---

## Prerequisites

- Go 1.21+
- Docker & Docker Compose
- [golang-migrate](https://github.com/golang-migrate/migrate) CLI + swag CLI

```bash
make install-tools
```

---

## Getting started

```bash
# 1. Copy env file and fill in JWT_SECRET (must match curiosity-api)
cp .env.example .env

# 2. Start PostgreSQL and run migrations
make db-setup

# 3. Generate Swagger docs
make docs

# 4. Run the server
make run
```

Server: `http://localhost:8081` — Swagger UI: `http://localhost:8081/swagger/`

---

## Environment variables

| Variable | Default | Description |
|---|---|---|
| `SERVER_PORT` | `8081` | HTTP server port |
| `DB_HOST` | `localhost` | PostgreSQL host |
| `DB_PORT` | `5434` | PostgreSQL port |
| `DB_USER` | `postgres` | PostgreSQL user |
| `DB_PASSWORD` | `postgres` | PostgreSQL password |
| `DB_NAME` | `curiosity_chat` | Database name |
| `DB_SSL_MODE` | `disable` | SSL mode |
| `JWT_SECRET` | — | **Required.** Must match `curiosity-api` |
| `CURIOSITY_API_URL` | `http://localhost:8080` | Base URL for follow-check calls |
| `LOG_LEVEL` | `info` | `debug`, `info`, `warn`, `error` |
| `LOG_FORMAT` | `console` | `console` or `json` |

---

## Complete flow

This section documents the full lifecycle of the chat feature end-to-end.

### 1. Prerequisites in curiosity-api

Before two users can chat they must mutually follow each other. This is enforced by the `curiosity-api` follow system.

```
User A  →  follows →  User B   (accepted)
User B  →  follows →  User A   (accepted)
```

When `POST /api/v1/conversations` is called, curiosity-chat-api calls
`curiosity-api`'s `/api/v1/users/{id}/followers` in both directions to verify mutual follow before allowing the conversation.

### 2. Authentication

All tokens are issued by `curiosity-api` (SSO login). The `JWT_SECRET` is shared between the two services.

Token structure (HS256, standard `RegisteredClaims`):
- `sub` → userID (UUID)
- `exp` → expiration timestamp

REST endpoints read the token from the `Authorization: Bearer <token>` header.
The WebSocket endpoint reads the token from the **first application frame** (never from the URL).

### 3. REST flow — start a conversation and send a message

```
1. Obtain a token from curiosity-api login
   POST http://localhost:8080/api/v1/auth/login
   → { "token": "<jwt>" }

2. Create (or retrieve) a conversation
   POST http://localhost:8081/api/v1/conversations
   Authorization: Bearer <jwt>
   { "target_user_id": "<uuid>" }
   → 201 { "data": { "id": "<conv-uuid>", "type": "conversations", ... } }
     (or 200 if the conversation already exists)

3. Send a text message via HTTP
   POST http://localhost:8081/api/v1/conversations/<conv-uuid>/messages
   Authorization: Bearer <jwt>
   { "type": "text", "content": "Hello!" }
   → 201 { "data": { "id": "<msg-uuid>", "type": "messages", ... } }

4. Share a POI via HTTP
   POST http://localhost:8081/api/v1/conversations/<conv-uuid>/messages
   Authorization: Bearer <jwt>
   { "type": "poi_share", "poi_id": "<poi-uuid>" }
   → 201

5. Fetch message history (newest first)
   GET http://localhost:8081/api/v1/conversations/<conv-uuid>/messages
   Authorization: Bearer <jwt>
   → 200 { "data": [...], "meta": { "total": 12 }, ... }
```

### 4. WebSocket flow — real-time messaging

The JWT token is **never sent in the URL** (query params appear in server logs and browser history). Authentication happens over the WebSocket connection itself via the first application frame.

```
┌─────────────────────────────────────────────────────────────────┐
│  Client A                          Server                        │
│                                                                  │
│  WS connect (no token in URL) ──→  Upgrade accepted             │
│                                                                  │
│  {"type":"auth",               ──→  Validate JWT                │
│   "token":"<jwt>"}                  Extract userID from sub      │
│                                                                  │
│                               ←──  {"type":"auth_ok",           │
│                                     "user_id":"<uuid>"}          │
│                                                                  │
│  {"type":"text",               ──→  Save to DB                  │
│   "conversation_id":"<uuid>",       Update last_message_at       │
│   "content":"Hello!"}               Broadcast to sender A        │
│                                     Broadcast to recipient B     │
│                                                                  │
│                               ←──  {"id":"<msg-uuid>",          │
│                                     "type":"text",               │
│                                     "conversation_id":"<uuid>",  │
│                                     "sender_id":"<uuid>",        │
│                                     "content":"Hello!",          │
│                                     "created_at":"..."}          │
│                                                                  │
│  {"type":"poi_share",          ──→  Save to DB                  │
│   "conversation_id":"<uuid>",       Broadcast to A and B        │
│   "poi_id":"<poi-uuid>"}                                         │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

**Auth timeout**: if the client does not send the auth frame within **10 seconds**, the server closes the connection with WebSocket close code `1008` (Policy Violation).

**Multi-device**: the Hub tracks all active connections per userID. Every device logged in as the same user receives broadcast messages.

### 5. Message types

| `type` | Required fields | Forbidden fields |
|---|---|---|
| `text` | `content` (non-empty string) | `poi_id` |
| `poi_share` | `poi_id` (UUID) | `content` |

This constraint is enforced at the use case layer **and** by a PostgreSQL `CHECK` constraint.

### 6. Conversation uniqueness

A conversation between user A and user B is stored as a single row regardless of who initiates it. The pair is normalized so `user1_id < user2_id` (lexicographic UUID comparison). This is enforced at the domain layer (`NewConversation`) and by a `UNIQUE` + `CHECK` constraint in the database.

```sql
CONSTRAINT conversations_unique_pair UNIQUE (user1_id, user2_id)
CONSTRAINT conversations_ordered     CHECK  (user1_id < user2_id)
```

Calling `POST /api/v1/conversations` twice with the same pair always returns the same conversation.

---

## Limits & Constraints

### Message Content Limits

| Field | Max Length | Description |
|-------|-----------|-------------|
| `content` (text messages) | **1000 characters** | Text message content |
| `content` (POI title) | **500 characters** | POI share title |
| `poi_id` | UUID format | Must be valid UUID |
| `share_intent` | Enum | `must_go`, `come_with_me`, `invite`, `invite_me` |

### Pagination Limits

| Endpoint | Default Limit | Max Limit |
|----------|--------------|-----------|
| `GET /conversations` | 20 | 100 |
| `GET /conversations/{id}/messages` | 50 | 100 |

### Request Body Limits

- **Max request body size**: 1MB for all POST endpoints
- Negative pagination values are normalized to defaults

### CORS Configuration

Allowed origins are configurable via the `ALLOWED_ORIGINS` environment variable (comma-separated).
Default for development:
- `http://localhost:3000`
- `http://localhost:8080`
- `https://localhost:3000`
- `https://localhost:8080`

**Note**: Wildcard (`*`) origins are **not** allowed in production.

### WebSocket Constraints

- **Auth timeout**: 10 seconds to send first auth frame
- **Buffer sizes**: 
  - Read: 1024 bytes
  - Write: 1024 bytes
  - Client send channel: 256 messages
- **DB operation timeout**: 10 seconds per message operation

---

## API endpoints

All data endpoints use [JSON:API](https://jsonapi.org/) format. WebSocket and health use plain JSON.

| Method | Path | Auth | Description |
|---|---|---|---|
| `GET` | `/api/v1/health` | No | Health check |
| `GET` | `/api/v1/ws` | Via first WS frame | WebSocket connection |
| `POST` | `/api/v1/conversations` | Bearer | Create or get existing conversation |
| `GET` | `/api/v1/conversations` | Bearer | List my conversations (paginated, newest first) |
| `GET` | `/api/v1/conversations/{id}` | Bearer | Get a conversation |
| `GET` | `/api/v1/conversations/{id}/messages` | Bearer | Message history (newest first, paginated) |
| `POST` | `/api/v1/conversations/{id}/messages` | Bearer | Send a message (HTTP fallback) |
| `GET` | `/swagger/` | No | Swagger UI |
| `GET` | `/swagger/doc.json` | No | OpenAPI spec |

---

## Makefile targets

```bash
make run              # Build (generates docs) and run
make build            # Compile binary to bin/
make test             # Run all tests with verbose output
make coverage         # Generate HTML coverage report
make docs             # Generate Swagger documentation
make install-tools    # Install migrate + swag CLIs

make migrate          # Run migrations on prod + test DBs
make migrate-up       # Run pending migrations on prod DB
make migrate-down     # Rollback last migration
make migrate-create NAME=my_migration

make db-setup         # Start postgres + migrate
make db-setup-test    # Start test postgres + migrate
make db-teardown      # Stop and remove containers
make db-reset         # Full teardown + setup
```

---

## Database schema

```sql
-- Ordered pair, guaranteed unique per user duo
CREATE TABLE conversations (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user1_id        UUID NOT NULL,  -- always the lexicographically smaller UUID
    user2_id        UUID NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_message_at TIMESTAMPTZ,
    CONSTRAINT conversations_unique_pair UNIQUE (user1_id, user2_id),
    CONSTRAINT conversations_ordered     CHECK  (user1_id < user2_id)
);

CREATE TABLE messages (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    conversation_id UUID NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
    sender_id       UUID NOT NULL,
    type            VARCHAR(20) NOT NULL CHECK (type IN ('text', 'poi_share')),
    content         TEXT,   -- non-null when type='text'
    poi_id          UUID,   -- non-null when type='poi_share'
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT messages_content_check CHECK (
        (type = 'text'      AND content IS NOT NULL AND poi_id IS NULL) OR
        (type = 'poi_share' AND poi_id  IS NOT NULL AND content IS NULL)
    )
);
```

---

## Project structure

```
.
├── cmd/api/main.go                         # Wiring & server startup
├── docs/                                   # Generated Swagger (swag init)
├── migrations/                             # SQL migrations
├── internal/
│   ├── domain/
│   │   ├── entity/                         # Conversation, Message
│   │   └── errors/                         # Domain error sentinels
│   ├── usecase/
│   │   ├── conversation/                   # CreateConversation, GetConversation, ListConversations
│   │   └── message/                        # SendMessage, GetMessages
│   ├── adapter/
│   │   ├── repository/postgres/            # SQL implementations
│   │   └── http/
│   │       ├── handler/                    # HTTP + WebSocket handlers
│   │       ├── middleware/                  # JWT auth middleware
│   │       ├── response/                   # JSON:API response helpers
│   │       └── router/                     # Chi router + Swagger UI
│   ├── infrastructure/
│   │   ├── auth/                           # JWT validation (HS256, sub claim)
│   │   ├── config/                         # Env-based config
│   │   ├── database/                       # sqlx connection pool
│   │   └── followclient/                   # HTTP client → curiosity-api
│   ├── ws/                                 # WebSocket hub (per-user connection set)
│   └── pkg/apperror/                       # Typed application errors → HTTP codes
└── test/mocks/                             # testify/mock implementations
```

---

## Running tests

```bash
make test
```

Unit tests use hand-written mocks (`test/mocks/`) and do not require a database connection.

---

## Recent Changes & Improvements

### Security Fixes

- **CORS**: Removed wildcard (`*`) origin allowlist. Now requires explicit `ALLOWED_ORIGINS` configuration
- **WebSocket CheckOrigin**: Implemented strict origin validation for WebSocket connections
- **Request Body Limits**: Added 1MB max body size limit to prevent DoS attacks

### Validation Improvements

- **Message Content**: Limited to 1000 characters maximum
- **POI Titles**: Limited to 500 characters maximum  
- **ShareIntent**: Now validates against allowed enum values (`must_go`, `come_with_me`, `invite`, `invite_me`)
- **UUID Validation**: Consolidated all UUID validations into `apperror.ValidateUUID()` helper

### Architecture Refactoring

- **Removed unused dependency**: `CreateConversation` no longer receives unused `followChecker` parameter
- **Fixed null pointer**: `Hub.IsOnline()` now safely handles nil map entries
- **Context management**: Fixed context leak in WebSocket read loop (defer cancel() → explicit cancel())
- **Extracted constants**: All magic numbers moved to named constants (buffer sizes, timeouts, pagination limits)
- **Extracted pagination logic**: Created `response.ParsePagination()` helper to eliminate duplication

### WebSocket Fixes

- **Error handling**: Replaced all `//nolint:errcheck` with proper error handling and logging
- **Context timeout**: WebSocket DB operations now use 10-second timeout
- **Connection safety**: Added `defer r.Body.Close()` in HTTP handlers
- **OtherUserID**: Fixed potential bug where non-participants could get wrong user ID (now returns error)

### Code Quality

- **Constants**: Added `DefaultConversationLimit`, `MaxConversationLimit`, `DefaultMessageLimit`, `MaxMessageLimit`
- **WebSocket constants**: `wsReadBufferSize`, `wsWriteBufferSize`, channel buffer sizes
- **Pagination constants**: Consolidated limit validation across handlers
- **Request body handling**: Added `defer r.Body.Close()` and size limits

### Testing

- Added 19+ new tests covering:
  - Message content length limits (max, over-limit, under-limit)
  - ShareIntent validation (valid and invalid values)
  - POI title length limits
  - Pagination parameter parsing
  - UUID validation helper
  - Hub online status with edge cases
  - OtherUserID error cases

### Configuration

- **New env var**: `ALLOWED_ORIGINS` - Comma-separated list of allowed CORS origins
- **Database**: Added connection max lifetime (1 hour) to prevent stale connections
