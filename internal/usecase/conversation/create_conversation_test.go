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
	uc := NewCreateConversation(repo)

	ctx := context.Background()
	userA := uuid.New().String()
	userB := uuid.New().String()

	repo.On("GetByParticipants", ctx, userA, userB).Return(nil, apperror.NotFound("not found", nil))
	repo.On("Create", ctx, mock.AnythingOfType("*entity.Conversation")).Return(nil)

	conv, err := uc.Execute(ctx, userA, userB)

	require.NoError(t, err)
	require.NotNil(t, conv)
	assert.NotEmpty(t, conv.ID)
	repo.AssertExpectations(t)
}

func TestCreateConversationReturnsExisting(t *testing.T) {
	repo := new(mocks.ConversationRepositoryMock)
	uc := NewCreateConversation(repo)

	ctx := context.Background()
	userA := uuid.New().String()
	userB := uuid.New().String()
	existing := entity.NewConversation(userA, userB)

	repo.On("GetByParticipants", ctx, userA, userB).Return(existing, nil)

	conv, err := uc.Execute(ctx, userA, userB)

	require.NoError(t, err)
	assert.Equal(t, existing.ID, conv.ID)
	repo.AssertNotCalled(t, "Create")
}

func TestCreateConversationEmptyRequesterID(t *testing.T) {
	uc := NewCreateConversation(new(mocks.ConversationRepositoryMock))

	_, err := uc.Execute(context.Background(), "", uuid.New().String())

	require.Error(t, err)
	assert.True(t, apperror.IsValidation(err))
	assert.Contains(t, err.Error(), "requester ID is required")
}

func TestCreateConversationInvalidRequesterID(t *testing.T) {
	uc := NewCreateConversation(new(mocks.ConversationRepositoryMock))

	_, err := uc.Execute(context.Background(), "not-a-uuid", uuid.New().String())

	require.Error(t, err)
	assert.True(t, apperror.IsValidation(err))
	assert.Contains(t, err.Error(), "invalid requester ID format")
}

func TestCreateConversationEmptyTargetID(t *testing.T) {
	uc := NewCreateConversation(new(mocks.ConversationRepositoryMock))

	_, err := uc.Execute(context.Background(), uuid.New().String(), "")

	require.Error(t, err)
	assert.True(t, apperror.IsValidation(err))
	assert.Contains(t, err.Error(), "target user ID is required")
}

func TestCreateConversationInvalidTargetID(t *testing.T) {
	uc := NewCreateConversation(new(mocks.ConversationRepositoryMock))

	_, err := uc.Execute(context.Background(), uuid.New().String(), "not-a-uuid")

	require.Error(t, err)
	assert.True(t, apperror.IsValidation(err))
	assert.Contains(t, err.Error(), "invalid target user ID format")
}

func TestCreateConversationSelfConversation(t *testing.T) {
	uc := NewCreateConversation(new(mocks.ConversationRepositoryMock))
	id := uuid.New().String()

	_, err := uc.Execute(context.Background(), id, id)

	require.Error(t, err)
	assert.True(t, apperror.IsValidation(err))
	assert.Contains(t, err.Error(), "yourself")
}

func TestCreateConversationGetByParticipantsFails(t *testing.T) {
	repo := new(mocks.ConversationRepositoryMock)
	uc := NewCreateConversation(repo)

	ctx := context.Background()
	userA := uuid.New().String()
	userB := uuid.New().String()

	repo.On("GetByParticipants", ctx, userA, userB).Return(nil, errors.New("db error"))

	_, err := uc.Execute(ctx, userA, userB)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to check existing conversation")
}

func TestCreateConversationCreateFails(t *testing.T) {
	repo := new(mocks.ConversationRepositoryMock)
	uc := NewCreateConversation(repo)

	ctx := context.Background()
	userA := uuid.New().String()
	userB := uuid.New().String()

	repo.On("GetByParticipants", ctx, userA, userB).Return(nil, apperror.NotFound("not found", nil))
	repo.On("Create", ctx, mock.AnythingOfType("*entity.Conversation")).Return(errors.New("db error"))

	_, err := uc.Execute(ctx, userA, userB)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create conversation")
}
