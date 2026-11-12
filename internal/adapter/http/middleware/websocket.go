package middleware

import (
	"net/http"
	"strings"
)

// SkipForWebSocket applies mw to every request except WebSocket upgrades, which need the
// raw connection and cannot pass through writers that buffer or transform the body.
func SkipForWebSocket(mw func(http.Handler) http.Handler) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		wrapped := mw(next)
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if strings.EqualFold(r.Header.Get("Upgrade"), "websocket") {
				next.ServeHTTP(w, r)
				return
			}
			wrapped.ServeHTTP(w, r)
		})
	}
}
