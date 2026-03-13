package handler

import (
	"time"
)

// JSONAPIResource is a base structure for JSON:API resources.
type JSONAPIResource struct {
	ID   string `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	Type string `json:"type" example:"conversations"`
}

// JSONAPILinks represents JSON:API links object.
type JSONAPILinks struct {
	Self string `json:"self" example:"/api/v1/conversations/550e8400-e29b-41d4-a716-446655440000"`
}

// JSONAPIMeta represents JSON:API meta information for pagination.
type JSONAPIMeta struct {
	Total  int `json:"total" example:"42"`
	Limit  int `json:"limit" example:"20"`
	Offset int `json:"offset" example:"0"`
}

// JSONAPIError represents a JSON:API error object.
type JSONAPIError struct {
	Status string `json:"status" example:"404"`
	Title  string `json:"title" example:"Not Found"`
	Detail string `json:"detail" example:"conversation not found"`
}

// JSONAPIErrorsResponse is the standard error response for JSON:API.
type JSONAPIErrorsResponse struct {
	Errors []JSONAPIError `json:"errors"`
}

// ConversationAttributes represents the attributes of a conversation.
type ConversationAttributes struct {
	User1ID       string     `json:"user1_id" example:"550e8400-e29b-41d4-a716-446655440001"`
	User2ID       string     `json:"user2_id" example:"550e8400-e29b-41d4-a716-446655440002"`
	CreatedAt     time.Time  `json:"created_at" example:"2025-01-15T10:30:00Z"`
	LastMessageAt *time.Time `json:"last_message_at,omitempty" example:"2025-01-15T14:20:00Z"`
}

// ConversationResponse is a JSON:API single resource response for conversations.
type ConversationResponse struct {
	Data struct {
		JSONAPIResource
		Attributes ConversationAttributes `json:"attributes"`
		Links      JSONAPILinks           `json:"links"`
	} `json:"data"`
}

// ConversationListResponse is a JSON:API collection response for conversations.
type ConversationListResponse struct {
	Data []struct {
		JSONAPIResource
		Attributes ConversationAttributes `json:"attributes"`
		Links      JSONAPILinks           `json:"links"`
	} `json:"data"`
	Meta  JSONAPIMeta `json:"meta"`
	Links struct {
		Self  string `json:"self" example:"/api/v1/conversations?page[limit]=20&page[offset]=0"`
		First string `json:"first" example:"/api/v1/conversations?page[limit]=20&page[offset]=0"`
		Next  string `json:"next,omitempty" example:"/api/v1/conversations?page[limit]=20&page[offset]=20"`
		Prev  string `json:"prev,omitempty" example:"/api/v1/conversations?page[limit]=20&page[offset]=0"`
	} `json:"links"`
}

// MessageAttributes represents the attributes of a message.
type MessageAttributes struct {
	ConversationID string    `json:"conversation_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	SenderID       string    `json:"sender_id" example:"550e8400-e29b-41d4-a716-446655440001"`
	Type           string    `json:"type" example:"text"`
	Content        *string   `json:"content,omitempty" example:"Hello! How are you?"`
	POIID          *string   `json:"poi_id,omitempty" example:"550e8400-e29b-41d4-a716-446655440003"`
	ShareIntent    *string   `json:"share_intent,omitempty" example:"visit"`
	Status         string    `json:"status" example:"sent"`
	CreatedAt      time.Time `json:"created_at" example:"2025-01-15T14:20:00Z"`
}

// MessageResponse is a JSON:API single resource response for messages.
type MessageResponse struct {
	Data struct {
		JSONAPIResource
		Attributes MessageAttributes `json:"attributes"`
		Links      JSONAPILinks      `json:"links"`
	} `json:"data"`
}

// MessageListResponse is a JSON:API collection response for messages.
type MessageListResponse struct {
	Data []struct {
		JSONAPIResource
		Attributes MessageAttributes `json:"attributes"`
		Links      JSONAPILinks      `json:"links"`
	} `json:"data"`
	Meta  JSONAPIMeta `json:"meta"`
	Links struct {
		Self  string `json:"self" example:"/api/v1/conversations/550e8400-e29b-41d4-a716-446655440000/messages?page[limit]=50&page[offset]=0"`
		First string `json:"first" example:"/api/v1/conversations/550e8400-e29b-41d4-a716-446655440000/messages?page[limit]=50&page[offset]=0"`
		Next  string `json:"next,omitempty"`
		Prev  string `json:"prev,omitempty"`
	} `json:"links"`
}

// HealthJSONAPIResponse is a JSON:API response for health endpoint.
type HealthJSONAPIResponse struct {
	Data struct {
		JSONAPIResource
		Attributes struct {
			Status string `json:"status" example:"ok"`
		} `json:"attributes"`
	} `json:"data"`
}
