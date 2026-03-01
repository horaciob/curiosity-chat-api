package message

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/horaciobranciforte/curiosity-chat-api/internal/domain/entity"
	"github.com/horaciobranciforte/curiosity-chat-api/internal/pkg/apperror"
	"github.com/horaciobranciforte/curiosity-chat-api/test/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetMessagesSuccess(t *testing.T) {
	msgRepo := new(mocks.MessageRepositoryMock)
	convRepo := new(mocks.ConversationRepositoryMock)
	uc := NewGetMessages(msgRepo, convRepo)

	ctx := context.Background()
	senderID := uuid.New().String()
	otherID := uuid.New().String()
	conv := entity.NewConversation(senderID, otherID)

	msgs := []*entity.Message{entity.NewTextMessage(conv.ID, senderID, "Hello")}

	convRepo.On("GetByID", ctx, conv.ID).Return(conv, nil)
	msgRepo.On("ListByConversation", ctx, conv.ID, 50, 0).Return(msgs, nil)
	msgRepo.On("CountByConversation", ctx, conv.ID).Return(1, nil)

	result, total, err := uc.Execute(ctx, conv.ID, senderID, 50, 0)

	require.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, 1, total)
	msgRepo.AssertExpectations(t)
}

func TestGetMessagesNotParticipant(t *testing.T) {
	msgRepo := new(mocks.MessageRepositoryMock)
	convRepo := new(mocks.ConversationRepositoryMock)
	uc := NewGetMessages(msgRepo, convRepo)

	ctx := context.Background()
	userA := uuid.New().String()
	userB := uuid.New().String()
	stranger := uuid.New().String()
	conv := entity.NewConversation(userA, userB)

	convRepo.On("GetByID", ctx, conv.ID).Return(conv, nil)

	_, _, err := uc.Execute(ctx, conv.ID, stranger, 50, 0)

	require.Error(t, err)
	assert.True(t, apperror.IsForbidden(err))
	msgRepo.AssertNotCalled(t, "ListByConversation")
}

func TestGetMessagesDefaultLimit(t *testing.T) {
	msgRepo := new(mocks.MessageRepositoryMock)
	convRepo := new(mocks.ConversationRepositoryMock)
	uc := NewGetMessages(msgRepo, convRepo)

	ctx := context.Background()
	senderID := uuid.New().String()
	conv := entity.NewConversation(senderID, uuid.New().String())

	convRepo.On("GetByID", ctx, conv.ID).Return(conv, nil)
	msgRepo.On("ListByConversation", ctx, conv.ID, 50, 0).Return([]*entity.Message{}, nil)
	msgRepo.On("CountByConversation", ctx, conv.ID).Return(0, nil)

	_, _, err := uc.Execute(ctx, conv.ID, senderID, 0, 0)

	require.NoError(t, err)
	msgRepo.AssertExpectations(t)
}

func TestGetMessagesEmptyConversationID(t *testing.T) {
	msgRepo := new(mocks.MessageRepositoryMock)
	convRepo := new(mocks.ConversationRepositoryMock)
	uc := NewGetMessages(msgRepo, convRepo)

	_, _, err := uc.Execute(context.Background(), "", uuid.New().String(), 50, 0)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "conversation ID is required")
}
