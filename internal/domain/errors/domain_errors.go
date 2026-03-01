package errors

import "errors"

var (
	ErrConversationNotFound = errors.New("conversation not found")
	ErrNotParticipant       = errors.New("user is not a participant in this conversation")
	ErrUsersCannotChat      = errors.New("users must follow each other to chat")
	ErrInvalidMessageType   = errors.New("invalid message type")
	ErrSelfConversation     = errors.New("cannot start a conversation with yourself")
	ErrInvalidUserID        = errors.New("invalid user ID")
	ErrInvalidConversationID = errors.New("invalid conversation ID")
)
