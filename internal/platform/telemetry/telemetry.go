package telemetry

import (
	"context"
	"errors"
	"livon/internal/config"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
)

// ShutdownFunc is a helper to clean up all providers on app exit
type ShutdownFunc func(context.Context) error

func InitTelemetry(ctx context.Context, cfg config.Config) (ShutdownFunc, error) {
	res, _ := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceNameKey.String(cfg.Service.Name),
		),
	)

	// Tracing (Tempo)
	traceExporter, _ := otlptracegrpc.New(ctx, otlptracegrpc.WithEndpoint(cfg.Tracer.Address), otlptracegrpc.WithInsecure())
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(traceExporter),
		sdktrace.WithResource(res),
	)
	otel.SetTracerProvider(tp)

	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	// Return a combined shutdown function
	return func(shutdownCtx context.Context) error {
		var err error
		err = errors.Join(err, tp.Shutdown(shutdownCtx)) // Flush Traces
		return err
	}, nil
}
