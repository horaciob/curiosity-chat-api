package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/horaciobranciforte/curiosity-chat-api/internal/adapter/http/response"
	"github.com/horaciobranciforte/curiosity-chat-api/internal/infrastructure/tokenstore"
)

const accessTokenExpiresIn = int(time.Hour / time.Second) // 3600

type tokenJWTSvc interface {
	Issue(userID string) (string, error)
	IssueRefreshToken(ctx context.Context, userID string, store tokenstore.Store) (string, error)
	ValidateAndRotateRefreshToken(ctx context.Context, rawToken string, store tokenstore.Store) (newAccessToken, newRawRefresh string, err error)
}

type TokenHandler struct {
	jwtSvc tokenJWTSvc
	store  tokenstore.Store
}

func NewTokenHandler(jwtSvc tokenJWTSvc, store tokenstore.Store) *TokenHandler {
	return &TokenHandler{jwtSvc: jwtSvc, store: store}
}

type tokenPairResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
}

// IssueInternal handles POST /internal/token.
// Protected by InternalAuthenticate middleware.
func (h *TokenHandler) IssueInternal(w http.ResponseWriter, r *http.Request) {
	var req struct {
		UserID string `json:"user_id"`
	}
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

	accessToken, err := h.jwtSvc.Issue(req.UserID)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "Internal Server Error", "failed to issue access token")
		return
	}
	refreshToken, err := h.jwtSvc.IssueRefreshToken(r.Context(), req.UserID, h.store)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "Internal Server Error", "failed to issue refresh token")
		return
	}

	response.JSON(w, http.StatusOK, tokenPairResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    accessTokenExpiresIn,
	})
}

// Refresh handles POST /api/v1/token/refresh.
// Public endpoint — rotates the refresh token and issues a new access+refresh pair.
func (h *TokenHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	var req struct {
		RefreshToken string `json:"refresh_token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "Bad Request", "invalid request body")
		return
	}
	if req.RefreshToken == "" {
		response.Error(w, http.StatusBadRequest, "Bad Request", "refresh_token is required")
		return
	}

	newAccess, newRefresh, err := h.jwtSvc.ValidateAndRotateRefreshToken(r.Context(), req.RefreshToken, h.store)
	if err != nil {
		if errors.Is(err, tokenstore.ErrTokenNotFound) {
			response.Error(w, http.StatusUnauthorized, "Unauthorized", "refresh token is invalid or expired")
			return
		}
		response.Error(w, http.StatusInternalServerError, "Internal Server Error", "failed to refresh token")
		return
	}

	response.JSON(w, http.StatusOK, tokenPairResponse{
		AccessToken:  newAccess,
		RefreshToken: newRefresh,
		ExpiresIn:    accessTokenExpiresIn,
	})
}
