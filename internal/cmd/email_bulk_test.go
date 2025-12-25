package cmd

import (
	"testing"
)

func TestEmailBulkDeleteCmd_RequiresArgs(t *testing.T) {
	// Create the root command with a minimal flags structure
	flags := &rootFlags{}
	cmd := newEmailBulkDeleteCmd(flags)

	// Set args to empty (no email IDs provided)
	cmd.SetArgs([]string{})

	// Execute should fail because bulk-delete requires at least 1 email ID
	err := cmd.Execute()
	if err == nil {
		t.Error("expected error when no email IDs provided, got nil")
	}

	// Verify the error is related to args validation
	// Cobra's MinimumNArgs returns an error like "requires at least 1 arg(s), only received 0"
	expectedErrPattern := "requires at least 1 arg"
	if err != nil && !contains(err.Error(), expectedErrPattern) {
		t.Errorf("expected error containing %q, got: %v", expectedErrPattern, err)
	}
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		func() bool {
			for i := 0; i <= len(s)-len(substr); i++ {
				if s[i:i+len(substr)] == substr {
					return true
				}
			}
			return false
		}())
}

// TestEmailBulkDeleteCmd_AcceptsMultipleArgs verifies that the command accepts multiple email IDs
func TestEmailBulkDeleteCmd_AcceptsMultipleArgs(t *testing.T) {
	flags := &rootFlags{}
	cmd := newEmailBulkDeleteCmd(flags)

	// Verify that Args validator allows multiple arguments
	argsValidator := cmd.Args
	if argsValidator == nil {
		t.Fatal("expected Args validator to be set")
	}

	// Test with 1 arg - should pass validation
	err := argsValidator(cmd, []string{"email1"})
	if err != nil {
		t.Errorf("expected Args validator to accept 1 arg, got error: %v", err)
	}

	// Test with multiple args - should pass validation
	err = argsValidator(cmd, []string{"email1", "email2", "email3"})
	if err != nil {
		t.Errorf("expected Args validator to accept multiple args, got error: %v", err)
	}

	// Test with 0 args - should fail validation
	err = argsValidator(cmd, []string{})
	if err == nil {
		t.Error("expected Args validator to reject 0 args, got nil error")
	}
}

// TestEmailBulkDeleteCmd_HasRequiredFlags verifies that the command has the expected flags
func TestEmailBulkDeleteCmd_HasRequiredFlags(t *testing.T) {
	flags := &rootFlags{}
	cmd := newEmailBulkDeleteCmd(flags)

	// Verify --dry-run flag exists
	dryRunFlag := cmd.Flags().Lookup("dry-run")
	if dryRunFlag == nil {
		t.Error("expected --dry-run flag to exist")
	}

	// Verify --yes flag exists
	yesFlag := cmd.Flags().Lookup("yes")
	if yesFlag == nil {
		t.Error("expected --yes flag to exist")
	}

	// Verify -y shorthand exists
	yShortFlag := cmd.Flags().ShorthandLookup("y")
	if yShortFlag == nil {
		t.Error("expected -y shorthand flag to exist")
	}
}

// TestEmailBulkDeleteCmd_CommandMetadata verifies command metadata is set correctly
func TestEmailBulkDeleteCmd_CommandMetadata(t *testing.T) {
	flags := &rootFlags{}
	cmd := newEmailBulkDeleteCmd(flags)

	if cmd.Use != "bulk-delete <emailId>..." {
		t.Errorf("expected Use to be 'bulk-delete <emailId>...', got: %q", cmd.Use)
	}

	if cmd.Short == "" {
		t.Error("expected Short description to be set")
	}

	if cmd.RunE == nil {
		t.Error("expected RunE function to be set")
	}

	// Verify it's using MinimumNArgs(1)
	if cmd.Args == nil {
		t.Error("expected Args validator to be set")
	}

	// Test the validator accepts 1+ args
	if err := cmd.Args(cmd, []string{"id1"}); err != nil {
		t.Errorf("Args validator should accept 1 arg: %v", err)
	}
	if err := cmd.Args(cmd, []string{}); err == nil {
		t.Error("Args validator should reject 0 args")
	}
}

