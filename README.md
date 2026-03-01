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
