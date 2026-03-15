package message

import (
	"context"
	"time"

	"github.com/horaciobranciforte/curiosity-chat-api/internal/domain/entity"
)

// Repository defines the data access contract for messages.
type Repository interface {
	Create(ctx context.Context, m *entity.Message) error
	ListByConversation(ctx context.Context, conversationID string, limit, offset int) ([]*entity.Message, error)
	CountByConversation(ctx context.Context, conversationID string) (int, error)
	// UpdateStatus updates the delivery status of a single message.
	UpdateStatus(ctx context.Context, messageID, status string) error
	// MarkConversationRead marks all unread messages from other participants as read.
	// Returns the ID of the most recently read message, or empty string if nothing changed.
	MarkConversationRead(ctx context.Context, conversationID, readerID string) (string, error)
	// CountUnreadByConversationForUser returns how many unread messages the user has in a conversation.
	CountUnreadByConversationForUser(ctx context.Context, conversationID, userID string) (int, error)
	// CountTotalUnreadForUser returns the total unread message count across all conversations for the user.
	CountTotalUnreadForUser(ctx context.Context, userID string) (int, error)
}

// ConversationRepository provides read/update access to conversations from message use cases.
type ConversationRepository interface {
	GetByID(ctx context.Context, id string) (*entity.Conversation, error)
	UpdateLastMessageAt(ctx context.Context, id string, t time.Time) error
}

// FollowChecker verifies whether two users are mutual follows.
type FollowChecker interface {
	AreFollowing(ctx context.Context, userA, userB string) (bool, error)
}
