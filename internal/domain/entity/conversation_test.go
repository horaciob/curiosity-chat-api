package entity

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

// ---- NewConversation ----

func TestNewConversationIDIsValidUUID(t *testing.T) {
	conv := NewConversation(uuid.New().String(), uuid.New().String())
	_, err := uuid.Parse(conv.ID)
	assert.NoError(t, err)
}

func TestNewConversationCreatedAtNotZero(t *testing.T) {
	conv := NewConversation(uuid.New().String(), uuid.New().String())
	assert.False(t, conv.CreatedAt.IsZero())
}

func TestNewConversationLastMessageAtIsNil(t *testing.T) {
	conv := NewConversation(uuid.New().String(), uuid.New().String())
	assert.Nil(t, conv.LastMessageAt)
}

func TestNewConversationNormalizesOrderWhenUserAGreater(t *testing.T) {
	// Use fixed strings so we can predict the lexicographic order.
	userA := "zzzzzzzz-0000-0000-0000-000000000000"
	userB := "aaaaaaaa-0000-0000-0000-000000000000"

	conv := NewConversation(userA, userB)

	assert.Equal(t, userB, conv.User1ID, "User1ID should be the lexicographically smaller ID")
	assert.Equal(t, userA, conv.User2ID)
}

func TestNewConversationKeepsOrderWhenUserASmaller(t *testing.T) {
	userA := "aaaaaaaa-0000-0000-0000-000000000000"
	userB := "zzzzzzzz-0000-0000-0000-000000000000"

	conv := NewConversation(userA, userB)

	assert.Equal(t, userA, conv.User1ID)
	assert.Equal(t, userB, conv.User2ID)
}

func TestNewConversationIsDeterministic(t *testing.T) {
	userA := uuid.New().String()
	userB := uuid.New().String()

	conv1 := NewConversation(userA, userB)
	conv2 := NewConversation(userB, userA)

	// Both should produce the same participant pair (different IDs, same users).
	assert.Equal(t, conv1.User1ID, conv2.User1ID)
	assert.Equal(t, conv1.User2ID, conv2.User2ID)
}

// ---- HasParticipant ----

func TestHasParticipantUser1(t *testing.T) {
	userA := uuid.New().String()
	conv := NewConversation(userA, uuid.New().String())
	assert.True(t, conv.HasParticipant(userA))
}

func TestHasParticipantUser2(t *testing.T) {
	userB := uuid.New().String()
	conv := NewConversation(uuid.New().String(), userB)
	assert.True(t, conv.HasParticipant(userB))
}

func TestHasParticipantOutsider(t *testing.T) {
	conv := NewConversation(uuid.New().String(), uuid.New().String())
	assert.False(t, conv.HasParticipant(uuid.New().String()))
}

func TestHasParticipantEmptyString(t *testing.T) {
	conv := NewConversation(uuid.New().String(), uuid.New().String())
	assert.False(t, conv.HasParticipant(""))
}

// ---- OtherUserID ----

func TestOtherUserIDFromUser1ReturnsUser2(t *testing.T) {
	userA := uuid.New().String()
	userB := uuid.New().String()
	conv := NewConversation(userA, userB)

	other := conv.OtherUserID(conv.User1ID)
	assert.Equal(t, conv.User2ID, other)
}

func TestOtherUserIDFromUser2ReturnsUser1(t *testing.T) {
	userA := uuid.New().String()
	userB := uuid.New().String()
	conv := NewConversation(userA, userB)

	other := conv.OtherUserID(conv.User2ID)
	assert.Equal(t, conv.User1ID, other)
}
