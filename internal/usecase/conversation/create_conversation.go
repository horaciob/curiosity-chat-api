package conversation

import (
	"context"

	"github.com/horaciobranciforte/curiosity-chat-api/internal/domain/entity"
	domerrors "github.com/horaciobranciforte/curiosity-chat-api/internal/domain/errors"
	"github.com/horaciobranciforte/curiosity-chat-api/internal/pkg/apperror"
)

// CreateConversation creates or returns an existing conversation between two users.
type CreateConversation struct {
	repo    Repository
	friends FriendChecker // when set, only friends can start a conversation
}

// FriendChecker tells whether two users are friends.
type FriendChecker interface {
	AreFriends(ctx context.Context, a, b string) (bool, error)
}

// WithFriendChecker allows conversations between friends only. Anyone used to be able
// to message any user ID (they travel in invite links and search results).
func (uc *CreateConversation) WithFriendChecker(f FriendChecker) *CreateConversation {
	uc.friends = f
	return uc
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

	if uc.friends != nil {
		ok, err := uc.friends.AreFriends(ctx, requesterID, targetID)
		if err != nil {
			return nil, apperror.Internal("failed to check friendship", err)
		}
		if !ok {
			return nil, apperror.Forbidden("you can only message friends", domerrors.ErrNotParticipant).WithCode("not_friends")
		}
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
		// Both people (or a double tap) creating it at once: the pair is unique, so the
		// loser's insert fails — hand back the one the other request made.
		if again, getErr := uc.repo.GetByParticipants(ctx, requesterID, targetID); getErr == nil && again != nil {
			return again, nil
		}
		return nil, apperror.Internal("failed to create conversation", err)
	}

	return conv, nil
}
