package router

import (
	"github.com/horaciobranciforte/curiosity-chat-api/internal/infrastructure/internalkeys"
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/horaciobranciforte/curiosity-chat-api/internal/adapter/http/handler"
	"github.com/horaciobranciforte/curiosity-chat-api/internal/ws"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubTokens struct{}

func (stubTokens) Validate(tok string) (string, error) {
	if tok == "good" {
		return "user-1", nil
	}
	return "", assert.AnError
}

func newTestRouter() *chi.Mux {
	hub := ws.NewHub()
	return NewRouter(handler.NewHealthHandler(), &handler.ConversationHandler{}, &handler.MessageHandler{},
		&handler.WSHandler{}, handler.NewInternalHandler(hub), stubTokens{}, internalkeys.Single("key"), nil)
}

// The route table is the API contract (API.md).
func TestRouteTableMatchesContract(t *testing.T) {
	var got []string
	require.NoError(t, chi.Walk(newTestRouter(), func(method, route string, _ http.Handler, _ ...func(http.Handler) http.Handler) error {
		if !strings.HasPrefix(route, "/swagger") {
			got = append(got, method+" "+strings.TrimSuffix(route, "/"))
		}
		return nil
	}))
	sort.Strings(got)
	want := []string{
		"GET /api/v1/conversations",
		"GET /api/v1/conversations/{id}",
		"GET /api/v1/conversations/{id}/messages",
		"GET /api/v1/ws",
		"GET /health",
		"GET /metrics",
		"POST /api/v1/conversations",
		"POST /api/v1/conversations/{id}/messages",
		"POST /internal/users/{userID}/events",
	}
	assert.Equal(t, want, got)
}

func TestConversationsRequireToken(t *testing.T) {
	r := newTestRouter()
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/api/v1/conversations", nil))
	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}

func TestInternalEventsRequireKeyAndValidJSON(t *testing.T) {
	r := newTestRouter()
	send := func(key, body string) int {
		req := httptest.NewRequest(http.MethodPost, "/internal/users/u1/events", strings.NewReader(body))
		if key != "" {
			req.Header.Set("X-Internal-Key", key)
		}
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		return rr.Code
	}
	assert.Equal(t, http.StatusUnauthorized, send("", `{"type":"notification"}`))
	assert.Equal(t, http.StatusUnauthorized, send("wrong", `{"type":"notification"}`))
	assert.Equal(t, http.StatusBadRequest, send("key", `not json`))
	assert.Equal(t, http.StatusNoContent, send("key", `{"type":"notification"}`))
}
