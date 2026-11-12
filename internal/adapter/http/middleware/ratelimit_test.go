package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func okHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("ok"))
	})
}

// A burst must pass and the excess must be refused, or the limiter is either useless or a
// denial of service against the app's own traffic.
func TestRateLimiterAllowsTheBurstThenRefuses(t *testing.T) {
	rl := NewRateLimiter(RateLimiterConfig{
		RequestsPerSecond: 5,
		BurstSize:         3,
		CleanupInterval:   time.Minute,
		InactiveDuration:  time.Minute,
	})
	h := rl.Middleware(okHandler())

	var allowed, refused int
	for i := 0; i < 10; i++ {
		req := httptest.NewRequest(http.MethodGet, "/pois", nil)
		req.RemoteAddr = "203.0.113.10:1234"
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)

		if rec.Code == http.StatusTooManyRequests {
			refused++
		} else {
			allowed++
		}
	}

	assert.GreaterOrEqual(t, allowed, 3, "the configured burst must get through")
	assert.Greater(t, refused, 0, "the excess must be refused")
}

// One noisy client must not spend another client's allowance.
func TestRateLimiterCountsEachClientSeparately(t *testing.T) {
	rl := NewRateLimiter(RateLimiterConfig{
		RequestsPerSecond: 1,
		BurstSize:         2,
		CleanupInterval:   time.Minute,
		InactiveDuration:  time.Minute,
	})
	h := rl.Middleware(okHandler())

	exhaust := func(ip string) {
		for i := 0; i < 5; i++ {
			req := httptest.NewRequest(http.MethodGet, "/pois", nil)
			req.RemoteAddr = ip + ":1234"
			h.ServeHTTP(httptest.NewRecorder(), req)
		}
	}
	exhaust("203.0.113.20")

	req := httptest.NewRequest(http.MethodGet, "/pois", nil)
	req.RemoteAddr = "203.0.113.21:1234"
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	assert.NotEqual(t, http.StatusTooManyRequests, rec.Code,
		"a second client must start with its own allowance")
}

func TestRateLimiterAnswersWithJSONSoClientsCanReadIt(t *testing.T) {
	rl := NewRateLimiter(RateLimiterConfig{
		RequestsPerSecond: 1,
		BurstSize:         1,
		CleanupInterval:   time.Minute,
		InactiveDuration:  time.Minute,
	})
	h := rl.Middleware(okHandler())

	var last *httptest.ResponseRecorder
	for i := 0; i < 5; i++ {
		req := httptest.NewRequest(http.MethodGet, "/pois", nil)
		req.RemoteAddr = "203.0.113.30:1234"
		last = httptest.NewRecorder()
		h.ServeHTTP(last, req)
	}

	require.Equal(t, http.StatusTooManyRequests, last.Code)
	assert.Contains(t, last.Header().Get("Content-Type"), "json")
	assert.NotEmpty(t, last.Body.String(), "a refusal must say why")
}

func TestDefaultRateLimiterConfigIsUsable(t *testing.T) {
	cfg := DefaultRateLimiterConfig()
	assert.Greater(t, cfg.RequestsPerSecond, float64(0))
	assert.Greater(t, cfg.BurstSize, 0)
	assert.GreaterOrEqual(t, cfg.BurstSize, int(cfg.RequestsPerSecond),
		"a burst smaller than the sustained rate would refuse traffic it then allows")
}
