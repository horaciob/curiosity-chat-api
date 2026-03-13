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

// Health handles GET /api/v1/health.
//
//	@Summary		Health check
//	@Description	Returns the service status in JSON:API format
//	@Tags			health
//	@Produce		json
//	@Success		200	{object}	HealthJSONAPIResponse	"Service status"
//	@Router			/health [get]
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
