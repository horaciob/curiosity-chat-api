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
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestCreateConversationSuccess(t *testing.T) {
	repo := new(mocks.ConversationRepositoryMock)
	checker := new(mocks.FollowCheckerMock)
	uc := NewCreateConversation(repo, checker)

	ctx := context.Background()
	userA := uuid.New().String()
	userB := uuid.New().String()

	checker.On("AreFollowing", ctx, userA, userB).Return(true, nil)
	repo.On("GetByParticipants", ctx, mock.Anything, mock.Anything).Return(nil, apperror.NotFound("not found", nil))
	repo.On("Create", ctx, mock.AnythingOfType("*entity.Conversation")).Return(nil)

	conv, err := uc.Execute(ctx, userA, userB)

	require.NoError(t, err)
	assert.NotNil(t, conv)
	assert.NotEmpty(t, conv.ID)
	repo.AssertExpectations(t)
	checker.AssertExpectations(t)
}

func TestCreateConversationReturnsExisting(t *testing.T) {
	repo := new(mocks.ConversationRepositoryMock)
	checker := new(mocks.FollowCheckerMock)
	uc := NewCreateConversation(repo, checker)

	ctx := context.Background()
	userA := uuid.New().String()
	userB := uuid.New().String()
	existing := entity.NewConversation(userA, userB)

	checker.On("AreFollowing", ctx, userA, userB).Return(true, nil)
	repo.On("GetByParticipants", ctx, mock.Anything, mock.Anything).Return(existing, nil)

	conv, err := uc.Execute(ctx, userA, userB)

	require.NoError(t, err)
	assert.Equal(t, existing.ID, conv.ID)
	repo.AssertNotCalled(t, "Create")
}

func TestCreateConversationEmptyRequesterID(t *testing.T) {
	repo := new(mocks.ConversationRepositoryMock)
	checker := new(mocks.FollowCheckerMock)
	uc := NewCreateConversation(repo, checker)

	_, err := uc.Execute(context.Background(), "", uuid.New().String())

	require.Error(t, err)
	assert.Contains(t, err.Error(), "requester ID is required")
	checker.AssertNotCalled(t, "AreFollowing")
}

func TestCreateConversationSelfConversation(t *testing.T) {
	repo := new(mocks.ConversationRepositoryMock)
	checker := new(mocks.FollowCheckerMock)
	uc := NewCreateConversation(repo, checker)

	id := uuid.New().String()
	_, err := uc.Execute(context.Background(), id, id)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot start a conversation with yourself")
	checker.AssertNotCalled(t, "AreFollowing")
}

func TestCreateConversationNotFollowing(t *testing.T) {
	repo := new(mocks.ConversationRepositoryMock)
	checker := new(mocks.FollowCheckerMock)
	uc := NewCreateConversation(repo, checker)

	ctx := context.Background()
	userA := uuid.New().String()
	userB := uuid.New().String()

	checker.On("AreFollowing", ctx, userA, userB).Return(false, nil)

	_, err := uc.Execute(ctx, userA, userB)

	require.Error(t, err)
	assert.True(t, apperror.IsForbidden(err))
	repo.AssertNotCalled(t, "Create")
}

func TestCreateConversationFollowCheckerError(t *testing.T) {
	repo := new(mocks.ConversationRepositoryMock)
	checker := new(mocks.FollowCheckerMock)
	uc := NewCreateConversation(repo, checker)

	ctx := context.Background()
	userA := uuid.New().String()
	userB := uuid.New().String()

	checker.On("AreFollowing", ctx, userA, userB).Return(false, errors.New("service unavailable"))

	_, err := uc.Execute(ctx, userA, userB)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to check follow relationship")
}
