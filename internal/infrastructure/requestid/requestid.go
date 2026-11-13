// Package requestid carries one request ID across services.
//
// chi's middleware.RequestID accepts an incoming X-Request-Id or makes one up. This package
// sends it back to the caller (Echo) and forwards it on every outbound call made with the
// request's context (Transport), so one ID ties together the log lines a single user action
// leaves in user-api, poi-api and chat-api.
//
// The same file lives in each Go service; keep the copies identical.
package requestid

import (
	"github.com/horaciobranciforte/curiosity-chat-api/internal/infrastructure/telemetry"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5/middleware"
)

// Header is the header the ID travels in (chi's spelling).
var Header = middleware.RequestIDHeader

// Echo writes the request ID into the response headers. Mount it after middleware.RequestID.
func Echo(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if id := middleware.GetReqID(r.Context()); id != "" {
			w.Header().Set(Header, id)
		}
		next.ServeHTTP(w, r)
	})
}

// Transport adds the request ID found in each outgoing request's context.
type Transport struct {
	Base http.RoundTripper
}

// RoundTrip implements http.RoundTripper.
func (t *Transport) RoundTrip(req *http.Request) (*http.Response, error) {
	base := t.Base
	if base == nil {
		base = http.DefaultTransport
	}
	if id := middleware.GetReqID(req.Context()); id != "" && req.Header.Get(Header) == "" {
		req = req.Clone(req.Context())
		req.Header.Set(Header, id)
	}
	return base.RoundTrip(req)
}

// NewClient returns an http.Client with the given timeout that forwards request IDs and
// trace context (traceparent).
func NewClient(timeout time.Duration) *http.Client {
	return &http.Client{Timeout: timeout, Transport: &Transport{Base: telemetry.Transport(nil)}}
}
