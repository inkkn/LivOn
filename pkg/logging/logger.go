package logging

import (
	"log/slog"
	"os"
	"strings"
)

func NewLogger(service string) *slog.Logger {
	level := slog.LevelInfo

	if env := strings.ToLower(os.Getenv("LOG_LEVEL")); env != "" {
		switch env {
		case "debug":
			level = slog.LevelDebug
		case "warn":
			level = slog.LevelWarn
		case "error":
			level = slog.LevelError
		}
	}

	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level:     level,
		AddSource: true, // critical for incident debugging
	})

	logger := slog.New(handler).With(
		slog.String("service", service),
		slog.Int("pid", os.Getpid()),
	)

	slog.SetDefault(logger)
	return logger
}
