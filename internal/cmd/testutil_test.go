package cmd

import (
	"bytes"
	"io"
	"os"
	"testing"
)

// setupTestEnvironment sets up the test environment with required environment variables.
// It returns a cleanup function that should be called with defer to restore the environment.
//
// Example usage:
//
//	func TestSomething(t *testing.T) {
//	    cleanup := setupTestEnvironment(t)
//	    defer cleanup()
//
//	    // Your test code here
//	}
//
//nolint:unused // test helper for future tests
func setupTestEnvironment(t *testing.T) func() {
	t.Helper()

	// Store original environment values for restoration
	origAccount := os.Getenv("FASTMAIL_ACCOUNT")
	origToken := os.Getenv("FASTMAIL_TOKEN")

	// Set test environment variables
	os.Setenv("FASTMAIL_ACCOUNT", "test@example.com")
	os.Setenv("FASTMAIL_TOKEN", "test-token-12345")

	// Return cleanup function
	return func() {
		if origAccount != "" {
			os.Setenv("FASTMAIL_ACCOUNT", origAccount)
		} else {
			os.Unsetenv("FASTMAIL_ACCOUNT")
		}

		if origToken != "" {
			os.Setenv("FASTMAIL_TOKEN", origToken)
		} else {
			os.Unsetenv("FASTMAIL_TOKEN")
		}
	}
}

// setupMinimalTestEnvironment sets up only the FASTMAIL_ACCOUNT environment variable.
// This is useful for tests that don't require authentication.
//
//nolint:unused // test helper for future tests
func setupMinimalTestEnvironment(t *testing.T) func() {
	t.Helper()

	// Store original environment value for restoration
	origAccount := os.Getenv("FASTMAIL_ACCOUNT")

	// Set test environment variable
	os.Setenv("FASTMAIL_ACCOUNT", "test@example.com")

	// Return cleanup function
	return func() {
		if origAccount != "" {
			os.Setenv("FASTMAIL_ACCOUNT", origAccount)
		} else {
			os.Unsetenv("FASTMAIL_ACCOUNT")
		}
	}
}

// captureStdout captures stdout output for assertions in tests.
//
//nolint:unused // shared test helper
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()

	stdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	os.Stdout = w

	fn()

	_ = w.Close()
	os.Stdout = stdout

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	_ = r.Close()

	return buf.String()
}

// captureStderr captures stderr output for assertions in tests.
//
//nolint:unused // shared test helper
func captureStderr(t *testing.T, fn func()) string {
	t.Helper()

	stderr := os.Stderr
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	os.Stderr = w

	fn()

	_ = w.Close()
	os.Stderr = stderr

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	_ = r.Close()

	return buf.String()
}

// newTestApp returns a minimal App for command unit tests.
func newTestApp() *App {
	return &App{Flags: &rootFlags{}}
}
