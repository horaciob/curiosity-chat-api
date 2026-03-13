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
	"github.com/stretchr/testify/require"
)

func makeMsg(convID, senderID, content string) *entity.Message {
	return entity.NewTextMessage(convID, senderID, content)
}

func TestGetMessagesSuccess(t *testing.T) {
	msgRepo := new(mocks.MessageRepositoryMock)
	convRepo := new(mocks.ConversationRepositoryMock)
	uc := NewGetMessages(msgRepo, convRepo)

	ctx := context.Background()
	requesterID := uuid.New().String()
	conv := makeConv(requesterID, uuid.New().String())
	msgs := []*entity.Message{
		makeMsg(conv.ID, requesterID, "hello"),
		makeMsg(conv.ID, requesterID, "world"),
	}

	convRepo.On("GetByID", ctx, conv.ID).Return(conv, nil)
	msgRepo.On("ListByConversation", ctx, conv.ID, 50, 0).Return(msgs, nil)
	msgRepo.On("CountByConversation", ctx, conv.ID).Return(2, nil)

	result, total, err := uc.Execute(ctx, conv.ID, requesterID, 50, 0)

	require.NoError(t, err)
	assert.Len(t, result, 2)
	assert.Equal(t, 2, total)
	msgRepo.AssertExpectations(t)
}

func TestGetMessagesEmpty(t *testing.T) {
	msgRepo := new(mocks.MessageRepositoryMock)
	convRepo := new(mocks.ConversationRepositoryMock)
	uc := NewGetMessages(msgRepo, convRepo)

	ctx := context.Background()
	requesterID := uuid.New().String()
	conv := makeConv(requesterID, uuid.New().String())

	convRepo.On("GetByID", ctx, conv.ID).Return(conv, nil)
	msgRepo.On("ListByConversation", ctx, conv.ID, 50, 0).Return([]*entity.Message{}, nil)
	msgRepo.On("CountByConversation", ctx, conv.ID).Return(0, nil)

	result, total, err := uc.Execute(ctx, conv.ID, requesterID, 50, 0)

	require.NoError(t, err)
	assert.Empty(t, result)
	assert.Zero(t, total)
}

func TestGetMessagesEmptyConversationID(t *testing.T) {
	uc := NewGetMessages(new(mocks.MessageRepositoryMock), new(mocks.ConversationRepositoryMock))

	_, _, err := uc.Execute(context.Background(), "", uuid.New().String(), 50, 0)

	require.Error(t, err)
	assert.True(t, apperror.IsValidation(err))
	assert.Contains(t, err.Error(), "conversation ID is required")
}

func TestGetMessagesInvalidConversationID(t *testing.T) {
	uc := NewGetMessages(new(mocks.MessageRepositoryMock), new(mocks.ConversationRepositoryMock))

	_, _, err := uc.Execute(context.Background(), "bad-id", uuid.New().String(), 50, 0)

	require.Error(t, err)
	assert.True(t, apperror.IsValidation(err))
	assert.Contains(t, err.Error(), "invalid conversation ID format")
}

func TestGetMessagesEmptyRequesterID(t *testing.T) {
	uc := NewGetMessages(new(mocks.MessageRepositoryMock), new(mocks.ConversationRepositoryMock))

	_, _, err := uc.Execute(context.Background(), uuid.New().String(), "", 50, 0)

	require.Error(t, err)
	assert.True(t, apperror.IsValidation(err))
	assert.Contains(t, err.Error(), "requester ID is required")
}

func TestGetMessagesDefaultLimit(t *testing.T) {
	msgRepo := new(mocks.MessageRepositoryMock)
	convRepo := new(mocks.ConversationRepositoryMock)
	uc := NewGetMessages(msgRepo, convRepo)

	ctx := context.Background()
	requesterID := uuid.New().String()
	conv := makeConv(requesterID, uuid.New().String())

	convRepo.On("GetByID", ctx, conv.ID).Return(conv, nil)
	msgRepo.On("ListByConversation", ctx, conv.ID, 50, 0).Return([]*entity.Message{}, nil)
	msgRepo.On("CountByConversation", ctx, conv.ID).Return(0, nil)

	_, _, err := uc.Execute(ctx, conv.ID, requesterID, 0, 0)

	require.NoError(t, err)
	msgRepo.AssertExpectations(t)
}

