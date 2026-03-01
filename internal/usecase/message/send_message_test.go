package message

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
	convRepo.On("UpdateLastMessageAt", ctx, conv.ID, mock.Anything).Return(nil)

	msg, err := uc.Execute(ctx, conv.ID, senderID, SendMessageInput{
		Type:    entity.MessageTypeText,
		Content: "Hello!",
	})

	require.NoError(t, err)
	assert.NotNil(t, msg)
	assert.Equal(t, entity.MessageTypeText, msg.Type)
	assert.Equal(t, "Hello!", *msg.Content)
	msgRepo.AssertExpectations(t)
	convRepo.AssertExpectations(t)
}

func TestSendPOIShareMessageSuccess(t *testing.T) {
	msgRepo := new(mocks.MessageRepositoryMock)
	convRepo := new(mocks.ConversationRepositoryMock)
	uc := NewSendMessage(msgRepo, convRepo)

	ctx := context.Background()
	senderID := uuid.New().String()
	otherID := uuid.New().String()
	poiID := uuid.New().String()
	conv := entity.NewConversation(senderID, otherID)

	convRepo.On("GetByID", ctx, conv.ID).Return(conv, nil)
	msgRepo.On("Create", ctx, mock.AnythingOfType("*entity.Message")).Return(nil)
	convRepo.On("UpdateLastMessageAt", ctx, conv.ID, mock.Anything).Return(nil)

	msg, err := uc.Execute(ctx, conv.ID, senderID, SendMessageInput{
		Type:  entity.MessageTypePOIShare,
		POIID: poiID,
	})

	require.NoError(t, err)
	assert.Equal(t, entity.MessageTypePOIShare, msg.Type)
	assert.Equal(t, poiID, *msg.POIID)
}

func TestSendMessageNotParticipant(t *testing.T) {
	msgRepo := new(mocks.MessageRepositoryMock)
	convRepo := new(mocks.ConversationRepositoryMock)
	uc := NewSendMessage(msgRepo, convRepo)

	ctx := context.Background()
	userA := uuid.New().String()
	userB := uuid.New().String()
	stranger := uuid.New().String()
	conv := entity.NewConversation(userA, userB)

	convRepo.On("GetByID", ctx, conv.ID).Return(conv, nil)

	_, err := uc.Execute(ctx, conv.ID, stranger, SendMessageInput{Type: entity.MessageTypeText, Content: "Hi"})

	require.Error(t, err)
	assert.True(t, apperror.IsForbidden(err))
	msgRepo.AssertNotCalled(t, "Create")
}

func TestSendMessageInvalidType(t *testing.T) {
	msgRepo := new(mocks.MessageRepositoryMock)
	convRepo := new(mocks.ConversationRepositoryMock)
	uc := NewSendMessage(msgRepo, convRepo)

	ctx := context.Background()
	senderID := uuid.New().String()
	otherID := uuid.New().String()
	conv := entity.NewConversation(senderID, otherID)

	convRepo.On("GetByID", ctx, conv.ID).Return(conv, nil)

	_, err := uc.Execute(ctx, conv.ID, senderID, SendMessageInput{Type: "invalid"})

	require.Error(t, err)
	assert.True(t, apperror.IsValidation(err))
}

func TestSendMessageEmptyConversationID(t *testing.T) {
	msgRepo := new(mocks.MessageRepositoryMock)
	convRepo := new(mocks.ConversationRepositoryMock)
	uc := NewSendMessage(msgRepo, convRepo)

	_, err := uc.Execute(context.Background(), "", uuid.New().String(), SendMessageInput{Type: entity.MessageTypeText, Content: "Hi"})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "conversation ID is required")
	convRepo.AssertNotCalled(t, "GetByID")
}

func TestSendTextMessageEmptyContent(t *testing.T) {
	msgRepo := new(mocks.MessageRepositoryMock)
	convRepo := new(mocks.ConversationRepositoryMock)
	uc := NewSendMessage(msgRepo, convRepo)

	ctx := context.Background()
	senderID := uuid.New().String()
	conv := entity.NewConversation(senderID, uuid.New().String())

	convRepo.On("GetByID", ctx, conv.ID).Return(conv, nil)

	_, err := uc.Execute(ctx, conv.ID, senderID, SendMessageInput{Type: entity.MessageTypeText, Content: ""})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "content is required")
}
