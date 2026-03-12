package handler

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/horaciobranciforte/curiosity-chat-api/internal/pkg/apperror"
	"github.com/stretchr/testify/assert"
)

func TestHandleUseCaseErrorValidation(t *testing.T) {
	rr := httptest.NewRecorder()
	err := apperror.Validation("invalid input", nil)

	handleUseCaseError(rr, err)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	assert.Contains(t, rr.Body.String(), "invalid input")
}

func TestHandleUseCaseErrorNotFound(t *testing.T) {
	rr := httptest.NewRecorder()
	err := apperror.NotFound("conversation not found", nil)

	handleUseCaseError(rr, err)

	assert.Equal(t, http.StatusNotFound, rr.Code)
	assert.Contains(t, rr.Body.String(), "conversation not found")
}

func TestHandleUseCaseErrorForbidden(t *testing.T) {
	rr := httptest.NewRecorder()
	err := apperror.Forbidden("access denied", nil)

	handleUseCaseError(rr, err)

	assert.Equal(t, http.StatusForbidden, rr.Code)
	assert.Contains(t, rr.Body.String(), "access denied")
}

func TestHandleUseCaseErrorConflict(t *testing.T) {
	rr := httptest.NewRecorder()
	err := apperror.Conflict("already exists", nil)

	handleUseCaseError(rr, err)

	assert.Equal(t, http.StatusConflict, rr.Code)
	assert.Contains(t, rr.Body.String(), "already exists")
}

func TestHandleUseCaseErrorInternal(t *testing.T) {
	rr := httptest.NewRecorder()
	err := apperror.Internal("database error", errors.New("connection failed"))

	handleUseCaseError(rr, err)

	assert.Equal(t, http.StatusInternalServerError, rr.Code)
	assert.Contains(t, rr.Body.String(), "database error")
}

func TestHandleUseCaseErrorGenericError(t *testing.T) {
	rr := httptest.NewRecorder()
	err := errors.New("some random error")

	handleUseCaseError(rr, err)

	assert.Equal(t, http.StatusInternalServerError, rr.Code)
	assert.Contains(t, rr.Body.String(), "some random error")
}

func TestHTTPTitle(t *testing.T) {
	tests := []struct {
		status   int
		expected string
	}{
		{http.StatusBadRequest, "Bad Request"},
		{http.StatusUnauthorized, "Unauthorized"},
		{http.StatusForbidden, "Forbidden"},
		{http.StatusNotFound, "Not Found"},
		{http.StatusConflict, "Conflict"},
		{http.StatusInternalServerError, "Internal Server Error"},
		{999, "Internal Server Error"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := httpTitle(tt.status)
			assert.Equal(t, tt.expected, result)
		})
	}
}
