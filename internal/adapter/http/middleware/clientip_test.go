package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func seenIP(t *testing.T, trustedSpec, remote, xff string) string {
	t.Helper()
	trusted, err := ParseTrustedProxies(trustedSpec)
	require.NoError(t, err)
	var got string
	h := TrustedRealIP(trusted)(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) { got = remoteIP(r) }))
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = remote
	if xff != "" {
		req.Header.Set("X-Forwarded-For", xff)
	}
	h.ServeHTTP(httptest.NewRecorder(), req)
	return got
}

func TestForwardedForIsIgnoredFromStrangers(t *testing.T) {
	assert.Equal(t, "203.0.113.9", seenIP(t, "", "203.0.113.9:5555", "1.2.3.4"))
	assert.Equal(t, "203.0.113.9", seenIP(t, "10.0.0.0/8", "203.0.113.9:5555", "1.2.3.4"))
}

func TestForwardedForCountsBehindOurProxy(t *testing.T) {
	assert.Equal(t, "198.51.100.7", seenIP(t, "10.0.0.0/8", "10.0.0.2:5555", "198.51.100.7"))
}

func TestASpoofedLeftmostHopDoesNotWin(t *testing.T) {
	// The client prepended a fake address; our proxy appended the real one.
	assert.Equal(t, "198.51.100.7", seenIP(t, "10.0.0.2", "10.0.0.2:5555", "6.6.6.6, 198.51.100.7"))
}

func TestBadTrustedProxiesIsAnError(t *testing.T) {
	_, err := ParseTrustedProxies("not-a-cidr/99")
	assert.Error(t, err)
}
