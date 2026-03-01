package handler

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/horaciobranciforte/curiosity-chat-api/internal/adapter/http/middleware"
	"github.com/horaciobranciforte/curiosity-chat-api/internal/usecase/message"
	"github.com/horaciobranciforte/curiosity-chat-api/internal/ws"
	"go.uber.org/zap"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

// WSHandler handles WebSocket connections.
type WSHandler struct {
	hub           *ws.Hub
	sendMessageUC *message.SendMessage
	convRepo      message.ConversationRepository
	jwtValidator  middleware.TokenValidator
	logger        *zap.Logger
}

// NewWSHandler creates a new WSHandler.
func NewWSHandler(
	hub *ws.Hub,
	sendMessageUC *message.SendMessage,
	convRepo message.ConversationRepository,
	jwtValidator middleware.TokenValidator,
	logger *zap.Logger,
) *WSHandler {
	return &WSHandler{
		hub:           hub,
		sendMessageUC: sendMessageUC,
		convRepo:      convRepo,
		jwtValidator:  jwtValidator,
		logger:        logger,
	}
}

// ServeWS handles GET /api/v1/ws — upgrades the connection to WebSocket.
// Authentication is done via the ?token=<jwt> query parameter.
func (h *WSHandler) ServeWS(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	if token == "" {
		http.Error(w, "missing token", http.StatusUnauthorized)
		return
	}

	userID, err := h.jwtValidator.Validate(token)
	if err != nil {
		http.Error(w, "invalid token", http.StatusUnauthorized)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		h.logger.Error("websocket upgrade failed", zap.Error(err))
		return
	}

	client := &ws.Client{
		UserID: userID,
		Conn:   conn,
		Send:   make(chan []byte, 256),
	}
	h.hub.Register(client)
	go client.WritePump()

	h.readPump(client)
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

		ctx := context.Background()

		msg, err := h.sendMessageUC.Execute(ctx, incoming.ConversationID, client.UserID, message.SendMessageInput{
			Type:    incoming.Type,
			Content: incoming.Content,
			POIID:   incoming.POIID,
		})
		if err != nil {
			h.logger.Warn("failed to save ws message",
				zap.String("user_id", client.UserID),
				zap.Error(err))
			continue
		}

		outgoing := ws.OutgoingMessage{
			ID:             msg.ID,
			Type:           msg.Type,
			ConversationID: msg.ConversationID,
			SenderID:       msg.SenderID,
			Content:        msg.Content,
			POIID:          msg.POIID,
			CreatedAt:      msg.CreatedAt,
		}

		// Push to sender (for multi-device support)
		h.hub.BroadcastTo(client.UserID, outgoing)

		// Push to the other participant in the conversation
		conv, err := h.convRepo.GetByID(ctx, msg.ConversationID)
		if err != nil {
			h.logger.Warn("failed to resolve conversation for broadcast",
				zap.String("conversation_id", msg.ConversationID),
				zap.Error(err))
			continue
		}
		h.hub.BroadcastTo(conv.OtherUserID(client.UserID), outgoing)
	}
}
