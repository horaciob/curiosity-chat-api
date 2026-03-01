package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/horaciobranciforte/curiosity-chat-api/internal/adapter/http/middleware"
	"github.com/horaciobranciforte/curiosity-chat-api/internal/adapter/http/response"
	"github.com/horaciobranciforte/curiosity-chat-api/internal/usecase/message"
)

// MessageHandler handles message endpoints.
type MessageHandler struct {
	sendMessageUC *message.SendMessage
	getMessagesUC *message.GetMessages
}

// NewMessageHandler creates a new MessageHandler.
func NewMessageHandler(
	sendMessageUC *message.SendMessage,
	getMessagesUC *message.GetMessages,
) *MessageHandler {
	return &MessageHandler{
		sendMessageUC: sendMessageUC,
		getMessagesUC: getMessagesUC,
	}
}

type sendMessageRequest struct {
	Type    string `json:"type"`
	Content string `json:"content,omitempty"`
	POIID   string `json:"poi_id,omitempty"`
}

// Send handles POST /api/v1/conversations/{id}/messages.
//
//	@Summary		Send a message
//	@Description	Sends a text message or POI share to the conversation. The authenticated user must be a participant.
//	@Tags			messages
//	@Accept			json
//	@Produce		json
//	@Param			id		path		string				true	"Conversation ID"
//	@Param			body	body		sendMessageRequest	true	"Message payload"
//	@Success		201		{object}	object				"Message created"
//	@Failure		400		{object}	object				"Bad request"
//	@Failure		401		{object}	object				"Unauthorized"
//	@Failure		403		{object}	object				"Forbidden — not a participant"
//	@Failure		404		{object}	object				"Conversation not found"
//	@Failure		500		{object}	object				"Internal server error"
//	@Security		BearerAuth
//	@Router			/conversations/{id}/messages [post]
func (h *MessageHandler) Send(w http.ResponseWriter, r *http.Request) {
	conversationID := chi.URLParam(r, "id")
	senderID := middleware.UserIDFromContext(r.Context())

	var req sendMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "Bad Request", "invalid request body")
		return
	}

	msg, err := h.sendMessageUC.Execute(r.Context(), conversationID, senderID, message.SendMessageInput{
		Type:    req.Type,
		Content: req.Content,
		POIID:   req.POIID,
	})
	if err != nil {
		handleUseCaseError(w, err)
		return
	}

	response.Created(w, response.NewMessageResponse(msg))
}

// List handles GET /api/v1/conversations/{id}/messages.
//
//	@Summary		List messages
//	@Description	Returns paginated message history for a conversation, ordered newest first. The authenticated user must be a participant.
//	@Tags			messages
//	@Produce		json
//	@Param			id				path		string	true	"Conversation ID"
//	@Param			page[limit]		query		int		false	"Max results (default 50)"
//	@Param			page[offset]	query		int		false	"Offset for pagination"
//	@Success		200				{object}	object	"Paginated list of messages"
//	@Failure		401				{object}	object	"Unauthorized"
//	@Failure		403				{object}	object	"Forbidden — not a participant"
//	@Failure		404				{object}	object	"Conversation not found"
//	@Failure		500				{object}	object	"Internal server error"
//	@Security		BearerAuth
//	@Router			/conversations/{id}/messages [get]
func (h *MessageHandler) List(w http.ResponseWriter, r *http.Request) {
	conversationID := chi.URLParam(r, "id")
	requesterID := middleware.UserIDFromContext(r.Context())
	q := r.URL.Query()
	limit, _ := strconv.Atoi(q.Get("page[limit]"))
	offset, _ := strconv.Atoi(q.Get("page[offset]"))
	if limit == 0 {
		limit = 50
	}

	msgs, total, err := h.getMessagesUC.Execute(r.Context(), conversationID, requesterID, limit, offset)
	if err != nil {
		handleUseCaseError(w, err)
		return
	}

	baseURL := "/api/v1/conversations/" + conversationID + "/messages"
	response.Collection(w, response.NewMessageListResponse(msgs), total, limit, offset, baseURL)
}
