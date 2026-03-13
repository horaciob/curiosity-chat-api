package response

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/google/jsonapi"
	"github.com/google/uuid"
	"github.com/horaciobranciforte/curiosity-chat-api/internal/domain/entity"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParsePaginationDefaults(t *testing.T) {
	q := url.Values{}
	result := ParsePagination(q, 20, 100)

	assert.Equal(t, 20, result.Limit)
	assert.Equal(t, 0, result.Offset)
}

func TestParsePaginationCustomValues(t *testing.T) {
	q := url.Values{
		"page[limit]":  []string{"50"},
		"page[offset]": []string{"100"},
	}
	result := ParsePagination(q, 20, 100)

	assert.Equal(t, 50, result.Limit)
	assert.Equal(t, 100, result.Offset)
}

func TestParsePaginationLimitExceedsMax(t *testing.T) {
	q := url.Values{
		"page[limit]": []string{"200"},
	}
	result := ParsePagination(q, 20, 100)

	assert.Equal(t, 100, result.Limit)
}

func TestParsePaginationNegativeLimit(t *testing.T) {
	q := url.Values{
		"page[limit]": []string{"-10"},
	}
	result := ParsePagination(q, 20, 100)

	assert.Equal(t, 20, result.Limit)
}

func TestParsePaginationNegativeOffset(t *testing.T) {
	q := url.Values{
		"page[offset]": []string{"-50"},
	}
	result := ParsePagination(q, 20, 100)

	assert.Equal(t, 0, result.Offset)
}

func TestParsePaginationInvalidValues(t *testing.T) {
	q := url.Values{
		"page[limit]":  []string{"invalid"},
		"page[offset]": []string{"invalid"},
	}
	result := ParsePagination(q, 20, 100)

	assert.Equal(t, 20, result.Limit)
	assert.Equal(t, 0, result.Offset)
}

func TestParsePaginationZeroLimit(t *testing.T) {
	q := url.Values{
		"page[limit]": []string{"0"},
	}
	result := ParsePagination(q, 20, 100)

	assert.Equal(t, 20, result.Limit)
}

func TestParsePaginationLargeOffset(t *testing.T) {
	q := url.Values{
		"page[offset]": []string{"999999"},
	}
	result := ParsePagination(q, 20, 100)

	assert.Equal(t, 999999, result.Offset)
}

func TestParsePaginationDifferentDefaults(t *testing.T) {
	q := url.Values{}
	result := ParsePagination(q, 50, 200)

	assert.Equal(t, 50, result.Limit)
}

func TestSuccessWritesJSONAPIResponse(t *testing.T) {
	w := httptest.NewRecorder()
	model := &ConversationResponse{
		ID:        uuid.New().String(),
		User1ID:   uuid.New().String(),
		User2ID:   uuid.New().String(),
		CreatedAt: time.Now().UTC().Format("2006-01-02T15:04:05Z07:00"),
	}

	Success(w, model)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, jsonapi.MediaType, w.Header().Get("Content-Type"))
	assert.Contains(t, w.Body.String(), "conversations")
}

func TestCreatedWritesJSONAPIResponse(t *testing.T) {
	w := httptest.NewRecorder()
	model := &ConversationResponse{
		ID:        uuid.New().String(),
		User1ID:   uuid.New().String(),
		User2ID:   uuid.New().String(),
		CreatedAt: time.Now().UTC().Format("2006-01-02T15:04:05Z07:00"),
	}

	Created(w, model)

	assert.Equal(t, http.StatusCreated, w.Code)
	assert.Equal(t, jsonapi.MediaType, w.Header().Get("Content-Type"))
	assert.Contains(t, w.Body.String(), "conversations")
}

func TestCollectionWritesPaginatedResponse(t *testing.T) {
	w := httptest.NewRecorder()
	convs := []*ConversationResponse{
		{
			ID:        uuid.New().String(),
			User1ID:   uuid.New().String(),
			User2ID:   uuid.New().String(),
			CreatedAt: time.Now().UTC().Format("2006-01-02T15:04:05Z07:00"),
		},
	}

	Collection(w, convs, 100, 20, 0, "/api/v1/conversations")

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, jsonapi.MediaType, w.Header().Get("Content-Type"))

	body := w.Body.String()
	assert.Contains(t, body, "conversations")
	assert.Contains(t, body, "meta")
	assert.Contains(t, body, "links")
}

func TestCollectionWithFirstPageOnly(t *testing.T) {
	w := httptest.NewRecorder()
	convs := []*ConversationResponse{
		{
			ID:        uuid.New().String(),
			User1ID:   uuid.New().String(),
			User2ID:   uuid.New().String(),
			CreatedAt: time.Now().UTC().Format("2006-01-02T15:04:05Z07:00"),
		},
	}

	// First page, no prev link
	Collection(w, convs, 100, 20, 0, "/api/v1/conversations")

	body := w.Body.String()
	assert.Contains(t, body, `"first"`)
	assert.Contains(t, body, `"next"`)
	assert.NotContains(t, body, `"prev"`)
}

