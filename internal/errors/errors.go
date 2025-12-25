// Package errors provides contextual error handling with user-facing suggestions.
package errors

import "fmt"

// Common suggestion constants for user-facing error messages
const (
	SuggestionReauth     = "Run 'fastmail auth' to re-authenticate"
	SuggestionCheckEmail = "Verify your email address is correct"
	SuggestionCheckNet   = "Check your network connection and try again"
)

// ContextError wraps an error with additional context and optional user-facing suggestion.
type ContextError struct {
	Context    string // Contextual information (e.g., "while fetching emails")
	Err        error  // The underlying error
	Suggestion string // Optional user-facing suggestion
}

// Error implements the error interface.
// Returns "context: error" format, or just the error message if no context.
func (e *ContextError) Error() string {
	if e.Err == nil {
		return ""
	}
	if e.Context == "" {
		return e.Err.Error()
	}
	return fmt.Sprintf("%s: %s", e.Context, e.Err.Error())
}

// Unwrap returns the underlying error for errors.Is and errors.As compatibility.
func (e *ContextError) Unwrap() error {
	return e.Err
}

// WithContext wraps an error with contextual information.
// Returns nil if the error is nil.
func WithContext(err error, context string) error {
	if err == nil {
		return nil
	}
	return &ContextError{
		Context: context,
		Err:     err,
	}
}

// WithSuggestion adds a user-facing suggestion to an error.
// Returns nil if the error is nil.
func WithSuggestion(err error, suggestion string) error {
	if err == nil {
		return nil
	}

	// If it's already a ContextError, add the suggestion to it
	if ce, ok := err.(*ContextError); ok {
		ce.Suggestion = suggestion
		return ce
	}

	// Otherwise, wrap it in a new ContextError with just the suggestion
	return &ContextError{
		Err:        err,
		Suggestion: suggestion,
	}
}

// ContainsSuggestion checks if an error has a user-facing suggestion.
// Returns false if the error is nil.
func ContainsSuggestion(err error) bool {
	if err == nil {
		return false
	}

	// Check direct ContextError
	if ce, ok := err.(*ContextError); ok {
		return ce.Suggestion != ""
	}

	return false
}

// GetSuggestion extracts the user-facing suggestion from an error.
// Returns an empty string if the error is nil or has no suggestion.
func GetSuggestion(err error) string {
	if err == nil {
		return ""
	}

	// Check direct ContextError
	if ce, ok := err.(*ContextError); ok {
		return ce.Suggestion
	}

	return ""
}
