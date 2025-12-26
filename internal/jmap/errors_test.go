package jmap

import (
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"
)

// TestValidationError_Error tests the Error() method
func TestValidationError_Error(t *testing.T) {
	tests := []struct {
		name     string
		err      *ValidationError
		expected string
	}{
		{
			name:     "with field",
			err:      &ValidationError{Field: "email", Message: "invalid format"},
			expected: "email: invalid format",
		},
		{
			name:     "without field",
			err:      &ValidationError{Message: "validation failed"},
			expected: "validation failed",
		},
		{
			name:     "empty field",
			err:      &ValidationError{Field: "", Message: "general error"},
			expected: "general error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.err.Error()
			if got != tt.expected {
				t.Errorf("ValidationError.Error() = %q, want %q", got, tt.expected)
			}
		})
	}
}

// TestRateLimitError_Error tests the Error() method
func TestRateLimitError_Error(t *testing.T) {
	tests := []struct {
		name     string
		err      *RateLimitError
		contains string
	}{
		{
			name:     "with seconds",
			err:      &RateLimitError{RetryAfter: 30 * time.Second},
			contains: "30s",
		},
		{
			name:     "with minutes",
			err:      &RateLimitError{RetryAfter: 5 * time.Minute},
			contains: "5m",
		},
		{
			name:     "zero duration",
			err:      &RateLimitError{RetryAfter: 0},
			contains: "rate limited",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.err.Error()
			if !strings.Contains(got, tt.contains) {
				t.Errorf("RateLimitError.Error() = %q, want to contain %q", got, tt.contains)
			}
			if !strings.Contains(got, "rate limited") {
				t.Errorf("RateLimitError.Error() = %q, want to contain 'rate limited'", got)
			}
		})
	}
}

// TestCircuitBreakerError_Error tests the Error() method
func TestCircuitBreakerError_Error(t *testing.T) {
	err := &CircuitBreakerError{}
	expected := "circuit breaker open: service temporarily unavailable"
	got := err.Error()
	if got != expected {
		t.Errorf("CircuitBreakerError.Error() = %q, want %q", got, expected)
	}
}

// TestAuthError_Error tests the Error() method
func TestAuthError_Error(t *testing.T) {
	tests := []struct {
		name     string
		err      *AuthError
		expected string
	}{
		{
			name:     "invalid token",
			err:      &AuthError{Message: "invalid token"},
			expected: "authentication error: invalid token",
		},
		{
			name:     "expired credentials",
			err:      &AuthError{Message: "credentials expired"},
			expected: "authentication error: credentials expired",
		},
		{
			name:     "empty message",
			err:      &AuthError{Message: ""},
			expected: "authentication error: ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.err.Error()
			if got != tt.expected {
				t.Errorf("AuthError.Error() = %q, want %q", got, tt.expected)
			}
		})
	}
}

// TestIsValidationError tests the helper function
func TestIsValidationError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "direct ValidationError",
			err:      &ValidationError{Field: "email", Message: "invalid"},
			expected: true,
		},
		{
			name:     "wrapped ValidationError",
			err:      fmt.Errorf("failed to validate: %w", &ValidationError{Field: "name", Message: "required"}),
			expected: true,
		},
		{
			name:     "other error type",
			err:      &AuthError{Message: "unauthorized"},
			expected: false,
		},
		{
			name:     "sentinel error",
			err:      ErrNoAccounts,
			expected: false,
		},
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
		{
			name:     "generic error",
			err:      errors.New("generic error"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsValidationError(tt.err)
			if got != tt.expected {
				t.Errorf("IsValidationError() = %v, want %v", got, tt.expected)
			}
		})
	}
}

// TestIsRateLimitError tests the helper function
func TestIsRateLimitError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "direct RateLimitError",
			err:      &RateLimitError{RetryAfter: 30 * time.Second},
			expected: true,
		},
		{
			name:     "wrapped RateLimitError",
			err:      fmt.Errorf("request failed: %w", &RateLimitError{RetryAfter: 60 * time.Second}),
			expected: true,
		},
		{
			name:     "other error type",
			err:      &ValidationError{Message: "invalid"},
			expected: false,
		},
		{
			name:     "sentinel error",
			err:      ErrEmailNotFound,
			expected: false,
		},
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
		{
			name:     "generic error",
			err:      errors.New("generic error"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsRateLimitError(tt.err)
			if got != tt.expected {
				t.Errorf("IsRateLimitError() = %v, want %v", got, tt.expected)
			}
		})
	}
}

