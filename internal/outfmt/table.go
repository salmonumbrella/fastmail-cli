package outfmt

import (
	"os"
	"strings"
	"text/tabwriter"
)

// NewTabWriter returns a tabwriter configured for stdout.
func NewTabWriter() *tabwriter.Writer {
	return tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
}

// SanitizeTab replaces tab characters with spaces for clean tabwriter output.
func SanitizeTab(s string) string {
	return strings.ReplaceAll(s, "\t", " ")
}
