package response

import (
	"fmt"

	"github.com/google/jsonapi"
	"github.com/horaciobranciforte/curiosity-chat-api/internal/domain/entity"
)

// ConversationResponse is the JSON:API DTO for a conversation.
type ConversationResponse struct {
	ID            string  `jsonapi:"primary,conversations"`
	User1ID       string  `jsonapi:"attr,user1_id"`
	User2ID       string  `jsonapi:"attr,user2_id"`
	CreatedAt     string  `jsonapi:"attr,created_at"`
	LastMessageAt *string `jsonapi:"attr,last_message_at,omitempty"`
	Links         *jsonapi.Links
}

// NewConversationResponse converts an entity to a JSON:API DTO.
func NewConversationResponse(c *entity.Conversation) *ConversationResponse {
	dto := &ConversationResponse{
		ID:        c.ID,
		User1ID:   c.User1ID,
		User2ID:   c.User2ID,
		CreatedAt: c.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		Links: &jsonapi.Links{
			"self": fmt.Sprintf("/api/v1/conversations/%s", c.ID),
		},
	}
	if c.LastMessageAt != nil {
		t := c.LastMessageAt.Format("2006-01-02T15:04:05Z07:00")
		dto.LastMessageAt = &t
	}
	return dto
}

// NewConversationListResponse converts a slice of entities to JSON:API DTOs.
func NewConversationListResponse(convs []*entity.Conversation) []*ConversationResponse {
	dtos := make([]*ConversationResponse, 0, len(convs))
	for _, c := range convs {
		dtos = append(dtos, NewConversationResponse(c))
	}
	return dtos
}
