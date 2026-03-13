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
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func makeConv(userA, userB string) *entity.Conversation {
	return entity.NewConversation(userA, userB)
}

func TestSendMessageTextSuccess(t *testing.T) {
	msgRepo := new(mocks.MessageRepositoryMock)
	convRepo := new(mocks.ConversationRepositoryMock)
	followChecker := new(mocks.FollowCheckerMock)
	uc := NewSendMessage(msgRepo, convRepo, followChecker)

	ctx := context.Background()
	senderID := uuid.New().String()
	conv := makeConv(senderID, uuid.New().String())

	convRepo.On("GetByID", ctx, conv.ID).Return(conv, nil)
	followChecker.On("AreFollowing", ctx, senderID, mock.Anything).Return(true, nil)
	msgRepo.On("Create", ctx, mock.AnythingOfType("*entity.Message")).Return(nil)
	convRepo.On("UpdateLastMessageAt", ctx, conv.ID, mock.Anything).Return(nil)

	msg, err := uc.Execute(ctx, conv.ID, senderID, SendMessageInput{Type: "text", Content: "hello"})

	require.NoError(t, err)
	require.NotNil(t, msg)
	assert.Equal(t, entity.MessageTypeText, msg.Type)
	assert.Equal(t, "hello", *msg.Content)
	msgRepo.AssertExpectations(t)
	convRepo.AssertExpectations(t)
	followChecker.AssertExpectations(t)
}

func TestSendMessagePOIShareSuccess(t *testing.T) {
	msgRepo := new(mocks.MessageRepositoryMock)
	convRepo := new(mocks.ConversationRepositoryMock)
	followChecker := new(mocks.FollowCheckerMock)
	uc := NewSendMessage(msgRepo, convRepo, followChecker)

	ctx := context.Background()
	senderID := uuid.New().String()
	poiID := uuid.New().String()
	conv := makeConv(senderID, uuid.New().String())

	convRepo.On("GetByID", ctx, conv.ID).Return(conv, nil)
	followChecker.On("AreFollowing", ctx, senderID, mock.Anything).Return(true, nil)
	msgRepo.On("Create", ctx, mock.AnythingOfType("*entity.Message")).Return(nil)
	convRepo.On("UpdateLastMessageAt", ctx, conv.ID, mock.Anything).Return(nil)

	msg, err := uc.Execute(ctx, conv.ID, senderID, SendMessageInput{Type: "poi_share", POIID: poiID})

	require.NoError(t, err)
	assert.Equal(t, entity.MessageTypePOIShare, msg.Type)
	assert.Equal(t, poiID, *msg.POIID)
}

func TestSendMessageEmptyConversationID(t *testing.T) {
	uc := NewSendMessage(new(mocks.MessageRepositoryMock), new(mocks.ConversationRepositoryMock), new(mocks.FollowCheckerMock))

	_, err := uc.Execute(context.Background(), "", uuid.New().String(), SendMessageInput{Type: "text", Content: "hi"})

	require.Error(t, err)
	assert.True(t, apperror.IsValidation(err))
	assert.Contains(t, err.Error(), "conversation ID is required")
}

func TestSendMessageInvalidConversationID(t *testing.T) {
	uc := NewSendMessage(new(mocks.MessageRepositoryMock), new(mocks.ConversationRepositoryMock), new(mocks.FollowCheckerMock))

	_, err := uc.Execute(context.Background(), "bad-id", uuid.New().String(), SendMessageInput{Type: "text", Content: "hi"})

	require.Error(t, err)
	assert.True(t, apperror.IsValidation(err))
	assert.Contains(t, err.Error(), "invalid conversation ID format")
}

func TestSendMessageEmptySenderID(t *testing.T) {
	uc := NewSendMessage(new(mocks.MessageRepositoryMock), new(mocks.ConversationRepositoryMock), new(mocks.FollowCheckerMock))

	_, err := uc.Execute(context.Background(), uuid.New().String(), "", SendMessageInput{Type: "text", Content: "hi"})

	require.Error(t, err)
	assert.True(t, apperror.IsValidation(err))
	assert.Contains(t, err.Error(), "sender ID is required")
}

