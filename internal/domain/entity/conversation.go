package entity

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

// ErrNotParticipant is returned when a user is not a participant in the conversation
var ErrNotParticipant = errors.New("user is not a participant in this conversation")

// Conversation represents a 1-on-1 chat between two users.
// user1_id is always lexicographically smaller than user2_id to ensure uniqueness.
type Conversation struct {
	ID            string
	User1ID       string
	User2ID       string
	CreatedAt     time.Time
	LastMessageAt *time.Time
}

// NewConversation creates a new Conversation, normalizing so user1_id < user2_id.
func NewConversation(userA, userB string) *Conversation {
	u1, u2 := userA, userB
	if u1 > u2 {
		u1, u2 = u2, u1
	}
	return &Conversation{
		ID:        uuid.New().String(),
		User1ID:   u1,
		User2ID:   u2,
		CreatedAt: time.Now().UTC(),
	}
}

// HasParticipant returns true if userID is one of the two participants.
func (c *Conversation) HasParticipant(userID string) bool {
	return c.User1ID == userID || c.User2ID == userID
}

// OtherUserID returns the ID of the participant that is not myID.
// Returns an error if myID is not a participant in the conversation.
func (c *Conversation) OtherUserID(myID string) (string, error) {
	if c.User1ID == myID {
		return c.User2ID, nil
	}
	if c.User2ID == myID {
		return c.User1ID, nil
	}
	return "", ErrNotParticipant
}