func TestGetMessagesLimitCapped(t *testing.T) {
	msgRepo := new(mocks.MessageRepositoryMock)
	convRepo := new(mocks.ConversationRepositoryMock)
	uc := NewGetMessages(msgRepo, convRepo)

	ctx := context.Background()
	requesterID := uuid.New().String()
	conv := makeConv(requesterID, uuid.New().String())

	convRepo.On("GetByID", ctx, conv.ID).Return(conv, nil)
	msgRepo.On("ListByConversation", ctx, conv.ID, 100, 0).Return([]*entity.Message{}, nil)
	msgRepo.On("CountByConversation", ctx, conv.ID).Return(0, nil)

	_, _, err := uc.Execute(ctx, conv.ID, requesterID, 999, 0)

	require.NoError(t, err)
	msgRepo.AssertExpectations(t)
}

func TestGetMessagesConversationNotFound(t *testing.T) {
	msgRepo := new(mocks.MessageRepositoryMock)
	convRepo := new(mocks.ConversationRepositoryMock)
	uc := NewGetMessages(msgRepo, convRepo)

	ctx := context.Background()
	convID := uuid.New().String()

	convRepo.On("GetByID", ctx, convID).Return(nil, apperror.NotFound("not found", nil))

	_, _, err := uc.Execute(ctx, convID, uuid.New().String(), 50, 0)

	require.Error(t, err)
	assert.True(t, apperror.IsNotFound(err))
}

func TestGetMessagesNotParticipant(t *testing.T) {
	msgRepo := new(mocks.MessageRepositoryMock)
	convRepo := new(mocks.ConversationRepositoryMock)
	uc := NewGetMessages(msgRepo, convRepo)

	ctx := context.Background()
	conv := makeConv(uuid.New().String(), uuid.New().String())
	outsider := uuid.New().String()

	convRepo.On("GetByID", ctx, conv.ID).Return(conv, nil)

	_, _, err := uc.Execute(ctx, conv.ID, outsider, 50, 0)

	require.Error(t, err)
	assert.True(t, apperror.IsForbidden(err))
}

func TestGetMessagesListFails(t *testing.T) {
	msgRepo := new(mocks.MessageRepositoryMock)
	convRepo := new(mocks.ConversationRepositoryMock)
	uc := NewGetMessages(msgRepo, convRepo)

	ctx := context.Background()
	requesterID := uuid.New().String()
	conv := makeConv(requesterID, uuid.New().String())

	convRepo.On("GetByID", ctx, conv.ID).Return(conv, nil)
	msgRepo.On("ListByConversation", ctx, conv.ID, 50, 0).Return(nil, errors.New("db error"))

	_, _, err := uc.Execute(ctx, conv.ID, requesterID, 50, 0)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get messages")
	msgRepo.AssertNotCalled(t, "CountByConversation")
}

func TestGetMessagesCountFails(t *testing.T) {
	msgRepo := new(mocks.MessageRepositoryMock)
	convRepo := new(mocks.ConversationRepositoryMock)
	uc := NewGetMessages(msgRepo, convRepo)

	ctx := context.Background()
	requesterID := uuid.New().String()
	conv := makeConv(requesterID, uuid.New().String())

	convRepo.On("GetByID", ctx, conv.ID).Return(conv, nil)
	msgRepo.On("ListByConversation", ctx, conv.ID, 50, 0).Return([]*entity.Message{}, nil)
	msgRepo.On("CountByConversation", ctx, conv.ID).Return(0, errors.New("db error"))

	_, _, err := uc.Execute(ctx, conv.ID, requesterID, 50, 0)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to count messages")
}

func TestGetMessagesInternalError(t *testing.T) {
	msgRepo := new(mocks.MessageRepositoryMock)
	convRepo := new(mocks.ConversationRepositoryMock)
	uc := NewGetMessages(msgRepo, convRepo)

	ctx := context.Background()
	convID := uuid.New().String()
	requesterID := uuid.New().String()

	convRepo.On("GetByID", ctx, convID).Return(nil, errors.New("database error"))

	_, _, err := uc.Execute(ctx, convID, requesterID, 50, 0)

	require.Error(t, err)
	assert.True(t, apperror.IsType(err, apperror.TypeInternal))
	assert.Contains(t, err.Error(), "failed to get conversation")
}
