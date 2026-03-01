package handler

import (
	"errors"
	"net/http"

	"github.com/horaciobranciforte/curiosity-chat-api/internal/adapter/http/response"
	"github.com/horaciobranciforte/curiosity-chat-api/internal/pkg/apperror"
)

// handleUseCaseError maps an apperror.Error to the appropriate HTTP response.
func handleUseCaseError(w http.ResponseWriter, err error) {
	var appErr *apperror.Error
	if errors.As(err, &appErr) {
		title := httpTitle(appErr.HTTPStatus())
		response.Error(w, appErr.HTTPStatus(), title, appErr.Message)
		return
	}
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
