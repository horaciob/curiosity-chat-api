package conversation

import (
	"context"

	"github.com/google/uuid"
	"github.com/horaciobranciforte/curiosity-chat-api/internal/domain/entity"
	domerrors "github.com/horaciobranciforte/curiosity-chat-api/internal/domain/errors"
	"github.com/horaciobranciforte/curiosity-chat-api/internal/pkg/apperror"
)

// CreateConversation creates or returns an existing conversation between two users.
type CreateConversation struct {
	repo          Repository
	followChecker FollowChecker
}

// NewCreateConversation creates a new CreateConversation use case.
func NewCreateConversation(repo Repository, followChecker FollowChecker) *CreateConversation {
	return &CreateConversation{repo: repo, followChecker: followChecker}
}

// Execute creates or retrieves the conversation between requesterID and targetID.
func (uc *CreateConversation) Execute(ctx context.Context, requesterID, targetID string) (*entity.Conversation, error) {
	if requesterID == "" {
		return nil, apperror.Validation("requester ID is required", domerrors.ErrInvalidUserID)
	}
	if _, err := uuid.Parse(requesterID); err != nil {
		return nil, apperror.Validation("invalid requester ID format", domerrors.ErrInvalidUserID)
	}
	if targetID == "" {
		return nil, apperror.Validation("target user ID is required", domerrors.ErrInvalidUserID)
	}
	if _, err := uuid.Parse(targetID); err != nil {
		return nil, apperror.Validation("invalid target user ID format", domerrors.ErrInvalidUserID)
	}
	if requesterID == targetID {
		return nil, apperror.Validation("cannot start a conversation with yourself", domerrors.ErrSelfConversation)
	}

	canChat, err := uc.followChecker.AreFollowing(ctx, requesterID, targetID)
	if err != nil {
		return nil, apperror.Internal("failed to check follow relationship", err)
	}
	if !canChat {
		return nil, apperror.Forbidden("users must follow each other to chat", domerrors.ErrUsersCannotChat)
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
