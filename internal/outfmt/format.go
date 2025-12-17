package outfmt

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
)

type Mode int

const (
	Text Mode = iota
	JSON
)

// WriteJSON writes v as indented JSON to w.
func WriteJSON(w io.Writer, v any) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

// PrintJSON prints v as JSON to stdout.
func PrintJSON(v any) error {
	return WriteJSON(os.Stdout, v)
}

// Errorf prints to stderr.
func Errorf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
}
