package logger

import (
	"livon/internal/config"
	"log/slog"
	"os"
)

func NewLogger(cfg config.Config) *slog.Logger {
	var level slog.Level
	var handler slog.Handler
	Level := cfg.Logger.Level
	Format := cfg.Logger.Format
	switch Level {
	case "INFO":
		level = slog.LevelInfo
	case "debug":
		level = slog.LevelDebug
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	}
	switch Format {
	case "TEXT":
		handler = slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			Level:     level,
			AddSource: true, // critical for incident debugging
		})
	case "JSON":
		handler = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level:     level,
			AddSource: true, // critical for incident debugging
		})
	}
	logger := slog.New(handler).With(
		slog.String("service", cfg.Service.Name),
		slog.String("env", cfg.Service.Env),
		slog.String("address", cfg.Service.Add),
		slog.Int("pid", os.Getpid()),
	)
	slog.SetDefault(logger)
	return logger
}
