package cmd

import (
	"os"
	"testing"
)

func TestConfirmPrompt_Accepted(t *testing.T) {
	stdin := os.Stdin
	defer func() { os.Stdin = stdin }()

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	os.Stdin = r

	_, _ = w.WriteString("YES\n")
	_ = w.Close()

	confirmed, err := confirmPrompt(os.Stdout, "Confirm? ", "y", "yes")
	if err != nil {
		t.Fatalf("confirmPrompt error: %v", err)
	}
	if !confirmed {
		t.Fatalf("confirmPrompt = false, want true")
	}
}

func TestConfirmPrompt_Denied(t *testing.T) {
	stdin := os.Stdin
	defer func() { os.Stdin = stdin }()

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	os.Stdin = r

	_, _ = w.WriteString("no\n")
	_ = w.Close()

	confirmed, err := confirmPrompt(os.Stdout, "Confirm? ", "y", "yes")
	if err != nil {
		t.Fatalf("confirmPrompt error: %v", err)
	}
	if confirmed {
		t.Fatalf("confirmPrompt = true, want false")
	}
}

func TestConfirmPrompt_EOF(t *testing.T) {
	stdin := os.Stdin
	defer func() { os.Stdin = stdin }()

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	os.Stdin = r
	_ = w.Close()

	_, err = confirmPrompt(os.Stdout, "Confirm? ", "y", "yes")
	if err == nil {
		t.Fatalf("confirmPrompt error = nil, want error")
	}
}