// Ensure bulk-delete is registered as a subcommand of email
func TestEmailCmd_HasBulkDeleteSubcommand(t *testing.T) {
	flags := &rootFlags{}
	emailCmd := newEmailCmd(flags)

	// Find bulk-delete subcommand
	var found bool
	for _, cmd := range emailCmd.Commands() {
		if cmd.Name() == "bulk-delete" {
			found = true
			break
		}
	}

	if !found {
		t.Error("expected 'bulk-delete' to be registered as a subcommand of 'email'")
	}
}

// TestEmailBulkMoveCmd_RequiresToFlag verifies that the --to flag is required
func TestEmailBulkMoveCmd_RequiresToFlag(t *testing.T) {
	flags := &rootFlags{}
	cmd := newEmailBulkMoveCmd(flags)

	// Set args with email IDs but no --to flag
	cmd.SetArgs([]string{"email1", "email2"})

	// Execute should fail because --to is required
	err := cmd.Execute()
	if err == nil {
		t.Error("expected error when --to flag is not provided, got nil")
	}

	// Verify the error is about the missing --to flag
	expectedErrPattern := "--to is required"
	if err != nil && !contains(err.Error(), expectedErrPattern) {
		t.Errorf("expected error containing %q, got: %v", expectedErrPattern, err)
	}
}

func TestEmailBulkMoveCmd_RequiresArgs(t *testing.T) {
	flags := &rootFlags{}
	cmd := newEmailBulkMoveCmd(flags)

	// Set args to empty (no email IDs provided)
	cmd.SetArgs([]string{})

	// Execute should fail because bulk-move requires at least 1 email ID
	err := cmd.Execute()
	if err == nil {
		t.Error("expected error when no email IDs provided, got nil")
	}

	// Verify the error is related to args validation
	expectedErrPattern := "requires at least 1 arg"
	if err != nil && !contains(err.Error(), expectedErrPattern) {
		t.Errorf("expected error containing %q, got: %v", expectedErrPattern, err)
	}
}

func TestEmailBulkMoveCmd_AcceptsMultipleArgs(t *testing.T) {
	flags := &rootFlags{}
	cmd := newEmailBulkMoveCmd(flags)

	// Verify that Args validator allows multiple arguments
	argsValidator := cmd.Args
	if argsValidator == nil {
		t.Fatal("expected Args validator to be set")
	}

	// Test with 1 arg - should pass validation
	err := argsValidator(cmd, []string{"email1"})
	if err != nil {
		t.Errorf("expected Args validator to accept 1 arg, got error: %v", err)
	}

	// Test with multiple args - should pass validation
	err = argsValidator(cmd, []string{"email1", "email2", "email3"})
	if err != nil {
		t.Errorf("expected Args validator to accept multiple args, got error: %v", err)
	}

	// Test with 0 args - should fail validation
	err = argsValidator(cmd, []string{})
	if err == nil {
		t.Error("expected Args validator to reject 0 args, got nil error")
	}
}

func TestEmailBulkMoveCmd_HasRequiredFlags(t *testing.T) {
	flags := &rootFlags{}
	cmd := newEmailBulkMoveCmd(flags)

	// Verify --to flag exists
	toFlag := cmd.Flags().Lookup("to")
	if toFlag == nil {
		t.Error("expected --to flag to exist")
	}

	// Verify --dry-run flag exists
	dryRunFlag := cmd.Flags().Lookup("dry-run")
	if dryRunFlag == nil {
		t.Error("expected --dry-run flag to exist")
	}

	// Verify --yes flag exists
	yesFlag := cmd.Flags().Lookup("yes")
	if yesFlag == nil {
		t.Error("expected --yes flag to exist")
	}

	// Verify -y shorthand exists
	yShortFlag := cmd.Flags().ShorthandLookup("y")
	if yShortFlag == nil {
		t.Error("expected -y shorthand flag to exist")
	}
}

func TestEmailBulkMoveCmd_CommandMetadata(t *testing.T) {
	flags := &rootFlags{}
	cmd := newEmailBulkMoveCmd(flags)

	if cmd.Use != "bulk-move <emailId>..." {
		t.Errorf("expected Use to be 'bulk-move <emailId>...', got: %q", cmd.Use)
	}

	if cmd.Short == "" {
		t.Error("expected Short description to be set")
	}

	if cmd.RunE == nil {
		t.Error("expected RunE function to be set")
	}

	// Verify it's using MinimumNArgs(1)
	if cmd.Args == nil {
		t.Error("expected Args validator to be set")
	}

	// Test the validator accepts 1+ args
	if err := cmd.Args(cmd, []string{"id1"}); err != nil {
		t.Errorf("Args validator should accept 1 arg: %v", err)
	}
	if err := cmd.Args(cmd, []string{}); err == nil {
		t.Error("Args validator should reject 0 args")
	}
}

