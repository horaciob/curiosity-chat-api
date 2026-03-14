package message

import (
	"context"

	"github.com/horaciobranciforte/curiosity-chat-api/internal/domain/entity"
	domerrors "github.com/horaciobranciforte/curiosity-chat-api/internal/domain/errors"
	"github.com/horaciobranciforte/curiosity-chat-api/internal/infrastructure/logger"
	"github.com/horaciobranciforte/curiosity-chat-api/internal/pkg/apperror"
	"go.uber.org/zap"
)

// SendMessageInput carries the parameters for sending a message.
type SendMessageInput struct {
	Type        string // "text" or "poi_share"
	Content     string // required for type=text; POI title for type=poi_share
	POIID       string // required for type=poi_share
	ShareIntent string // "must_go" | "come_with_me" | "invite" — optional, only for type=poi_share
}

const (
	// MaxMessageContentLength is the maximum allowed length for message content (1000 characters)
	MaxMessageContentLength = 1000
	// MaxPOITitleLength is the maximum allowed length for POI title (500 characters)
	MaxPOITitleLength = 500
)

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
// All participants can send unlimited messages without any mutual follow requirement.
func (uc *SendMessage) Execute(ctx context.Context, conversationID, senderID string, in SendMessageInput) (*entity.Message, error) {
	logger.Info("[SEND_MESSAGE] Entering use case",
		zap.String("conversation_id", conversationID),
		zap.String("sender_id", senderID),
		zap.String("message_type", in.Type),
	)

	if err := apperror.ValidateUUID(conversationID, "conversation ID", domerrors.ErrInvalidConversationID); err != nil {
		logger.Error("[SEND_MESSAGE] Invalid conversation ID",
			zap.String("conversation_id", conversationID),
			zap.Error(err),
		)
		return nil, err
	}
	if senderID == "" {
		logger.Error("[SEND_MESSAGE] Sender ID is empty")
		return nil, apperror.Validation("sender ID is required", domerrors.ErrInvalidUserID)
	}

	conv, err := uc.convRepo.GetByID(ctx, conversationID)
	if err != nil {
		if apperror.IsNotFound(err) {
			logger.Warn("[SEND_MESSAGE] Conversation not found",
				zap.String("conversation_id", conversationID),
			)
			return nil, err
		}
		logger.Error("[SEND_MESSAGE] Failed to get conversation",
			zap.String("conversation_id", conversationID),
			zap.Error(err),
		)
		return nil, apperror.Internal("failed to get conversation", err)
	}

	logger.Info("[SEND_MESSAGE] Conversation retrieved",
		zap.String("conversation_id", conversationID),
		zap.String("user1_id", conv.User1ID),
		zap.String("user2_id", conv.User2ID),
	)

	if !conv.HasParticipant(senderID) {
		logger.Error("[SEND_MESSAGE] 403 Forbidden - User is not a participant",
			zap.String("conversation_id", conversationID),
			zap.String("sender_id", senderID),
			zap.String("user1_id", conv.User1ID),
			zap.String("user2_id", conv.User2ID),
		)
		return nil, apperror.Forbidden("you are not a participant in this conversation", domerrors.ErrNotParticipant)
	}

	logger.Info("[SEND_MESSAGE] Participant check passed - unlimited messaging enabled",
		zap.String("conversation_id", conversationID),
		zap.String("sender_id", senderID),
	)

	var msg *entity.Message
	switch in.Type {
	case entity.MessageTypeText:
		if in.Content == "" {
			return nil, apperror.Validation("content is required for text messages", domerrors.ErrInvalidMessageType)
		}
		if len(in.Content) > MaxMessageContentLength {
			return nil, apperror.Validation("content exceeds maximum length", domerrors.ErrInvalidMessageType)
		}
		msg = entity.NewTextMessage(conversationID, senderID, in.Content)
	case entity.MessageTypePOIShare:
		if err := apperror.ValidateUUID(in.POIID, "poi_id", domerrors.ErrInvalidMessageType); err != nil {
			return nil, err
		}
		if in.Content != "" && len(in.Content) > MaxPOITitleLength {
			return nil, apperror.Validation("poi title exceeds maximum length", domerrors.ErrInvalidMessageType)
		}
		if in.ShareIntent != "" {
			validIntents := map[string]bool{
				entity.ShareIntentMustGo:     true,
				entity.ShareIntentComeWithMe: true,
				entity.ShareIntentInvite:     true,
				entity.ShareIntentInviteMe:   true,
			}
			if !validIntents[in.ShareIntent] {
				return nil, apperror.Validation("invalid share_intent value", domerrors.ErrInvalidMessageType)
			}
		}
		msg = entity.NewPOIShareMessage(conversationID, senderID, in.POIID, in.Content, in.ShareIntent)
	default:
		return nil, apperror.Validation("type must be 'text' or 'poi_share'", domerrors.ErrInvalidMessageType)
	}

	if err := uc.repo.Create(ctx, msg); err != nil {
		logger.Error("[SEND_MESSAGE] Failed to save message",
			zap.String("conversation_id", conversationID),
			zap.String("sender_id", senderID),
			zap.Error(err),
		)
		return nil, apperror.Internal("failed to save message", err)
	}

	if err := uc.convRepo.UpdateLastMessageAt(ctx, conversationID, msg.CreatedAt); err != nil {
		logger.Error("[SEND_MESSAGE] Failed to update conversation",
			zap.String("conversation_id", conversationID),
			zap.Error(err),
		)
		return nil, apperror.Internal("failed to update conversation", err)
	}

	logger.Info("[SEND_MESSAGE] Message sent successfully",
		zap.String("conversation_id", conversationID),
		zap.String("sender_id", senderID),
		zap.String("message_id", msg.ID),
		zap.String("message_type", msg.Type),
	)

	return msg, nil
}
