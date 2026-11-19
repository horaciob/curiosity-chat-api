package middleware

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// RateLimiterConfig holds configuration for the rate limiter.
type RateLimiterConfig struct {
	RequestsPerSecond float64
	BurstSize         int
	CleanupInterval   time.Duration
	InactiveDuration  time.Duration
	// BypassSecret is checked against the X-Bypass-Ratelimit request header.
	// When non-empty and matching, the request skips rate limiting entirely.
	BypassSecret string
}

// DefaultRateLimiterConfig returns a default configuration.
func DefaultRateLimiterConfig() RateLimiterConfig {
	return RateLimiterConfig{
		RequestsPerSecond: 10,
		BurstSize:         20,
		CleanupInterval:   5 * time.Minute,
		InactiveDuration:  10 * time.Minute,
	}
}

// ipLimiter holds the rate limiter and last seen time for an IP.
type ipLimiter struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// RateLimiter is an HTTP middleware that performs per-IP and per-user rate limiting.
type RateLimiter struct {
	config   RateLimiterConfig
	visitors map[string]*ipLimiter
	mu       sync.RWMutex
	cleaner  *time.Ticker
}

// NewRateLimiter creates a new RateLimiter middleware.
func NewRateLimiter(config RateLimiterConfig) *RateLimiter {
	rl := &RateLimiter{
		config:   config,
		visitors: make(map[string]*ipLimiter),
		cleaner:  time.NewTicker(config.CleanupInterval),
	}

	go rl.cleanupLoop()

	return rl
}

// Stop stops the cleanup goroutine.
func (rl *RateLimiter) Stop() {
	rl.cleaner.Stop()
}

// cleanupLoop periodically removes inactive limiters.
func (rl *RateLimiter) cleanupLoop() {
	for range rl.cleaner.C {
		rl.cleanup()
	}
}

// cleanup removes inactive visitors.
func (rl *RateLimiter) cleanup() {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	for ip, v := range rl.visitors {
		if time.Since(v.lastSeen) > rl.config.InactiveDuration {
			delete(rl.visitors, ip)
		}
	}
}

// getLimiter returns the rate limiter for the given identifier (IP or user).
func (rl *RateLimiter) getLimiter(identifier string) *rate.Limiter {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	v, exists := rl.visitors[identifier]
	if !exists {
		limiter := rate.NewLimiter(rate.Limit(rl.config.RequestsPerSecond), rl.config.BurstSize)
		rl.visitors[identifier] = &ipLimiter{
			limiter:  limiter,
			lastSeen: time.Now(),
		}
		return limiter
	}

	v.lastSeen = time.Now()
	return v.limiter
}


// splitAndTrim splits a string by separator and trims whitespace from each part.
func splitAndTrim(s, sep string) []string {
	parts := []string{}
	for _, part := range splitString(s, sep) {
		parts = append(parts, trimSpace(part))
	}
	return parts
}

// splitString splits a string by separator.
func splitString(s, sep string) []string {
	if s == "" {
		return nil
	}
	var result []string
	start := 0
	for i := 0; i < len(s); i++ {
		if i < len(s)-len(sep)+1 && s[i:i+len(sep)] == sep {
			result = append(result, s[start:i])
			start = i + len(sep)
		}
	}
	result = append(result, s[start:])
	return result
}

// trimSpace removes leading and trailing whitespace.
func trimSpace(s string) string {
	start := 0
	end := len(s)
	for start < end && (s[start] == ' ' || s[start] == '\t' || s[start] == '\n' || s[start] == '\r') {
		start++
	}
	for end > start && (s[end-1] == ' ' || s[end-1] == '\t' || s[end-1] == '\n' || s[end-1] == '\r') {
		end--
	}
	return s[start:end]
}

// Middleware returns the HTTP middleware function.
func (rl *RateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Allow requests that present the bypass secret header.
		if rl.config.BypassSecret != "" && r.Header.Get("X-Bypass-Ratelimit") == rl.config.BypassSecret {
			next.ServeHTTP(w, r)
			return
		}

		// Get identifier: prefer user ID if authenticated, otherwise use IP
		identifier := remoteIP(r)
		// chat-api's helper returns the id alone, empty when unauthenticated.
		if userID := UserIDFromContext(r.Context()); userID != "" {
			identifier = userID
		}

		limiter := rl.getLimiter(identifier)

		if !limiter.Allow() {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusTooManyRequests)
			json.NewEncoder(w).Encode(map[string]string{
				"error": "rate limit exceeded - too many requests",
			})
			return
		}

		next.ServeHTTP(w, r)
	})
}