func TestCollectionWithMiddlePage(t *testing.T) {
	w := httptest.NewRecorder()
	convs := []*ConversationResponse{
		{
			ID:        uuid.New().String(),
			User1ID:   uuid.New().String(),
			User2ID:   uuid.New().String(),
			CreatedAt: time.Now().UTC().Format("2006-01-02T15:04:05Z07:00"),
		},
	}

	// Middle page, has both prev and next
	Collection(w, convs, 100, 20, 20, "/api/v1/conversations")

	body := w.Body.String()
	assert.Contains(t, body, `"first"`)
	assert.Contains(t, body, `"prev"`)
	assert.Contains(t, body, `"next"`)
}

func TestCollectionWithLastPage(t *testing.T) {
	w := httptest.NewRecorder()
	convs := []*ConversationResponse{
		{
			ID:        uuid.New().String(),
			User1ID:   uuid.New().String(),
			User2ID:   uuid.New().String(),
			CreatedAt: time.Now().UTC().Format("2006-01-02T15:04:05Z07:00"),
		},
	}

	// Last page, no next link
	Collection(w, convs, 25, 20, 20, "/api/v1/conversations")

	body := w.Body.String()
	assert.Contains(t, body, `"first"`)
	assert.Contains(t, body, `"prev"`)
	assert.NotContains(t, body, `"next"`)
}

func TestCollectionWithNegativePrevOffset(t *testing.T) {
	w := httptest.NewRecorder()
	convs := []*ConversationResponse{
		{
			ID:        uuid.New().String(),
			User1ID:   uuid.New().String(),
			User2ID:   uuid.New().String(),
			CreatedAt: time.Now().UTC().Format("2006-01-02T15:04:05Z07:00"),
		},
	}

	// offset=5, limit=20 would make prevOffset = -15, should be clamped to 0
	Collection(w, convs, 100, 20, 5, "/api/v1/conversations")

	body := w.Body.String()
	assert.Contains(t, body, `"prev"`)
	assert.Contains(t, body, "page[offset]=0")
}

func TestErrorWritesJSONAPIErrorResponse(t *testing.T) {
	w := httptest.NewRecorder()

	Error(w, http.StatusNotFound, "Not Found", "resource not found")

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Equal(t, jsonapi.MediaType, w.Header().Get("Content-Type"))

	body := w.Body.String()
	assert.Contains(t, body, `"status":"404"`)
	assert.Contains(t, body, `"title":"Not Found"`)
	assert.Contains(t, body, `"detail":"resource not found"`)
}

func TestErrorWithDifferentStatusCodes(t *testing.T) {
	tests := []struct {
		name   string
		status int
		title  string
		detail string
	}{
		{"bad request", http.StatusBadRequest, "Bad Request", "invalid input"},
		{"unauthorized", http.StatusUnauthorized, "Unauthorized", "missing token"},
		{"forbidden", http.StatusForbidden, "Forbidden", "access denied"},
		{"not found", http.StatusNotFound, "Not Found", "not found"},
		{"conflict", http.StatusConflict, "Conflict", "already exists"},
		{"internal", http.StatusInternalServerError, "Internal Error", "server error"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			Error(w, tt.status, tt.title, tt.detail)

			assert.Equal(t, tt.status, w.Code)

			var payload jsonapi.ErrorsPayload
			err := json.NewDecoder(w.Body).Decode(&payload)
			require.NoError(t, err)
			require.Len(t, payload.Errors, 1)
			assert.Equal(t, tt.title, payload.Errors[0].Title)
			assert.Equal(t, tt.detail, payload.Errors[0].Detail)
		})
	}
}

func TestJSONWritesPlainJSONResponse(t *testing.T) {
	w := httptest.NewRecorder()
	data := map[string]string{"status": "ok"}

	JSON(w, http.StatusOK, data)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

	var result map[string]string
	err := json.NewDecoder(w.Body).Decode(&result)
	require.NoError(t, err)
	assert.Equal(t, "ok", result["status"])
}

