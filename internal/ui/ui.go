// Package ui provides terminal UI utilities with color support.
// It handles color output with automatic detection, respects NO_COLOR,
// and provides Success, Error, Warning, and Info message helpers.
package ui

import (
	"context"
	"fmt"
	"os"

	"github.com/muesli/termenv"
)

type UI struct {
	out   *termenv.Output
	color bool
}

type contextKey struct{}

// New creates a new UI with the specified color mode.
// colorMode can be "never", "always", or "auto".
// The NO_COLOR environment variable overrides color=true.
func New(colorMode string) *UI {
	out := termenv.NewOutput(os.Stderr)
	var color bool

	switch colorMode {
	case "never":
		color = false
	case "always":
		color = true
	default: // auto
		color = out.ColorProfile() != termenv.Ascii
	}

	if os.Getenv("NO_COLOR") != "" {
		color = false
	}

	return &UI{out: out, color: color}
}

// Success prints a success message in green to stderr.
func (u *UI) Success(msg string) {
	if u.color {
		fmt.Fprintln(os.Stderr, u.out.String(msg).Foreground(u.out.Color("2")))
	} else {
		fmt.Fprintln(os.Stderr, msg)
	}
}

// Error prints an error message in red to stderr.
func (u *UI) Error(msg string) {
	if u.color {
		fmt.Fprintln(os.Stderr, u.out.String(msg).Foreground(u.out.Color("1")))
	} else {
		fmt.Fprintln(os.Stderr, msg)
	}
}

// Warning prints a warning message in yellow to stderr.
func (u *UI) Warning(msg string) {
	if u.color {
		fmt.Fprintln(os.Stderr, u.out.String(msg).Foreground(u.out.Color("3")))
	} else {
		fmt.Fprintln(os.Stderr, msg)
	}
}

// Info prints an informational message to stderr.
func (u *UI) Info(msg string) {
	fmt.Fprintln(os.Stderr, msg)
}

// WithUI stores the UI in the context.
func WithUI(ctx context.Context, u *UI) context.Context {
	return context.WithValue(ctx, contextKey{}, u)
}

// FromContext retrieves the UI from the context.
// If no UI is found in the context, returns New("auto").
func FromContext(ctx context.Context) *UI {
	if u, ok := ctx.Value(contextKey{}).(*UI); ok {
		return u
	}
	return New("auto")
}