func TestSendMessageConversationNotFound(t *testing.T) {
	msgRepo := new(mocks.MessageRepositoryMock)
	convRepo := new(mocks.ConversationRepositoryMock)
	followChecker := new(mocks.FollowCheckerMock)
	uc := NewSendMessage(msgRepo, convRepo, followChecker)

	ctx := context.Background()
	convID := uuid.New().String()

	convRepo.On("GetByID", ctx, convID).Return(nil, apperror.NotFound("not found", nil))

	_, err := uc.Execute(ctx, convID, uuid.New().String(), SendMessageInput{Type: "text", Content: "hi"})

	require.Error(t, err)
	assert.True(t, apperror.IsNotFound(err))
}

func TestSendMessageNotParticipant(t *testing.T) {
	msgRepo := new(mocks.MessageRepositoryMock)
	convRepo := new(mocks.ConversationRepositoryMock)
	followChecker := new(mocks.FollowCheckerMock)
	uc := NewSendMessage(msgRepo, convRepo, followChecker)

	ctx := context.Background()
	conv := makeConv(uuid.New().String(), uuid.New().String())
	outsider := uuid.New().String()

	convRepo.On("GetByID", ctx, conv.ID).Return(conv, nil)

	_, err := uc.Execute(ctx, conv.ID, outsider, SendMessageInput{Type: "text", Content: "hi"})

	require.Error(t, err)
	assert.True(t, apperror.IsForbidden(err))
}

func TestSendMessageEmptyContent(t *testing.T) {
	msgRepo := new(mocks.MessageRepositoryMock)
	convRepo := new(mocks.ConversationRepositoryMock)
	followChecker := new(mocks.FollowCheckerMock)
	uc := NewSendMessage(msgRepo, convRepo, followChecker)

	ctx := context.Background()
	senderID := uuid.New().String()
	conv := makeConv(senderID, uuid.New().String())

	convRepo.On("GetByID", ctx, conv.ID).Return(conv, nil)
	followChecker.On("AreFollowing", ctx, senderID, mock.Anything).Return(true, nil)

	_, err := uc.Execute(ctx, conv.ID, senderID, SendMessageInput{Type: "text", Content: ""})

	require.Error(t, err)
	assert.True(t, apperror.IsValidation(err))
	assert.Contains(t, err.Error(), "content is required")
}

func TestSendMessagePOIShareMissingPOIID(t *testing.T) {
	msgRepo := new(mocks.MessageRepositoryMock)
	convRepo := new(mocks.ConversationRepositoryMock)
	followChecker := new(mocks.FollowCheckerMock)
	uc := NewSendMessage(msgRepo, convRepo, followChecker)

	ctx := context.Background()
	senderID := uuid.New().String()
	conv := makeConv(senderID, uuid.New().String())

	convRepo.On("GetByID", ctx, conv.ID).Return(conv, nil)
	followChecker.On("AreFollowing", ctx, senderID, mock.Anything).Return(true, nil)

	_, err := uc.Execute(ctx, conv.ID, senderID, SendMessageInput{Type: "poi_share", POIID: ""})

	require.Error(t, err)
	assert.True(t, apperror.IsValidation(err))
	assert.Contains(t, err.Error(), "poi_id is required")
}

func TestSendMessagePOIShareInvalidPOIID(t *testing.T) {
	msgRepo := new(mocks.MessageRepositoryMock)
	convRepo := new(mocks.ConversationRepositoryMock)
	followChecker := new(mocks.FollowCheckerMock)
	uc := NewSendMessage(msgRepo, convRepo, followChecker)

	ctx := context.Background()
	senderID := uuid.New().String()
	conv := makeConv(senderID, uuid.New().String())

	convRepo.On("GetByID", ctx, conv.ID).Return(conv, nil)
	followChecker.On("AreFollowing", ctx, senderID, mock.Anything).Return(true, nil)

	_, err := uc.Execute(ctx, conv.ID, senderID, SendMessageInput{Type: "poi_share", POIID: "not-a-uuid"})

	require.Error(t, err)
	assert.True(t, apperror.IsValidation(err))
	assert.Contains(t, err.Error(), "invalid poi_id format")
}

