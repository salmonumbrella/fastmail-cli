package cmd

import "github.com/salmonumbrella/fastmail-cli/internal/outfmt"

func printCancelled() {
	outfmt.Errorf("Cancelled")
}
