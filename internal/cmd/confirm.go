package cmd

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"golang.org/x/term"
)

func confirmPrompt(w io.Writer, prompt string, accepted ...string) (bool, error) {
	fmt.Fprint(w, prompt)
	scanner := bufio.NewScanner(os.Stdin)
	if !scanner.Scan() {
		if err := scanner.Err(); err != nil {
			return false, fmt.Errorf("failed to read confirmation: %w", err)
		}
		if !term.IsTerminal(int(os.Stdin.Fd())) {
			return false, Suggest(fmt.Errorf("confirmation required in non-interactive mode"), "Re-run with --yes to skip confirmation")
		}
		return false, fmt.Errorf("cancelled")
	}
	response := strings.ToLower(strings.TrimSpace(scanner.Text()))
	for _, ok := range accepted {
		if response == ok {
			return true, nil
		}
	}
	return false, nil
}
