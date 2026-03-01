package handler

import (
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
	"github.com/horaciobranciforte/curiosity-chat-api/internal/adapter/http/response"
)

// TokenIssuer can issue a signed JWT for a given user ID.
type TokenIssuer interface {
	Issue(userID string) (string, error)
}

// TokenHandler handles POST /api/v1/token.
type TokenHandler struct {
	issuer TokenIssuer
}

// NewTokenHandler creates a new TokenHandler.
func NewTokenHandler(issuer TokenIssuer) *TokenHandler {
	return &TokenHandler{issuer: issuer}
}

type issueTokenRequest struct {
	UserID string `json:"user_id"`
}

// Issue handles POST /api/v1/token.
//
//	@Summary		Issue a chat JWT
//	@Description	Issues a signed JWT for the given user ID. Used by mobile clients that authenticate via the main curiosity-api SSO flow and need a token to call the chat API.
//	@Tags			auth
//	@Accept			json
//	@Produce		json
//	@Param			body	body		issueTokenRequest	true	"User ID"
//	@Success		200		{object}	map[string]string	"token"
//	@Failure		400		{object}	map[string]string	"Bad request"
//	@Failure		500		{object}	map[string]string	"Internal error"
//	@Router			/token [post]
func (h *TokenHandler) Issue(w http.ResponseWriter, r *http.Request) {
	var req issueTokenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "Bad Request", "invalid request body")
		return
	}

	if req.UserID == "" {
		response.Error(w, http.StatusBadRequest, "Bad Request", "user_id is required")
		return
	}

	if _, err := uuid.Parse(req.UserID); err != nil {
		response.Error(w, http.StatusBadRequest, "Bad Request", "user_id must be a valid UUID")
		return
	}

	token, err := h.issuer.Issue(req.UserID)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "Internal Server Error", "failed to issue token")
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{"token": token})
}
