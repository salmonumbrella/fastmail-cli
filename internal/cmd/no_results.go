package cmd

import "github.com/salmonumbrella/fastmail-cli/internal/outfmt"

func printNoResults(format string, args ...any) {
	outfmt.Errorf(format, args...)
}
