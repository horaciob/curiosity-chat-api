# Architecture Patterns

## Entity Pattern

Entities have identity and behavior. No external dependencies.

### Conversation

```go
type Conversation struct {
    ID            string
    User1ID       string     // always the lexicographically smaller UUID
    User2ID       string
    CreatedAt     time.Time
    LastMessageAt *time.Time
}

// NewConversation normalizes so user1 < user2 regardless of call order.
func NewConversation(userA, userB string) *Conversation {
    u1, u2 := userA, userB
    if u1 > u2 {
        u1, u2 = u2, u1
    }
    return &Conversation{
        ID:        uuid.New().String(),
        User1ID:   u1,
        User2ID:   u2,
        CreatedAt: time.Now(),
    }
}

func (c *Conversation) HasParticipant(userID string) bool {
    return c.User1ID == userID || c.User2ID == userID
}

func (c *Conversation) OtherUserID(myID string) string {
    if c.User1ID == myID {
        return c.User2ID
    }
    return c.User1ID
}
```

### Message

```go
const (
    MessageTypeText     = "text"
    MessageTypePOIShare = "poi_share"
)

type Message struct {
    ID             string
    ConversationID string
    SenderID       string
    Type           string
    Content        *string   // non-nil for type=text
    POIID          *string   // non-nil for type=poi_share
    CreatedAt      time.Time
}

func NewTextMessage(conversationID, senderID, content string) *Message {
    return &Message{
        ID:             uuid.New().String(),
        ConversationID: conversationID,
        SenderID:       senderID,
        Type:           MessageTypeText,
        Content:        &content,
        CreatedAt:      time.Now(),
    }
}

func NewPOIShareMessage(conversationID, senderID, poiID string) *Message {
    return &Message{
        ID:             uuid.New().String(),
        ConversationID: conversationID,
        SenderID:       senderID,
        Type:           MessageTypePOIShare,
        POIID:          &poiID,
        CreatedAt:      time.Now(),
    }
}
```

---

## Use Case Pattern

One struct per use case, constructor injects repository interfaces, single `Execute` method.

```go
// interface.go — defined in same package as use case
type Repository interface {
    Create(ctx context.Context, c *entity.Conversation) error
    GetByID(ctx context.Context, id string) (*entity.Conversation, error)
    GetByParticipants(ctx context.Context, userA, userB string) (*entity.Conversation, error)
    ListByUser(ctx context.Context, userID string, limit, offset int) ([]*entity.Conversation, error)
    CountByUser(ctx context.Context, userID string) (int, error)
    UpdateLastMessageAt(ctx context.Context, id string, t time.Time) error
}

type FollowChecker interface {
    AreFollowing(ctx context.Context, userA, userB string) (bool, error)
}

// create_conversation.go
type CreateConversation struct {
    repo          Repository
    followChecker FollowChecker
}

func NewCreateConversation(repo Repository, followChecker FollowChecker) *CreateConversation {
    return &CreateConversation{repo: repo, followChecker: followChecker}
}

func (uc *CreateConversation) Execute(ctx context.Context, requesterID, targetID string) (*entity.Conversation, error) {
    // 1. Validate inputs
    if requesterID == "" {
        return nil, apperror.Validation("requester ID is required", nil)
    }
    if _, err := uuid.Parse(requesterID); err != nil {
        return nil, apperror.Validation("invalid requester ID format", err)
    }
    // ...

    // 2. Business rules
    if requesterID == targetID {
        return nil, apperror.Validation("cannot start a conversation with yourself", domerrors.ErrSelfConversation)
    }

    // 3. Check follow relationship
    following, err := uc.followChecker.AreFollowing(ctx, requesterID, targetID)
    if err != nil {
        return nil, apperror.Internal("failed to check follow relationship", err)
    }
    if !following {
        return nil, apperror.Forbidden("users must follow each other to chat", domerrors.ErrUsersCannotChat)
    }

    // 4. Return existing if found
    existing, err := uc.repo.GetByParticipants(ctx, requesterID, targetID)
    if err == nil {
        return existing, nil
    }
    if !apperror.IsNotFound(err) {
        return nil, apperror.Internal("failed to check existing conversation", err)
    }

    // 5. Create new
    conv := entity.NewConversation(requesterID, targetID)
    if err := uc.repo.Create(ctx, conv); err != nil {
        return nil, apperror.Internal("failed to create conversation", err)
    }
    return conv, nil
}
```

