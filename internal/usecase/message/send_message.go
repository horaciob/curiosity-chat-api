package message

import (
	"context"

	"github.com/google/uuid"
	"github.com/horaciobranciforte/curiosity-chat-api/internal/domain/entity"
	domerrors "github.com/horaciobranciforte/curiosity-chat-api/internal/domain/errors"
	"github.com/horaciobranciforte/curiosity-chat-api/internal/pkg/apperror"
)

// SendMessageInput carries the parameters for sending a message.
type SendMessageInput struct {
	Type        string // "text" or "poi_share"
	Content     string // required for type=text; POI title for type=poi_share
	POIID       string // required for type=poi_share
	ShareIntent string // "must_go" | "come_with_me" | "invite" — optional, only for type=poi_share
}

// SendMessage sends a message to an existing conversation.
type SendMessage struct {
	repo          Repository
	convRepo      ConversationRepository
	followChecker FollowChecker
}

// NewSendMessage creates a new SendMessage use case.
func NewSendMessage(repo Repository, convRepo ConversationRepository, followChecker FollowChecker) *SendMessage {
	return &SendMessage{repo: repo, convRepo: convRepo, followChecker: followChecker}
}

// Execute creates and persists a message, updating the conversation's last_message_at.
// Non-mutual-follow users can exchange only one message each; after that they must be
// friends (mutual follow) to continue the conversation.
func (uc *SendMessage) Execute(ctx context.Context, conversationID, senderID string, in SendMessageInput) (*entity.Message, error) {
	if conversationID == "" {
		return nil, apperror.Validation("conversation ID is required", domerrors.ErrInvalidConversationID)
	}
	if _, err := uuid.Parse(conversationID); err != nil {
		return nil, apperror.Validation("invalid conversation ID format", domerrors.ErrInvalidConversationID)
	}
	if senderID == "" {
		return nil, apperror.Validation("sender ID is required", domerrors.ErrInvalidUserID)
	}

	conv, err := uc.convRepo.GetByID(ctx, conversationID)
	if err != nil {
		if apperror.IsNotFound(err) {
			return nil, err
		}
		return nil, apperror.Internal("failed to get conversation", err)
	}

	if !conv.HasParticipant(senderID) {
		return nil, apperror.Forbidden("access denied", domerrors.ErrNotParticipant)
	}

	// Enforce the stranger message limit: non-mutual-follow users may only send
	// one message each (two total). Once the conversation has 2+ messages, both
	// participants must be mutual follows to continue.
	otherID := conv.OtherUserID(senderID)
	mutual, err := uc.followChecker.AreFollowing(ctx, senderID, otherID)
	if err != nil {
		return nil, apperror.Internal("failed to check follow relationship", err)
	}
	if !mutual {
		count, err := uc.repo.CountByConversation(ctx, conversationID)
		if err != nil {
			return nil, apperror.Internal("failed to count messages", err)
		}
		if count >= 2 {
			return nil, apperror.Forbidden("become friends to continue this conversation", domerrors.ErrUsersCannotChat)
		}
	}

	var msg *entity.Message
	switch in.Type {
	case entity.MessageTypeText:
		if in.Content == "" {
			return nil, apperror.Validation("content is required for text messages", domerrors.ErrInvalidMessageType)
		}
		msg = entity.NewTextMessage(conversationID, senderID, in.Content)
	case entity.MessageTypePOIShare:
		if in.POIID == "" {
			return nil, apperror.Validation("poi_id is required for poi_share messages", domerrors.ErrInvalidMessageType)
		}
		if _, err := uuid.Parse(in.POIID); err != nil {
			return nil, apperror.Validation("invalid poi_id format", domerrors.ErrInvalidMessageType)
		}
		msg = entity.NewPOIShareMessage(conversationID, senderID, in.POIID, in.Content, in.ShareIntent)
	default:
		return nil, apperror.Validation("type must be 'text' or 'poi_share'", domerrors.ErrInvalidMessageType)
	}

	if err := uc.repo.Create(ctx, msg); err != nil {
		return nil, apperror.Internal("failed to save message", err)
	}

	if err := uc.convRepo.UpdateLastMessageAt(ctx, conversationID, msg.CreatedAt); err != nil {
		return nil, apperror.Internal("failed to update conversation", err)
	}

	return msg, nil
}
