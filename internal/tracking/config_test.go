package tracking

import (
	"path/filepath"
	"testing"
)

func TestConfigPath(t *testing.T) {
	path, err := ConfigPath()
	if err != nil {
		t.Fatalf("ConfigPath: %v", err)
	}

	if !filepath.IsAbs(path) {
		t.Errorf("path should be absolute: %s", path)
	}

	if filepath.Base(path) != "config.json" {
		t.Errorf("unexpected filename: %s", filepath.Base(path))
	}
}

func TestIsConfigured(t *testing.T) {
	tests := []struct {
		name     string
		cfg      Config
		expected bool
	}{
		{
			name:     "fully configured",
			cfg:      Config{Enabled: true, WorkerURL: "https://test.dev", TrackingKey: "key123"},
			expected: true,
		},
		{
			name:     "disabled",
			cfg:      Config{Enabled: false, WorkerURL: "https://test.dev", TrackingKey: "key123"},
			expected: false,
		},
		{
			name:     "missing worker URL",
			cfg:      Config{Enabled: true, WorkerURL: "", TrackingKey: "key123"},
			expected: false,
		},
		{
			name:     "missing tracking key",
			cfg:      Config{Enabled: true, WorkerURL: "https://test.dev", TrackingKey: ""},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.cfg.IsConfigured(); got != tt.expected {
				t.Errorf("IsConfigured() = %v, want %v", got, tt.expected)
			}
		})
	}
}
