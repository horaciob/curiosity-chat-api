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
	Type    string // "text" or "poi_share"
	Content string // required for type=text
	POIID   string // required for type=poi_share
}

// SendMessage sends a message to an existing conversation.
type SendMessage struct {
	repo     Repository
	convRepo ConversationRepository
}

// NewSendMessage creates a new SendMessage use case.
func NewSendMessage(repo Repository, convRepo ConversationRepository) *SendMessage {
	return &SendMessage{repo: repo, convRepo: convRepo}
}

// Execute creates and persists a message, updating the conversation's last_message_at.
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
		msg = entity.NewPOIShareMessage(conversationID, senderID, in.POIID)
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
