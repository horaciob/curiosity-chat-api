package handler

import (
	"context"
	"encoding/json"
	"golang.org/x/time/rate"
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
	// maxWSFrameBytes bounds one incoming frame. Without it a client could send frames of
	// any size and the server would buffer them whole.
	maxWSFrameBytes = 16 << 10
	// closeTokenExpired is sent when the token a connection authenticated with expires;
	// the apps refresh and reconnect.
	closeTokenExpired = 4001
)

// newUpgrader builds a websocket upgrader that accepts the same origins the REST CORS
// middleware does. Browser clients send an Origin header; native clients (the iOS app)
// send none and are always allowed.
func newUpgrader(allowedOrigins []string) websocket.Upgrader {
	return websocket.Upgrader{
		ReadBufferSize:  wsReadBufferSize,
		WriteBufferSize: wsWriteBufferSize,
		CheckOrigin: func(r *http.Request) bool {
			origin := r.Header.Get("Origin")
			// Allow requests with no origin (non-browser clients).
			if origin == "" {
				return true
			}
			for _, allowed := range allowedOrigins {
				if allowed == "*" || origin == allowed {
					return true
				}
			}
			return false
		},
	}
}

// WSHandler handles WebSocket connections.
type WSHandler struct {
	hub           *ws.Hub
	sendMessageUC *message.SendMessage
	convRepo      message.ConversationRepository
	msgRepo       message.Repository
	jwtValidator  middleware.TokenValidator
	logger        *zap.Logger
	upgrader      websocket.Upgrader
	pusher        ChatPusher // optional
}

// WithChatPusher enables pushes to recipients who have no open connection.
func (h *WSHandler) WithChatPusher(p ChatPusher) *WSHandler {
	h.pusher = p
	return h
}

// NewWSHandler creates a new WSHandler.
func NewWSHandler(
	hub *ws.Hub,
	sendMessageUC *message.SendMessage,
	convRepo message.ConversationRepository,
	msgRepo message.Repository,
	jwtValidator middleware.TokenValidator,
	logger *zap.Logger,
	allowedOrigins []string,
) *WSHandler {
	return &WSHandler{
		hub:           hub,
		sendMessageUC: sendMessageUC,
		convRepo:      convRepo,
		msgRepo:       msgRepo,
		jwtValidator:  jwtValidator,
		logger:        logger,
		upgrader:      newUpgrader(allowedOrigins),
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
	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err == nil {
		conn.SetReadLimit(maxWSFrameBytes)
	}
	if err != nil {
		h.logger.Error("websocket upgrade failed", zap.Error(err))
		return
	}

	userID, expires, err := h.authenticate(conn)
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

	// The connection is only as good as the token it opened with. Close it when that
	// expires (or, for a deleted account, can no longer be refreshed): the apps refresh
	// and reconnect. It used to stay open for as long as the socket lived.
	if !expires.IsZero() {
		timer := time.AfterFunc(time.Until(expires), func() {
			_ = conn.WriteControl(websocket.CloseMessage,
				websocket.FormatCloseMessage(closeTokenExpired, "token expired"), time.Now().Add(writeWaitControl))
			_ = conn.Close()
		})
		defer timer.Stop()
	}

	h.readPump(client)
}

const writeWaitControl = 5 * time.Second

type expiringValidator interface {
	ValidateWithExpiry(token string) (string, time.Time, error)
}

// authenticate reads the first WebSocket frame and validates the auth token.
// The client must send {"type":"auth","token":"<jwt>"} within AuthDeadline.
func (h *WSHandler) authenticate(conn *websocket.Conn) (string, time.Time, error) {
	if err := conn.SetReadDeadline(time.Now().Add(ws.AuthDeadline)); err != nil {
		return "", time.Time{}, err
	}
	defer func() {
		if err := conn.SetReadDeadline(time.Time{}); err != nil {
			h.logger.Debug("failed to reset read deadline", zap.Error(err))
		}
	}()

	_, data, err := conn.ReadMessage()
	if err != nil {
		return "", time.Time{}, err
	}

	var frame ws.IncomingMessage
	if err := json.Unmarshal(data, &frame); err != nil {
		return "", time.Time{}, err
	}

	if frame.Type != ws.MessageTypeAuth || frame.Token == "" {
		return "", time.Time{}, websocket.ErrCloseSent
	}

	var userID string
	var expires time.Time
	if v, ok := h.jwtValidator.(expiringValidator); ok {
		userID, expires, err = v.ValidateWithExpiry(frame.Token)
	} else {
		userID, err = h.jwtValidator.Validate(frame.Token)
	}
	if err != nil {
		return "", time.Time{}, err
	}

	ack, err := json.Marshal(map[string]string{"type": "auth_ok", "user_id": userID})
	if err != nil {
		return "", time.Time{}, err
	}
	if err := conn.WriteMessage(websocket.TextMessage, ack); err != nil {
		h.logger.Debug("failed to write auth ack", zap.Error(err))
		return "", time.Time{}, err
	}

	return userID, expires, nil
}

func (h *WSHandler) readPump(client *ws.Client) {
	defer func() {
		h.hub.Unregister(client)
		client.Conn.Close()
	}()

	// Keep the connection alive: reset read deadline on every pong received.
	// HTTP requests go through the rate limiter; WebSocket frames used not to, so a
	// script could send messages as fast as the socket allowed. Per connection: a
	// generous ceiling on frames (typing, read) and a human pace for chat messages.
	frames := rate.NewLimiter(20, 40)
	messages := rate.NewLimiter(rate.Every(300*time.Millisecond), 5)

	client.Conn.SetReadDeadline(time.Now().Add(60 * time.Second)) //nolint:errcheck
	client.Conn.SetPongHandler(func(string) error {
		client.Conn.SetReadDeadline(time.Now().Add(60 * time.Second)) //nolint:errcheck
		return nil
	})

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

		if !frames.Allow() {
			continue
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
			if !messages.Allow() {
				h.tellRateLimited(client, incoming.ConversationID)
				break
			}
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

	// Broadcast updated unread counts back to the reader (their count for this conversation is now 0)
	totalUnread, _ := h.msgRepo.CountTotalUnreadForUser(ctx, client.UserID)
	h.hub.BroadcastJSON(client.UserID, ws.UnreadCountEvent{
		Type:           "unread_count_update",
		ConversationID: incoming.ConversationID,
		UnreadCount:    0,
		TotalUnread:    totalUnread,
	})
}

func (h *WSHandler) handleChatMessage(ctx context.Context, client *ws.Client, incoming ws.IncomingMessage) {
	msg, err := h.sendMessageUC.Execute(ctx, incoming.ConversationID, client.UserID, message.SendMessageInput{
		Type:        incoming.Type,
		Content:     incoming.Content,
		POIID:       incoming.POIID,
		ShareIntent: incoming.ShareIntent,
	})
	if err != nil {
		h.logger.Warn("ws message refused", zap.String("conversation_id", incoming.ConversationID), zap.Error(err))
		return
	}
	deliverNewMessage(h.hub, h.pusher, h.convRepo, h.msgRepo, client.UserID, msg)
}

// tellRateLimited lets the sender know a message was dropped, without blocking.
func (h *WSHandler) tellRateLimited(client *ws.Client, conversationID string) {
	frame, _ := json.Marshal(map[string]string{"type": "error", "code": "rate_limited", "conversation_id": conversationID})
	select {
	case client.Send <- frame:
	default:
	}
}
