package errors

import (
	"errors"
	"fmt"
	"testing"
)

func TestContextError_Message(t *testing.T) {
	tests := []struct {
		name     string
		context  string
		err      error
		expected string
	}{
		{
			name:     "with context",
			context:  "while fetching emails",
			err:      errors.New("connection refused"),
			expected: "while fetching emails: connection refused",
		},
		{
			name:     "without context",
			context:  "",
			err:      errors.New("connection refused"),
			expected: "connection refused",
		},
		{
			name:     "nil error",
			context:  "some context",
			err:      nil,
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var err error
			if tt.err != nil {
				err = WithContext(tt.err, tt.context)
			}

			var got string
			if err != nil {
				got = err.Error()
			}

			if got != tt.expected {
				t.Errorf("Error() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestContextError_Suggestion(t *testing.T) {
	tests := []struct {
		name               string
		err                error
		suggestion         string
		hasSuggestion      bool
		expectedError      string
		expectedSuggestion string
	}{
		{
			name:               "with suggestion",
			err:                errors.New("authentication failed"),
			suggestion:         SuggestionReauth,
			hasSuggestion:      true,
			expectedError:      "authentication failed",
			expectedSuggestion: SuggestionReauth,
		},
		{
			name:               "without suggestion",
			err:                errors.New("some error"),
			suggestion:         "",
			hasSuggestion:      false,
			expectedError:      "some error",
			expectedSuggestion: "",
		},
		{
			name:               "context and suggestion",
			err:                WithContext(errors.New("token expired"), "while listing calendars"),
			suggestion:         SuggestionReauth,
			hasSuggestion:      true,
			expectedError:      "while listing calendars: token expired",
			expectedSuggestion: SuggestionReauth,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var err error
			if tt.suggestion != "" {
				err = WithSuggestion(tt.err, tt.suggestion)
			} else {
				err = tt.err
			}

			if err.Error() != tt.expectedError {
				t.Errorf("Error() = %q, want %q", err.Error(), tt.expectedError)
			}

			if ContainsSuggestion(err) != tt.hasSuggestion {
				t.Errorf("ContainsSuggestion() = %v, want %v", ContainsSuggestion(err), tt.hasSuggestion)
			}

			got := GetSuggestion(err)
			if got != tt.expectedSuggestion {
				t.Errorf("GetSuggestion() = %q, want %q", got, tt.expectedSuggestion)
			}
		})
	}
}

func TestContextError_Unwrap(t *testing.T) {
	baseErr := errors.New("base error")
	wrappedErr := WithContext(baseErr, "while processing")

	// Test that errors.Is works through the wrapper
	if !errors.Is(wrappedErr, baseErr) {
		t.Error("errors.Is() should find the base error through wrapping")
	}

	// Test with suggestion
	suggestionErr := WithSuggestion(baseErr, SuggestionCheckEmail)
	if !errors.Is(suggestionErr, baseErr) {
		t.Error("errors.Is() should find the base error through suggestion wrapper")
	}

	// Test with both context and suggestion
	bothErr := WithSuggestion(WithContext(baseErr, "operation failed"), SuggestionCheckNet)
	if !errors.Is(bothErr, baseErr) {
		t.Error("errors.Is() should find the base error through multiple wrappers")
	}
}

func TestContextError_ChainedWrapping(t *testing.T) {
	baseErr := errors.New("network timeout")
	contextErr := WithContext(baseErr, "while fetching data")
	finalErr := WithSuggestion(contextErr, SuggestionCheckNet)

	expectedMsg := "while fetching data: network timeout"
	if finalErr.Error() != expectedMsg {
		t.Errorf("Error() = %q, want %q", finalErr.Error(), expectedMsg)
	}

	if !ContainsSuggestion(finalErr) {
		t.Error("ContainsSuggestion() should return true for chained error")
	}

	if GetSuggestion(finalErr) != SuggestionCheckNet {
		t.Errorf("GetSuggestion() = %q, want %q", GetSuggestion(finalErr), SuggestionCheckNet)
	}

	if !errors.Is(finalErr, baseErr) {
		t.Error("errors.Is() should find base error through chain")
	}
}

func TestSuggestionConstants(t *testing.T) {
	// Verify that suggestion constants are non-empty
	constants := []struct {
		name  string
		value string
	}{
		{"SuggestionReauth", SuggestionReauth},
		{"SuggestionCheckEmail", SuggestionCheckEmail},
		{"SuggestionCheckNet", SuggestionCheckNet},
	}

	for _, c := range constants {
		if c.value == "" {
			t.Errorf("%s should not be empty", c.name)
		}
	}
}

func TestContextError_WrappedSuggestion(t *testing.T) {
	// Create a ContextError with a suggestion
	baseErr := errors.New("auth failed")
	contextErr := WithSuggestion(baseErr, SuggestionReauth)

	// Wrap it with fmt.Errorf (simulates how errors get wrapped in practice)
	wrappedErr := fmt.Errorf("outer context: %w", contextErr)

	// ContainsSuggestion should find the suggestion through the wrapper
	if !ContainsSuggestion(wrappedErr) {
		t.Error("ContainsSuggestion() should return true for wrapped ContextError with suggestion")
	}

	// GetSuggestion should extract the suggestion through the wrapper
	got := GetSuggestion(wrappedErr)
	if got != SuggestionReauth {
		t.Errorf("GetSuggestion() = %q, want %q", got, SuggestionReauth)
	}

	// Double-wrapped should also work
	doubleWrapped := fmt.Errorf("even more outer: %w", wrappedErr)
	if !ContainsSuggestion(doubleWrapped) {
		t.Error("ContainsSuggestion() should return true for double-wrapped ContextError")
	}
	if GetSuggestion(doubleWrapped) != SuggestionReauth {
		t.Errorf("GetSuggestion() on double-wrapped = %q, want %q", GetSuggestion(doubleWrapped), SuggestionReauth)
	}
}
