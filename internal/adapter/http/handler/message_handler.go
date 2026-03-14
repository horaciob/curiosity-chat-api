package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/horaciobranciforte/curiosity-chat-api/internal/adapter/http/middleware"
	"github.com/horaciobranciforte/curiosity-chat-api/internal/adapter/http/response"
	"github.com/horaciobranciforte/curiosity-chat-api/internal/infrastructure/logger"
	"github.com/horaciobranciforte/curiosity-chat-api/internal/usecase/message"
	"go.uber.org/zap"
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
	Type        string `json:"type"`
	Content     string `json:"content,omitempty"`
	POIID       string `json:"poi_id,omitempty"`
	ShareIntent string `json:"share_intent,omitempty"`
}

// Send handles POST /api/v1/conversations/{id}/messages.
//
//	@Summary		Send a message
//	@Description	Sends a text message or POI share to the conversation. The authenticated user must be a participant. Returns JSON:API compliant response.
//	@Tags			messages
//	@Accept			json
//	@Produce		json
//	@Param			id		path		string					true	"Conversation ID"
//	@Param			body	body		sendMessageRequest		true	"Message payload"
//	@Success		201		{object}	MessageResponse		"Message created"
//	@Failure		400		{object}	JSONAPIErrorsResponse	"Bad request"
//	@Failure		401		{object}	JSONAPIErrorsResponse	"Unauthorized"
//	@Failure		403		{object}	JSONAPIErrorsResponse	"Forbidden — not a participant"
//	@Failure		404		{object}	JSONAPIErrorsResponse	"Conversation not found"
//	@Failure		500		{object}	JSONAPIErrorsResponse	"Internal server error"
//	@Security		BearerAuth
//	@Router			/conversations/{id}/messages [post]
func (h *MessageHandler) Send(w http.ResponseWriter, r *http.Request) {
	conversationID := chi.URLParam(r, "id")
	senderID := middleware.UserIDFromContext(r.Context())

	logger.Info("[HANDLER] Send message endpoint called",
		zap.String("conversation_id", conversationID),
		zap.String("sender_id", senderID),
		zap.String("method", r.Method),
		zap.String("path", r.URL.Path),
	)

	var req sendMessageRequest
	if !decodeRequestBody(w, r, &req) {
		logger.Error("[HANDLER] Failed to decode request body",
			zap.String("conversation_id", conversationID),
			zap.String("sender_id", senderID),
		)
		return
	}

	logger.Info("[HANDLER] Request body decoded",
		zap.String("conversation_id", conversationID),
		zap.String("sender_id", senderID),
		zap.String("message_type", req.Type),
	)

	msg, err := h.sendMessageUC.Execute(r.Context(), conversationID, senderID, message.SendMessageInput{
		Type:        req.Type,
		Content:     req.Content,
		POIID:       req.POIID,
		ShareIntent: req.ShareIntent,
	})
	if err != nil {
		logger.Error("[HANDLER] SendMessage use case failed",
			zap.String("conversation_id", conversationID),
			zap.String("sender_id", senderID),
			zap.Error(err),
		)
		handleUseCaseError(w, err)
		return
	}

	logger.Info("[HANDLER] Message sent successfully",
		zap.String("conversation_id", conversationID),
		zap.String("sender_id", senderID),
		zap.String("message_id", msg.ID),
	)

	response.Created(w, response.NewMessageResponse(msg))
}

// List handles GET /api/v1/conversations/{id}/messages.
//
//	@Summary		List messages
//	@Description	Returns paginated message history for a conversation, ordered newest first. The authenticated user must be a participant. Returns JSON:API compliant response with meta and links for pagination.
//	@Tags			messages
//	@Produce		json
//	@Param			id				path		string	true	"Conversation ID"
//	@Param			page[limit]		query		int		false	"Max results (default 50)"
//	@Param			page[offset]	query		int		false	"Offset for pagination"
//	@Success		200				{object}	MessageListResponse	"Paginated list of messages"
//	@Failure		401				{object}	JSONAPIErrorsResponse	"Unauthorized"
//	@Failure		403				{object}	JSONAPIErrorsResponse	"Forbidden — not a participant"
//	@Failure		404				{object}	JSONAPIErrorsResponse	"Conversation not found"
//	@Failure		500				{object}	JSONAPIErrorsResponse	"Internal server error"
//	@Security		BearerAuth
//	@Router			/conversations/{id}/messages [get]
func (h *MessageHandler) List(w http.ResponseWriter, r *http.Request) {
	conversationID := chi.URLParam(r, "id")
	requesterID := middleware.UserIDFromContext(r.Context())
	pagination := response.ParsePagination(r.URL.Query(), message.DefaultMessageLimit, message.MaxMessageLimit)

	msgs, total, err := h.getMessagesUC.Execute(r.Context(), conversationID, requesterID, pagination.Limit, pagination.Offset)
	if err != nil {
		handleUseCaseError(w, err)
		return
	}

	baseURL := "/api/v1/conversations/" + conversationID + "/messages"
	response.Collection(w, response.NewMessageListResponse(msgs), total, pagination.Limit, pagination.Offset, baseURL)
}
