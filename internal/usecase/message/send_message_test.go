package message

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/horaciobranciforte/curiosity-chat-api/internal/domain/entity"
	"github.com/horaciobranciforte/curiosity-chat-api/internal/pkg/apperror"
	"github.com/horaciobranciforte/curiosity-chat-api/test/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func makeConv(userA, userB string) *entity.Conversation {
	return entity.NewConversation(userA, userB)
}

func TestSendMessageTextSuccess(t *testing.T) {
	msgRepo := new(mocks.MessageRepositoryMock)
	convRepo := new(mocks.ConversationRepositoryMock)
	uc := NewSendMessage(msgRepo, convRepo)

	ctx := context.Background()
	senderID := uuid.New().String()
	conv := makeConv(senderID, uuid.New().String())

	convRepo.On("GetByID", ctx, conv.ID).Return(conv, nil)
	msgRepo.On("Create", ctx, mock.AnythingOfType("*entity.Message")).Return(nil)
	convRepo.On("UpdateLastMessageAt", ctx, conv.ID, mock.Anything).Return(nil)

	msg, err := uc.Execute(ctx, conv.ID, senderID, SendMessageInput{Type: "text", Content: "hello"})

	require.NoError(t, err)
	require.NotNil(t, msg)
	assert.Equal(t, entity.MessageTypeText, msg.Type)
	assert.Equal(t, "hello", *msg.Content)
	msgRepo.AssertExpectations(t)
	convRepo.AssertExpectations(t)
}

func TestSendMessagePOIShareSuccess(t *testing.T) {
	msgRepo := new(mocks.MessageRepositoryMock)
	convRepo := new(mocks.ConversationRepositoryMock)
	uc := NewSendMessage(msgRepo, convRepo)

	ctx := context.Background()
	senderID := uuid.New().String()
	poiID := uuid.New().String()
	conv := makeConv(senderID, uuid.New().String())

	convRepo.On("GetByID", ctx, conv.ID).Return(conv, nil)
	msgRepo.On("Create", ctx, mock.AnythingOfType("*entity.Message")).Return(nil)
	convRepo.On("UpdateLastMessageAt", ctx, conv.ID, mock.Anything).Return(nil)

	msg, err := uc.Execute(ctx, conv.ID, senderID, SendMessageInput{Type: "poi_share", POIID: poiID})

	require.NoError(t, err)
	assert.Equal(t, entity.MessageTypePOIShare, msg.Type)
	assert.Equal(t, poiID, *msg.POIID)
}

func TestSendMessageEmptyConversationID(t *testing.T) {
	uc := NewSendMessage(new(mocks.MessageRepositoryMock), new(mocks.ConversationRepositoryMock))

	_, err := uc.Execute(context.Background(), "", uuid.New().String(), SendMessageInput{Type: "text", Content: "hi"})

	require.Error(t, err)
	assert.True(t, apperror.IsValidation(err))
	assert.Contains(t, err.Error(), "conversation ID is required")
}

func TestSendMessageInvalidConversationID(t *testing.T) {
	uc := NewSendMessage(new(mocks.MessageRepositoryMock), new(mocks.ConversationRepositoryMock))

	_, err := uc.Execute(context.Background(), "bad-id", uuid.New().String(), SendMessageInput{Type: "text", Content: "hi"})

	require.Error(t, err)
	assert.True(t, apperror.IsValidation(err))
	assert.Contains(t, err.Error(), "invalid conversation ID format")
}

func TestSendMessageEmptySenderID(t *testing.T) {
	uc := NewSendMessage(new(mocks.MessageRepositoryMock), new(mocks.ConversationRepositoryMock))

	_, err := uc.Execute(context.Background(), uuid.New().String(), "", SendMessageInput{Type: "text", Content: "hi"})

	require.Error(t, err)
	assert.True(t, apperror.IsValidation(err))
	assert.Contains(t, err.Error(), "sender ID is required")
}

func TestSendMessageConversationNotFound(t *testing.T) {
	msgRepo := new(mocks.MessageRepositoryMock)
	convRepo := new(mocks.ConversationRepositoryMock)
	uc := NewSendMessage(msgRepo, convRepo)

	ctx := context.Background()
	convID := uuid.New().String()

	convRepo.On("GetByID", ctx, convID).Return(nil, apperror.NotFound("not found", nil))

	_, err := uc.Execute(ctx, convID, uuid.New().String(), SendMessageInput{Type: "text", Content: "hi"})

	require.Error(t, err)
	assert.True(t, apperror.IsNotFound(err))
}

