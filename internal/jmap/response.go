package jmap

import (
	"encoding/json"
	"fmt"
)

// decodeMethodResponse decodes a JMAP method response into the provided type.
func decodeMethodResponse[T any](resp *Response, index int) (T, error) {
	var zero T
	if resp == nil || len(resp.MethodResponses) <= index {
		return zero, fmt.Errorf("empty response from server")
	}

	methodName, ok := resp.MethodResponses[index][0].(string)
	if !ok {
		return zero, fmt.Errorf("invalid response format")
	}
	if methodName == "error" {
		return zero, parseJMAPError(resp.MethodResponses[index][1])
	}

	resultJSON, err := json.Marshal(resp.MethodResponses[index][1])
	if err != nil {
		return zero, fmt.Errorf("failed to marshal response: %w", err)
	}

	var result T
	if err := json.Unmarshal(resultJSON, &result); err != nil {
		return zero, fmt.Errorf("failed to parse response: %w", err)
	}

	return result, nil
}

func parseJMAPError(payload any) error {
	if payload == nil {
		return fmt.Errorf("API error: empty response")
	}
	if m, ok := payload.(map[string]any); ok {
		errType, _ := m["type"].(string)
		description, _ := m["description"].(string)
		return fmt.Errorf("API error: %w", &JMAPError{Type: errType, Description: description})
	}
	return fmt.Errorf("API error: %v", payload)
}
