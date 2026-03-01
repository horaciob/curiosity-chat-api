package conversation

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/horaciobranciforte/curiosity-chat-api/internal/domain/entity"
	"github.com/horaciobranciforte/curiosity-chat-api/test/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListConversationsSuccess(t *testing.T) {
	repo := new(mocks.ConversationRepositoryMock)
	uc := NewListConversations(repo)

	ctx := context.Background()
	userID := uuid.New().String()
	convs := []*entity.Conversation{entity.NewConversation(userID, uuid.New().String())}

	repo.On("ListByUser", ctx, userID, 20, 0).Return(convs, nil)
	repo.On("CountByUser", ctx, userID).Return(1, nil)

	result, total, err := uc.Execute(ctx, userID, 20, 0)

	require.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, 1, total)
	repo.AssertExpectations(t)
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

func TestListConversationsEmptyUserID(t *testing.T) {
	repo := new(mocks.ConversationRepositoryMock)
	uc := NewListConversations(repo)

	_, _, err := uc.Execute(context.Background(), "", 20, 0)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "user ID is required")
	repo.AssertNotCalled(t, "ListByUser")
}
