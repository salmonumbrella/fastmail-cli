package format

import "testing"

func TestTruncate(t *testing.T) {
	cases := []struct {
		name   string
		in     string
		maxLen int
		want   string
	}{
		{"short", "hello", 10, "hello"},
		{"exact", "hello", 5, "hello"},
		{"long", "hello", 4, "h..."},
		{"longer", "abcdefghij", 6, "abc..."},
		{"cjk_short", "日本語", 10, "日本語"},
		{"cjk_truncate", "日本語テストです", 6, "日本語..."},
		{"emoji", "Hello 🌍🌎🌏!", 10, "Hello 🌍🌎🌏!"},
		{"emoji_truncate", "Hello 🌍🌎🌏 World", 10, "Hello 🌍..."},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			if got := Truncate(tt.in, tt.maxLen); got != tt.want {
				t.Fatalf("Truncate(%q, %d) = %q, want %q", tt.in, tt.maxLen, got, tt.want)
			}
		})
	}
}
