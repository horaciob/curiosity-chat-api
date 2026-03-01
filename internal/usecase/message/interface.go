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
}

// ConversationRepository provides read/update access to conversations from message use cases.
type ConversationRepository interface {
	GetByID(ctx context.Context, id string) (*entity.Conversation, error)
	UpdateLastMessageAt(ctx context.Context, id string, t time.Time) error
}
