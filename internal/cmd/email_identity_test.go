package cmd

import (
	"testing"
)

func TestIdentitySetDefaultCmd_RequiresArg(t *testing.T) {
	app := newTestApp()
	cmd := newIdentitySetDefaultCmd(app)

	// Set args to empty (no email provided)
	cmd.SetArgs([]string{})

	// Execute should fail because identity-set-default requires exactly 1 arg
	err := cmd.Execute()
	if err == nil {
		t.Error("expected error when no email provided, got nil")
	}
}

func TestIdentitySetDefaultCmd_CommandMetadata(t *testing.T) {
	app := newTestApp()
	cmd := newIdentitySetDefaultCmd(app)

	// Verify Use
	expectedUse := "identity-set-default <email>"
	if cmd.Use != expectedUse {
		t.Errorf("Use = %q, want %q", cmd.Use, expectedUse)
	}

	// Verify Short description is set
	if cmd.Short == "" {
		t.Error("Short description should not be empty")
	}

	// Verify Long description is set
	if cmd.Long == "" {
		t.Error("Long description should not be empty")
	}
}

func TestIdentitySetDefaultCmd_AcceptsExactlyOneArg(t *testing.T) {
	app := newTestApp()
	cmd := newIdentitySetDefaultCmd(app)

	argsValidator := cmd.Args
	if argsValidator == nil {
		t.Fatal("expected Args validator to be set")
	}

	// Test with 1 arg - should pass validation
	err := argsValidator(cmd, []string{"user@example.com"})
	if err != nil {
		t.Errorf("expected Args validator to accept 1 arg, got error: %v", err)
	}

	// Test with 0 args - should fail validation
	err = argsValidator(cmd, []string{})
	if err == nil {
		t.Error("expected Args validator to reject 0 args, got nil error")
	}

	// Test with 2 args - should fail validation
	err = argsValidator(cmd, []string{"user@example.com", "extra@example.com"})
	if err == nil {
		t.Error("expected Args validator to reject 2 args, got nil error")
	}
}

func TestEmailCmd_HasIdentitySetDefaultSubcommand(t *testing.T) {
	app := newTestApp()
	emailCmd := newEmailCmd(app)

	// Find the identity-set-default subcommand
	var found bool
	for _, sub := range emailCmd.Commands() {
		if sub.Name() == "identity-set-default" {
			found = true
			break
		}
	}

	if !found {
		t.Error("email command should have identity-set-default subcommand")
	}
}