func TestJSONWithDifferentStatusCodes(t *testing.T) {
	w := httptest.NewRecorder()
	data := map[string]interface{}{"count": 42}

	JSON(w, http.StatusCreated, data)

	assert.Equal(t, http.StatusCreated, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
}

func TestNewConversationResponse(t *testing.T) {
	now := time.Now().UTC()
	conv := &entity.Conversation{
		ID:            uuid.New().String(),
		User1ID:       uuid.New().String(),
		User2ID:       uuid.New().String(),
		CreatedAt:     now,
		LastMessageAt: nil,
	}

	resp := NewConversationResponse(conv)

	assert.Equal(t, conv.ID, resp.ID)
	assert.Equal(t, conv.User1ID, resp.User1ID)
	assert.Equal(t, conv.User2ID, resp.User2ID)
	assert.Equal(t, conv.CreatedAt.Format("2006-01-02T15:04:05Z07:00"), resp.CreatedAt)
	assert.Nil(t, resp.LastMessageAt)
	assert.NotNil(t, resp.Links)
	assert.Contains(t, (*resp.Links)["self"], conv.ID)
}

func TestNewConversationResponseWithLastMessage(t *testing.T) {
	now := time.Now().UTC()
	lastMsg := now.Add(time.Hour)
	conv := &entity.Conversation{
		ID:            uuid.New().String(),
		User1ID:       uuid.New().String(),
		User2ID:       uuid.New().String(),
		CreatedAt:     now,
		LastMessageAt: &lastMsg,
	}

	resp := NewConversationResponse(conv)

	assert.NotNil(t, resp.LastMessageAt)
	assert.Equal(t, lastMsg.Format("2006-01-02T15:04:05Z07:00"), *resp.LastMessageAt)
}

func TestNewConversationListResponse(t *testing.T) {
	convs := []*entity.Conversation{
		{
			ID:        uuid.New().String(),
			User1ID:   uuid.New().String(),
			User2ID:   uuid.New().String(),
			CreatedAt: time.Now().UTC(),
		},
		{
			ID:        uuid.New().String(),
			User1ID:   uuid.New().String(),
			User2ID:   uuid.New().String(),
			CreatedAt: time.Now().UTC(),
		},
	}

	resp := NewConversationListResponse(convs)

	assert.Len(t, resp, 2)
	assert.Equal(t, convs[0].ID, resp[0].ID)
	assert.Equal(t, convs[1].ID, resp[1].ID)
}

func TestNewConversationListResponseEmpty(t *testing.T) {
	resp := NewConversationListResponse([]*entity.Conversation{})

	assert.Empty(t, resp)
	assert.NotNil(t, resp)
}

func TestNewMessageResponse(t *testing.T) {
	content := "Hello!"
	msg := &entity.Message{
		ID:             uuid.New().String(),
		ConversationID: uuid.New().String(),
		SenderID:       uuid.New().String(),
		Type:           entity.MessageTypeText,
		Content:        &content,
		POIID:          nil,
		ShareIntent:    nil,
		Status:         entity.MessageStatusSent,
		CreatedAt:      time.Now().UTC(),
	}

	resp := NewMessageResponse(msg)

	assert.Equal(t, msg.ID, resp.ID)
	assert.Equal(t, msg.ConversationID, resp.ConversationID)
	assert.Equal(t, msg.SenderID, resp.SenderID)
	assert.Equal(t, msg.Type, resp.Type)
	assert.Equal(t, *msg.Content, *resp.Content)
	assert.Nil(t, resp.POIID)
	assert.Nil(t, resp.ShareIntent)
	assert.Equal(t, msg.Status, resp.Status)
	assert.NotNil(t, resp.Links)
	assert.Contains(t, (*resp.Links)["self"], msg.ID)
}

func TestNewMessageResponseWithPOI(t *testing.T) {
	poiID := uuid.New().String()
	shareIntent := entity.ShareIntentMustGo
	msg := &entity.Message{
		ID:             uuid.New().String(),
		ConversationID: uuid.New().String(),
		SenderID:       uuid.New().String(),
		Type:           entity.MessageTypePOIShare,
		Content:        nil,
		POIID:          &poiID,
		ShareIntent:    &shareIntent,
		Status:         entity.MessageStatusDelivered,
		CreatedAt:      time.Now().UTC(),
	}

	resp := NewMessageResponse(msg)

	assert.Equal(t, entity.MessageTypePOIShare, resp.Type)
	assert.Equal(t, poiID, *resp.POIID)
	assert.Equal(t, shareIntent, *resp.ShareIntent)
	assert.Nil(t, resp.Content)
}

func TestNewMessageListResponse(t *testing.T) {
	content := "Hello!"
	msgs := []*entity.Message{
		{
			ID:             uuid.New().String(),
			ConversationID: uuid.New().String(),
			SenderID:       uuid.New().String(),
			Type:           entity.MessageTypeText,
			Content:        &content,
			Status:         entity.MessageStatusSent,
			CreatedAt:      time.Now().UTC(),
		},
		{
			ID:             uuid.New().String(),
			ConversationID: uuid.New().String(),
			SenderID:       uuid.New().String(),
			Type:           entity.MessageTypeText,
			Content:        &content,
			Status:         entity.MessageStatusRead,
			CreatedAt:      time.Now().UTC(),
		},
	}

	resp := NewMessageListResponse(msgs)

	assert.Len(t, resp, 2)
	assert.Equal(t, msgs[0].ID, resp[0].ID)
	assert.Equal(t, msgs[1].ID, resp[1].ID)
}

func TestNewMessageListResponseEmpty(t *testing.T) {
	resp := NewMessageListResponse([]*entity.Message{})

	assert.Empty(t, resp)
	assert.NotNil(t, resp)
}
