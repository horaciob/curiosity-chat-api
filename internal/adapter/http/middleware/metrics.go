package middleware

import (
	"bufio"
	"errors"
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Metrics records request counts and latencies for every HTTP route.
//
// The services had no metrics of any kind: the only way to find out that an endpoint had
// become slow, or had started failing, was for someone to notice in the app. These three
// series answer the questions that actually get asked during an incident — what is failing,
// how often, and how slow is it.
var (
	requestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Requests handled, by route, method and status class.",
		},
		[]string{"route", "method", "status"},
	)

	requestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name: "http_request_duration_seconds",
			Help: "Request latency by route and method.",
			// Buckets span a fast local read to a slow AI call, which is the real range
			// this service covers.
			Buckets: []float64{0.005, 0.025, 0.1, 0.25, 0.5, 1, 2.5, 5, 10, 30},
		},
		[]string{"route", "method"},
	)

	requestsInFlight = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "http_requests_in_flight",
			Help: "Requests currently being served.",
		},
	)
)

// Metrics is the middleware. Mount it high in the stack so it sees every request, including
// the ones later middleware rejects.
func Metrics(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		requestsInFlight.Inc()
		defer requestsInFlight.Dec()

		rec := &metricsRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rec, r)

		// The chi route pattern, not the raw path: /pois/{id} rather than one time series
		// per POI, which would be an unbounded label and would take the metrics store down
		// long before it told anyone anything useful.
		route := chi.RouteContext(r.Context()).RoutePattern()
		if route == "" {
			route = "unmatched"
		}

		requestsTotal.WithLabelValues(route, r.Method, strconv.Itoa(rec.status)).Inc()
		requestDuration.WithLabelValues(route, r.Method).Observe(time.Since(start).Seconds())
	})
}

type metricsRecorder struct {
	http.ResponseWriter
	status  int
	written bool
}

func (m *metricsRecorder) WriteHeader(status int) {
	if !m.written {
		m.status = status
		m.written = true
	}
	m.ResponseWriter.WriteHeader(status)
}

func (m *metricsRecorder) Write(p []byte) (int, error) {
	m.written = true
	return m.ResponseWriter.Write(p)
}

// Flush keeps streaming responses working through the recorder.
func (m *metricsRecorder) Flush() {
	if f, ok := m.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

// Hijack lets the WebSocket upgrade take over the connection. Without it every
// /api/v1/ws request failed with "http.Hijacker is unavailable on the writer", so no
// client could open the realtime socket.
func (m *metricsRecorder) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	h, ok := m.ResponseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, errors.New("metrics: underlying ResponseWriter does not support hijacking")
	}
	m.written = true
	m.status = http.StatusSwitchingProtocols
	return h.Hijack()
}

// Unwrap exposes the original writer to http.ResponseController.
func (m *metricsRecorder) Unwrap() http.ResponseWriter { return m.ResponseWriter }