func TestSendMessageInvalidType(t *testing.T) {
	msgRepo := new(mocks.MessageRepositoryMock)
	convRepo := new(mocks.ConversationRepositoryMock)
	followChecker := new(mocks.FollowCheckerMock)
	uc := NewSendMessage(msgRepo, convRepo, followChecker)

	ctx := context.Background()
	senderID := uuid.New().String()
	conv := makeConv(senderID, uuid.New().String())

	convRepo.On("GetByID", ctx, conv.ID).Return(conv, nil)
	followChecker.On("AreFollowing", ctx, senderID, mock.Anything).Return(true, nil)

	_, err := uc.Execute(ctx, conv.ID, senderID, SendMessageInput{Type: "video"})

	require.Error(t, err)
	assert.True(t, apperror.IsValidation(err))
	assert.Contains(t, err.Error(), "text")
}

func TestSendMessageSaveFails(t *testing.T) {
	msgRepo := new(mocks.MessageRepositoryMock)
	convRepo := new(mocks.ConversationRepositoryMock)
	followChecker := new(mocks.FollowCheckerMock)
	uc := NewSendMessage(msgRepo, convRepo, followChecker)

	ctx := context.Background()
	senderID := uuid.New().String()
	conv := makeConv(senderID, uuid.New().String())

	convRepo.On("GetByID", ctx, conv.ID).Return(conv, nil)
	followChecker.On("AreFollowing", ctx, senderID, mock.Anything).Return(true, nil)
	msgRepo.On("Create", ctx, mock.AnythingOfType("*entity.Message")).Return(errors.New("db error"))

	_, err := uc.Execute(ctx, conv.ID, senderID, SendMessageInput{Type: "text", Content: "hi"})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to save message")
}

func TestSendMessageUpdateLastMessageAtFails(t *testing.T) {
	msgRepo := new(mocks.MessageRepositoryMock)
	convRepo := new(mocks.ConversationRepositoryMock)
	followChecker := new(mocks.FollowCheckerMock)
	uc := NewSendMessage(msgRepo, convRepo, followChecker)

	ctx := context.Background()
	senderID := uuid.New().String()
	conv := makeConv(senderID, uuid.New().String())

	convRepo.On("GetByID", ctx, conv.ID).Return(conv, nil)
	followChecker.On("AreFollowing", ctx, senderID, mock.Anything).Return(true, nil)
	msgRepo.On("Create", ctx, mock.AnythingOfType("*entity.Message")).Return(nil)
	convRepo.On("UpdateLastMessageAt", ctx, conv.ID, mock.Anything).Return(errors.New("db error"))

	_, err := uc.Execute(ctx, conv.ID, senderID, SendMessageInput{Type: "text", Content: "hi"})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to update conversation")
}

func TestSendMessageStrangerLimitAllowed(t *testing.T) {
	msgRepo := new(mocks.MessageRepositoryMock)
	convRepo := new(mocks.ConversationRepositoryMock)
	followChecker := new(mocks.FollowCheckerMock)
	uc := NewSendMessage(msgRepo, convRepo, followChecker)

	ctx := context.Background()
	senderID := uuid.New().String()
	conv := makeConv(senderID, uuid.New().String())

	convRepo.On("GetByID", ctx, conv.ID).Return(conv, nil)
	followChecker.On("AreFollowing", ctx, senderID, mock.Anything).Return(false, nil)
	msgRepo.On("CountByConversation", ctx, conv.ID).Return(1, nil)
	msgRepo.On("Create", ctx, mock.AnythingOfType("*entity.Message")).Return(nil)
	convRepo.On("UpdateLastMessageAt", ctx, conv.ID, mock.Anything).Return(nil)

	msg, err := uc.Execute(ctx, conv.ID, senderID, SendMessageInput{Type: "text", Content: "hello"})

	require.NoError(t, err)
	assert.Equal(t, "text", msg.Type)
}

