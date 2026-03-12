package conversation

import (
	"context"

	"github.com/horaciobranciforte/curiosity-chat-api/internal/domain/entity"
	domerrors "github.com/horaciobranciforte/curiosity-chat-api/internal/domain/errors"
	"github.com/horaciobranciforte/curiosity-chat-api/internal/pkg/apperror"
)

const (
	// DefaultConversationLimit is the default number of conversations to return
	DefaultConversationLimit = 20
	// MaxConversationLimit is the maximum allowed limit for pagination
	MaxConversationLimit = 100
)

// ListConversations returns a paginated list of conversations for a user.
type ListConversations struct {
	repo Repository
}

// NewListConversations creates a new ListConversations use case.
func NewListConversations(repo Repository) *ListConversations {
	return &ListConversations{repo: repo}
}

// Execute returns conversations for userID ordered by last_message_at DESC.
func (uc *ListConversations) Execute(ctx context.Context, userID string, limit, offset int) ([]*entity.Conversation, int, error) {
	if err := apperror.ValidateUUID(userID, "user ID", domerrors.ErrInvalidUserID); err != nil {
		return nil, 0, err
	}

	if limit <= 0 {
		limit = DefaultConversationLimit
	}
	if limit > MaxConversationLimit {
		limit = MaxConversationLimit
	}

	convs, err := uc.repo.ListByUser(ctx, userID, limit, offset)
	if err != nil {
		return nil, 0, apperror.Internal("failed to list conversations", err)
	}

	total, err := uc.repo.CountByUser(ctx, userID)
	if err != nil {
		return nil, 0, apperror.Internal("failed to count conversations", err)
	}

	return convs, total, nil
}
