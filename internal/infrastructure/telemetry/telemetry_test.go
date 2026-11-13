package telemetry

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

func TestSetupIsANoOpWithoutAnEndpoint(t *testing.T) {
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "")
	t.Setenv("OTEL_EXPORTER_OTLP_TRACES_ENDPOINT", "")

	shutdown, err := Setup(context.Background(), "svc")

	require.NoError(t, err)
	assert.False(t, Enabled())
	assert.NoError(t, shutdown(context.Background()))
}

func TestSetupExportsWhenAnEndpointIsSet(t *testing.T) {
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "http://127.0.0.1:1")

	shutdown, err := Setup(context.Background(), "svc")

	require.NoError(t, err)
	assert.True(t, Enabled())
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // nothing to flush; don't wait on the unreachable endpoint
	_ = shutdown(ctx)
}

// withRecorder installs an in-memory tracer provider for the test.
func withRecorder(t *testing.T) *tracetest.SpanRecorder {
	t.Helper()
	rec := tracetest.NewSpanRecorder()
	prev := otel.GetTracerProvider()
	otel.SetTracerProvider(sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(rec)))
	t.Cleanup(func() { otel.SetTracerProvider(prev) })
	_, _ = Setup(context.Background(), "svc") // propagator only (no endpoint in tests)
	return rec
}

func TestRequestSpansAreNamedAfterTheRoute(t *testing.T) {
	rec := withRecorder(t)
	r := chi.NewRouter()
	r.Use(RouteName)
	r.Get("/pois/{id}", func(http.ResponseWriter, *http.Request) {})

	Handler(r, "svc").ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/pois/123", nil))

	spans := rec.Ended()
	require.Len(t, spans, 1)
	assert.Equal(t, "GET /pois/{id}", spans[0].Name())
}

func TestOutboundCallsCarryTheTrace(t *testing.T) {
	rec := withRecorder(t)
	var traceparent string
	downstream := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		traceparent = r.Header.Get("traceparent")
	}))
	defer downstream.Close()

	client := &http.Client{Transport: Transport(nil)}
	handler := Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		req, _ := http.NewRequestWithContext(r.Context(), http.MethodGet, downstream.URL, nil)
		resp, err := client.Do(req)
		if err == nil {
			resp.Body.Close()
		}
	}), "svc")
	handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))

	require.NotEmpty(t, traceparent, "the downstream service can continue the trace")
	spans := rec.Ended()
	require.Len(t, spans, 2, "server span + client span")
	assert.Equal(t, spans[0].SpanContext().TraceID(), spans[1].SpanContext().TraceID())
}

func serviceNameOf(t *testing.T, serviceName string) string {
	t.Helper()
	res, err := newResource(serviceName)
	require.NoError(t, err)
	v, _ := res.Set().Value("service.name")
	return v.AsString()
}

func TestResourceIsNamedAfterTheService(t *testing.T) {
	t.Setenv("OTEL_SERVICE_NAME", "")
	assert.Equal(t, "poi-api", serviceNameOf(t, "poi-api"), "not unknown_service:<binary>")

	t.Setenv("OTEL_SERVICE_NAME", "poi-api-canary")
	assert.Equal(t, "poi-api-canary", serviceNameOf(t, "poi-api"))
}
