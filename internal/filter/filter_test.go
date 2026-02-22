package filter

import (
	"reflect"
	"testing"
)

func TestApply(t *testing.T) {
	tests := []struct {
		name       string
		data       any
		expression string
		want       any
		wantErr    bool
	}{
		{
			name:       "empty expression returns data unchanged",
			data:       map[string]string{"key": "value"},
			expression: "",
			want:       map[string]string{"key": "value"},
		},
		{
			name:       "select field",
			data:       map[string]any{"name": "test", "id": 123},
			expression: ".name",
			want:       "test",
		},
		{
			name:       "select nested field",
			data:       map[string]any{"user": map[string]any{"name": "alice"}},
			expression: ".user.name",
			want:       "alice",
		},
		{
			name:       "array index",
			data:       map[string]any{"items": []any{"a", "b", "c"}},
			expression: ".items[0]",
			want:       "a",
		},
		{
			name: "map over array",
			data: map[string]any{"emails": []any{
				map[string]any{"subject": "Hello"},
				map[string]any{"subject": "World"},
			}},
			expression: ".emails[].subject",
			want:       []any{"Hello", "World"},
		},
		{
			name:       "invalid expression",
			data:       map[string]string{"key": "value"},
			expression: ".invalid[",
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Apply(tt.data, tt.expression)
			if (err != nil) != tt.wantErr {
				t.Errorf("Apply() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Apply() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestApplyToJSON(t *testing.T) {
	input := []byte(`{"emails":[{"subject":"Hello"},{"subject":"World"}]}`)

	result, err := ApplyToJSON(input, ".emails[].subject")
	if err != nil {
		t.Fatalf("ApplyToJSON() error = %v", err)
	}

	expected := `[
  "Hello",
  "World"
]`
	if string(result) != expected {
		t.Errorf("ApplyToJSON() = %s, want %s", result, expected)
	}
}
