package handler

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/horaciobranciforte/curiosity-chat-api/internal/adapter/http/middleware"
	"github.com/horaciobranciforte/curiosity-chat-api/internal/adapter/http/response"
	"github.com/horaciobranciforte/curiosity-chat-api/internal/infrastructure/logger"
	"github.com/horaciobranciforte/curiosity-chat-api/internal/usecase/message"
	"github.com/horaciobranciforte/curiosity-chat-api/internal/ws"
	"go.uber.org/zap"
)

// MessageHandler handles message endpoints.
type MessageHandler struct {
	sendMessageUC *message.SendMessage
	getMessagesUC *message.GetMessages
	hub           *ws.Hub
	convRepo      message.ConversationRepository
	msgRepo       message.Repository
}

// NewMessageHandler creates a new MessageHandler.
func NewMessageHandler(
	sendMessageUC *message.SendMessage,
	getMessagesUC *message.GetMessages,
	hub *ws.Hub,
	convRepo message.ConversationRepository,
	msgRepo message.Repository,
) *MessageHandler {
	return &MessageHandler{
		sendMessageUC: sendMessageUC,
		getMessagesUC: getMessagesUC,
		hub:           hub,
		convRepo:      convRepo,
		msgRepo:       msgRepo,
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

	// Broadcast the new message via WebSocket to both participants.
	// Use a background context — the request context is cancelled once the response is written.
	broadcastCtx, cancelBroadcast := context.WithTimeout(context.Background(), wsOperationTimeout)
	defer cancelBroadcast()

	outgoing := ws.OutgoingMessage{
		ID:             msg.ID,
		Type:           msg.Type,
		ConversationID: msg.ConversationID,
		SenderID:       msg.SenderID,
		Content:        msg.Content,
		POIID:          msg.POIID,
		ShareIntent:    msg.ShareIntent,
		Status:         msg.Status,
		CreatedAt:      msg.CreatedAt,
	}

	// Echo to sender for multi-device support
	h.hub.BroadcastTo(senderID, outgoing)

	// Resolve recipient
	conv, convErr := h.convRepo.GetByID(broadcastCtx, msg.ConversationID)
	if convErr != nil {
		logger.Warn("[HANDLER] Failed to resolve conversation for WS broadcast",
			zap.String("conversation_id", msg.ConversationID),
			zap.Error(convErr),
		)
		return
	}
	otherUserID, otherErr := conv.OtherUserID(senderID)
	if otherErr != nil {
		logger.Warn("[HANDLER] Failed to get other participant for WS broadcast",
			zap.String("conversation_id", msg.ConversationID),
			zap.String("sender_id", senderID),
			zap.Error(otherErr),
		)
		return
	}

	// Push new message to recipient — this is what makes their screen refresh
	h.hub.BroadcastTo(otherUserID, outgoing)
	logger.Info("[HANDLER] WS broadcast sent to recipient",
		zap.String("conversation_id", msg.ConversationID),
		zap.String("recipient_id", otherUserID),
		zap.String("message_id", msg.ID),
	)

	// Immediate delivery confirmation if recipient is online
	if h.hub.IsOnline(otherUserID) {
		if updateErr := h.msgRepo.UpdateStatus(broadcastCtx, msg.ID, "delivered"); updateErr == nil {
			h.hub.BroadcastJSON(senderID, ws.DeliveredEvent{
				Type:           "delivered",
				ConversationID: msg.ConversationID,
				MessageID:      msg.ID,
			})
		}
	}

	// Broadcast unread count update to recipient
	unreadCtx, cancelUnread := context.WithTimeout(context.Background(), wsOperationTimeout)
	defer cancelUnread()
	convUnread, convUnreadErr := h.msgRepo.CountUnreadByConversationForUser(unreadCtx, msg.ConversationID, otherUserID)
	if convUnreadErr == nil {
		totalUnread, _ := h.msgRepo.CountTotalUnreadForUser(unreadCtx, otherUserID)
		h.hub.BroadcastJSON(otherUserID, ws.UnreadCountEvent{
			Type:           "unread_count_update",
			ConversationID: msg.ConversationID,
			UnreadCount:    convUnread,
			TotalUnread:    totalUnread,
		})
	}
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
