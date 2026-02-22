// Package filter provides JQ-compatible filtering for JSON output.
package filter

import (
	"encoding/json"
	"fmt"

	"github.com/itchyny/gojq"
)

// Apply applies a JQ filter expression to the input data.
func Apply(data any, expression string) (any, error) {
	if expression == "" {
		return data, nil
	}

	query, err := gojq.Parse(expression)
	if err != nil {
		return nil, fmt.Errorf("invalid filter expression: %w", err)
	}

	iter := query.Run(data)

	var results []any
	for {
		v, ok := iter.Next()
		if !ok {
			break
		}
		if err, ok := v.(error); ok {
			return nil, fmt.Errorf("filter error: %w", err)
		}
		results = append(results, v)
	}

	// Return single result unwrapped, multiple as array
	if len(results) == 1 {
		return results[0], nil
	}
	return results, nil
}

// ApplyToJSON applies filter to JSON bytes and returns filtered JSON bytes.
func ApplyToJSON(jsonData []byte, expression string) ([]byte, error) {
	if expression == "" {
		return jsonData, nil
	}

	var data any
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("invalid JSON: %w", err)
	}

	result, err := Apply(data, expression)
	if err != nil {
		return nil, err
	}

	return json.MarshalIndent(result, "", "  ")
}