---

## Repository Pattern

Internal `xxxRow` struct for DB mapping, `toEntity()` converter. `ConversationRepository` implements both `conversation.Repository` and `message.ConversationRepository`.

```go
type ConversationRepository struct {
    db *sqlx.DB
}

type conversationRow struct {
    ID            string     `db:"id"`
    User1ID       string     `db:"user1_id"`
    User2ID       string     `db:"user2_id"`
    CreatedAt     time.Time  `db:"created_at"`
    LastMessageAt *time.Time `db:"last_message_at"`
}

func (r conversationRow) toEntity() *entity.Conversation {
    return &entity.Conversation{
        ID:            r.ID,
        User1ID:       r.User1ID,
        User2ID:       r.User2ID,
        CreatedAt:     r.CreatedAt,
        LastMessageAt: r.LastMessageAt,
    }
}

func (r *ConversationRepository) GetByID(ctx context.Context, id string) (*entity.Conversation, error) {
    q := `SELECT id, user1_id, user2_id, created_at, last_message_at
          FROM conversations WHERE id = $1`
    var row conversationRow
    if err := r.db.GetContext(ctx, &row, q, id); err != nil {
        if errors.Is(err, sql.ErrNoRows) {
            return nil, apperror.NotFound("conversation not found", domerrors.ErrConversationNotFound)
        }
        return nil, err
    }
    return row.toEntity(), nil
}
```

---

## Handler Pattern

Handlers inject use cases directly (not interfaces). Read `userID` from context via `middleware.UserIDFromContext`.

```go
type ConversationHandler struct {
    createConversationUC *conversation.CreateConversation
    getConversationUC    *conversation.GetConversation
    listConversationsUC  *conversation.ListConversations
}

func (h *ConversationHandler) Create(w http.ResponseWriter, r *http.Request) {
    requesterID := middleware.UserIDFromContext(r.Context())

    var req createConversationRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        response.Error(w, http.StatusBadRequest, "Bad Request", "invalid request body")
        return
    }

    conv, err := h.createConversationUC.Execute(r.Context(), requesterID, req.TargetUserID)
    if err != nil {
        handleUseCaseError(w, err)  // maps *apperror.Error → HTTP status
        return
    }

    response.Created(w, response.NewConversationResponse(conv))
}
```

---

## WebSocket Handler Pattern

Auth via first frame (never query params). `authenticate()` sets a read deadline then reads exactly one frame.

```go
func (h *WSHandler) ServeWS(w http.ResponseWriter, r *http.Request) {
    conn, _ := upgrader.Upgrade(w, r, nil)

    userID, err := h.authenticate(conn)
    if err != nil {
        conn.WriteMessage(websocket.CloseMessage,
            websocket.FormatCloseMessage(websocket.ClosePolicyViolation, "authentication failed"))
        conn.Close()
        return
    }

    client := &ws.Client{UserID: userID, Conn: conn, Send: make(chan []byte, 256)}
    h.hub.Register(client)
    go client.WritePump()
    h.readPump(client)
}

func (h *WSHandler) authenticate(conn *websocket.Conn) (string, error) {
    conn.SetReadDeadline(time.Now().Add(ws.AuthDeadline))
    defer conn.SetReadDeadline(time.Time{})

    _, data, err := conn.ReadMessage()
    // ... unmarshal, check type=="auth", validate token
    return userID, nil
}
```
