package middleware

import (
	"net/http"

	"github.com/horaciobranciforte/curiosity-chat-api/internal/adapter/http/response"
)

const internalKeyHeader = "X-Internal-Key"

// InternalAuthenticate returns a middleware that requires a valid X-Internal-Key header.
// Use this to protect routes that should only be called by other internal services.
func InternalAuthenticate(key string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Header.Get(internalKeyHeader) != key {
				response.Error(w, http.StatusUnauthorized, "Unauthorized", "valid X-Internal-Key header required")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
