package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestInternalAuthenticateMissingKey(t *testing.T) {
	handler := InternalAuthenticate("secret-key")
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/internal", nil)

	handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("next handler must not be called without header")
	})).ServeHTTP(rr, req)

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}

func TestInternalAuthenticateWrongKey(t *testing.T) {
	handler := InternalAuthenticate("secret-key")
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/internal", nil)
	req.Header.Set("X-Internal-Key", "wrong-key")

	handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("next handler must not be called with wrong key")
	})).ServeHTTP(rr, req)

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}

func TestInternalAuthenticateEmptyConfiguredKey(t *testing.T) {
	// Even with an empty configured key, a missing header still fails.
	handler := InternalAuthenticate("")
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/internal", nil)

	handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("next handler must not be called without header")
	})).ServeHTTP(rr, req)

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}

func TestInternalAuthenticateSuccess(t *testing.T) {
	const key = "1nt3rn4l"
	handler := InternalAuthenticate(key)

	nextCalled := false
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/internal/something", nil)
	req.Header.Set("X-Internal-Key", key)

	handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
		w.WriteHeader(http.StatusOK)
	})).ServeHTTP(rr, req)

	assert.True(t, nextCalled)
	assert.Equal(t, http.StatusOK, rr.Code)
}