func TestSendMessageNotParticipant(t *testing.T) {
	msgRepo := new(mocks.MessageRepositoryMock)
	convRepo := new(mocks.ConversationRepositoryMock)
	uc := NewSendMessage(msgRepo, convRepo)

	ctx := context.Background()
	conv := makeConv(uuid.New().String(), uuid.New().String())
	outsider := uuid.New().String()

	convRepo.On("GetByID", ctx, conv.ID).Return(conv, nil)

	_, err := uc.Execute(ctx, conv.ID, outsider, SendMessageInput{Type: "text", Content: "hi"})

	require.Error(t, err)
	assert.True(t, apperror.IsForbidden(err))
}

func TestSendMessageEmptyContent(t *testing.T) {
	msgRepo := new(mocks.MessageRepositoryMock)
	convRepo := new(mocks.ConversationRepositoryMock)
	uc := NewSendMessage(msgRepo, convRepo)

	ctx := context.Background()
	senderID := uuid.New().String()
	conv := makeConv(senderID, uuid.New().String())

	convRepo.On("GetByID", ctx, conv.ID).Return(conv, nil)

	_, err := uc.Execute(ctx, conv.ID, senderID, SendMessageInput{Type: "text", Content: ""})

	require.Error(t, err)
	assert.True(t, apperror.IsValidation(err))
	assert.Contains(t, err.Error(), "content is required")
}

func TestSendMessagePOIShareMissingPOIID(t *testing.T) {
	msgRepo := new(mocks.MessageRepositoryMock)
	convRepo := new(mocks.ConversationRepositoryMock)
	uc := NewSendMessage(msgRepo, convRepo)

	ctx := context.Background()
	senderID := uuid.New().String()
	conv := makeConv(senderID, uuid.New().String())

	convRepo.On("GetByID", ctx, conv.ID).Return(conv, nil)

	_, err := uc.Execute(ctx, conv.ID, senderID, SendMessageInput{Type: "poi_share", POIID: ""})

	require.Error(t, err)
	assert.True(t, apperror.IsValidation(err))
	assert.Contains(t, err.Error(), "poi_id is required")
}

func TestSendMessagePOIShareInvalidPOIID(t *testing.T) {
	msgRepo := new(mocks.MessageRepositoryMock)
	convRepo := new(mocks.ConversationRepositoryMock)
	uc := NewSendMessage(msgRepo, convRepo)

	ctx := context.Background()
	senderID := uuid.New().String()
	conv := makeConv(senderID, uuid.New().String())

	convRepo.On("GetByID", ctx, conv.ID).Return(conv, nil)

	_, err := uc.Execute(ctx, conv.ID, senderID, SendMessageInput{Type: "poi_share", POIID: "not-a-uuid"})

	require.Error(t, err)
	assert.True(t, apperror.IsValidation(err))
	assert.Contains(t, err.Error(), "invalid poi_id format")
}

func TestSendMessageInvalidType(t *testing.T) {
	msgRepo := new(mocks.MessageRepositoryMock)
	convRepo := new(mocks.ConversationRepositoryMock)
	uc := NewSendMessage(msgRepo, convRepo)

	ctx := context.Background()
	senderID := uuid.New().String()
	conv := makeConv(senderID, uuid.New().String())

	convRepo.On("GetByID", ctx, conv.ID).Return(conv, nil)

	_, err := uc.Execute(ctx, conv.ID, senderID, SendMessageInput{Type: "video"})

	require.Error(t, err)
	assert.True(t, apperror.IsValidation(err))
	assert.Contains(t, err.Error(), "text")
}

func TestSendMessageSaveFails(t *testing.T) {
	msgRepo := new(mocks.MessageRepositoryMock)
	convRepo := new(mocks.ConversationRepositoryMock)
	uc := NewSendMessage(msgRepo, convRepo)

	ctx := context.Background()
	senderID := uuid.New().String()
	conv := makeConv(senderID, uuid.New().String())

	convRepo.On("GetByID", ctx, conv.ID).Return(conv, nil)
	msgRepo.On("Create", ctx, mock.AnythingOfType("*entity.Message")).Return(errors.New("db error"))

	_, err := uc.Execute(ctx, conv.ID, senderID, SendMessageInput{Type: "text", Content: "hi"})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to save message")
}

func TestSendMessageUpdateLastMessageAtFails(t *testing.T) {
	msgRepo := new(mocks.MessageRepositoryMock)
	convRepo := new(mocks.ConversationRepositoryMock)
	uc := NewSendMessage(msgRepo, convRepo)

	ctx := context.Background()
	senderID := uuid.New().String()
	conv := makeConv(senderID, uuid.New().String())

	convRepo.On("GetByID", ctx, conv.ID).Return(conv, nil)
	msgRepo.On("Create", ctx, mock.AnythingOfType("*entity.Message")).Return(nil)
	convRepo.On("UpdateLastMessageAt", ctx, conv.ID, mock.Anything).Return(errors.New("db error"))

	_, err := uc.Execute(ctx, conv.ID, senderID, SendMessageInput{Type: "text", Content: "hi"})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to update conversation")
}
