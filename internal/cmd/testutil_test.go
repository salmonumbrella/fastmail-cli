package cmd

import (
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