func TestSendMessageStrangerLimitExceeded(t *testing.T) {
	msgRepo := new(mocks.MessageRepositoryMock)
	convRepo := new(mocks.ConversationRepositoryMock)
	followChecker := new(mocks.FollowCheckerMock)
	uc := NewSendMessage(msgRepo, convRepo, followChecker)

	ctx := context.Background()
	senderID := uuid.New().String()
	conv := makeConv(senderID, uuid.New().String())

	convRepo.On("GetByID", ctx, conv.ID).Return(conv, nil)
	followChecker.On("AreFollowing", ctx, senderID, mock.Anything).Return(false, nil)
	msgRepo.On("CountByConversation", ctx, conv.ID).Return(2, nil)

	_, err := uc.Execute(ctx, conv.ID, senderID, SendMessageInput{Type: "text", Content: "hello"})

	require.Error(t, err)
	assert.True(t, apperror.IsForbidden(err))
	assert.Contains(t, err.Error(), "become friends")
}

func TestSendMessageFollowCheckFails(t *testing.T) {
	msgRepo := new(mocks.MessageRepositoryMock)
	convRepo := new(mocks.ConversationRepositoryMock)
	followChecker := new(mocks.FollowCheckerMock)
	uc := NewSendMessage(msgRepo, convRepo, followChecker)

	ctx := context.Background()
	senderID := uuid.New().String()
	conv := makeConv(senderID, uuid.New().String())

	convRepo.On("GetByID", ctx, conv.ID).Return(conv, nil)
	followChecker.On("AreFollowing", ctx, senderID, mock.Anything).Return(false, errors.New("service down"))

	_, err := uc.Execute(ctx, conv.ID, senderID, SendMessageInput{Type: "text", Content: "hello"})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to check follow relationship")
}

func TestSendMessageCountFails(t *testing.T) {
	msgRepo := new(mocks.MessageRepositoryMock)
	convRepo := new(mocks.ConversationRepositoryMock)
	followChecker := new(mocks.FollowCheckerMock)
	uc := NewSendMessage(msgRepo, convRepo, followChecker)

	ctx := context.Background()
	senderID := uuid.New().String()
	conv := makeConv(senderID, uuid.New().String())

	convRepo.On("GetByID", ctx, conv.ID).Return(conv, nil)
	followChecker.On("AreFollowing", ctx, senderID, mock.Anything).Return(false, nil)
	msgRepo.On("CountByConversation", ctx, conv.ID).Return(0, errors.New("db error"))

	_, err := uc.Execute(ctx, conv.ID, senderID, SendMessageInput{Type: "text", Content: "hello"})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to count messages")
}

func TestSendMessageContentTooLong(t *testing.T) {
	msgRepo := new(mocks.MessageRepositoryMock)
	convRepo := new(mocks.ConversationRepositoryMock)
	followChecker := new(mocks.FollowCheckerMock)
	uc := NewSendMessage(msgRepo, convRepo, followChecker)

	ctx := context.Background()
	senderID := uuid.New().String()
	conv := makeConv(senderID, uuid.New().String())

	convRepo.On("GetByID", ctx, conv.ID).Return(conv, nil)
	followChecker.On("AreFollowing", ctx, senderID, mock.Anything).Return(true, nil)

	longContent := make([]byte, MaxMessageContentLength+1)
	for i := range longContent {
		longContent[i] = 'a'
	}

	_, err := uc.Execute(ctx, conv.ID, senderID, SendMessageInput{Type: "text", Content: string(longContent)})

	require.Error(t, err)
	assert.True(t, apperror.IsValidation(err))
	assert.Contains(t, err.Error(), "content exceeds maximum length")
}

