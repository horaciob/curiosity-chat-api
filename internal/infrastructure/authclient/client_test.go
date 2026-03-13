package authclient

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
	client := NewClient("http://auth.example.com", "secret-key")

	assert.NotNil(t, client)
	assert.Equal(t, "http://auth.example.com", client.baseURL)
	assert.Equal(t, "secret-key", client.internalKey)
	assert.NotNil(t, client.httpClient)
	assert.Equal(t, 5*time.Second, client.httpClient.Timeout)
}

func TestNewClientTrimsTrailingSlash(t *testing.T) {
	client := NewClient("http://auth.example.com/", "secret-key")

	assert.Equal(t, "http://auth.example.com", client.baseURL)
}

func TestValidateSuccess(t *testing.T) {
	expectedUserID := "user-123"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/internal/token/validate", r.URL.Path)
		assert.Equal(t, "test-internal-key", r.Header.Get("X-Internal-Key"))
		assert.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"user_id": expectedUserID})
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-internal-key")
	userID, err := client.Validate("test-token")

	require.NoError(t, err)
	assert.Equal(t, expectedUserID, userID)
}

func TestValidateCtxSuccess(t *testing.T) {
	expectedUserID := "user-456"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"user_id": expectedUserID})
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-internal-key")
	ctx := context.Background()
	userID, err := client.ValidateCtx(ctx, "test-token")

	require.NoError(t, err)
	assert.Equal(t, expectedUserID, userID)
}

func TestValidateUnauthorized(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-internal-key")
	userID, err := client.Validate("invalid-token")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid or expired token")
	assert.Empty(t, userID)
}

func TestValidateBadRequest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-internal-key")
	userID, err := client.Validate("test-token")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "auth-api returned status 400")
	assert.Empty(t, userID)
}

func TestValidateServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-internal-key")
	userID, err := client.Validate("test-token")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "auth-api returned status 500")
	assert.Empty(t, userID)
}

func TestValidateEmptyUserID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"user_id": ""})
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-internal-key")
	userID, err := client.Validate("test-token")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "auth-api returned empty user_id")
	assert.Empty(t, userID)
}

func TestValidateInvalidJSONResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("invalid json"))
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-internal-key")
	userID, err := client.Validate("test-token")

	assert.Error(t, err)
	assert.Empty(t, userID)
}

func TestValidateNetworkError(t *testing.T) {
	client := NewClient("http://localhost:59999", "test-internal-key")
	userID, err := client.Validate("test-token")

	assert.Error(t, err)
	assert.Empty(t, userID)
}

func TestValidateCtxCanceled(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"user_id": "user-123"})
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-internal-key")
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()

	// Wait for context to expire
	time.Sleep(10 * time.Millisecond)

	userID, err := client.ValidateCtx(ctx, "test-token")

	assert.Error(t, err)
	assert.Empty(t, userID)
}

func TestValidateMissingUserIDField(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		// Return JSON without user_id field
		json.NewEncoder(w).Encode(map[string]string{"other_field": "value"})
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-internal-key")
	userID, err := client.Validate("test-token")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "auth-api returned empty user_id")
	assert.Empty(t, userID)
}

func TestClientImplementsTokenValidator(t *testing.T) {
	// Verify that Client satisfies the expected interface
	client := NewClient("http://auth.example.com", "secret-key")

	// This test ensures the Validate method has the correct signature
	// The client should work with middleware that expects Validate(string) (string, error)
	var validateFunc func(string) (string, error) = client.Validate
	assert.NotNil(t, validateFunc)
}

func TestValidateWithDifferentStatusCodes(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		wantErr    bool
		errContain string
	}{
		{"forbidden", http.StatusForbidden, true, "auth-api returned status 403"},
		{"not found", http.StatusNotFound, true, "auth-api returned status 404"},
		{"conflict", http.StatusConflict, true, "auth-api returned status 409"},
		{"gateway error", http.StatusBadGateway, true, "auth-api returned status 502"},
		{"service unavailable", http.StatusServiceUnavailable, true, "auth-api returned status 503"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
			}))
			defer server.Close()

			client := NewClient(server.URL, "test-internal-key")
			userID, err := client.Validate("test-token")

			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContain)
				assert.Empty(t, userID)
			}
		})
	}
}
