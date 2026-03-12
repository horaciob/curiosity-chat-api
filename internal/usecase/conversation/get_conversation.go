package conversation

import (
	"context"

	"github.com/horaciobranciforte/curiosity-chat-api/internal/domain/entity"
	domerrors "github.com/horaciobranciforte/curiosity-chat-api/internal/domain/errors"
	"github.com/horaciobranciforte/curiosity-chat-api/internal/pkg/apperror"
)

// GetConversation retrieves a conversation by ID, verifying the requester is a participant.
type GetConversation struct {
	repo Repository
}

// NewGetConversation creates a new GetConversation use case.
func NewGetConversation(repo Repository) *GetConversation {
	return &GetConversation{repo: repo}
}

// Execute returns the conversation with the given ID if requesterID is a participant.
func (uc *GetConversation) Execute(ctx context.Context, conversationID, requesterID string) (*entity.Conversation, error) {
	if err := apperror.ValidateUUID(conversationID, "conversation ID", domerrors.ErrInvalidConversationID); err != nil {
		return nil, err
	}
	if requesterID == "" {
		return nil, apperror.Validation("requester ID is required", domerrors.ErrInvalidUserID)
	}

	conv, err := uc.repo.GetByID(ctx, conversationID)
	if err != nil {
		if apperror.IsNotFound(err) {
			return nil, err
		}
		return nil, apperror.Internal("failed to get conversation", err)
	}

	if !conv.HasParticipant(requesterID) {
		return nil, apperror.Forbidden("access denied", domerrors.ErrNotParticipant)
	}

	return conv, nil
}
