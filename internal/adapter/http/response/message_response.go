package response

import (
	"fmt"

	"github.com/google/jsonapi"
	"github.com/horaciobranciforte/curiosity-chat-api/internal/domain/entity"
)

// MessageResponse is the JSON:API DTO for a message.
type MessageResponse struct {
	ID             string  `jsonapi:"primary,messages"`
	ConversationID string  `jsonapi:"attr,conversation_id"`
	SenderID       string  `jsonapi:"attr,sender_id"`
	Type           string  `jsonapi:"attr,type"`
	Content        *string `jsonapi:"attr,content,omitempty"`
	POIID          *string `jsonapi:"attr,poi_id,omitempty"`
	Status         string  `jsonapi:"attr,status"`
	CreatedAt      string  `jsonapi:"attr,created_at"`
	Links          *jsonapi.Links
}

// NewMessageResponse converts an entity to a JSON:API DTO.
func NewMessageResponse(m *entity.Message) *MessageResponse {
	return &MessageResponse{
		ID:             m.ID,
		ConversationID: m.ConversationID,
		SenderID:       m.SenderID,
		Type:           m.Type,
		Content:        m.Content,
		POIID:          m.POIID,
		Status:         m.Status,
		CreatedAt:      m.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		Links: &jsonapi.Links{
			"self": fmt.Sprintf("/api/v1/conversations/%s/messages/%s", m.ConversationID, m.ID),
		},
	}
}

// NewMessageListResponse converts a slice of entities to JSON:API DTOs.
func NewMessageListResponse(msgs []*entity.Message) []*MessageResponse {
	dtos := make([]*MessageResponse, 0, len(msgs))
	for _, m := range msgs {
		dtos = append(dtos, NewMessageResponse(m))
	}
	return dtos
}
