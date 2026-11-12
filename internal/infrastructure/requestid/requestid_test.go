package requestid

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEchoReturnsTheIncomingRequestID(t *testing.T) {
	h := middleware.RequestID(Echo(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {})))
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set(Header, "abc-123")
	w := httptest.NewRecorder()

	h.ServeHTTP(w, req)

	assert.Equal(t, "abc-123", w.Header().Get(Header))
}

func TestEchoReturnsAGeneratedIDWhenNoneCame(t *testing.T) {
	h := middleware.RequestID(Echo(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {})))
	w := httptest.NewRecorder()

	h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/", nil))

	assert.NotEmpty(t, w.Header().Get(Header))
}

func TestClientForwardsTheContextRequestID(t *testing.T) {
	var got string
	srv := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) { got = r.Header.Get(Header) }))
	defer srv.Close()

	ctx := context.WithValue(context.Background(), middleware.RequestIDKey, "trace-me")
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, srv.URL, nil)
	require.NoError(t, err)
	resp, err := NewClient(time.Second).Do(req)
	require.NoError(t, err)
	resp.Body.Close()

	assert.Equal(t, "trace-me", got)
	assert.Empty(t, req.Header.Get(Header), "the caller's request is not mutated")
}

func TestClientKeepsAnExplicitHeaderAndSkipsEmptyIDs(t *testing.T) {
	var got []string
	srv := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) { got = append(got, r.Header.Get(Header)) }))
	defer srv.Close()
	c := NewClient(time.Second)

	ctx := context.WithValue(context.Background(), middleware.RequestIDKey, "from-ctx")
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, srv.URL, nil)
	req.Header.Set(Header, "explicit")
	resp, err := c.Do(req)
	require.NoError(t, err)
	resp.Body.Close()

	req2, _ := http.NewRequest(http.MethodGet, srv.URL, nil)
	resp, err = c.Do(req2)
	require.NoError(t, err)
	resp.Body.Close()

	assert.Equal(t, []string{"explicit", ""}, got)
}
