package outfmt

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/salmonumbrella/fastmail-cli/internal/filter"
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

// WriteJSONFiltered writes v as indented JSON to w, applying a JQ filter expression.
// If query is empty, behaves like WriteJSON.
func WriteJSONFiltered(w io.Writer, v any, query string) error {
	if query == "" {
		return WriteJSON(w, v)
	}

	// gojq expects JSON-compatible types (maps, slices, primitives), not Go structs.
	// Marshal to JSON and unmarshal back to get JSON-compatible types.
	jsonBytes, err := json.Marshal(v)
	if err != nil {
		return fmt.Errorf("failed to marshal data for filtering: %w", err)
	}

	var jsonData any
	if err = json.Unmarshal(jsonBytes, &jsonData); err != nil {
		return fmt.Errorf("failed to unmarshal data for filtering: %w", err)
	}

	result, err := filter.Apply(jsonData, query)
	if err != nil {
		return err
	}

	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(result)
}

// PrintJSONFiltered prints v as JSON to stdout, applying a JQ filter expression.
// If query is empty, behaves like PrintJSON.
func PrintJSONFiltered(v any, query string) error {
	return WriteJSONFiltered(os.Stdout, v, query)
}

// Errorf prints to stderr.
func Errorf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
}
