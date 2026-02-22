package logging

import (
	"context"
	"log/slog"
	"os"
)

type contextKey struct{}

// Setup creates a new slog.Logger with the appropriate log level.
// If debug is true, the logger will log at Debug level.
// If debug is false, the logger will log at Info level.
// The logger writes to stderr using a text handler.
func Setup(debug bool) *slog.Logger {
	level := slog.LevelInfo
	if debug {
		level = slog.LevelDebug
	}

	handler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: level,
	})

	return slog.New(handler)
}

// WithLogger stores the logger in the context.
func WithLogger(ctx context.Context, logger *slog.Logger) context.Context {
	return context.WithValue(ctx, contextKey{}, logger)
}

// FromContext retrieves the logger from the context.
// If no logger is found in the context, returns slog.Default().
func FromContext(ctx context.Context) *slog.Logger {
	if logger, ok := ctx.Value(contextKey{}).(*slog.Logger); ok {
		return logger
	}
	return slog.Default()
}
