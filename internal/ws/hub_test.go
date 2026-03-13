package ws

import (
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
)

type mockConn struct {
	writeMessages [][]byte
	closed        bool
}

func (m *mockConn) WriteMessage(messageType int, data []byte) error {
	m.writeMessages = append(m.writeMessages, data)
	return nil
}

func (m *mockConn) Close() error {
	m.closed = true
	return nil
}

func TestHubIsOnlineEmptyHub(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	assert.False(t, hub.IsOnline("user-1"))
}

func TestHubIsOnlineAfterRegister(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	client := &Client{
		UserID: "user-1",
		Conn:   &websocket.Conn{},
		Send:   make(chan []byte, 256),
	}

	hub.Register(client)
	time.Sleep(10 * time.Millisecond) // Allow hub to process

	assert.True(t, hub.IsOnline("user-1"))
}

func TestHubIsOnlineAfterUnregister(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	client := &Client{
		UserID: "user-1",
		Conn:   &websocket.Conn{},
		Send:   make(chan []byte, 256),
	}

	hub.Register(client)
	time.Sleep(10 * time.Millisecond)
	assert.True(t, hub.IsOnline("user-1"))

	hub.Unregister(client)
	time.Sleep(10 * time.Millisecond)
	assert.False(t, hub.IsOnline("user-1"))
}

func TestHubIsOnlineMultipleUsers(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	client1 := &Client{
		UserID: "user-1",
		Conn:   &websocket.Conn{},
		Send:   make(chan []byte, 256),
	}
	client2 := &Client{
		UserID: "user-2",
		Conn:   &websocket.Conn{},
		Send:   make(chan []byte, 256),
	}

	hub.Register(client1)
	hub.Register(client2)
	time.Sleep(10 * time.Millisecond)

	assert.True(t, hub.IsOnline("user-1"))
	assert.True(t, hub.IsOnline("user-2"))
	assert.False(t, hub.IsOnline("user-3"))
}

func TestHubBroadcastTo(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	client := &Client{
		UserID: "user-1",
		Conn:   &websocket.Conn{},
		Send:   make(chan []byte, 256),
	}

	hub.Register(client)
	time.Sleep(10 * time.Millisecond)

	msg := OutgoingMessage{
		ID:             "msg-1",
		Type:           "text",
		ConversationID: "conv-1",
		SenderID:       "user-1",
		Status:         "sent",
		CreatedAt:      time.Now(),
	}

	hub.BroadcastTo("user-1", msg)

	select {
	case data := <-client.Send:
		assert.NotEmpty(t, data)
	case <-time.After(100 * time.Millisecond):
		t.Fatal("expected message to be sent")
	}
}

func TestHubBroadcastJSON(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	client := &Client{
		UserID: "user-1",
		Conn:   &websocket.Conn{},
		Send:   make(chan []byte, 256),
	}

	hub.Register(client)
	time.Sleep(10 * time.Millisecond)

	event := TypingEvent{
		Type:           "typing",
		ConversationID: "conv-1",
		SenderID:       "user-1",
		IsTyping:       true,
	}

	hub.BroadcastJSON("user-1", event)

	select {
	case data := <-client.Send:
		assert.NotEmpty(t, data)
	case <-time.After(100 * time.Millisecond):
		t.Fatal("expected message to be sent")
	}
}

func TestHubBroadcastToNonExistentUser(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	msg := OutgoingMessage{
		ID:             "msg-1",
		Type:           "text",
		ConversationID: "conv-1",
		SenderID:       "user-1",
		Status:         "sent",
		CreatedAt:      time.Now(),
	}

	// Should not panic when broadcasting to non-existent user
	hub.BroadcastTo("non-existent-user", msg)
	time.Sleep(10 * time.Millisecond)
}

func TestHubBroadcastJSONToNonExistentUser(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	event := TypingEvent{
		Type:           "typing",
		ConversationID: "conv-1",
		SenderID:       "user-1",
		IsTyping:       true,
	}

	// Should not panic when broadcasting to non-existent user
	hub.BroadcastJSON("non-existent-user", event)
	time.Sleep(10 * time.Millisecond)
}

func TestHubMultipleConnectionsForSameUser(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	client1 := &Client{
		UserID: "user-1",
		Conn:   &websocket.Conn{},
		Send:   make(chan []byte, 256),
	}
	client2 := &Client{
		UserID: "user-1",
		Conn:   &websocket.Conn{},
		Send:   make(chan []byte, 256),
	}

	hub.Register(client1)
	hub.Register(client2)
	time.Sleep(10 * time.Millisecond)

	msg := OutgoingMessage{
		ID:             "msg-1",
		Type:           "text",
		ConversationID: "conv-1",
		SenderID:       "user-2",
		Status:         "sent",
		CreatedAt:      time.Now(),
	}

	hub.BroadcastTo("user-1", msg)

	// Both clients should receive the message
	select {
	case <-client1.Send:
		// Success
	case <-time.After(100 * time.Millisecond):
		t.Fatal("expected client1 to receive message")
	}

	select {
	case <-client2.Send:
		// Success
	case <-time.After(100 * time.Millisecond):
		t.Fatal("expected client2 to receive message")
	}
}

func TestHubUnregisterOneOfManyConnections(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	client1 := &Client{
		UserID: "user-1",
		Conn:   &websocket.Conn{},
		Send:   make(chan []byte, 256),
	}
	client2 := &Client{
		UserID: "user-1",
		Conn:   &websocket.Conn{},
		Send:   make(chan []byte, 256),
	}

	hub.Register(client1)
	hub.Register(client2)
	time.Sleep(10 * time.Millisecond)

	// Unregister only one client
	hub.Unregister(client1)
	time.Sleep(10 * time.Millisecond)

	// User should still be online
	assert.True(t, hub.IsOnline("user-1"))

	// Unregister the other client
	hub.Unregister(client2)
	time.Sleep(10 * time.Millisecond)

	// Now user should be offline
	assert.False(t, hub.IsOnline("user-1"))
}

func TestHubSlowClientDropMessage(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	// Create a client with a full buffer (no reader)
	client := &Client{
		UserID: "user-1",
		Conn:   &websocket.Conn{},
		Send:   make(chan []byte, 1),
	}

	hub.Register(client)
	time.Sleep(10 * time.Millisecond)

	// Fill the buffer
	client.Send <- []byte("existing-message")

	// This message should be dropped (slow client)
	msg := OutgoingMessage{
		ID:             "msg-1",
		Type:           "text",
		ConversationID: "conv-1",
		SenderID:       "user-2",
		Status:         "sent",
		CreatedAt:      time.Now(),
	}

	hub.BroadcastTo("user-1", msg)
	time.Sleep(10 * time.Millisecond)

	// Only the first message should be in the channel
	select {
	case data := <-client.Send:
		assert.Equal(t, "existing-message", string(data))
	case <-time.After(100 * time.Millisecond):
		t.Fatal("expected existing message")
	}

	// No more messages
	select {
	case <-client.Send:
		t.Fatal("expected no more messages (slow client)")
	case <-time.After(50 * time.Millisecond):
		// Success - message was dropped
	}
}
