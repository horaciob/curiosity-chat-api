package ws

import (
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
}

// NewHub creates a new Hub.
func NewHub() *Hub {
	return &Hub{
		clients:    make(map[string]map[*Client]bool),
		register:   make(chan *Client, registerChanBufferSize),
		unregister: make(chan *Client, unregisterChanBufferSize),
		broadcast:  make(chan broadcastEnvelope, broadcastChanBufferSize),
	}
}

// Run starts the hub event loop. Call as a goroutine.
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			if h.clients[client.UserID] == nil {
				h.clients[client.UserID] = make(map[*Client]bool)
			}
			h.clients[client.UserID][client] = true
			h.mu.Unlock()

		case client := <-h.unregister:
			h.mu.Lock()
			if conns, ok := h.clients[client.UserID]; ok {
				delete(conns, client)
				if len(conns) == 0 {
					delete(h.clients, client.UserID)
				}
			}
			h.mu.Unlock()
			close(client.Send)

		case env := <-h.broadcast:
			h.mu.RLock()
			conns := h.clients[env.targetUserID]
			h.mu.RUnlock()
			for client := range conns {
				select {
				case client.Send <- env.payload:
				default:
					// Slow client — drop message rather than blocking
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
	h.broadcast <- broadcastEnvelope{targetUserID: userID, payload: data}
}

// BroadcastJSON serializes v as JSON and sends it to all connections of the given user.
// Use this for control frames (typing, delivered, read_receipt).
func (h *Hub) BroadcastJSON(userID string, v any) {
	data, err := json.Marshal(v)
	if err != nil {
		return
	}
	h.broadcast <- broadcastEnvelope{targetUserID: userID, payload: data}
}

// IsOnline returns true if the user has at least one active WebSocket connection.
func (h *Hub) IsOnline(userID string) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	if conns, ok := h.clients[userID]; ok {
		return len(conns) > 0
	}
	return false
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
