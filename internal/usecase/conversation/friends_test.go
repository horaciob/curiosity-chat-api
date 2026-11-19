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

type friendAnswer bool

func (f friendAnswer) AreFriends(context.Context, string, string) (bool, error) { return bool(f), nil }

func TestOnlyFriendsCanStartAConversation(t *testing.T) {
	repo := new(mocks.ConversationRepositoryMock)
	uc := NewCreateConversation(repo).WithFriendChecker(friendAnswer(false))

	_, err := uc.Execute(context.Background(), uuid.New().String(), uuid.New().String())

	assert.True(t, apperror.IsForbidden(err), "got %v", err)
	repo.AssertNotCalled(t, "Create", mock.Anything, mock.Anything)
}

func TestFriendsCanStartAConversation(t *testing.T) {
	repo := new(mocks.ConversationRepositoryMock)
	uc := NewCreateConversation(repo).WithFriendChecker(friendAnswer(true))
	ctx := context.Background()
	a, b := uuid.New().String(), uuid.New().String()
	repo.On("GetByParticipants", ctx, a, b).Return(nil, apperror.NotFound("none", nil))
	repo.On("Create", ctx, mock.AnythingOfType("*entity.Conversation")).Return(nil)

	conv, err := uc.Execute(ctx, a, b)
	require.NoError(t, err)
	assert.NotEmpty(t, conv.ID)
}

func TestCreatingTheSameConversationTwiceAtOnceReturnsTheOneThatWon(t *testing.T) {
	repo := new(mocks.ConversationRepositoryMock)
	uc := NewCreateConversation(repo)
	ctx := context.Background()
	a, b := uuid.New().String(), uuid.New().String()
	winner := entity.NewConversation(a, b)
	repo.On("GetByParticipants", ctx, a, b).Return(nil, apperror.NotFound("none", nil)).Once()
	repo.On("Create", ctx, mock.AnythingOfType("*entity.Conversation")).Return(errors.New("duplicate key value violates unique constraint \"conversations_unique_pair\""))
	repo.On("GetByParticipants", ctx, a, b).Return(winner, nil).Once()

	conv, err := uc.Execute(ctx, a, b)
	require.NoError(t, err) // used to be a 500
	assert.Equal(t, winner.ID, conv.ID)
}
