package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSkipForWebSocketBypassesOnlyUpgrades(t *testing.T) {
	applied := 0
	mw := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			applied++
			next.ServeHTTP(w, r)
		})
	}
	h := SkipForWebSocket(mw)(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))

	upgrade := httptest.NewRequest(http.MethodGet, "/api/v1/ws", nil)
	upgrade.Header.Set("Upgrade", "websocket")
	h.ServeHTTP(httptest.NewRecorder(), upgrade)
	assert.Equal(t, 0, applied)

	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/api/v1/conversations", nil))
	assert.Equal(t, 1, applied)
}

// The recorder must pass hijacking through, or WebSocket upgrades fail behind Metrics.
func TestMetricsRecorderSupportsHijack(t *testing.T) {
	var _ http.Hijacker = &metricsRecorder{}
}