func TestSendMessageContentMaxLength(t *testing.T) {
	msgRepo := new(mocks.MessageRepositoryMock)
	convRepo := new(mocks.ConversationRepositoryMock)
	followChecker := new(mocks.FollowCheckerMock)
	uc := NewSendMessage(msgRepo, convRepo, followChecker)

	ctx := context.Background()
	senderID := uuid.New().String()
	conv := makeConv(senderID, uuid.New().String())

	convRepo.On("GetByID", ctx, conv.ID).Return(conv, nil)
	followChecker.On("AreFollowing", ctx, senderID, mock.Anything).Return(true, nil)
	msgRepo.On("Create", ctx, mock.AnythingOfType("*entity.Message")).Return(nil)
	convRepo.On("UpdateLastMessageAt", ctx, conv.ID, mock.Anything).Return(nil)

	maxContent := make([]byte, MaxMessageContentLength)
	for i := range maxContent {
		maxContent[i] = 'a'
	}

	msg, err := uc.Execute(ctx, conv.ID, senderID, SendMessageInput{Type: "text", Content: string(maxContent)})

	require.NoError(t, err)
	assert.Equal(t, "text", msg.Type)
	assert.Equal(t, MaxMessageContentLength, len(*msg.Content))
}

func TestSendMessageContentOneOverMaxLength(t *testing.T) {
	msgRepo := new(mocks.MessageRepositoryMock)
	convRepo := new(mocks.ConversationRepositoryMock)
	followChecker := new(mocks.FollowCheckerMock)
	uc := NewSendMessage(msgRepo, convRepo, followChecker)

	ctx := context.Background()
	senderID := uuid.New().String()
	conv := makeConv(senderID, uuid.New().String())

	convRepo.On("GetByID", ctx, conv.ID).Return(conv, nil)
	followChecker.On("AreFollowing", ctx, senderID, mock.Anything).Return(true, nil)

	// Create content that is exactly 1 character over the limit (1001 chars)
	overLimitContent := make([]byte, MaxMessageContentLength+1)
	for i := range overLimitContent {
		overLimitContent[i] = 'b'
	}

	_, err := uc.Execute(ctx, conv.ID, senderID, SendMessageInput{Type: "text", Content: string(overLimitContent)})

	require.Error(t, err)
	assert.True(t, apperror.IsValidation(err))
	assert.Contains(t, err.Error(), "content exceeds maximum length")
}

func TestSendMessageContentOneUnderMaxLength(t *testing.T) {
	msgRepo := new(mocks.MessageRepositoryMock)
	convRepo := new(mocks.ConversationRepositoryMock)
	followChecker := new(mocks.FollowCheckerMock)
	uc := NewSendMessage(msgRepo, convRepo, followChecker)

	ctx := context.Background()
	senderID := uuid.New().String()
	conv := makeConv(senderID, uuid.New().String())

	convRepo.On("GetByID", ctx, conv.ID).Return(conv, nil)
	followChecker.On("AreFollowing", ctx, senderID, mock.Anything).Return(true, nil)
	msgRepo.On("Create", ctx, mock.AnythingOfType("*entity.Message")).Return(nil)
	convRepo.On("UpdateLastMessageAt", ctx, conv.ID, mock.Anything).Return(nil)

	// Create content that is exactly 1 character under the limit (999 chars)
	underLimitContent := make([]byte, MaxMessageContentLength-1)
	for i := range underLimitContent {
		underLimitContent[i] = 'c'
	}

	msg, err := uc.Execute(ctx, conv.ID, senderID, SendMessageInput{Type: "text", Content: string(underLimitContent)})

	require.NoError(t, err)
	assert.Equal(t, "text", msg.Type)
	assert.Equal(t, MaxMessageContentLength-1, len(*msg.Content))
}

