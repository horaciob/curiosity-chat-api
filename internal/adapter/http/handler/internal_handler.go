package handler

import (
	"encoding/json"
	"net/http"

	"github.com/horaciobranciforte/curiosity-chat-api/internal/ws"
)

// InternalHandler handles internal service-to-service endpoints.
type InternalHandler struct {
	hub *ws.Hub
}

// NewInternalHandler creates a new InternalHandler.
func NewInternalHandler(hub *ws.Hub) *InternalHandler {
	return &InternalHandler{hub: hub}
}

type pushRequest struct {
	UserID  string          `json:"user_id"`
	Payload json.RawMessage `json:"payload"`
}

// Push handles POST /internal/push — receives a JSON payload from another service
// and broadcasts it to the target user via WebSocket.
func (h *InternalHandler) Push(w http.ResponseWriter, r *http.Request) {
	var req pushRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.UserID == "" || len(req.Payload) == 0 {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	h.hub.BroadcastRaw(req.UserID, req.Payload)
	w.WriteHeader(http.StatusNoContent)
}
