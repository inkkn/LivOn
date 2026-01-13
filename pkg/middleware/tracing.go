package middleware

import (
	"bufio"
	"errors"
	"net"
	"net/http"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
	"go.opentelemetry.io/otel/trace"
)

type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	h, ok := rw.ResponseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, errors.New("hijacker not supported")
	}
	return h.Hijack()
}

func TracerMiddleware(app string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// extract existing context from headers (Propagation)
			ctx := otel.GetTextMapPropagator().Extract(r.Context(), propagation.HeaderCarrier(r.Header))
			// Span
			tracer := otel.Tracer(app)
			ctx, span := tracer.Start(ctx,
				r.Method+" "+r.URL.Path,
				trace.WithAttributes(
					semconv.ServiceName(app),
					semconv.HTTPMethodKey.String(r.Method),
					semconv.HTTPTargetKey.String(r.URL.Path),
				),
				trace.WithSpanKind(trace.SpanKindServer),
			)
			defer span.End()
			wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}
			next.ServeHTTP(wrapped, r.WithContext(ctx))
			span.SetAttributes(semconv.HTTPStatusCodeKey.Int(wrapped.statusCode))
			if wrapped.statusCode >= 400 {
				span.SetStatus(1, "request failed")
			}
		})
	}
}
