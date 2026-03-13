package followclient

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewClient(t *testing.T) {
	client := NewClient("http://user.example.com", "secret-key")

	assert.NotNil(t, client)
	assert.Equal(t, "http://user.example.com", client.baseURL)
	assert.Equal(t, "secret-key", client.internalKey)
	assert.NotNil(t, client.httpClient)
	assert.Equal(t, 5*time.Second, client.httpClient.Timeout)
}

func TestAreFollowingSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/internal/users/user-a/is-mutual-following/user-b", r.URL.Path)
		assert.Equal(t, "test-internal-key", r.Header.Get("X-Internal-Key"))

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(mutualFollowResponse{Mutual: true})
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-internal-key")
	mutual, err := client.AreFollowing(context.Background(), "user-a", "user-b")

	require.NoError(t, err)
	assert.True(t, mutual)
}

func TestAreFollowingNotMutual(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(mutualFollowResponse{Mutual: false})
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-internal-key")
	mutual, err := client.AreFollowing(context.Background(), "user-a", "user-b")

	require.NoError(t, err)
	assert.False(t, mutual)
}

func TestAreFollowingBadRequest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-internal-key")
	mutual, err := client.AreFollowing(context.Background(), "user-a", "user-b")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "user-api returned status 400")
	assert.False(t, mutual)
}

func TestAreFollowingUnauthorized(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-internal-key")
	mutual, err := client.AreFollowing(context.Background(), "user-a", "user-b")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "user-api returned status 401")
	assert.False(t, mutual)
}

func TestAreFollowingServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-internal-key")
	mutual, err := client.AreFollowing(context.Background(), "user-a", "user-b")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "user-api returned status 500")
	assert.False(t, mutual)
}

func TestAreFollowingInvalidJSONResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("invalid json"))
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-internal-key")
	mutual, err := client.AreFollowing(context.Background(), "user-a", "user-b")

	assert.Error(t, err)
	assert.False(t, mutual)
}

func TestAreFollowingNetworkError(t *testing.T) {
	client := NewClient("http://localhost:59999", "test-internal-key")
	mutual, err := client.AreFollowing(context.Background(), "user-a", "user-b")

	assert.Error(t, err)
	assert.False(t, mutual)
}

func TestAreFollowingCtxCanceled(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(mutualFollowResponse{Mutual: true})
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-internal-key")
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()

	// Wait for context to expire
	time.Sleep(10 * time.Millisecond)

	mutual, err := client.AreFollowing(ctx, "user-a", "user-b")

	assert.Error(t, err)
	assert.False(t, mutual)
}

func TestAreFollowingWithDifferentStatusCodes(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		wantErr    bool
		errContain string
	}{
		{"not found", http.StatusNotFound, true, "user-api returned status 404"},
		{"forbidden", http.StatusForbidden, true, "user-api returned status 403"},
		{"conflict", http.StatusConflict, true, "user-api returned status 409"},
		{"gateway error", http.StatusBadGateway, true, "user-api returned status 502"},
		{"service unavailable", http.StatusServiceUnavailable, true, "user-api returned status 503"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
			}))
			defer server.Close()

			client := NewClient(server.URL, "test-internal-key")
			mutual, err := client.AreFollowing(context.Background(), "user-a", "user-b")

			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContain)
				assert.False(t, mutual)
			}
		})
	}
}

// NoopFollowChecker tests

func TestNoopFollowCheckerAlwaysReturnsTrue(t *testing.T) {
	checker := NoopFollowChecker{}

	mutual, err := checker.AreFollowing(context.Background(), "any-user", "any-other-user")

	require.NoError(t, err)
	assert.True(t, mutual)
}

func TestNoopFollowCheckerIgnoresContext(t *testing.T) {
	checker := NoopFollowChecker{}
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	// Should still return true even with canceled context
	mutual, err := checker.AreFollowing(ctx, "user-a", "user-b")

	require.NoError(t, err)
	assert.True(t, mutual)
}

func TestNoopFollowCheckerIgnoresUserIDs(t *testing.T) {
	checker := NoopFollowChecker{}

	// Should return true for any combination of user IDs
	tests := []struct {
		userA string
		userB string
	}{
		{"", ""},
		{"user-1", ""},
		{"", "user-2"},
		{"user-1", "user-2"},
		{"same", "same"},
	}

	for _, tt := range tests {
		mutual, err := checker.AreFollowing(context.Background(), tt.userA, tt.userB)
		require.NoError(t, err)
		assert.True(t, mutual)
	}
}
