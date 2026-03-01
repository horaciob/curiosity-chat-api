# curiosity-chat-api

Real-time 1-on-1 chat microservice for the Curiosity platform. Users who mutually follow each other can exchange text messages and share Points of Interest (POIs) via WebSockets, with full PostgreSQL persistence.

## Overview

- **Language**: Go
- **Architecture**: Clean Architecture (entity → use case → repository → HTTP adapter)
- **Transport**: REST (JSON:API) + WebSockets (gorilla/websocket)
- **Database**: PostgreSQL 15
- **Port**: `8081` (curiosity-api runs on `8080`)
- **Auth**: JWT — same token issued by `curiosity-api` (shared `JWT_SECRET`)

## Prerequisites

- Go 1.21+
- Docker & Docker Compose
- [golang-migrate](https://github.com/golang-migrate/migrate) CLI

```bash
brew install golang-migrate
```

## Getting started

```bash
# 1. Copy env file and fill in JWT_SECRET
cp .env.example .env

# 2. Start PostgreSQL and run migrations
make db-setup

# 3. Run the server
make run
```

Server starts at `http://localhost:8081`.

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

## API endpoints

All data endpoints use [JSON:API](https://jsonapi.org/) format. The WebSocket and health endpoints use plain JSON.

| Method | Path | Auth | Description |
|---|---|---|---|
| `GET` | `/api/v1/health` | No | Health check |
| `GET` | `/api/v1/ws?token=<jwt>` | Via query param | WebSocket connection |
| `POST` | `/api/v1/conversations` | Bearer | Create or get existing conversation |
| `GET` | `/api/v1/conversations` | Bearer | List my conversations (paginated) |
| `GET` | `/api/v1/conversations/{id}` | Bearer | Get a conversation |
| `GET` | `/api/v1/conversations/{id}/messages` | Bearer | Message history (newest first, paginated) |
| `POST` | `/api/v1/conversations/{id}/messages` | Bearer | Send a message (HTTP fallback) |

### Create a conversation

```bash
POST /api/v1/conversations
Authorization: Bearer <jwt>

{ "target_user_id": "<uuid>" }
```

Returns `201 Created` with the conversation (or `200` if it already exists). Both users must mutually follow each other in `curiosity-api`.

### Send a text message

```bash
POST /api/v1/conversations/{id}/messages
Authorization: Bearer <jwt>

{ "type": "text", "content": "Hello!" }
```

### Share a POI

```bash
POST /api/v1/conversations/{id}/messages
Authorization: Bearer <jwt>

{ "type": "poi_share", "poi_id": "<uuid>" }
```

## WebSocket

Connect with a valid JWT token as a query parameter:

```
ws://localhost:8081/api/v1/ws?token=<jwt>
```

### Client → Server (send a message)

```json
{
  "type": "text",
  "conversation_id": "<uuid>",
  "content": "Hello!"
}
```

```json
{
  "type": "poi_share",
  "conversation_id": "<uuid>",
  "poi_id": "<uuid>"
}
```

### Server → Client (delivery to both sender and recipient)

```json
{
  "id": "<uuid>",
  "type": "text",
  "conversation_id": "<uuid>",
  "sender_id": "<uuid>",
  "content": "Hello!",
  "created_at": "2025-01-01T12:00:00Z"
}
```

## Makefile targets

```bash
make run              # Build and run
make build            # Compile binary to bin/
make test             # Run all tests
make coverage         # Generate HTML coverage report

make db-setup         # Start postgres + migrate
make db-setup-test    # Start test postgres + migrate
make db-teardown      # Stop and remove containers
make db-reset         # Full teardown + setup

make migrate-up       # Run pending migrations
make migrate-down     # Rollback last migration
make migrate-create NAME=my_migration
```

## Database schema

```
conversations
  id            UUID PK
  user1_id      UUID   -- always the lexicographically smaller UUID
  user2_id      UUID
  created_at    TIMESTAMPTZ
  last_message_at TIMESTAMPTZ

messages
  id              UUID PK
  conversation_id UUID FK → conversations(id) ON DELETE CASCADE
  sender_id       UUID
  type            VARCHAR(20)  -- 'text' | 'poi_share'
  content         TEXT         -- set when type='text'
  poi_id          UUID         -- set when type='poi_share'
  created_at      TIMESTAMPTZ
```

## Project structure

```
.
├── cmd/api/main.go                         # Wiring & server startup
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
│   │       └── router/                     # Chi router wiring
│   ├── infrastructure/
│   │   ├── auth/                           # JWT validation
│   │   ├── config/                         # Env-based config
│   │   ├── database/                       # sqlx connection pool
│   │   └── followclient/                   # HTTP client → curiosity-api
│   ├── ws/                                 # WebSocket hub
│   └── pkg/apperror/                       # Typed application errors
└── test/mocks/                             # testify mock implementations
```

## Running tests

```bash
make test
```

Unit tests use hand-written mocks (`test/mocks/`) and do not require a database.
