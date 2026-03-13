package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/jsonapi"
	"github.com/stretchr/testify/assert"
)

func TestHealthHandlerHealth(t *testing.T) {
	handler := NewHealthHandler()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
	rr := httptest.NewRecorder()

	handler.Health(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, jsonapi.MediaType, rr.Header().Get("Content-Type"))
	assert.Contains(t, rr.Body.String(), "ok")
	assert.Contains(t, rr.Body.String(), "health")
}
