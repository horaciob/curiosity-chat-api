package ws

import (
	"encoding/json"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// IncomingMessage is the JSON structure sent by a WebSocket client.
type IncomingMessage struct {
	Type           string `json:"type"`             // "text" | "poi_share"
	ConversationID string `json:"conversation_id"`
	Content        string `json:"content,omitempty"`
	POIID          string `json:"poi_id,omitempty"`
}

// OutgoingMessage is the JSON structure pushed to WebSocket clients.
type OutgoingMessage struct {
	ID             string    `json:"id"`
	Type           string    `json:"type"`
	ConversationID string    `json:"conversation_id"`
	SenderID       string    `json:"sender_id"`
	Content        *string   `json:"content,omitempty"`
	POIID          *string   `json:"poi_id,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
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
		register:   make(chan *Client, 64),
		unregister: make(chan *Client, 64),
		broadcast:  make(chan broadcastEnvelope, 256),
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

// WritePump pumps messages from the hub to the WebSocket connection.
func (c *Client) WritePump() {
	defer c.Conn.Close()
	for msg := range c.Send {
		if err := c.Conn.WriteMessage(websocket.TextMessage, msg); err != nil {
			return
		}
	}
}
