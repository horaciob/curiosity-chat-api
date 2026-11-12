package userclient

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNamesSendsKeyAndDecodes(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/internal/users", r.URL.Path)
		assert.Equal(t, "a,b", r.URL.Query().Get("ids"))
		assert.Equal(t, "secret", r.Header.Get("X-Internal-Key"))
		_ = json.NewEncoder(w).Encode(map[string]any{"names": map[string]string{"a": "Ana"}})
	}))
	defer srv.Close()

	names, err := NewClient(srv.URL+"/", "secret").Names(context.Background(), []string{"a", "b"})

	require.NoError(t, err)
	assert.Equal(t, map[string]string{"a": "Ana"}, names)
}

func TestChatPushPostsMessage(t *testing.T) {
	var got map[string]string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/internal/users/r1/pushes", r.URL.Path)
		_ = json.NewDecoder(r.Body).Decode(&got)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	err := NewClient(srv.URL, "k").ChatPush(context.Background(), "r1", "s1", "c1", "hola")

	require.NoError(t, err)
	assert.Equal(t, map[string]string{"sender_id": "s1", "conversation_id": "c1", "text": "hola"}, got)
}

func TestChatPushReportsServerErrors(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	assert.Error(t, NewClient(srv.URL, "k").ChatPush(context.Background(), "r1", "s1", "c1", ""))
}
