package entity

import (
	"time"

	"github.com/google/uuid"
)

const (
	MessageTypeText     = "text"
	MessageTypePOIShare = "poi_share"
)

const (
	ShareIntentMustGo     = "must_go"
	ShareIntentComeWithMe = "come_with_me"
	ShareIntentInvite     = "invite"
)

const (
	MessageStatusSent      = "sent"
	MessageStatusDelivered = "delivered"
	MessageStatusRead      = "read"
)

// Message represents a single chat message in a conversation.
type Message struct {
	ID             string
	ConversationID string
	SenderID       string
	Type           string
	Content        *string // text content for type=text; POI title for type=poi_share
	POIID          *string // non-nil for type=poi_share
	ShareIntent    *string // "must_go" | "come_with_me" | "invite" — only for type=poi_share
	Status         string
	CreatedAt      time.Time
}

// NewTextMessage creates a text message.
func NewTextMessage(conversationID, senderID, content string) *Message {
	c := content
	return &Message{
		ID:             uuid.New().String(),
		ConversationID: conversationID,
		SenderID:       senderID,
		Type:           MessageTypeText,
		Content:        &c,
		Status:         MessageStatusSent,
		CreatedAt:      time.Now().UTC(),
	}
}

// NewPOIShareMessage creates a poi_share message.
// poiTitle is stored in Content so clients can render the card without a separate API call.
// shareIntent is one of ShareIntentMustGo, ShareIntentComeWithMe, ShareIntentInvite (may be empty).
func NewPOIShareMessage(conversationID, senderID, poiID, poiTitle, shareIntent string) *Message {
	p := poiID
	msg := &Message{
		ID:             uuid.New().String(),
		ConversationID: conversationID,
		SenderID:       senderID,
		Type:           MessageTypePOIShare,
		POIID:          &p,
		Status:         MessageStatusSent,
		CreatedAt:      time.Now().UTC(),
	}
	if poiTitle != "" {
		t := poiTitle
		msg.Content = &t
	}
	if shareIntent != "" {
		s := shareIntent
		msg.ShareIntent = &s
	}
	return msg
}
