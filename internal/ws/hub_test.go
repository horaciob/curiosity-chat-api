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
