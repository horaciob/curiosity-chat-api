# API Endpoints Reference

Base URL: `http://localhost:8081`
Format: JSON:API for data endpoints, plain JSON for health/ws.
Auth: `Authorization: Bearer <jwt>` — token issued by curiosity-api (shared JWT_SECRET).

---

## Health

### GET /api/v1/health

```
Response 200:
{ "status": "ok" }
```

---

## Conversations

### POST /api/v1/conversations
Create a conversation between the authenticated user and a target. Returns existing if already present. Both users must mutually follow each other in curiosity-api.

```
Request:
Authorization: Bearer <jwt>
Content-Type: application/json

{ "target_user_id": "<uuid>" }

Response 201 (created) / 200 (existing):
{
  "data": {
    "id": "<uuid>",
    "type": "conversations",
    "attributes": {
      "user1_id": "<uuid>",
      "user2_id": "<uuid>",
      "created_at": "2025-01-01T12:00:00Z",
      "last_message_at": null
    }
  }
}

Errors:
  400 — missing/invalid target_user_id or self-conversation
  401 — missing/invalid token
  403 — users do not mutually follow each other
  500 — internal error
```

### GET /api/v1/conversations
List conversations for the authenticated user, ordered by most recent activity.

```
Request:
Authorization: Bearer <jwt>

Query params:
  page[limit]   int  (default 20, max 100)
  page[offset]  int  (default 0)

Response 200:
{
  "data": [ { "id": "...", "type": "conversations", "attributes": { ... } } ],
  "meta": { "total": 5 },
  "links": { "self": "/api/v1/conversations?page[limit]=20&page[offset]=0" }
}
```

### GET /api/v1/conversations/{id}
Get a single conversation. Requester must be a participant.

```
Response 200: single conversation object (same shape as POST response)
Errors:
  401 — unauthorized
  403 — not a participant
  404 — conversation not found
```

---

## Messages

### POST /api/v1/conversations/{id}/messages
Send a message (HTTP fallback — WebSocket preferred for real-time).

```
Request:
Authorization: Bearer <jwt>
Content-Type: application/json

Text message:
{ "type": "text", "content": "Hello!" }

POI share:
{ "type": "poi_share", "poi_id": "<uuid>" }

Response 201:
{
  "data": {
    "id": "<uuid>",
    "type": "messages",
    "attributes": {
      "conversation_id": "<uuid>",
      "sender_id": "<uuid>",
      "type": "text",
      "content": "Hello!",
      "poi_id": null,
      "created_at": "2025-01-01T12:00:00Z"
    }
  }
}

Errors:
  400 — invalid type, missing content/poi_id
  401 — unauthorized
  403 — not a participant
  404 — conversation not found
```

### GET /api/v1/conversations/{id}/messages
Paginated message history, newest first.

```
Query params:
  page[limit]   int  (default 50, max 100)
  page[offset]  int  (default 0)

Response 200:
{
  "data": [ { message objects... } ],
  "meta": { "total": 42 },
  "links": { "self": "/api/v1/conversations/<id>/messages?page[limit]=50&page[offset]=0" }
}
```

---

## WebSocket

### GET /api/v1/ws
Upgrade to WebSocket. Auth via first application frame (token never in URL).

**Protocol:**
```
1. Connect
   ws://localhost:8081/api/v1/ws

2. Client → Server (auth frame, within 10 seconds)
   { "type": "auth", "token": "<jwt>" }

3. Server → Client (success)
   { "type": "auth_ok", "user_id": "<uuid>" }
   — or close(1008) on failure/timeout

4. Client → Server (text message)
   {
     "type": "text",
     "conversation_id": "<uuid>",
     "content": "Hello!"
   }

5. Client → Server (poi share)
   {
     "type": "poi_share",
     "conversation_id": "<uuid>",
     "poi_id": "<uuid>"
   }

6. Server → Client (broadcast to sender + recipient)
   {
     "id": "<uuid>",
     "type": "text",
     "conversation_id": "<uuid>",
     "sender_id": "<uuid>",
     "content": "Hello!",
     "created_at": "2025-01-01T12:00:00Z"
   }
```

---

## Error Response Format

```json
{
  "error": "Not Found",
  "message": "conversation not found",
  "code": 404
}
```
