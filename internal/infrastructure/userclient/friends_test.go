package userclient

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAreFriendsAsksUserAPI(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/internal/friendships", r.URL.Path)
		assert.Equal(t, "a,b", r.URL.Query().Get("users"))
		assert.Equal(t, "k", r.Header.Get("X-Internal-Key"))
		_, _ = w.Write([]byte(`{"friends":true}`))
	}))
	defer srv.Close()

	ok, err := NewClient(srv.URL, "k").AreFriends(context.Background(), "a", "b")
	require.NoError(t, err)
	assert.True(t, ok)
}

type countingFriends struct {
	calls   int
	answer  bool
	failing bool
}

func (c *countingFriends) AreFriends(context.Context, string, string) (bool, error) {
	c.calls++
	if c.failing {
		return false, errors.New("user-api down")
	}
	return c.answer, nil
}

func TestFriendCacheAsksOncePerPairUntilItExpires(t *testing.T) {
	inner := &countingFriends{answer: true}
	cache := NewFriendCache(inner, time.Minute)
	now := time.Now()
	cache.now = func() time.Time { return now }

	for i := 0; i < 3; i++ {
		ok, err := cache.AreFriends(context.Background(), "a", "b")
		require.NoError(t, err)
		assert.True(t, ok)
	}
	_, _ = cache.AreFriends(context.Background(), "b", "a") // same pair, either order
	assert.Equal(t, 1, inner.calls)

	now = now.Add(2 * time.Minute)
	_, _ = cache.AreFriends(context.Background(), "a", "b")
	assert.Equal(t, 2, inner.calls)
}

func TestFriendCacheDoesNotCacheErrors(t *testing.T) {
	inner := &countingFriends{failing: true}
	cache := NewFriendCache(inner, time.Minute)

	_, err := cache.AreFriends(context.Background(), "a", "b")
	require.Error(t, err)
	_, _ = cache.AreFriends(context.Background(), "a", "b")
	assert.Equal(t, 2, inner.calls)
}
