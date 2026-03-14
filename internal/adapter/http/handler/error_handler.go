package handler

import (
	"errors"
	"net/http"

	"github.com/horaciobranciforte/curiosity-chat-api/internal/adapter/http/response"
	"github.com/horaciobranciforte/curiosity-chat-api/internal/infrastructure/logger"
	"github.com/horaciobranciforte/curiosity-chat-api/internal/pkg/apperror"
	"go.uber.org/zap"
)

// handleUseCaseError maps an apperror.Error to the appropriate HTTP response.
func handleUseCaseError(w http.ResponseWriter, err error) {
	var appErr *apperror.Error
	if errors.As(err, &appErr) {
		title := httpTitle(appErr.HTTPStatus())

		// Log 403 Forbidden responses with detailed context
		if appErr.HTTPStatus() == http.StatusForbidden {
			logger.Error("[ERROR] 403 Forbidden response",
				zap.String("title", title),
				zap.String("detail", appErr.Message),
				zap.String("error_code", appErr.Code),
			)
		}

		response.Error(w, appErr.HTTPStatus(), title, appErr.Message)
		return
	}

	// Log unhandled errors
	logger.Error("[ERROR] Unhandled error in use case",
		zap.Error(err),
	)

	response.Error(w, http.StatusInternalServerError, "Internal Server Error", err.Error())
}

func httpTitle(status int) string {
	switch status {
	case http.StatusBadRequest:
		return "Bad Request"
	case http.StatusUnauthorized:
		return "Unauthorized"
	case http.StatusForbidden:
		return "Forbidden"
	case http.StatusNotFound:
		return "Not Found"
	case http.StatusConflict:
		return "Conflict"
	default:
		return "Internal Server Error"
	}
}
