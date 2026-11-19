package handler

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/horaciobranciforte/curiosity-chat-api/internal/ws"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// expiringValidatorStub accepts "tok" for user u1 and says it expires at exp.
type expiringValidatorStub struct{ exp time.Time }

func (s expiringValidatorStub) Validate(token string) (string, error) {
	id, _, err := s.ValidateWithExpiry(token)
	return id, err
}

func (s expiringValidatorStub) ValidateWithExpiry(token string) (string, time.Time, error) {
	if token != "tok" {
		return "", time.Time{}, errors.New("bad token")
	}
	return "u1", s.exp, nil
}

func authedConn(t *testing.T, exp time.Time) *websocket.Conn {
	t.Helper()
	hub := ws.NewHub()
	go hub.Run()
	h := NewWSHandler(hub, nil, nil, nil, expiringValidatorStub{exp: exp}, zap.NewNop(), []string{"*"})
	srv := httptest.NewServer(http.HandlerFunc(h.ServeWS))
	t.Cleanup(srv.Close)

	conn := dialWS(t, "ws"+strings.TrimPrefix(srv.URL, "http"))
	require.NoError(t, conn.WriteJSON(map[string]string{"type": "auth", "token": "tok"}))
	require.NoError(t, conn.SetReadDeadline(time.Now().Add(2*time.Second)))
	_, _, err := conn.ReadMessage() // auth_ok
	require.NoError(t, err)
	return conn
}

func TestTheConnectionClosesWhenItsTokenExpires(t *testing.T) {
	conn := authedConn(t, time.Now().Add(200*time.Millisecond))

	require.NoError(t, conn.SetReadDeadline(time.Now().Add(3*time.Second)))
	_, _, err := conn.ReadMessage()

	var closeErr *websocket.CloseError
	require.ErrorAs(t, err, &closeErr)
	assert.Equal(t, closeTokenExpired, closeErr.Code)
}

func TestAnOversizedFrameEndsTheConnection(t *testing.T) {
	conn := authedConn(t, time.Now().Add(time.Hour))

	huge := `{"type":"text","content":"` + strings.Repeat("x", maxWSFrameBytes*2) + `"}`
	if err := conn.WriteMessage(websocket.TextMessage, []byte(huge)); err != nil {
		return // the server hung up while we were still writing: that is the limit at work
	}

	require.NoError(t, conn.SetReadDeadline(time.Now().Add(3*time.Second)))
	_, _, err := conn.ReadMessage()
	assert.Error(t, err, "the server should drop a client sending frames over the limit")
}
