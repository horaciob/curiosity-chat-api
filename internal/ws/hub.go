package ws

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const (
	// MessageTypeAuth is the first frame the client must send to authenticate.
	MessageTypeAuth = "auth"
	// AuthDeadline is the time the server waits for the auth frame before closing.
	AuthDeadline = 10 * time.Second

	// Hub channel buffer sizes
	registerChanBufferSize   = 64
	unregisterChanBufferSize = 64
	broadcastChanBufferSize  = 256

	// ClientSendChanBufferSize is the buffer size for client send channels
	ClientSendChanBufferSize = 256

	// writeWait is the time allowed to write a message to the peer.
	writeWait = 10 * time.Second
	// pingPeriod is how often to send a ping frame to keep the connection alive.
	pingPeriod = 30 * time.Second
)

// IncomingMessage is the JSON structure sent by a WebSocket client.
type IncomingMessage struct {
	Type           string `json:"type"`            // "auth" | "text" | "poi_share" | "typing" | "read"
	Token          string `json:"token,omitempty"` // only for type="auth"
	ConversationID string `json:"conversation_id,omitempty"`
	Content        string `json:"content,omitempty"`
	POIID          string `json:"poi_id,omitempty"`
	IsTyping       bool   `json:"is_typing,omitempty"` // for type="typing"
	ShareIntent    string `json:"share_intent,omitempty"` // for type="poi_share"
}

// OutgoingMessage is the JSON structure pushed to WebSocket clients for chat messages.
type OutgoingMessage struct {
	ID             string    `json:"id"`
	Type           string    `json:"type"`
	ConversationID string    `json:"conversation_id"`
	SenderID       string    `json:"sender_id"`
	Content        *string   `json:"content,omitempty"`
	POIID          *string   `json:"poi_id,omitempty"`
	ShareIntent    *string   `json:"share_intent,omitempty"`
	Status         string    `json:"status"`
	CreatedAt      time.Time `json:"created_at"`
}

// TypingEvent is pushed to the recipient when the sender starts/stops typing.
type TypingEvent struct {
	Type           string `json:"type"` // "typing"
	ConversationID string `json:"conversation_id"`
	SenderID       string `json:"sender_id"`
	IsTyping       bool   `json:"is_typing"`
}

// DeliveredEvent is pushed to the sender when the message is delivered to the recipient.
type DeliveredEvent struct {
	Type           string `json:"type"` // "delivered"
	ConversationID string `json:"conversation_id"`
	MessageID      string `json:"message_id"`
}

// ReadReceiptEvent is pushed to the sender when the recipient reads the messages.
type ReadReceiptEvent struct {
	Type              string `json:"type"` // "read_receipt"
	ConversationID    string `json:"conversation_id"`
	LastReadMessageID string `json:"last_read_message_id"`
	ReaderID          string `json:"reader_id"`
}

// UnreadCountEvent is broadcast to a user when their unread message count changes.
type UnreadCountEvent struct {
	Type           string `json:"type"` // "unread_count_update"
	ConversationID string `json:"conversation_id"`
	UnreadCount    int    `json:"unread_count"` // unread in this conversation
	TotalUnread    int    `json:"total_unread"` // across all conversations
}

// NotificationPushEvent relays an app notification from user-api to a connected user.
type NotificationPushEvent struct {
	Type             string `json:"type"` // "notification"
	NotificationID   string `json:"notification_id"`
	NotificationType string `json:"notification_type"`
	ActorID          string `json:"actor_id,omitempty"`
	ActorName        string `json:"actor_name,omitempty"`
	Title            string `json:"title"`
	Body             string `json:"body"`
	CreatedAt        string `json:"created_at"`
}

// NotificationCountEvent is broadcast when the user's notification badge count changes.
type NotificationCountEvent struct {
	Type        string `json:"type"` // "notification_count_update"
	UnreadCount int    `json:"unread_count"`
}

type broadcastEnvelope struct {
	targetUserID string
	payload      []byte
}

// Client represents a connected WebSocket client.
type Client struct {
	hub    *Hub
	UserID string
	Conn   *websocket.Conn
	Send   chan []byte
}

// Hub maintains the set of active WebSocket clients and routes messages.
type Hub struct {
	clients    map[string]map[*Client]bool // userID → set of connections
	register   chan *Client
	unregister chan *Client
	broadcast  chan broadcastEnvelope
	mu         sync.RWMutex
	cluster    Cluster // optional: fan deliveries and presence out across replicas
	onSlow     func(*Client) // what to do with a client whose buffer is full
}

// disconnect closes a client's connection; its read loop then unregisters it.
func disconnect(c *Client) {
	if c.Conn == nil || c.Conn.UnderlyingConn() == nil {
		return
	}
	_ = c.Conn.Close()
}

// WithCluster makes the hub one of several replicas sharing deliveries and presence.
// Call before Run and StartCluster.
func (h *Hub) WithCluster(c Cluster) *Hub {
	h.cluster = c
	return h
}

