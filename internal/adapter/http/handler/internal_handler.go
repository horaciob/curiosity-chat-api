package handler

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/horaciobranciforte/curiosity-chat-api/internal/ws"
)

// maxEventSize bounds relayed events; they are small notification payloads.
const maxEventSize = 64 << 10

// InternalHandler handles internal service-to-service endpoints.
type InternalHandler struct {
	hub *ws.Hub
}

// NewInternalHandler creates a new InternalHandler.
func NewInternalHandler(hub *ws.Hub) *InternalHandler {
	return &InternalHandler{hub: hub}
}

// Events handles POST /internal/users/{userID}/events: the body is a JSON event that is
// delivered as-is to every WebSocket connection the user has open (user-api relays new
// notifications this way). A user with nothing open is not an error.
func (h *InternalHandler) Events(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "userID")
	payload, err := io.ReadAll(io.LimitReader(r.Body, maxEventSize))
	if err != nil || userID == "" || !json.Valid(payload) {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	h.hub.BroadcastRaw(userID, payload)
	w.WriteHeader(http.StatusNoContent)
}
