package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
	"github.com/horaciobranciforte/curiosity-chat-api/internal/adapter/http/middleware"
	"github.com/horaciobranciforte/curiosity-chat-api/internal/usecase/message"
	"github.com/horaciobranciforte/curiosity-chat-api/internal/ws"
	"go.uber.org/zap"
)

const (
	// WebSocket buffer sizes
	wsReadBufferSize  = 1024
	wsWriteBufferSize = 1024

	// WebSocket operation timeout for DB operations
	wsOperationTimeout = 10 * time.Second
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  wsReadBufferSize,
	WriteBufferSize: wsWriteBufferSize,
	CheckOrigin: func(r *http.Request) bool {
		origin := r.Header.Get("Origin")
		// Allow requests with no origin (non-browser clients)
		if origin == "" {
			return true
		}
		// For production, configure ALLOWED_ORIGINS env var
		// For development, allow localhost
		allowedOrigins := []string{
			"http://localhost:3000",
			"http://localhost:8080",
			"https://localhost:3000",
			"https://localhost:8080",
		}
		for _, allowed := range allowedOrigins {
			if origin == allowed {
				return true
			}
		}
		return false
	},
}

// WSHandler handles WebSocket connections.
type WSHandler struct {
	hub           *ws.Hub
	sendMessageUC *message.SendMessage
	convRepo      message.ConversationRepository
	msgRepo       message.Repository
	jwtValidator  middleware.TokenValidator
	logger        *zap.Logger
}

// NewWSHandler creates a new WSHandler.
func NewWSHandler(
	hub *ws.Hub,
	sendMessageUC *message.SendMessage,
	convRepo message.ConversationRepository,
	msgRepo message.Repository,
	jwtValidator middleware.TokenValidator,
	logger *zap.Logger,
) *WSHandler {
	return &WSHandler{
		hub:           hub,
		sendMessageUC: sendMessageUC,
		convRepo:      convRepo,
		msgRepo:       msgRepo,
		jwtValidator:  jwtValidator,
		logger:        logger,
	}
}

// ServeWS handles GET /api/v1/ws — upgrades the connection to WebSocket.
//
// Authentication flow (token never appears in the URL):
//
//  1. Client connects — no credentials needed in the URL.
//
//  2. Server upgrades to WebSocket immediately.
//
//  3. Client must send {"type":"auth","token":"<jwt>"} within 10 seconds.
//
//  4. Server validates the token. On success it replies {"type":"auth_ok"}.
//
//  5. On failure or timeout the connection is closed with a policy-violation code.
//
//     @Summary		Open a WebSocket connection
//     @Description	Upgrades to WebSocket. The first client frame must be an auth message: {"type":"auth","token":"<jwt>"}
//     @Tags			websocket
//     @Success		101	"Switching Protocols"
//     @Failure		401	"Unauthorized"
//     @Router			/ws [get]
func (h *WSHandler) ServeWS(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		h.logger.Error("websocket upgrade failed", zap.Error(err))
		return
	}

	userID, err := h.authenticate(conn)
	if err != nil {
		h.logger.Warn("websocket auth failed", zap.Error(err))
		if writeErr := conn.WriteMessage(
			websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.ClosePolicyViolation, "authentication failed"),
		); writeErr != nil {
			h.logger.Debug("failed to write close message", zap.Error(writeErr))
		}
		conn.Close()
		return
	}

	client := &ws.Client{
		UserID: userID,
		Conn:   conn,
		Send:   make(chan []byte, ws.ClientSendChanBufferSize),
	}
	h.hub.Register(client)
	go client.WritePump()

	h.readPump(client)
}

// authenticate reads the first WebSocket frame and validates the auth token.
// The client must send {"type":"auth","token":"<jwt>"} within AuthDeadline.
func (h *WSHandler) authenticate(conn *websocket.Conn) (string, error) {
	if err := conn.SetReadDeadline(time.Now().Add(ws.AuthDeadline)); err != nil {
		return "", err
	}
	defer func() {
		if err := conn.SetReadDeadline(time.Time{}); err != nil {
			h.logger.Debug("failed to reset read deadline", zap.Error(err))
		}
	}()

	_, data, err := conn.ReadMessage()
	if err != nil {
		return "", err
	}

	var frame ws.IncomingMessage
	if err := json.Unmarshal(data, &frame); err != nil {
		return "", err
	}

	if frame.Type != ws.MessageTypeAuth || frame.Token == "" {
		return "", websocket.ErrCloseSent
	}

	userID, err := h.jwtValidator.Validate(frame.Token)
	if err != nil {
		return "", err
	}

	ack, err := json.Marshal(map[string]string{"type": "auth_ok", "user_id": userID})
	if err != nil {
		return "", err
	}
	if err := conn.WriteMessage(websocket.TextMessage, ack); err != nil {
		h.logger.Debug("failed to write auth ack", zap.Error(err))
		return "", err
	}

	return userID, nil
}