// TestIsCircuitBreakerError tests the helper function
func TestIsCircuitBreakerError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "direct CircuitBreakerError",
			err:      &CircuitBreakerError{},
			expected: true,
		},
		{
			name:     "wrapped CircuitBreakerError",
			err:      fmt.Errorf("circuit broken: %w", &CircuitBreakerError{}),
			expected: true,
		},
		{
			name:     "other error type",
			err:      &AuthError{Message: "unauthorized"},
			expected: false,
		},
		{
			name:     "sentinel error",
			err:      ErrMailboxNotFound,
			expected: false,
		},
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
		{
			name:     "generic error",
			err:      errors.New("generic error"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsCircuitBreakerError(tt.err)
			if got != tt.expected {
				t.Errorf("IsCircuitBreakerError() = %v, want %v", got, tt.expected)
			}
		})
	}
}

// TestIsAuthError tests the helper function
func TestIsAuthError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "direct AuthError",
			err:      &AuthError{Message: "invalid token"},
			expected: true,
		},
		{
			name:     "wrapped AuthError",
			err:      fmt.Errorf("authentication failed: %w", &AuthError{Message: "expired"}),
			expected: true,
		},
		{
			name:     "other error type",
			err:      &ValidationError{Message: "invalid"},
			expected: false,
		},
		{
			name:     "sentinel error",
			err:      ErrInvalidFromAddress,
			expected: false,
		},
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
		{
			name:     "generic error",
			err:      errors.New("generic error"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsAuthError(tt.err)
			if got != tt.expected {
				t.Errorf("IsAuthError() = %v, want %v", got, tt.expected)
			}
		})
	}
}

// TestErrorsAs_Unwrapping tests that errors.As properly unwraps our error types
func TestErrorsAs_Unwrapping(t *testing.T) {
	t.Run("unwrap ValidationError", func(t *testing.T) {
		original := &ValidationError{Field: "email", Message: "invalid format"}
		wrapped := fmt.Errorf("outer: %w", original)

		var ve *ValidationError
		if !errors.As(wrapped, &ve) {
			t.Fatal("errors.As() failed to unwrap ValidationError")
		}
		if ve.Field != "email" || ve.Message != "invalid format" {
			t.Errorf("unwrapped ValidationError = {Field: %q, Message: %q}, want {Field: %q, Message: %q}",
				ve.Field, ve.Message, "email", "invalid format")
		}
	})

	t.Run("unwrap RateLimitError", func(t *testing.T) {
		original := &RateLimitError{RetryAfter: 45 * time.Second}
		wrapped := fmt.Errorf("outer: %w", original)

		var rle *RateLimitError
		if !errors.As(wrapped, &rle) {
			t.Fatal("errors.As() failed to unwrap RateLimitError")
		}
		if rle.RetryAfter != 45*time.Second {
			t.Errorf("unwrapped RateLimitError.RetryAfter = %v, want %v", rle.RetryAfter, 45*time.Second)
		}
	})

	t.Run("unwrap AuthError", func(t *testing.T) {
		original := &AuthError{Message: "token expired"}
		wrapped := fmt.Errorf("outer: %w", original)

		var ae *AuthError
		if !errors.As(wrapped, &ae) {
			t.Fatal("errors.As() failed to unwrap AuthError")
		}
		if ae.Message != "token expired" {
			t.Errorf("unwrapped AuthError.Message = %q, want %q", ae.Message, "token expired")
		}
	})
}

// TestWrappedErrors tests multiple levels of error wrapping
func TestWrappedErrors(t *testing.T) {
	t.Run("double wrapped ValidationError", func(t *testing.T) {
		original := &ValidationError{Field: "name", Message: "required"}
		wrapped1 := fmt.Errorf("layer1: %w", original)
		wrapped2 := fmt.Errorf("layer2: %w", wrapped1)

		if !IsValidationError(wrapped2) {
			t.Error("IsValidationError() = false for double wrapped error, want true")
		}

		var ve *ValidationError
		if !errors.As(wrapped2, &ve) {
			t.Fatal("errors.As() failed on double wrapped ValidationError")
		}
		if ve.Field != "name" {
			t.Errorf("got Field = %q, want %q", ve.Field, "name")
		}
	})

	t.Run("triple wrapped RateLimitError", func(t *testing.T) {
		original := &RateLimitError{RetryAfter: 120 * time.Second}
		wrapped1 := fmt.Errorf("layer1: %w", original)
		wrapped2 := fmt.Errorf("layer2: %w", wrapped1)
		wrapped3 := fmt.Errorf("layer3: %w", wrapped2)

		if !IsRateLimitError(wrapped3) {
			t.Error("IsRateLimitError() = false for triple wrapped error, want true")
		}
	})
}

