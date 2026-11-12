package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/horaciobranciforte/curiosity-chat-api/internal/ws"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// stubTokenValidator accepts one known token and rejects everything else.
type stubTokenValidator struct {
	validToken string
	userID     string
}

func (s stubTokenValidator) Validate(token string) (string, error) {
	if token == s.validToken {
		return s.userID, nil
	}
	return "", errors.New("invalid token")
}

// MARK: - Origin checking

func TestNewUpgraderAllowsRequestWithoutOrigin(t *testing.T) {
	// Native clients (the iOS app) send no Origin header and must always be allowed.
	up := newUpgrader([]string{"https://admin.curiosity.app"})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/ws", nil)

	assert.True(t, up.CheckOrigin(req))
}

func TestNewUpgraderAllowsConfiguredOrigin(t *testing.T) {
	up := newUpgrader([]string{"http://localhost:3000", "https://admin.curiosity.app"})

	for _, origin := range []string{"http://localhost:3000", "https://admin.curiosity.app"} {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/ws", nil)
		req.Header.Set("Origin", origin)

		assert.True(t, up.CheckOrigin(req), "expected %s to be allowed", origin)
	}
}

func TestNewUpgraderRejectsUnknownOrigin(t *testing.T) {
	up := newUpgrader([]string{"https://admin.curiosity.app"})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/ws", nil)
	req.Header.Set("Origin", "https://evil.example.com")

	assert.False(t, up.CheckOrigin(req))
}

func TestNewUpgraderWildcardAllowsAnyOrigin(t *testing.T) {
	up := newUpgrader([]string{"*"})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/ws", nil)
	req.Header.Set("Origin", "https://anything.example.com")

	assert.True(t, up.CheckOrigin(req))
}

func TestNewUpgraderEmptyAllowListRejectsBrowsers(t *testing.T) {
	// No configured origins must not silently mean "allow everything".
	up := newUpgrader(nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/ws", nil)
	req.Header.Set("Origin", "http://localhost:3000")

	assert.False(t, up.CheckOrigin(req))
}

// MARK: - Auth handshake

// newWSTestServer spins up a real WebSocket server backed by the handler under test and
// returns its ws:// URL.
func newWSTestServer(t *testing.T, validator stubTokenValidator) string {
	t.Helper()

	h := NewWSHandler(
		ws.NewHub(),
		nil, // send-message use case is not exercised by the handshake tests
		nil,
		nil,
		validator,
		zap.NewNop(),
		[]string{"*"},
	)

	srv := httptest.NewServer(http.HandlerFunc(h.ServeWS))
	t.Cleanup(srv.Close)

	return "ws" + strings.TrimPrefix(srv.URL, "http")
}

func dialWS(t *testing.T, url string) *websocket.Conn {
	t.Helper()

	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	require.NoError(t, err)
	t.Cleanup(func() { conn.Close() })

	return conn
}

func TestServeWSAuthSuccessReturnsAuthOK(t *testing.T) {
	userID := uuid.New().String()
	token := uuid.New().String()
	conn := dialWS(t, newWSTestServer(t, stubTokenValidator{validToken: token, userID: userID}))

	require.NoError(t, conn.WriteJSON(map[string]string{"type": "auth", "token": token}))

	require.NoError(t, conn.SetReadDeadline(time.Now().Add(2*time.Second)))
	_, data, err := conn.ReadMessage()
	require.NoError(t, err)

	var ack map[string]string
	require.NoError(t, json.Unmarshal(data, &ack))

	assert.Equal(t, "auth_ok", ack["type"])
	assert.Equal(t, userID, ack["user_id"])
}

func TestServeWSAuthWithInvalidTokenClosesConnection(t *testing.T) {
	conn := dialWS(t, newWSTestServer(t, stubTokenValidator{
		validToken: uuid.New().String(),
		userID:     uuid.New().String(),
	}))

	require.NoError(t, conn.WriteJSON(map[string]string{"type": "auth", "token": "not-the-token"}))

	require.NoError(t, conn.SetReadDeadline(time.Now().Add(2*time.Second)))
	_, _, err := conn.ReadMessage()

	require.Error(t, err)
	assert.True(t,
		websocket.IsCloseError(err, websocket.ClosePolicyViolation),
		"expected a policy-violation close, got %v", err)
}

func TestServeWSFirstFrameMustBeAuth(t *testing.T) {
	token := uuid.New().String()
	conn := dialWS(t, newWSTestServer(t, stubTokenValidator{
		validToken: token,
		userID:     uuid.New().String(),
	}))

	// A valid token in a non-auth frame must not be accepted.
	require.NoError(t, conn.WriteJSON(map[string]string{"type": "text", "token": token}))

	require.NoError(t, conn.SetReadDeadline(time.Now().Add(2*time.Second)))
	_, _, err := conn.ReadMessage()

	assert.Error(t, err)
}

func TestServeWSAuthWithEmptyTokenIsRejected(t *testing.T) {
	conn := dialWS(t, newWSTestServer(t, stubTokenValidator{
		validToken: uuid.New().String(),
		userID:     uuid.New().String(),
	}))

	require.NoError(t, conn.WriteJSON(map[string]string{"type": "auth", "token": ""}))

	require.NoError(t, conn.SetReadDeadline(time.Now().Add(2*time.Second)))
	_, _, err := conn.ReadMessage()

	assert.Error(t, err)
}

func TestServeWSMalformedFirstFrameIsRejected(t *testing.T) {
	conn := dialWS(t, newWSTestServer(t, stubTokenValidator{
		validToken: uuid.New().String(),
		userID:     uuid.New().String(),
	}))

	require.NoError(t, conn.WriteMessage(websocket.TextMessage, []byte("this is not json")))

	require.NoError(t, conn.SetReadDeadline(time.Now().Add(2*time.Second)))
	_, _, err := conn.ReadMessage()

	assert.Error(t, err)
}
