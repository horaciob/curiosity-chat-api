package message

import (
	"context"

	"github.com/horaciobranciforte/curiosity-chat-api/internal/domain/entity"
	domerrors "github.com/horaciobranciforte/curiosity-chat-api/internal/domain/errors"
	"github.com/horaciobranciforte/curiosity-chat-api/internal/pkg/apperror"
)

const (
	// DefaultMessageLimit is the default number of messages to return
	DefaultMessageLimit = 50
	// MaxMessageLimit is the maximum allowed limit for pagination
	MaxMessageLimit = 100
)

// GetMessages retrieves paginated messages for a conversation.
type GetMessages struct {
	repo     Repository
	convRepo ConversationRepository
}

// NewGetMessages creates a new GetMessages use case.
func NewGetMessages(repo Repository, convRepo ConversationRepository) *GetMessages {
	return &GetMessages{repo: repo, convRepo: convRepo}
}

// Execute returns messages for a conversation, newest first. Verifies requester is a participant.
func (uc *GetMessages) Execute(ctx context.Context, conversationID, requesterID string, limit, offset int) ([]*entity.Message, int, error) {
	if err := apperror.ValidateUUID(conversationID, "conversation ID", domerrors.ErrInvalidConversationID); err != nil {
		return nil, 0, err
	}
	if requesterID == "" {
		return nil, 0, apperror.Validation("requester ID is required", domerrors.ErrInvalidUserID)
	}

	if limit <= 0 {
		limit = DefaultMessageLimit
	}
	if limit > MaxMessageLimit {
		limit = MaxMessageLimit
	}

	conv, err := uc.convRepo.GetByID(ctx, conversationID)
	if err != nil {
		if apperror.IsNotFound(err) {
			return nil, 0, err
		}
		return nil, 0, apperror.Internal("failed to get conversation", err)
	}

	if !conv.HasParticipant(requesterID) {
		return nil, 0, apperror.Forbidden("access denied", domerrors.ErrNotParticipant)
	}

	msgs, err := uc.repo.ListByConversation(ctx, conversationID, limit, offset)
	if err != nil {
		return nil, 0, apperror.Internal("failed to get messages", err)
	}

	total, err := uc.repo.CountByConversation(ctx, conversationID)
	if err != nil {
		return nil, 0, apperror.Internal("failed to count messages", err)
	}

	return msgs, total, nil
}
