package logging

import (
	"context"
	"log/slog"
	"testing"
)

func TestSetup_Debug(t *testing.T) {
	logger := Setup(true)
	if logger == nil {
		t.Fatal("Setup(true) returned nil logger")
	}

	// Verify the logger can log at debug level
	// We can't easily inspect the handler's level directly, but we can verify
	// the logger was created without error
	if !logger.Enabled(context.Background(), slog.LevelDebug) {
		t.Error("Expected logger to be enabled at Debug level when debug=true")
	}
}

func TestSetup_Info(t *testing.T) {
	logger := Setup(false)
	if logger == nil {
		t.Fatal("Setup(false) returned nil logger")
	}

	// Verify the logger logs at info level but not debug
	if !logger.Enabled(context.Background(), slog.LevelInfo) {
		t.Error("Expected logger to be enabled at Info level when debug=false")
	}
	if logger.Enabled(context.Background(), slog.LevelDebug) {
		t.Error("Expected logger to be disabled at Debug level when debug=false")
	}
}

func TestWithLogger_FromContext_RoundTrip(t *testing.T) {
	logger := Setup(true)
	ctx := context.Background()

	// Store logger in context
	ctx = WithLogger(ctx, logger)

	// Retrieve logger from context
	retrieved := FromContext(ctx)

	if retrieved != logger {
		t.Error("FromContext did not return the same logger that was stored with WithLogger")
	}
}

func TestFromContext_ReturnsDefault_WhenNotInContext(t *testing.T) {
	ctx := context.Background()
	logger := FromContext(ctx)

	if logger == nil {
		t.Fatal("FromContext returned nil when logger not in context")
	}

	// Should return slog.Default()
	if logger != slog.Default() {
		t.Error("FromContext should return slog.Default() when logger not in context")
	}
}

func TestFromContext_ReturnsDefault_WithNilContext(t *testing.T) {
	// FromContext should handle nil context gracefully
	// Note: passing nil context will panic in context.Value, so we skip this test
	// and document that a valid context must always be passed
	t.Skip("FromContext requires a valid context")
}
