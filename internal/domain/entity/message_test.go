package entity

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---- NewTextMessage ----

func TestNewTextMessageSetsType(t *testing.T) {
	msg := NewTextMessage(uuid.New().String(), uuid.New().String(), "hello")
	assert.Equal(t, MessageTypeText, msg.Type)
}

func TestNewTextMessageSetsContent(t *testing.T) {
	msg := NewTextMessage(uuid.New().String(), uuid.New().String(), "hello world")
	require.NotNil(t, msg.Content)
	assert.Equal(t, "hello world", *msg.Content)
}

func TestNewTextMessageIDIsValidUUID(t *testing.T) {
	msg := NewTextMessage(uuid.New().String(), uuid.New().String(), "hi")
	_, err := uuid.Parse(msg.ID)
	assert.NoError(t, err)
}

func TestNewTextMessageStatusIsSent(t *testing.T) {
	msg := NewTextMessage(uuid.New().String(), uuid.New().String(), "hi")
	assert.Equal(t, MessageStatusSent, msg.Status)
}

func TestNewTextMessageNoPOIID(t *testing.T) {
	msg := NewTextMessage(uuid.New().String(), uuid.New().String(), "hi")
	assert.Nil(t, msg.POIID)
}

func TestNewTextMessageNoShareIntent(t *testing.T) {
	msg := NewTextMessage(uuid.New().String(), uuid.New().String(), "hi")
	assert.Nil(t, msg.ShareIntent)
}

func TestNewTextMessageCreatedAtNotZero(t *testing.T) {
	msg := NewTextMessage(uuid.New().String(), uuid.New().String(), "hi")
	assert.False(t, msg.CreatedAt.IsZero())
}

// ---- NewPOIShareMessage ----

func TestNewPOIShareMessageSetsType(t *testing.T) {
	msg := NewPOIShareMessage(uuid.New().String(), uuid.New().String(), uuid.New().String(), "", "")
	assert.Equal(t, MessageTypePOIShare, msg.Type)
}

func TestNewPOIShareMessageSetsPOIID(t *testing.T) {
	poiID := uuid.New().String()
	msg := NewPOIShareMessage(uuid.New().String(), uuid.New().String(), poiID, "", "")
	require.NotNil(t, msg.POIID)
	assert.Equal(t, poiID, *msg.POIID)
}

func TestNewPOIShareMessageWithTitle(t *testing.T) {
	msg := NewPOIShareMessage(uuid.New().String(), uuid.New().String(), uuid.New().String(), "La Sagrada Família", "")
	require.NotNil(t, msg.Content)
	assert.Equal(t, "La Sagrada Família", *msg.Content)
}

func TestNewPOIShareMessageEmptyTitleIsNilContent(t *testing.T) {
	msg := NewPOIShareMessage(uuid.New().String(), uuid.New().String(), uuid.New().String(), "", "")
	assert.Nil(t, msg.Content)
}

func TestNewPOIShareMessageEmptyShareIntentIsNil(t *testing.T) {
	msg := NewPOIShareMessage(uuid.New().String(), uuid.New().String(), uuid.New().String(), "", "")
	assert.Nil(t, msg.ShareIntent)
}

func TestNewPOIShareMessageWithShareIntents(t *testing.T) {
	intents := []string{
		ShareIntentMustGo,
		ShareIntentComeWithMe,
		ShareIntentInvite,
		ShareIntentInviteMe,
	}
	for _, intent := range intents {
		t.Run(intent, func(t *testing.T) {
			msg := NewPOIShareMessage(uuid.New().String(), uuid.New().String(), uuid.New().String(), "Plaza Mayor", intent)
			require.NotNil(t, msg.ShareIntent)
			assert.Equal(t, intent, *msg.ShareIntent)
		})
	}
}

func TestNewPOIShareMessageStatusIsSent(t *testing.T) {
	msg := NewPOIShareMessage(uuid.New().String(), uuid.New().String(), uuid.New().String(), "", "")
	assert.Equal(t, MessageStatusSent, msg.Status)
}

func TestNewPOIShareMessageIDIsValidUUID(t *testing.T) {
	msg := NewPOIShareMessage(uuid.New().String(), uuid.New().String(), uuid.New().String(), "", "")
	_, err := uuid.Parse(msg.ID)
	assert.NoError(t, err)
}

func TestNewPOIShareMessageCreatedAtNotZero(t *testing.T) {
	msg := NewPOIShareMessage(uuid.New().String(), uuid.New().String(), uuid.New().String(), "", "")
	assert.False(t, msg.CreatedAt.IsZero())
}

// ---- ShareIntent constants ----

func TestShareIntentConstantsHaveExpectedValues(t *testing.T) {
	assert.Equal(t, "must_go", ShareIntentMustGo)
	assert.Equal(t, "come_with_me", ShareIntentComeWithMe)
	assert.Equal(t, "invite", ShareIntentInvite)
	assert.Equal(t, "invite_me", ShareIntentInviteMe)
}
