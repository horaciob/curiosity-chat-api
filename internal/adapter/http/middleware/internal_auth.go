package middleware

import (
	"github.com/horaciobranciforte/curiosity-chat-api/internal/infrastructure/internalkeys"
	"net/http"

	"github.com/horaciobranciforte/curiosity-chat-api/internal/adapter/http/response"
)

const internalKeyHeader = "X-Internal-Key"

// InternalAuthenticate requires the single shared X-Internal-Key.
func InternalAuthenticate(key string) func(http.Handler) http.Handler {
	return InternalAuthenticateKeys(internalkeys.Single(key))
}

// InternalAuthenticateKeys requires an X-Internal-Key from keys and records which caller
// it belongs to. Use it on routes only other internal services may call.
func InternalAuthenticateKeys(keys *internalkeys.Set) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			caller, ok := keys.Match(r.Header.Get(internalKeyHeader))
			if !ok {
				response.Error(w, http.StatusUnauthorized, "Unauthorized", "valid X-Internal-Key header required")
				return
			}
			next.ServeHTTP(w, r.WithContext(internalkeys.WithCaller(r.Context(), caller)))
		})
	}
}
