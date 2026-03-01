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
//
// Authentication flow (token never appears in the URL):
//  1. Client connects — no credentials needed in the URL.
//  2. Server upgrades to WebSocket immediately.
//  3. Client must send {"type":"auth","token":"<jwt>"} within 10 seconds.
//  4. Server validates the token. On success it replies {"type":"auth_ok"}.
//  5. On failure or timeout the connection is closed with a policy-violation code.
//
//	@Summary		Open a WebSocket connection
//	@Description	Upgrades to WebSocket. The first client frame must be an auth message: {"type":"auth","token":"<jwt>"}
//	@Tags			websocket
//	@Success		101	"Switching Protocols"
//	@Failure		401	"Unauthorized"
//	@Router			/ws [get]
func (h *WSHandler) ServeWS(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		h.logger.Error("websocket upgrade failed", zap.Error(err))
		return
	}

	userID, err := h.authenticate(conn)
	if err != nil {
		h.logger.Warn("websocket auth failed", zap.Error(err))
		conn.WriteMessage( //nolint:errcheck
			websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.ClosePolicyViolation, "authentication failed"),
		)
		conn.Close()
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

// authenticate reads the first WebSocket frame and validates the auth token.
// The client must send {"type":"auth","token":"<jwt>"} within AuthDeadline.
func (h *WSHandler) authenticate(conn *websocket.Conn) (string, error) {
	conn.SetReadDeadline(time.Now().Add(ws.AuthDeadline)) //nolint:errcheck
	defer conn.SetReadDeadline(time.Time{})               //nolint:errcheck

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

	ack, _ := json.Marshal(map[string]string{"type": "auth_ok", "user_id": userID})
	conn.WriteMessage(websocket.TextMessage, ack) //nolint:errcheck

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