// StartCluster receives deliveries published by any replica and keeps this replica's
// presence claims fresh, until ctx is done. A no-op without a cluster.
func (h *Hub) StartCluster(ctx context.Context, onError func(error)) {
	if h.cluster == nil {
		return
	}
	go func() {
		for ctx.Err() == nil {
			err := h.cluster.Subscribe(ctx, func(userID string, payload []byte) {
				h.broadcast <- broadcastEnvelope{targetUserID: userID, payload: payload}
			})
			if err != nil && ctx.Err() == nil {
				if onError != nil {
					onError(err)
				}
				time.Sleep(time.Second) // Redis restarting: retry, don't spin
			}
		}
	}()
	go func() {
		t := time.NewTicker(presenceRefresh)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				if err := h.cluster.SetOnline(ctx, h.localUsers()...); err != nil && onError != nil {
					onError(err)
				}
			}
		}
	}()
}

func (h *Hub) localUsers() []string {
	h.mu.RLock()
	defer h.mu.RUnlock()
	users := make([]string, 0, len(h.clients))
	for u := range h.clients {
		users = append(users, u)
	}
	return users
}

// deliver sends payload to every connection of userID: through the cluster when there is
// one (every replica, this one included, receives it), locally otherwise or if publishing
// fails — a replica that loses Redis still serves its own connections.
func (h *Hub) deliver(userID string, payload []byte) {
	if h.cluster != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		err := h.cluster.Publish(ctx, userID, payload)
		cancel()
		if err == nil {
			return
		}
	}
	h.broadcast <- broadcastEnvelope{targetUserID: userID, payload: payload}
}

func (h *Hub) clusterPresence(userID string, online bool) {
	if h.cluster == nil {
		return
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		if online {
			_ = h.cluster.SetOnline(ctx, userID)
		} else {
			_ = h.cluster.SetOffline(ctx, userID)
		}
	}()
}

// NewHub creates a new Hub.
func NewHub() *Hub {
	return &Hub{
		clients:    make(map[string]map[*Client]bool),
		register:   make(chan *Client, registerChanBufferSize),
		unregister: make(chan *Client, unregisterChanBufferSize),
		broadcast:  make(chan broadcastEnvelope, broadcastChanBufferSize),
		onSlow:     disconnect,
	}
}

// Run starts the hub event loop. Call as a goroutine.
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			first := h.clients[client.UserID] == nil
			if first {
				h.clients[client.UserID] = make(map[*Client]bool)
			}
			h.clients[client.UserID][client] = true
			h.mu.Unlock()
			if first {
				h.clusterPresence(client.UserID, true)
			}

		case client := <-h.unregister:
			h.mu.Lock()
			last := false
			if conns, ok := h.clients[client.UserID]; ok {
				delete(conns, client)
				if len(conns) == 0 {
					delete(h.clients, client.UserID)
					last = true
				}
			}
			h.mu.Unlock()
			close(client.Send)
			if last {
				h.clusterPresence(client.UserID, false)
			}

		case env := <-h.broadcast:
			h.mu.RLock()
			conns := h.clients[env.targetUserID]
			h.mu.RUnlock()
			for client := range conns {
				select {
				case client.Send <- env.payload:
				default:
					// Its buffer is full: it is not reading. Dropping the frame silently left
					// it showing a stale chat for good. Close the connection instead; its read
					// loop unregisters it, and the app reconnects and reloads the messages.
					go h.onSlow(client)
				}
			}
		}
	}
}

// Register adds a client to the hub.
func (h *Hub) Register(c *Client) { h.register <- c }

// Unregister removes a client from the hub.
func (h *Hub) Unregister(c *Client) { h.unregister <- c }

// BroadcastTo sends an OutgoingMessage to all connections of the given user.
func (h *Hub) BroadcastTo(userID string, msg OutgoingMessage) {
	data, err := json.Marshal(msg)
	if err != nil {
		return
	}
	h.deliver(userID, data)
}

// BroadcastJSON serializes v as JSON and sends it to all connections of the given user.
// Use this for control frames (typing, delivered, read_receipt).
func (h *Hub) BroadcastJSON(userID string, v any) {
	data, err := json.Marshal(v)
	if err != nil {
		return
	}
	h.deliver(userID, data)
}

// BroadcastRaw sends a pre-serialized JSON payload to all connections of the given user.
func (h *Hub) BroadcastRaw(userID string, data []byte) {
	h.deliver(userID, data)
}

// IsOnline returns true if the user has at least one active WebSocket connection — on
// this replica, or on any replica when the hub is part of a cluster.
func (h *Hub) IsOnline(userID string) bool {
	h.mu.RLock()
	local := len(h.clients[userID]) > 0
	h.mu.RUnlock()
	if local || h.cluster == nil {
		return local
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	return h.cluster.IsOnline(ctx, userID)
}

// WritePump pumps messages from the hub to the WebSocket connection.
// It also sends periodic ping frames to keep the connection alive through
// NAT timeouts and load balancer idle limits.
func (c *Client) WritePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.Conn.Close()
	}()
	for {
		select {
		case msg, ok := <-c.Send:
			c.Conn.SetWriteDeadline(time.Now().Add(writeWait)) //nolint:errcheck
			if !ok {
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{}) //nolint:errcheck
				return
			}
			if err := c.Conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				return
			}
		case <-ticker.C:
			c.Conn.SetWriteDeadline(time.Now().Add(writeWait)) //nolint:errcheck
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
