package cmd

import "strings"

// sanitizeTab replaces tab characters with spaces for clean tabwriter output
func sanitizeTab(s string) string {
	return strings.ReplaceAll(s, "\t", " ")
}