func TestSendMessagePOIShareInvalidShareIntent(t *testing.T) {
	msgRepo := new(mocks.MessageRepositoryMock)
	convRepo := new(mocks.ConversationRepositoryMock)
	followChecker := new(mocks.FollowCheckerMock)
	uc := NewSendMessage(msgRepo, convRepo, followChecker)

	ctx := context.Background()
	senderID := uuid.New().String()
	conv := makeConv(senderID, uuid.New().String())
	poiID := uuid.New().String()

	convRepo.On("GetByID", ctx, conv.ID).Return(conv, nil)
	followChecker.On("AreFollowing", ctx, senderID, mock.Anything).Return(true, nil)

	_, err := uc.Execute(ctx, conv.ID, senderID, SendMessageInput{Type: "poi_share", POIID: poiID, ShareIntent: "invalid_intent"})

	require.Error(t, err)
	assert.True(t, apperror.IsValidation(err))
	assert.Contains(t, err.Error(), "invalid share_intent")
}

func TestSendMessagePOIShareValidShareIntents(t *testing.T) {
	validIntents := []string{"must_go", "come_with_me", "invite", "invite_me"}

	for _, intent := range validIntents {
		t.Run(intent, func(t *testing.T) {
			msgRepo := new(mocks.MessageRepositoryMock)
			convRepo := new(mocks.ConversationRepositoryMock)
			followChecker := new(mocks.FollowCheckerMock)
			uc := NewSendMessage(msgRepo, convRepo, followChecker)

			ctx := context.Background()
			senderID := uuid.New().String()
			conv := makeConv(senderID, uuid.New().String())
			poiID := uuid.New().String()

			convRepo.On("GetByID", ctx, conv.ID).Return(conv, nil)
			followChecker.On("AreFollowing", ctx, senderID, mock.Anything).Return(true, nil)
			msgRepo.On("Create", ctx, mock.AnythingOfType("*entity.Message")).Return(nil)
			convRepo.On("UpdateLastMessageAt", ctx, conv.ID, mock.Anything).Return(nil)

			msg, err := uc.Execute(ctx, conv.ID, senderID, SendMessageInput{
				Type:        "poi_share",
				POIID:       poiID,
				Content:     "Test POI",
				ShareIntent: intent,
			})

			require.NoError(t, err)
			assert.Equal(t, "poi_share", msg.Type)
			if msg.ShareIntent != nil {
				assert.Equal(t, intent, *msg.ShareIntent)
			}
		})
	}
}

func TestSendMessagePOIShareTitleTooLong(t *testing.T) {
	msgRepo := new(mocks.MessageRepositoryMock)
	convRepo := new(mocks.ConversationRepositoryMock)
	followChecker := new(mocks.FollowCheckerMock)
	uc := NewSendMessage(msgRepo, convRepo, followChecker)

	ctx := context.Background()
	senderID := uuid.New().String()
	conv := makeConv(senderID, uuid.New().String())
	poiID := uuid.New().String()

	convRepo.On("GetByID", ctx, conv.ID).Return(conv, nil)
	followChecker.On("AreFollowing", ctx, senderID, mock.Anything).Return(true, nil)

	longTitle := make([]byte, MaxPOITitleLength+1)
	for i := range longTitle {
		longTitle[i] = 'a'
	}

	_, err := uc.Execute(ctx, conv.ID, senderID, SendMessageInput{Type: "poi_share", POIID: poiID, Content: string(longTitle)})

	require.Error(t, err)
	assert.True(t, apperror.IsValidation(err))
	assert.Contains(t, err.Error(), "poi title exceeds maximum length")
}

func TestSendMessageInternalError(t *testing.T) {
	msgRepo := new(mocks.MessageRepositoryMock)
	convRepo := new(mocks.ConversationRepositoryMock)
	followChecker := new(mocks.FollowCheckerMock)
	uc := NewSendMessage(msgRepo, convRepo, followChecker)

	ctx := context.Background()
	convID := uuid.New().String()

	convRepo.On("GetByID", ctx, convID).Return(nil, errors.New("database error"))

	_, err := uc.Execute(ctx, convID, uuid.New().String(), SendMessageInput{Type: "text", Content: "hi"})

	require.Error(t, err)
	assert.True(t, apperror.IsType(err, apperror.TypeInternal))
	assert.Contains(t, err.Error(), "failed to get conversation")
}
