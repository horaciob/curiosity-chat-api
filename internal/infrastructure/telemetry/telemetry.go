// Package telemetry sets up OpenTelemetry tracing.
//
// Tracing is on only when an OTLP endpoint is configured (OTEL_EXPORTER_OTLP_ENDPOINT or
// OTEL_EXPORTER_OTLP_TRACES_ENDPOINT, e.g. http://jaeger:4318); otherwise spans are never
// recorded or exported. W3C trace context is propagated either way, so a trace started by
// a caller that does trace stays connected across services that do too.
//
// The same file lives in each Go service; keep the copies identical.
package telemetry

import (
	"context"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

// Enabled reports whether an OTLP endpoint is configured.
func Enabled() bool {
	return os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT") != "" || os.Getenv("OTEL_EXPORTER_OTLP_TRACES_ENDPOINT") != ""
}

// Setup installs the global propagator and, when Enabled, a tracer provider exporting
// over OTLP/HTTP. The returned shutdown flushes pending spans; call it on exit.
// OTEL_SERVICE_NAME, when set, wins over serviceName.
func Setup(ctx context.Context, serviceName string) (shutdown func(context.Context) error, err error) {
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))
	if !Enabled() {
		return func(context.Context) error { return nil }, nil
	}
	exp, err := otlptracehttp.New(ctx)
	if err != nil {
		return nil, err
	}
	res, err := newResource(serviceName)
	if err != nil {
		return nil, err
	}
	tp := sdktrace.NewTracerProvider(sdktrace.WithBatcher(exp), sdktrace.WithResource(res))
	otel.SetTracerProvider(tp)
	return tp.Shutdown, nil
}

// newResource names the service. In resource.Merge the second argument wins, and the
// default resource always carries service.name ("unknown_service:<binary>"), so ours goes
// second — with OTEL_SERVICE_NAME still taking precedence when set.
func newResource(serviceName string) (*resource.Resource, error) {
	if env := os.Getenv("OTEL_SERVICE_NAME"); env != "" {
		serviceName = env
	}
	return resource.Merge(resource.Default(),
		resource.NewSchemaless(attribute.String("service.name", serviceName)))
}

// Handler wraps the whole router: one server span per request, continuing the caller's
// trace when it sent a traceparent header.
func Handler(next http.Handler, serviceName string) http.Handler {
	return otelhttp.NewHandler(next, serviceName)
}

// RouteName is chi middleware that names the request span after the matched route
// pattern (GET /api/v1/pois/{id}) rather than the raw path, which would create one span
// name per resource ID. Mount it with r.Use.
func RouteName(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(w, r)
		if rctx := chi.RouteContext(r.Context()); rctx != nil {
			if pattern := rctx.RoutePattern(); pattern != "" {
				span := trace.SpanFromContext(r.Context())
				span.SetName(r.Method + " " + pattern)
				span.SetAttributes(attribute.String("http.route", pattern))
			}
		}
	})
}

// Transport makes outbound calls child spans and sends traceparent downstream.
func Transport(base http.RoundTripper) http.RoundTripper {
	if base == nil {
		base = http.DefaultTransport
	}
	return otelhttp.NewTransport(base)
}