// TestErrorTypes_NilError tests that helper functions handle nil errors correctly
func TestErrorTypes_NilError(t *testing.T) {
	// When passed a nil error interface (not a typed nil), these should return false
	var err error = nil

	if IsValidationError(err) {
		t.Error("IsValidationError(nil) = true, want false")
	}
	if IsRateLimitError(err) {
		t.Error("IsRateLimitError(nil) = true, want false")
	}
	if IsCircuitBreakerError(err) {
		t.Error("IsCircuitBreakerError(nil) = true, want false")
	}
	if IsAuthError(err) {
		t.Error("IsAuthError(nil) = true, want false")
	}
}

// TestErrorTypes_TypedNilPointer tests behavior with typed nil pointers
// Note: errors.As returns true for typed nil pointers, which is expected behavior
func TestErrorTypes_TypedNilPointer(t *testing.T) {
	// When passed a typed nil pointer, errors.As will match the type
	// This is standard Go behavior and is intentional
	var nilValidation error = (*ValidationError)(nil)
	var nilRateLimit error = (*RateLimitError)(nil)
	var nilCircuitBreaker error = (*CircuitBreakerError)(nil)
	var nilAuth error = (*AuthError)(nil)

	// These return true because the type matches, even though the value is nil
	// This is expected behavior for errors.As
	if !IsValidationError(nilValidation) {
		t.Error("IsValidationError(typed nil) = false, want true (expected errors.As behavior)")
	}
	if !IsRateLimitError(nilRateLimit) {
		t.Error("IsRateLimitError(typed nil) = false, want true (expected errors.As behavior)")
	}
	if !IsCircuitBreakerError(nilCircuitBreaker) {
		t.Error("IsCircuitBreakerError(typed nil) = false, want true (expected errors.As behavior)")
	}
	if !IsAuthError(nilAuth) {
		t.Error("IsAuthError(typed nil) = false, want true (expected errors.As behavior)")
	}
}

// TestJMAPError tests the Error() method for JMAPError
func TestJMAPError(t *testing.T) {
	tests := []struct {
		name     string
		err      *JMAPError
		expected string
	}{
		{
			name:     "with description",
			err:      &JMAPError{Type: "invalidArguments", Description: "unknown property: foo"},
			expected: "JMAP error (invalidArguments): unknown property: foo",
		},
		{
			name:     "without description",
			err:      &JMAPError{Type: "serverFail"},
			expected: "JMAP error: serverFail",
		},
		{
			name:     "empty description",
			err:      &JMAPError{Type: "notFound", Description: ""},
			expected: "JMAP error: notFound",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.err.Error(); got != tt.expected {
				t.Errorf("JMAPError.Error() = %q, want %q", got, tt.expected)
			}
		})
	}
}

// TestNotFoundError tests the Error() method for NotFoundError
func TestNotFoundError(t *testing.T) {
	tests := []struct {
		name     string
		err      *NotFoundError
		expected string
	}{
		{
			name:     "with ID",
			err:      &NotFoundError{Resource: "email", ID: "abc123"},
			expected: "email not found: abc123",
		},
		{
			name:     "without ID",
			err:      &NotFoundError{Resource: "mailbox"},
			expected: "mailbox not found",
		},
		{
			name:     "empty ID",
			err:      &NotFoundError{Resource: "contact", ID: ""},
			expected: "contact not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.err.Error(); got != tt.expected {
				t.Errorf("NotFoundError.Error() = %q, want %q", got, tt.expected)
			}
		})
	}
}

