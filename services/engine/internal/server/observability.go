package server

import (
	"context"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"log/slog"
	"os"
)

func init() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
		if a.Key == slog.LevelKey {
			a.Key = "severity"
		}
		if a.Key == slog.TimeKey {
			a.Key = "timestamp"
		}
		return a
	}})))
}

// ConfigureTracing exports only when a collector endpoint is configured. Runtime
// logs and Cloud Run request metrics remain available without an OTLP collector.
func ConfigureTracing(ctx context.Context) (func(context.Context) error, error) {
	if os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT") == "" {
		return func(context.Context) error { return nil }, nil
	}
	exporter, e := otlptracehttp.New(ctx)
	if e != nil {
		return nil, e
	}
	p := sdktrace.NewTracerProvider(sdktrace.WithBatcher(exporter), sdktrace.WithResource(resource.NewWithAttributes("", attribute.String("service.name", "optitrace-api"))), sdktrace.WithSampler(sdktrace.ParentBased(sdktrace.TraceIDRatioBased(.1))))
	otel.SetTracerProvider(p)
	return p.Shutdown, nil
}
