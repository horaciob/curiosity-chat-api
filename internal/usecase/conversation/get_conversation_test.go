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
}

func TestGetConversationEmptyConversationID(t *testing.T) {
	uc := NewGetConversation(new(mocks.ConversationRepositoryMock))

	_, err := uc.Execute(context.Background(), "", uuid.New().String())

	require.Error(t, err)
	assert.True(t, apperror.IsValidation(err))
	assert.Contains(t, err.Error(), "conversation ID is required")
}

func TestGetConversationInvalidConversationID(t *testing.T) {
	uc := NewGetConversation(new(mocks.ConversationRepositoryMock))

	_, err := uc.Execute(context.Background(), "not-a-uuid", uuid.New().String())

	require.Error(t, err)
	assert.True(t, apperror.IsValidation(err))
	assert.Contains(t, err.Error(), "invalid conversation ID format")
}

func TestGetConversationEmptyRequesterID(t *testing.T) {
	uc := NewGetConversation(new(mocks.ConversationRepositoryMock))

	_, err := uc.Execute(context.Background(), uuid.New().String(), "")

	require.Error(t, err)
	assert.True(t, apperror.IsValidation(err))
	assert.Contains(t, err.Error(), "requester ID is required")
}

func TestGetConversationNotFound(t *testing.T) {
	repo := new(mocks.ConversationRepositoryMock)
	uc := NewGetConversation(repo)

	ctx := context.Background()
	convID := uuid.New().String()

	repo.On("GetByID", ctx, convID).Return(nil, apperror.NotFound("not found", nil))

	_, err := uc.Execute(ctx, convID, uuid.New().String())

	require.Error(t, err)
	assert.True(t, apperror.IsNotFound(err))
}

func TestGetConversationNotParticipant(t *testing.T) {
	repo := new(mocks.ConversationRepositoryMock)
	uc := NewGetConversation(repo)

	ctx := context.Background()
	userA := uuid.New().String()
	userB := uuid.New().String()
	outsider := uuid.New().String()
	conv := entity.NewConversation(userA, userB)

	repo.On("GetByID", ctx, conv.ID).Return(conv, nil)

	_, err := uc.Execute(ctx, conv.ID, outsider)

	require.Error(t, err)
	assert.True(t, apperror.IsForbidden(err))
}

func TestGetConversationRepoFails(t *testing.T) {
	repo := new(mocks.ConversationRepositoryMock)
	uc := NewGetConversation(repo)

	ctx := context.Background()
	convID := uuid.New().String()

	repo.On("GetByID", ctx, convID).Return(nil, errors.New("db error"))

	_, err := uc.Execute(ctx, convID, uuid.New().String())

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get conversation")
}
