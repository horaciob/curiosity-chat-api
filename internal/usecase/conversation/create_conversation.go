package conversation

import (
	"context"

	"github.com/horaciobranciforte/curiosity-chat-api/internal/domain/entity"
	domerrors "github.com/horaciobranciforte/curiosity-chat-api/internal/domain/errors"
	"github.com/horaciobranciforte/curiosity-chat-api/internal/pkg/apperror"
)

// CreateConversation creates or returns an existing conversation between two users.
type CreateConversation struct {
	repo Repository
}

// NewCreateConversation creates a new CreateConversation use case.
func NewCreateConversation(repo Repository) *CreateConversation {
	return &CreateConversation{repo: repo}
}

// Execute creates or retrieves the conversation between requesterID and targetID.
func (uc *CreateConversation) Execute(ctx context.Context, requesterID, targetID string) (*entity.Conversation, error) {
	if err := apperror.ValidateUUID(requesterID, "requester ID", domerrors.ErrInvalidUserID); err != nil {
		return nil, err
	}
	if err := apperror.ValidateUUID(targetID, "target user ID", domerrors.ErrInvalidUserID); err != nil {
		return nil, err
	}
	if requesterID == targetID {
		return nil, apperror.Validation("cannot start a conversation with yourself", domerrors.ErrSelfConversation)
	}

	existing, err := uc.repo.GetByParticipants(ctx, requesterID, targetID)
	if err != nil && !apperror.IsNotFound(err) {
		return nil, apperror.Internal("failed to check existing conversation", err)
	}
	if existing != nil {
		return existing, nil
	}

	conv := entity.NewConversation(requesterID, targetID)
	if err := uc.repo.Create(ctx, conv); err != nil {
		return nil, apperror.Internal("failed to create conversation", err)
	}

	return conv, nil
}