// TestIsNotFoundError tests the IsNotFoundError helper function
func TestIsNotFoundError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"NotFoundError type", &NotFoundError{Resource: "email", ID: "x"}, true},
		{"ErrEmailNotFound sentinel", ErrEmailNotFound, true},
		{"ErrContactNotFound sentinel", ErrContactNotFound, true},
		{"ErrThreadNotFound sentinel", ErrThreadNotFound, true},
		{"ErrMailboxNotFound sentinel", ErrMailboxNotFound, true},
		{"ErrEventNotFound sentinel", ErrEventNotFound, true},
		{"wrapped NotFoundError", fmt.Errorf("failed: %w", &NotFoundError{Resource: "thread", ID: "y"}), true},
		{"other error", fmt.Errorf("random error"), false},
		{"nil error", nil, false},
		{"ValidationError", &ValidationError{Message: "invalid"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsNotFoundError(tt.err); got != tt.want {
				t.Errorf("IsNotFoundError() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestRequestContext tests the RequestContext error wrapper
func TestRequestContext(t *testing.T) {
	inner := &JMAPError{Type: "serverFail", Description: "oops"}
	err := &RequestContext{
		Method: "Email/get",
		Err:    inner,
	}

	// Test Error() contains method name
	if !strings.Contains(err.Error(), "Email/get") {
		t.Errorf("error should contain method name, got: %s", err.Error())
	}

	// Test Unwrap returns inner error
	if err.Unwrap() != inner {
		t.Error("Unwrap() should return inner error")
	}

	// Test errors.As can unwrap to JMAPError
	var jmapErr *JMAPError
	if !errors.As(err, &jmapErr) {
		t.Error("should unwrap to JMAPError")
	}
	if jmapErr.Type != "serverFail" {
		t.Errorf("unwrapped JMAPError.Type = %q, want %q", jmapErr.Type, "serverFail")
	}
}

// TestIsJMAPError tests the IsJMAPError helper function
func TestIsJMAPError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "direct JMAPError",
			err:      &JMAPError{Type: "invalidArguments", Description: "bad input"},
			expected: true,
		},
		{
			name:     "wrapped JMAPError",
			err:      fmt.Errorf("request failed: %w", &JMAPError{Type: "serverFail"}),
			expected: true,
		},
		{
			name:     "JMAPError in RequestContext",
			err:      &RequestContext{Method: "Email/get", Err: &JMAPError{Type: "notFound"}},
			expected: true,
		},
		{
			name:     "other error type",
			err:      &ValidationError{Message: "invalid"},
			expected: false,
		},
		{
			name:     "sentinel error",
			err:      ErrEmailNotFound,
			expected: false,
		},
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsJMAPError(tt.err); got != tt.expected {
				t.Errorf("IsJMAPError() = %v, want %v", got, tt.expected)
			}
		})
	}
}

// TestRequestContext_NestedUnwrap tests nested error unwrapping through RequestContext
func TestRequestContext_NestedUnwrap(t *testing.T) {
	// Create a deeply nested error
	innermost := &NotFoundError{Resource: "email", ID: "abc123"}
	middle := fmt.Errorf("inner wrap: %w", innermost)
	outer := &RequestContext{Method: "Email/get", Err: middle}

	// Should be able to unwrap to NotFoundError
	var nfe *NotFoundError
	if !errors.As(outer, &nfe) {
		t.Error("should unwrap to NotFoundError through RequestContext")
	}
	if nfe.ID != "abc123" {
		t.Errorf("unwrapped NotFoundError.ID = %q, want %q", nfe.ID, "abc123")
	}

	// IsNotFoundError should work too
	if !IsNotFoundError(outer) {
		t.Error("IsNotFoundError should return true for wrapped NotFoundError in RequestContext")
	}
}

// TestSentinelErrors_Unchanged ensures existing sentinel errors still work
func TestSentinelErrors_Unchanged(t *testing.T) {
	sentinels := []struct {
		name string
		err  error
	}{
		{"ErrNoAccounts", ErrNoAccounts},
		{"ErrEmailNotFound", ErrEmailNotFound},
		{"ErrContactNotFound", ErrContactNotFound},
		{"ErrThreadNotFound", ErrThreadNotFound},
		{"ErrMailboxNotFound", ErrMailboxNotFound},
		{"ErrContactsNotEnabled", ErrContactsNotEnabled},
		{"ErrNoIdentities", ErrNoIdentities},
		{"ErrInvalidFromAddress", ErrInvalidFromAddress},
		{"ErrNoDraftsMailbox", ErrNoDraftsMailbox},
		{"ErrNoSentMailbox", ErrNoSentMailbox},
		{"ErrNoTrashMailbox", ErrNoTrashMailbox},
		{"ErrNoBody", ErrNoBody},
	}

	for _, tt := range sentinels {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err == nil {
				t.Errorf("%s is nil", tt.name)
			}
			if tt.err.Error() == "" {
				t.Errorf("%s.Error() is empty", tt.name)
			}
			// Ensure they work with errors.Is
			if !errors.Is(tt.err, tt.err) {
				t.Errorf("errors.Is(%s, %s) = false, want true", tt.name, tt.name)
			}
		})
	}
}
