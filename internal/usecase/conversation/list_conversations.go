package conversation

import (
	"context"

	"github.com/google/uuid"
	"github.com/horaciobranciforte/curiosity-chat-api/internal/domain/entity"
	domerrors "github.com/horaciobranciforte/curiosity-chat-api/internal/domain/errors"
	"github.com/horaciobranciforte/curiosity-chat-api/internal/pkg/apperror"
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
	if userID == "" {
		return nil, 0, apperror.Validation("user ID is required", domerrors.ErrInvalidUserID)
	}
	if _, err := uuid.Parse(userID); err != nil {
		return nil, 0, apperror.Validation("invalid user ID format", domerrors.ErrInvalidUserID)
	}

	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
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