func TestEmailCmd_HasBulkMoveSubcommand(t *testing.T) {
	flags := &rootFlags{}
	emailCmd := newEmailCmd(flags)

	// Find bulk-move subcommand
	var found bool
	for _, cmd := range emailCmd.Commands() {
		if cmd.Name() == "bulk-move" {
			found = true
			break
		}
	}

	if !found {
		t.Error("expected 'bulk-move' to be registered as a subcommand of 'email'")
	}
}

func TestEmailBulkMarkReadCmd_RequiresArgs(t *testing.T) {
	flags := &rootFlags{}
	cmd := newEmailBulkMarkReadCmd(flags)

	// Set args to empty (no email IDs provided)
	cmd.SetArgs([]string{})

	// Execute should fail because bulk-mark-read requires at least 1 email ID
	err := cmd.Execute()
	if err == nil {
		t.Error("expected error when no email IDs provided, got nil")
	}

	// Verify the error is related to args validation
	expectedErrPattern := "requires at least 1 arg"
	if err != nil && !contains(err.Error(), expectedErrPattern) {
		t.Errorf("expected error containing %q, got: %v", expectedErrPattern, err)
	}
}

func TestEmailBulkMarkReadCmd_AcceptsMultipleArgs(t *testing.T) {
	flags := &rootFlags{}
	cmd := newEmailBulkMarkReadCmd(flags)

	// Verify that Args validator allows multiple arguments
	argsValidator := cmd.Args
	if argsValidator == nil {
		t.Fatal("expected Args validator to be set")
	}

	// Test with 1 arg - should pass validation
	err := argsValidator(cmd, []string{"email1"})
	if err != nil {
		t.Errorf("expected Args validator to accept 1 arg, got error: %v", err)
	}

	// Test with multiple args - should pass validation
	err = argsValidator(cmd, []string{"email1", "email2", "email3"})
	if err != nil {
		t.Errorf("expected Args validator to accept multiple args, got error: %v", err)
	}

	// Test with 0 args - should fail validation
	err = argsValidator(cmd, []string{})
	if err == nil {
		t.Error("expected Args validator to reject 0 args, got nil error")
	}
}

func TestEmailBulkMarkReadCmd_HasRequiredFlags(t *testing.T) {
	flags := &rootFlags{}
	cmd := newEmailBulkMarkReadCmd(flags)

	// Verify --unread flag exists
	unreadFlag := cmd.Flags().Lookup("unread")
	if unreadFlag == nil {
		t.Error("expected --unread flag to exist")
	}

	// Verify --dry-run flag exists
	dryRunFlag := cmd.Flags().Lookup("dry-run")
	if dryRunFlag == nil {
		t.Error("expected --dry-run flag to exist")
	}
}

func TestEmailBulkMarkReadCmd_CommandMetadata(t *testing.T) {
	flags := &rootFlags{}
	cmd := newEmailBulkMarkReadCmd(flags)

	if cmd.Use != "bulk-mark-read <emailId>..." {
		t.Errorf("expected Use to be 'bulk-mark-read <emailId>...', got: %q", cmd.Use)
	}

	if cmd.Short == "" {
		t.Error("expected Short description to be set")
	}

	if cmd.RunE == nil {
		t.Error("expected RunE function to be set")
	}

	// Verify it's using MinimumNArgs(1)
	if cmd.Args == nil {
		t.Error("expected Args validator to be set")
	}

	// Test the validator accepts 1+ args
	if err := cmd.Args(cmd, []string{"id1"}); err != nil {
		t.Errorf("Args validator should accept 1 arg: %v", err)
	}
	if err := cmd.Args(cmd, []string{}); err == nil {
		t.Error("Args validator should reject 0 args")
	}
}

func TestEmailCmd_HasBulkMarkReadSubcommand(t *testing.T) {
	flags := &rootFlags{}
	emailCmd := newEmailCmd(flags)

	// Find bulk-mark-read subcommand
	var found bool
	for _, cmd := range emailCmd.Commands() {
		if cmd.Name() == "bulk-mark-read" {
			found = true
			break
		}
	}

	if !found {
		t.Error("expected 'bulk-mark-read' to be registered as a subcommand of 'email'")
	}
}
