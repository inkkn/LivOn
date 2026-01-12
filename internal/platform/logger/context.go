package logger

import (
	"context"
	"livon/pkg/middleware"
	"log/slog"
)

func FromContext(ctx context.Context) *slog.Logger {
	if l, ok := ctx.Value(middleware.LoggerKey).(*slog.Logger); ok && l != nil {
		return l
	}
	return slog.Default()
}
