package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/horaciobranciforte/curiosity-chat-api/internal/adapter/http/response"
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
			if authHeader == "" {
				response.Error(w, http.StatusUnauthorized, "Unauthorized", "missing authorization header")
				return
			}

			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
				response.Error(w, http.StatusUnauthorized, "Unauthorized", "invalid authorization header format")
				return
			}

			userID, err := validator.Validate(parts[1])
			if err != nil {
				response.Error(w, http.StatusUnauthorized, "Unauthorized", "invalid or expired token")
				return
			}

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
