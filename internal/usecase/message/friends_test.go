package message

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/horaciobranciforte/curiosity-chat-api/internal/pkg/apperror"
	"github.com/horaciobranciforte/curiosity-chat-api/test/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type friendCheck struct {
	ok  bool
	err error
}

func (f friendCheck) AreFriends(context.Context, string, string) (bool, error) { return f.ok, f.err }

func TestAnExFriendCannotKeepMessaging(t *testing.T) {
	msgRepo := new(mocks.MessageRepositoryMock)
	convRepo := new(mocks.ConversationRepositoryMock)
	uc := NewSendMessage(msgRepo, convRepo).WithFriendChecker(friendCheck{ok: false})
	ctx := context.Background()
	sender := uuid.New().String()
	conv := makeConv(sender, uuid.New().String())
	convRepo.On("GetByID", ctx, conv.ID).Return(conv, nil)

	_, err := uc.Execute(ctx, conv.ID, sender, SendMessageInput{Type: "text", Content: "hi"})

	assert.True(t, apperror.IsForbidden(err), "got %v", err)
	msgRepo.AssertNotCalled(t, "Create", mock.Anything, mock.Anything)
}

func TestAMessageStillGoesThroughWhenUserAPIIsDown(t *testing.T) {
	msgRepo := new(mocks.MessageRepositoryMock)
	convRepo := new(mocks.ConversationRepositoryMock)
	uc := NewSendMessage(msgRepo, convRepo).WithFriendChecker(friendCheck{err: errors.New("down")})
	ctx := context.Background()
	sender := uuid.New().String()
	conv := makeConv(sender, uuid.New().String())
	convRepo.On("GetByID", ctx, conv.ID).Return(conv, nil)
	msgRepo.On("Create", ctx, mock.AnythingOfType("*entity.Message")).Return(nil)
	convRepo.On("UpdateLastMessageAt", ctx, conv.ID, mock.Anything).Return(nil)

	_, err := uc.Execute(ctx, conv.ID, sender, SendMessageInput{Type: "text", Content: "hi"})
	require.NoError(t, err)
}
