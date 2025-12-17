package jmap

import "testing"

func TestGetString(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]any
		key      string
		expected string
	}{
		{
			name:     "valid string value",
			input:    map[string]any{"foo": "bar"},
			key:      "foo",
			expected: "bar",
		},
		{
			name:     "missing key",
			input:    map[string]any{"foo": "bar"},
			key:      "baz",
			expected: "",
		},
		{
			name:     "wrong type - int",
			input:    map[string]any{"foo": 123},
			key:      "foo",
			expected: "",
		},
		{
			name:     "wrong type - bool",
			input:    map[string]any{"foo": true},
			key:      "foo",
			expected: "",
		},
		{
			name:     "wrong type - nil",
			input:    map[string]any{"foo": nil},
			key:      "foo",
			expected: "",
		},
		{
			name:     "empty string",
			input:    map[string]any{"foo": ""},
			key:      "foo",
			expected: "",
		},
		{
			name:     "empty map",
			input:    map[string]any{},
			key:      "foo",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getString(tt.input, tt.key)
			if got != tt.expected {
				t.Errorf("getString() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestGetInt(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]any
		key      string
		expected int
	}{
		{
			name:     "valid float64 value",
			input:    map[string]any{"count": float64(42)},
			key:      "count",
			expected: 42,
		},
		{
			name:     "zero value",
			input:    map[string]any{"count": float64(0)},
			key:      "count",
			expected: 0,
		},
		{
			name:     "negative value",
			input:    map[string]any{"count": float64(-10)},
			key:      "count",
			expected: -10,
		},
		{
			name:     "missing key",
			input:    map[string]any{"count": float64(42)},
			key:      "missing",
			expected: 0,
		},
		{
			name:     "wrong type - string",
			input:    map[string]any{"count": "42"},
			key:      "count",
			expected: 0,
		},
		{
			name:     "wrong type - int",
			input:    map[string]any{"count": 42},
			key:      "count",
			expected: 0,
		},
		{
			name:     "wrong type - bool",
			input:    map[string]any{"count": true},
			key:      "count",
			expected: 0,
		},
		{
			name:     "wrong type - nil",
			input:    map[string]any{"count": nil},
			key:      "count",
			expected: 0,
		},
		{
			name:     "large value",
			input:    map[string]any{"count": float64(999999)},
			key:      "count",
			expected: 999999,
		},
		{
			name:     "empty map",
			input:    map[string]any{},
			key:      "count",
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getInt(tt.input, tt.key)
			if got != tt.expected {
				t.Errorf("getInt() = %d, want %d", got, tt.expected)
			}
		})
	}
}

func TestGetInt64(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]any
		key      string
		expected int64
	}{
		{
			name:     "valid float64 value",
			input:    map[string]any{"size": float64(12345)},
			key:      "size",
			expected: 12345,
		},
		{
			name:     "zero value",
			input:    map[string]any{"size": float64(0)},
			key:      "size",
			expected: 0,
		},
		{
			name:     "negative value",
			input:    map[string]any{"size": float64(-100)},
			key:      "size",
			expected: -100,
		},
		{
			name:     "large value",
			input:    map[string]any{"size": float64(9007199254740992)}, // 2^53, largest int exactly representable as float64
			key:      "size",
			expected: 9007199254740992,
		},
		{
			name:     "missing key",
			input:    map[string]any{"size": float64(12345)},
			key:      "missing",
			expected: 0,
		},
		{
			name:     "wrong type - string",
			input:    map[string]any{"size": "12345"},
			key:      "size",
			expected: 0,
		},
		{
			name:     "wrong type - int64",
			input:    map[string]any{"size": int64(12345)},
			key:      "size",
			expected: 0,
		},
		{
			name:     "wrong type - bool",
			input:    map[string]any{"size": false},
			key:      "size",
			expected: 0,
		},
		{
			name:     "wrong type - nil",
			input:    map[string]any{"size": nil},
			key:      "size",
			expected: 0,
		},
		{
			name:     "empty map",
			input:    map[string]any{},
			key:      "size",
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getInt64(tt.input, tt.key)
			if got != tt.expected {
				t.Errorf("getInt64() = %d, want %d", got, tt.expected)
			}
		})
	}
}

func TestGetBool(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]any
		key      string
		expected bool
	}{
		{
			name:     "true value",
			input:    map[string]any{"flag": true},
			key:      "flag",
			expected: true,
		},
		{
			name:     "false value",
			input:    map[string]any{"flag": false},
			key:      "flag",
			expected: false,
		},
		{
			name:     "missing key",
			input:    map[string]any{"flag": true},
			key:      "missing",
			expected: false,
		},
		{
			name:     "wrong type - string",
			input:    map[string]any{"flag": "true"},
			key:      "flag",
			expected: false,
		},
		{
			name:     "wrong type - int",
			input:    map[string]any{"flag": 1},
			key:      "flag",
			expected: false,
		},
		{
			name:     "wrong type - nil",
			input:    map[string]any{"flag": nil},
			key:      "flag",
			expected: false,
		},
		{
			name:     "empty map",
			input:    map[string]any{},
			key:      "flag",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getBool(tt.input, tt.key)
			if got != tt.expected {
				t.Errorf("getBool() = %t, want %t", got, tt.expected)
			}
		})
	}
}
