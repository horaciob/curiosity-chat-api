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
	Content        *string // non-nil for type=text
	POIID          *string // non-nil for type=poi_share
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
func NewPOIShareMessage(conversationID, senderID, poiID string) *Message {
	p := poiID
	return &Message{
		ID:             uuid.New().String(),
		ConversationID: conversationID,
		SenderID:       senderID,
		Type:           MessageTypePOIShare,
		POIID:          &p,
		Status:         MessageStatusSent,
		CreatedAt:      time.Now().UTC(),
	}
}
