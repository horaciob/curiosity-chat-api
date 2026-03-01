package conversation

import (
	"context"
	"time"

	"github.com/horaciobranciforte/curiosity-chat-api/internal/domain/entity"
)

// Repository defines the data access contract for conversations.
type Repository interface {
	Create(ctx context.Context, c *entity.Conversation) error
	GetByID(ctx context.Context, id string) (*entity.Conversation, error)
	GetByParticipants(ctx context.Context, userA, userB string) (*entity.Conversation, error)
	ListByUser(ctx context.Context, userID string, limit, offset int) ([]*entity.Conversation, error)
	CountByUser(ctx context.Context, userID string) (int, error)
	UpdateLastMessageAt(ctx context.Context, id string, t time.Time) error
}

// FollowChecker verifies follow relationships between users.
type FollowChecker interface {
	AreFollowing(ctx context.Context, userA, userB string) (bool, error)
}
