package conversation

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

func TestListConversationsSuccess(t *testing.T) {
	repo := new(mocks.ConversationRepositoryMock)
	uc := NewListConversations(repo)

	ctx := context.Background()
	userID := uuid.New().String()
	convs := []*entity.Conversation{
		entity.NewConversation(userID, uuid.New().String()),
		entity.NewConversation(userID, uuid.New().String()),
	}

	repo.On("ListByUser", ctx, userID, 20, 0).Return(convs, nil)
	repo.On("CountByUser", ctx, userID).Return(2, nil)

	result, total, err := uc.Execute(ctx, userID, 20, 0)

	require.NoError(t, err)
	assert.Len(t, result, 2)
	assert.Equal(t, 2, total)
	repo.AssertExpectations(t)
}

func TestListConversationsEmpty(t *testing.T) {
	repo := new(mocks.ConversationRepositoryMock)
	uc := NewListConversations(repo)

	ctx := context.Background()
	userID := uuid.New().String()

	repo.On("ListByUser", ctx, userID, 20, 0).Return([]*entity.Conversation{}, nil)
	repo.On("CountByUser", ctx, userID).Return(0, nil)

	result, total, err := uc.Execute(ctx, userID, 20, 0)

	require.NoError(t, err)
	assert.Empty(t, result)
	assert.Zero(t, total)
}

func TestListConversationsEmptyUserID(t *testing.T) {
	uc := NewListConversations(new(mocks.ConversationRepositoryMock))

	_, _, err := uc.Execute(context.Background(), "", 20, 0)

	require.Error(t, err)
	assert.True(t, apperror.IsValidation(err))
	assert.Contains(t, err.Error(), "user ID is required")
}

func TestListConversationsInvalidUserID(t *testing.T) {
	uc := NewListConversations(new(mocks.ConversationRepositoryMock))

	_, _, err := uc.Execute(context.Background(), "not-a-uuid", 20, 0)

	require.Error(t, err)
	assert.True(t, apperror.IsValidation(err))
	assert.Contains(t, err.Error(), "invalid user ID format")
}

func TestListConversationsDefaultLimit(t *testing.T) {
	repo := new(mocks.ConversationRepositoryMock)
	uc := NewListConversations(repo)

	ctx := context.Background()
	userID := uuid.New().String()

	repo.On("ListByUser", ctx, userID, 20, 0).Return([]*entity.Conversation{}, nil)
	repo.On("CountByUser", ctx, userID).Return(0, nil)

	_, _, err := uc.Execute(ctx, userID, 0, 0)

	require.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestListConversationsLimitCapped(t *testing.T) {
	repo := new(mocks.ConversationRepositoryMock)
	uc := NewListConversations(repo)

	ctx := context.Background()
	userID := uuid.New().String()

	repo.On("ListByUser", ctx, userID, 100, 0).Return([]*entity.Conversation{}, nil)
	repo.On("CountByUser", ctx, userID).Return(0, nil)

	_, _, err := uc.Execute(ctx, userID, 999, 0)

	require.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestListConversationsListFails(t *testing.T) {
	repo := new(mocks.ConversationRepositoryMock)
	uc := NewListConversations(repo)

	ctx := context.Background()
	userID := uuid.New().String()

	repo.On("ListByUser", ctx, userID, 20, 0).Return(nil, errors.New("db error"))

	_, _, err := uc.Execute(ctx, userID, 20, 0)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to list conversations")
	repo.AssertNotCalled(t, "CountByUser")
}

func TestListConversationsCountFails(t *testing.T) {
	repo := new(mocks.ConversationRepositoryMock)
	uc := NewListConversations(repo)

	ctx := context.Background()
	userID := uuid.New().String()

	repo.On("ListByUser", ctx, userID, 20, 0).Return([]*entity.Conversation{}, nil)
	repo.On("CountByUser", ctx, userID).Return(0, errors.New("db error"))

	_, _, err := uc.Execute(ctx, userID, 20, 0)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to count conversations")
}
