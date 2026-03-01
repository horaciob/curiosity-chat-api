package conversation

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

func TestGetConversationSuccess(t *testing.T) {
	repo := new(mocks.ConversationRepositoryMock)
	uc := NewGetConversation(repo)

	ctx := context.Background()
	userA := uuid.New().String()
	userB := uuid.New().String()
	conv := entity.NewConversation(userA, userB)

	repo.On("GetByID", ctx, conv.ID).Return(conv, nil)

	result, err := uc.Execute(ctx, conv.ID, userA)

	require.NoError(t, err)
	assert.Equal(t, conv.ID, result.ID)
	repo.AssertExpectations(t)
}

func TestGetConversationNotParticipant(t *testing.T) {
	repo := new(mocks.ConversationRepositoryMock)
	uc := NewGetConversation(repo)

	ctx := context.Background()
	userA := uuid.New().String()
	userB := uuid.New().String()
	stranger := uuid.New().String()
	conv := entity.NewConversation(userA, userB)

	repo.On("GetByID", ctx, conv.ID).Return(conv, nil)

	_, err := uc.Execute(ctx, conv.ID, stranger)

	require.Error(t, err)
	assert.True(t, apperror.IsForbidden(err))
}

func TestGetConversationNotFound(t *testing.T) {
	repo := new(mocks.ConversationRepositoryMock)
	uc := NewGetConversation(repo)

	ctx := context.Background()
	id := uuid.New().String()

	repo.On("GetByID", ctx, id).Return(nil, apperror.NotFound("not found", nil))

	_, err := uc.Execute(ctx, id, uuid.New().String())

	require.Error(t, err)
	assert.True(t, apperror.IsNotFound(err))
}

func TestGetConversationEmptyID(t *testing.T) {
	repo := new(mocks.ConversationRepositoryMock)
	uc := NewGetConversation(repo)

	_, err := uc.Execute(context.Background(), "", uuid.New().String())

	require.Error(t, err)
	assert.Contains(t, err.Error(), "conversation ID is required")
	repo.AssertNotCalled(t, "GetByID")
}
