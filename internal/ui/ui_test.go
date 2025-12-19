package ui

import (
	"context"
	"os"
	"testing"
)

func TestNew_ColorModeNever(t *testing.T) {
	u := New("never")
	if u == nil {
		t.Fatal("expected UI, got nil")
	}
	if u.color {
		t.Error("expected color=false for mode 'never', got true")
	}
}

func TestNew_ColorModeAlways(t *testing.T) {
	u := New("always")
	if u == nil {
		t.Fatal("expected UI, got nil")
	}
	if !u.color {
		t.Error("expected color=true for mode 'always', got false")
	}
}

func TestNew_ColorModeAuto(t *testing.T) {
	u := New("auto")
	if u == nil {
		t.Fatal("expected UI, got nil")
	}
	// Note: color value depends on terminal capabilities, so we just verify it's created
}

func TestNew_NOCOLOROverride(t *testing.T) {
	t.Helper()

	// Save original NO_COLOR value
	originalNoColor := os.Getenv("NO_COLOR")
	defer func() {
		if originalNoColor == "" {
			if err := os.Unsetenv("NO_COLOR"); err != nil {
				t.Fatalf("failed to unset NO_COLOR: %v", err)
			}
		} else {
			if err := os.Setenv("NO_COLOR", originalNoColor); err != nil {
				t.Fatalf("failed to restore NO_COLOR: %v", err)
			}
		}
	}()

	// Set NO_COLOR
	if err := os.Setenv("NO_COLOR", "1"); err != nil {
		t.Fatalf("failed to set NO_COLOR: %v", err)
	}

	u := New("always")
	if u == nil {
		t.Fatal("expected UI, got nil")
	}
	if u.color {
		t.Error("expected NO_COLOR to override color=always, but color=true")
	}
}

func TestWithUI_FromContext_RoundTrip(t *testing.T) {
	ctx := context.Background()
	u := New("never")

	// Store UI in context
	ctx = WithUI(ctx, u)

	// Retrieve UI from context
	retrieved := FromContext(ctx)
	if retrieved != u {
		t.Error("expected to retrieve the same UI instance from context")
	}
}

func TestFromContext_NotInContext(t *testing.T) {
	ctx := context.Background()

	// Retrieve from empty context
	u := FromContext(ctx)
	if u == nil {
		t.Fatal("expected default UI, got nil")
	}

	// Should return auto mode UI
	// We can't directly check the mode, but we can verify it's a valid UI
	if u.out == nil {
		t.Error("expected valid UI with output, got nil output")
	}
}

func TestUI_Methods(t *testing.T) {
	// Test that all methods can be called without panicking
	u := New("never")

	// These should not panic
	u.Success("success message")
	u.Error("error message")
	u.Warning("warning message")
	u.Info("info message")
}
