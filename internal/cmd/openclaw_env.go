package cmd

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const (
	openClawDirName     = ".openclaw"
	openClawEnvFileName = ".env"
)

// loadOpenClawEnvIfPresent loads ~/.openclaw/.env when available.
// Existing process environment variables are not overwritten.
func loadOpenClawEnvIfPresent() error {
	homeDir, err := os.UserHomeDir()
	if err != nil || strings.TrimSpace(homeDir) == "" {
		return nil
	}

	path := filepath.Join(homeDir, openClawDirName, openClawEnvFileName)
	return loadEnvFileIfPresent(path)
}

func loadEnvFileIfPresent(path string) error {
	f, err := os.Open(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("open %s: %w", path, err)
	}
	defer func() { _ = f.Close() }()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		key, value, ok := parseDotEnvLine(scanner.Text())
		if !ok {
			continue
		}
		if _, exists := os.LookupEnv(key); exists {
			continue
		}
		if err := os.Setenv(key, value); err != nil {
			return fmt.Errorf("set env %s: %w", key, err)
		}
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("read %s: %w", path, err)
	}
	return nil
}

func parseDotEnvLine(line string) (key, value string, ok bool) {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" || strings.HasPrefix(trimmed, "#") {
		return "", "", false
	}

	if strings.HasPrefix(trimmed, "export ") {
		trimmed = strings.TrimSpace(strings.TrimPrefix(trimmed, "export "))
	}

	idx := strings.IndexRune(trimmed, '=')
	if idx <= 0 {
		return "", "", false
	}

	key = strings.TrimSpace(trimmed[:idx])
	if key == "" {
		return "", "", false
	}

	value = strings.TrimSpace(trimmed[idx+1:])
	if len(value) >= 2 {
		if value[0] == '"' && value[len(value)-1] == '"' {
			if unquoted, err := strconv.Unquote(value); err == nil {
				value = unquoted
			} else {
				value = strings.Trim(value, "\"")
			}
		} else if value[0] == '\'' && value[len(value)-1] == '\'' {
			value = strings.TrimSuffix(strings.TrimPrefix(value, "'"), "'")
		}
	}

	return key, value, true
}
