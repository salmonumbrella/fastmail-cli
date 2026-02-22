package cmd

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadOpenClawEnvIfPresent_LoadsValues(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	unsetEnvWithRestore(t, "FASTMAIL_ACCOUNT")
	unsetEnvWithRestore(t, "FASTMAIL_OUTPUT")
	unsetEnvWithRestore(t, "FASTMAIL_CREDENTIALS_DIR")
	unsetEnvWithRestore(t, "OPENCLAW_CREDENTIALS_DIR")

	openClawDir := filepath.Join(home, openClawDirName)
	if err := os.MkdirAll(openClawDir, 0o700); err != nil {
		t.Fatalf("failed to create OpenClaw dir: %v", err)
	}

	content := "" +
		"# comment\n" +
		"FASTMAIL_ACCOUNT=account_from_openclaw\n" +
		"export FASTMAIL_OUTPUT=json\n" +
		"FASTMAIL_CREDENTIALS_DIR=\"/tmp/fastmail-creds\"\n" +
		"OPENCLAW_CREDENTIALS_DIR='/tmp/shared-creds'\n"
	path := filepath.Join(openClawDir, openClawEnvFileName)
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("failed to write OpenClaw env file: %v", err)
	}

	if err := loadOpenClawEnvIfPresent(); err != nil {
		t.Fatalf("loadOpenClawEnvIfPresent() error = %v", err)
	}

	if got := os.Getenv("FASTMAIL_ACCOUNT"); got != "account_from_openclaw" {
		t.Fatalf("FASTMAIL_ACCOUNT = %q, want %q", got, "account_from_openclaw")
	}
	if got := os.Getenv("FASTMAIL_OUTPUT"); got != "json" {
		t.Fatalf("FASTMAIL_OUTPUT = %q, want %q", got, "json")
	}
	if got := os.Getenv("FASTMAIL_CREDENTIALS_DIR"); got != "/tmp/fastmail-creds" {
		t.Fatalf("FASTMAIL_CREDENTIALS_DIR = %q, want %q", got, "/tmp/fastmail-creds")
	}
	if got := os.Getenv("OPENCLAW_CREDENTIALS_DIR"); got != "/tmp/shared-creds" {
		t.Fatalf("OPENCLAW_CREDENTIALS_DIR = %q, want %q", got, "/tmp/shared-creds")
	}
}

func TestLoadOpenClawEnvIfPresent_DoesNotOverrideExisting(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("FASTMAIL_ACCOUNT", "already-set")

	openClawDir := filepath.Join(home, openClawDirName)
	if err := os.MkdirAll(openClawDir, 0o700); err != nil {
		t.Fatalf("failed to create OpenClaw dir: %v", err)
	}
	path := filepath.Join(openClawDir, openClawEnvFileName)
	if err := os.WriteFile(path, []byte("FASTMAIL_ACCOUNT=from-file\n"), 0o600); err != nil {
		t.Fatalf("failed to write OpenClaw env file: %v", err)
	}

	if err := loadOpenClawEnvIfPresent(); err != nil {
		t.Fatalf("loadOpenClawEnvIfPresent() error = %v", err)
	}

	if got := os.Getenv("FASTMAIL_ACCOUNT"); got != "already-set" {
		t.Fatalf("FASTMAIL_ACCOUNT = %q, want %q", got, "already-set")
	}
}

func TestLoadOpenClawEnvIfPresent_MissingFile(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	if err := loadOpenClawEnvIfPresent(); err != nil {
		t.Fatalf("loadOpenClawEnvIfPresent() error = %v", err)
	}
}

func TestParseDotEnvLine(t *testing.T) {
	tests := []struct {
		name  string
		line  string
		key   string
		value string
		ok    bool
	}{
		{name: "simple", line: "A=B", key: "A", value: "B", ok: true},
		{name: "export", line: "export A=B", key: "A", value: "B", ok: true},
		{name: "double quoted", line: "A=\"B C\"", key: "A", value: "B C", ok: true},
		{name: "single quoted", line: "A='B C'", key: "A", value: "B C", ok: true},
		{name: "comment", line: "# test", ok: false},
		{name: "invalid", line: "A", ok: false},
		{name: "empty key", line: "=value", ok: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key, value, ok := parseDotEnvLine(tt.line)
			if ok != tt.ok {
				t.Fatalf("parseDotEnvLine(%q) ok = %v, want %v", tt.line, ok, tt.ok)
			}
			if !ok {
				return
			}
			if key != tt.key {
				t.Fatalf("parseDotEnvLine(%q) key = %q, want %q", tt.line, key, tt.key)
			}
			if value != tt.value {
				t.Fatalf("parseDotEnvLine(%q) value = %q, want %q", tt.line, value, tt.value)
			}
		})
	}
}

func unsetEnvWithRestore(t *testing.T, key string) {
	t.Helper()

	original, existed := os.LookupEnv(key)
	_ = os.Unsetenv(key)
	t.Cleanup(func() {
		if existed {
			_ = os.Setenv(key, original)
		} else {
			_ = os.Unsetenv(key)
		}
	})
}
