package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/horaciobranciforte/curiosity-chat-api/internal/adapter/http/response"
	"github.com/horaciobranciforte/curiosity-chat-api/internal/infrastructure/logger"
	"go.uber.org/zap"
)

type contextKey string

const UserIDKey contextKey = "user_id"

// TokenValidator validates a JWT token and returns the user ID.
type TokenValidator interface {
	Validate(token string) (string, error)
}

// Authenticate is a middleware that validates the Authorization Bearer token.
func Authenticate(validator TokenValidator) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")

			// Log incoming request details
			authStatus := "missing"
			if authHeader != "" {
				authStatus = "present"
			}

			logger.Info("[AUTH] Incoming request",
				zap.String("method", r.Method),
				zap.String("path", r.URL.Path),
				zap.String("authorization", authStatus),
				zap.String("content_type", r.Header.Get("Content-Type")),
				zap.String("user_agent", r.UserAgent()),
				zap.String("request_id", r.Header.Get("X-Request-ID")),
			)

			if authHeader == "" {
				logger.Warn("[AUTH] Missing authorization header",
					zap.String("path", r.URL.Path),
					zap.String("method", r.Method),
				)
				response.Error(w, http.StatusUnauthorized, "Unauthorized", "missing authorization header")
				return
			}

			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
				logger.Warn("[AUTH] Invalid authorization header format",
					zap.String("path", r.URL.Path),
					zap.String("method", r.Method),
					zap.String("header_prefix", parts[0]),
				)
				response.Error(w, http.StatusUnauthorized, "Unauthorized", "invalid authorization header format")
				return
			}

			userID, err := validator.Validate(parts[1])
			if err != nil {
				logger.Warn("[AUTH] Invalid or expired token",
					zap.String("path", r.URL.Path),
					zap.String("method", r.Method),
					zap.Error(err),
				)
				response.Error(w, http.StatusUnauthorized, "Unauthorized", "invalid or expired token")
				return
			}

			logger.Info("[AUTH] Authentication successful",
				zap.String("user_id", userID),
				zap.String("path", r.URL.Path),
				zap.String("method", r.Method),
			)

			ctx := context.WithValue(r.Context(), UserIDKey, userID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// UserIDFromContext extracts the user ID from the request context.
func UserIDFromContext(ctx context.Context) string {
	if v, ok := ctx.Value(UserIDKey).(string); ok {
		return v
	}
	return ""
}
