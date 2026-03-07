package middleware

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

// stubValidator is a simple TokenValidator for testing.
type stubValidator struct {
	userID string
	err    error
}

func (s *stubValidator) Validate(_ string) (string, error) {
	return s.userID, s.err
}

func nextHandlerCapturingUserID(capturedID *string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		*capturedID = UserIDFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	})
}

// ---- Authenticate middleware ----

func TestAuthenticateMissingAuthorizationHeader(t *testing.T) {
	handler := Authenticate(&stubValidator{userID: "uid"})
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("next handler must not be called when header is missing")
	})).ServeHTTP(rr, req)

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}

func TestAuthenticateInvalidHeaderNoBearerPrefix(t *testing.T) {
	handler := Authenticate(&stubValidator{userID: "uid"})
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Token sometoken")

	handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("next handler must not be called with wrong scheme")
	})).ServeHTTP(rr, req)

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}

func TestAuthenticateInvalidHeaderNoSpace(t *testing.T) {
	handler := Authenticate(&stubValidator{userID: "uid"})
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "baretoken")

	handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("next handler must not be called with malformed header")
	})).ServeHTTP(rr, req)

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}

func TestAuthenticateInvalidToken(t *testing.T) {
	validator := &stubValidator{err: errors.New("invalid token")}
	handler := Authenticate(validator)
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer badtoken")

	handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("next handler must not be called with invalid token")
	})).ServeHTTP(rr, req)

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}

func TestAuthenticateSuccessCallsNextWithUserID(t *testing.T) {
	userID := "user-123"
	validator := &stubValidator{userID: userID}
	handler := Authenticate(validator)

	var capturedID string
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer validtoken")

	handler(nextHandlerCapturingUserID(&capturedID)).ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, userID, capturedID)
}

func TestAuthenticateBearerCaseInsensitive(t *testing.T) {
	validator := &stubValidator{userID: "uid-abc"}
	handler := Authenticate(validator)

	var capturedID string
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "BEARER validtoken")

	handler(nextHandlerCapturingUserID(&capturedID)).ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "uid-abc", capturedID)
}

// ---- UserIDFromContext ----

func TestUserIDFromContextReturnsEmptyWhenNotSet(t *testing.T) {
	assert.Empty(t, UserIDFromContext(context.Background()))
}

func TestUserIDFromContextReturnsStoredValue(t *testing.T) {
	ctx := context.WithValue(context.Background(), UserIDKey, "user-xyz")
	assert.Equal(t, "user-xyz", UserIDFromContext(ctx))
}

func TestUserIDFromContextIgnoresWrongType(t *testing.T) {
	ctx := context.WithValue(context.Background(), UserIDKey, 12345)
	assert.Empty(t, UserIDFromContext(ctx))
}
