package message

import (
	"context"

	"github.com/google/uuid"
	"github.com/horaciobranciforte/curiosity-chat-api/internal/domain/entity"
	domerrors "github.com/horaciobranciforte/curiosity-chat-api/internal/domain/errors"
	"github.com/horaciobranciforte/curiosity-chat-api/internal/pkg/apperror"
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
	if conversationID == "" {
		return nil, 0, apperror.Validation("conversation ID is required", domerrors.ErrInvalidConversationID)
	}
	if _, err := uuid.Parse(conversationID); err != nil {
		return nil, 0, apperror.Validation("invalid conversation ID format", domerrors.ErrInvalidConversationID)
	}
	if requesterID == "" {
		return nil, 0, apperror.Validation("requester ID is required", domerrors.ErrInvalidUserID)
	}

	if limit <= 0 {
		limit = 50
	}
	if limit > 100 {
		limit = 100
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
