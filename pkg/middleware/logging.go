package middleware

import (
	"context"
	"log/slog"
	"net/http"
)

// type for context keys
type loggerKeyType struct{}

var LoggerKey = loggerKeyType{}

// RequestLogger creates a middleware that logs requests and injects the logger.
func RequestLogger(log *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

			// child logger with request details
			reqLog := log.With(
				slog.String("method", r.Method),
				slog.String("path", r.URL.Path),
				slog.String("remote_addr", r.RemoteAddr),
			)

			// inject this new logger into the context
			ctx := context.WithValue(r.Context(), LoggerKey, reqLog)

			// log the incoming request
			reqLog.Info("request started")

			// call the next handler with the NEW context
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
