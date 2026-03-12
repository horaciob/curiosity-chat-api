# Testing Patterns

## Test Naming Convention

**MANDATORY: CamelCase without underscores.**

Pattern: `Test<Subject><Scenario>`

```go
// Use case tests
func TestCreateConversationSuccess(t *testing.T) {}
func TestCreateConversationSelfConversation(t *testing.T) {}
func TestCreateConversationNotFollowing(t *testing.T) {}
func TestSendTextMessageSuccess(t *testing.T) {}
func TestSendMessageNotParticipant(t *testing.T) {}
func TestGetMessagesDefaultLimit(t *testing.T) {}

// NEVER use underscores
// func TestCreateConversation_Success(t *testing.T) {}  ← WRONG
```

---

## Mock Pattern

Mocks are hand-written in `test/mocks/` using `testify/mock`.

```go
// test/mocks/conversation_repository_mock.go
package mocks

import (
    "context"
    "time"

    "github.com/horaciobranciforte/curiosity-chat-api/internal/domain/entity"
    "github.com/stretchr/testify/mock"
)

type ConversationRepositoryMock struct {
    mock.Mock
}

func (m *ConversationRepositoryMock) GetByID(ctx context.Context, id string) (*entity.Conversation, error) {
    args := m.Called(ctx, id)
    if args.Get(0) == nil {
        return nil, args.Error(1)
    }
    return args.Get(0).(*entity.Conversation), args.Error(1)
}

func (m *ConversationRepositoryMock) UpdateLastMessageAt(ctx context.Context, id string, t time.Time) error {
    args := m.Called(ctx, id, t)
    return args.Error(0)
}
// ... other methods
```

---

## Unit Test — Use Case

```go
package conversation

import (
    "context"
    "testing"

    "github.com/google/uuid"
    "github.com/horaciobranciforte/curiosity-chat-api/internal/domain/entity"
    "github.com/horaciobranciforte/curiosity-chat-api/internal/pkg/apperror"
    "github.com/horaciobranciforte/curiosity-chat-api/test/mocks"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/mock"
    "github.com/stretchr/testify/require"
)

func TestCreateConversationSuccess(t *testing.T) {
    convRepo := new(mocks.ConversationRepositoryMock)
    followChecker := new(mocks.FollowCheckerMock)
    uc := NewCreateConversation(convRepo, followChecker)

    ctx := context.Background()
    requesterID := uuid.New().String()
    targetID := uuid.New().String()

    followChecker.On("AreFollowing", ctx, requesterID, targetID).Return(true, nil)
    convRepo.On("GetByParticipants", ctx, requesterID, targetID).
        Return(nil, apperror.NotFound("not found", nil))
    convRepo.On("Create", ctx, mock.AnythingOfType("*entity.Conversation")).Return(nil)

    conv, err := uc.Execute(ctx, requesterID, targetID)

    require.NoError(t, err)
    assert.NotEmpty(t, conv.ID)
    convRepo.AssertExpectations(t)
    followChecker.AssertExpectations(t)
}

func TestCreateConversationSelfConversation(t *testing.T) {
    uc := NewCreateConversation(
        new(mocks.ConversationRepositoryMock),
        new(mocks.FollowCheckerMock),
    )
    id := uuid.New().String()

    _, err := uc.Execute(context.Background(), id, id)

    require.Error(t, err)
    assert.True(t, apperror.IsValidation(err) || apperror.IsForbidden(err))
}
```

## Unit Test — Send Message

```go
func TestSendTextMessageSuccess(t *testing.T) {
    msgRepo := new(mocks.MessageRepositoryMock)
    convRepo := new(mocks.ConversationRepositoryMock)
    uc := NewSendMessage(msgRepo, convRepo)

    ctx := context.Background()
    senderID := uuid.New().String()
    otherID := uuid.New().String()
    conv := entity.NewConversation(senderID, otherID)

    convRepo.On("GetByID", ctx, conv.ID).Return(conv, nil)
    msgRepo.On("Create", ctx, mock.AnythingOfType("*entity.Message")).Return(nil)
    convRepo.On("UpdateLastMessageAt", ctx, conv.ID, mock.AnythingOfType("time.Time")).Return(nil)

    msg, err := uc.Execute(ctx, conv.ID, senderID, SendMessageInput{
        Type:    "text",
        Content: "Hello!",
    })

    require.NoError(t, err)
    assert.Equal(t, "text", msg.Type)
    assert.Equal(t, senderID, msg.SenderID)
    require.NotNil(t, msg.Content)
    assert.Equal(t, "Hello!", *msg.Content)
    msgRepo.AssertExpectations(t)
    convRepo.AssertExpectations(t)
}
```

---

## Common Assertions

```go
// Check error type
assert.True(t, apperror.IsNotFound(err))
assert.True(t, apperror.IsForbidden(err))
assert.True(t, apperror.IsValidation(err))

// Check error message
assert.Contains(t, err.Error(), "user ID is required")

// Verify mock not called
mockRepo.AssertNotCalled(t, "Create")

// Verify all expectations met
mockRepo.AssertExpectations(t)

// Match any value of a type
mock.AnythingOfType("*entity.Message")
mock.AnythingOfType("time.Time")
mock.Anything  // any value
```
