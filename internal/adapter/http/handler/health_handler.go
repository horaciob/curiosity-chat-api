package handler

import (
	"net/http"

	"github.com/google/jsonapi"
	"github.com/horaciobranciforte/curiosity-chat-api/internal/adapter/http/response"
)

// HealthResponse is the JSON:API DTO for health status.
type HealthResponse struct {
	ID     string `jsonapi:"primary,health"`
	Status string `jsonapi:"attr,status"`
}

// HealthHandler handles the health check endpoint.
type HealthHandler struct{}

// NewHealthHandler creates a new HealthHandler.
func NewHealthHandler() *HealthHandler { return &HealthHandler{} }

// Health handles GET /health.
func (h *HealthHandler) Health(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", jsonapi.MediaType)
	w.WriteHeader(http.StatusOK)

	health := &HealthResponse{
		ID:     "status",
		Status: "ok",
	}

	if err := jsonapi.MarshalPayload(w, health); err != nil {
		response.Error(w, http.StatusInternalServerError, "Internal Server Error", "failed to marshal health response")
	}
}