func (h *WSHandler) readPump(client *ws.Client) {
	defer func() {
		h.hub.Unregister(client)
		client.Conn.Close()
	}()

	for {
		_, data, err := client.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				h.logger.Warn("websocket unexpected close",
					zap.String("user_id", client.UserID),
					zap.Error(err))
			}
			return
		}

		var incoming ws.IncomingMessage
		if err := json.Unmarshal(data, &incoming); err != nil {
			h.logger.Warn("invalid ws message",
				zap.String("user_id", client.UserID),
				zap.Error(err))
			continue
		}

		// Use context with timeout for DB operations
		ctx, cancel := context.WithTimeout(context.Background(), wsOperationTimeout)
		switch incoming.Type {
		case "typing":
			h.handleTyping(ctx, client, incoming)
		case "read":
			h.handleRead(ctx, client, incoming)
		default:
			h.handleChatMessage(ctx, client, incoming)
		}
		cancel()
	}
}

func (h *WSHandler) handleTyping(ctx context.Context, client *ws.Client, incoming ws.IncomingMessage) {
	if incoming.ConversationID == "" {
		return
	}
	conv, err := h.convRepo.GetByID(ctx, incoming.ConversationID)
	if err != nil || !conv.HasParticipant(client.UserID) {
		return
	}
	otherID, err := conv.OtherUserID(client.UserID)
	if err != nil {
		h.logger.Warn("failed to get other participant",
			zap.String("conversation_id", incoming.ConversationID),
			zap.String("user_id", client.UserID),
			zap.Error(err))
		return
	}
	h.hub.BroadcastJSON(otherID, ws.TypingEvent{
		Type:           "typing",
		ConversationID: incoming.ConversationID,
		SenderID:       client.UserID,
		IsTyping:       incoming.IsTyping,
	})
}

func (h *WSHandler) handleRead(ctx context.Context, client *ws.Client, incoming ws.IncomingMessage) {
	if incoming.ConversationID == "" {
		return
	}
	conv, err := h.convRepo.GetByID(ctx, incoming.ConversationID)
	if err != nil || !conv.HasParticipant(client.UserID) {
		return
	}
	lastID, err := h.msgRepo.MarkConversationRead(ctx, incoming.ConversationID, client.UserID)
	if err != nil {
		h.logger.Warn("failed to mark conversation as read",
			zap.String("conversation_id", incoming.ConversationID),
			zap.Error(err))
		return
	}
	if lastID == "" {
		return // nothing to notify
	}
	otherID, err := conv.OtherUserID(client.UserID)
	if err != nil {
		h.logger.Warn("failed to get other participant",
			zap.String("conversation_id", incoming.ConversationID),
			zap.String("user_id", client.UserID),
			zap.Error(err))
		return
	}
	h.hub.BroadcastJSON(otherID, ws.ReadReceiptEvent{
		Type:              "read_receipt",
		ConversationID:    incoming.ConversationID,
		LastReadMessageID: lastID,
		ReaderID:          client.UserID,
	})
}

func (h *WSHandler) handleChatMessage(ctx context.Context, client *ws.Client, incoming ws.IncomingMessage) {
	msg, err := h.sendMessageUC.Execute(ctx, incoming.ConversationID, client.UserID, message.SendMessageInput{
		Type:    incoming.Type,
		Content: incoming.Content,
		POIID:   incoming.POIID,
	})
	if err != nil {
		h.logger.Warn("failed to save ws message",
			zap.String("user_id", client.UserID),
			zap.Error(err))
		return
	}

	outgoing := ws.OutgoingMessage{
		ID:             msg.ID,
		Type:           msg.Type,
		ConversationID: msg.ConversationID,
		SenderID:       msg.SenderID,
		Content:        msg.Content,
		POIID:          msg.POIID,
		Status:         msg.Status,
		CreatedAt:      msg.CreatedAt,
	}

	// Push echo to sender (multi-device support)
	h.hub.BroadcastTo(client.UserID, outgoing)

	// Resolve conversation to find the other participant
	conv, err := h.convRepo.GetByID(ctx, msg.ConversationID)
	if err != nil {
		h.logger.Warn("failed to resolve conversation for broadcast",
			zap.String("conversation_id", msg.ConversationID),
			zap.Error(err))
		return
	}

	otherUserID, err := conv.OtherUserID(client.UserID)
	if err != nil {
		h.logger.Warn("failed to get other participant",
			zap.String("conversation_id", msg.ConversationID),
			zap.String("user_id", client.UserID),
			zap.Error(err))
		return
	}
	h.hub.BroadcastTo(otherUserID, outgoing)

	// If the recipient is online, mark as delivered immediately
	if h.hub.IsOnline(otherUserID) {
		if updateErr := h.msgRepo.UpdateStatus(ctx, msg.ID, "delivered"); updateErr == nil {
			h.hub.BroadcastJSON(client.UserID, ws.DeliveredEvent{
				Type:           "delivered",
				ConversationID: msg.ConversationID,
				MessageID:      msg.ID,
			})
		}
	}
}
