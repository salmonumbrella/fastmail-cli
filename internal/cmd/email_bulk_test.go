package cmd

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/salmonumbrella/fastmail-cli/internal/jmap"
	"github.com/spf13/cobra"
)

func TestEmailBulkDeleteCmd_RequiresArgs(t *testing.T) {
	// Create the root command with a minimal flags structure
	app := newTestApp()
	cmd := newEmailBulkDeleteCmd(app)

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
	if err != nil && !strings.Contains(err.Error(), expectedErrPattern) {
		t.Errorf("expected error containing %q, got: %v", expectedErrPattern, err)
	}
}

// TestEmailBulkDeleteCmd_AcceptsMultipleArgs verifies that the command accepts multiple email IDs
func TestEmailBulkDeleteCmd_AcceptsMultipleArgs(t *testing.T) {
	app := newTestApp()
	cmd := newEmailBulkDeleteCmd(app)

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

func TestEmailBulkDeleteCmd_AcceptsInputFlagsWithoutPositionalArgs(t *testing.T) {
	app := newTestApp()

	cmdStdin := newEmailBulkDeleteCmd(app)
	if err := cmdStdin.Flags().Set("stdin", "true"); err != nil {
		t.Fatalf("set --stdin: %v", err)
	}
	if err := cmdStdin.Args(cmdStdin, []string{}); err != nil {
		t.Fatalf("expected --stdin to satisfy args requirement, got: %v", err)
	}

	cmdFile := newEmailBulkDeleteCmd(app)
	if err := cmdFile.Flags().Set("ids-file", "/tmp/fm-ids.txt"); err != nil {
		t.Fatalf("set --ids-file: %v", err)
	}
	if err := cmdFile.Args(cmdFile, []string{}); err != nil {
		t.Fatalf("expected --ids-file to satisfy args requirement, got: %v", err)
	}
}

// TestEmailBulkDeleteCmd_HasRequiredFlags verifies that the command has the expected flags
func TestEmailBulkDeleteCmd_HasRequiredFlags(t *testing.T) {
	app := newTestApp()
	root := NewRootCmd(app)
	emailCmd := root.Commands()[0]
	for _, c := range root.Commands() {
		if c.Name() == "email" {
			emailCmd = c
			break
		}
	}

	var cmd *cobra.Command
	for _, c := range emailCmd.Commands() {
		if c.Name() == "bulk-delete" {
			cmd = c
			break
		}
	}
	if cmd == nil {
		t.Fatal("expected bulk-delete command to exist under email")
	}

	// Verify --dry-run flag exists
	dryRunFlag := cmd.Flags().Lookup("dry-run")
	if dryRunFlag == nil {
		t.Error("expected --dry-run flag to exist")
	}

	// Verify bulk input flags exist
	if cmd.Flags().Lookup("stdin") == nil {
		t.Error("expected --stdin flag to exist")
	}
	if cmd.Flags().Lookup("ids-file") == nil {
		t.Error("expected --ids-file flag to exist")
	}
	if cmd.Flags().Lookup("batch-size") == nil {
		t.Error("expected --batch-size flag to exist")
	}

	// Verify --yes flag exists (inherited from root persistent flags)
	yesFlag := cmd.InheritedFlags().Lookup("yes")
	if yesFlag == nil {
		t.Error("expected --yes flag to exist")
	}

	// Verify -y shorthand exists
	yShortFlag := cmd.InheritedFlags().ShorthandLookup("y")
	if yShortFlag == nil {
		t.Error("expected -y shorthand flag to exist")
	}
}

// TestEmailBulkDeleteCmd_CommandMetadata verifies command metadata is set correctly
func TestEmailBulkDeleteCmd_CommandMetadata(t *testing.T) {
	app := newTestApp()
	cmd := newEmailBulkDeleteCmd(app)

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
	app := newTestApp()
	emailCmd := newEmailCmd(app)

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
	app := newTestApp()
	cmd := newEmailBulkMoveCmd(app)

	// Set args with email IDs but no --to flag
	cmd.SetArgs([]string{"email1", "email2"})

	// Execute should fail because --to is required
	err := cmd.Execute()
	if err == nil {
		t.Error("expected error when --to flag is not provided, got nil")
	}

	// Verify the error is about the missing --to flag
	expectedErrPattern := "--to is required"
	if err != nil && !strings.Contains(err.Error(), expectedErrPattern) {
		t.Errorf("expected error containing %q, got: %v", expectedErrPattern, err)
	}
}

func TestRunEmailBulkMove_RequiresMailbox(t *testing.T) {
	app := newTestApp()

	err := runEmailBulkMove(&cobra.Command{}, []string{"email1"}, app, "", false, bulkInputOptions{BatchSize: defaultBulkBatchSize})
	if err == nil {
		t.Fatal("expected error when target mailbox is empty")
	}
	if !strings.Contains(err.Error(), "--to is required") {
		t.Fatalf("expected '--to is required' error, got: %v", err)
	}
}

func TestRunEmailBulkMove_DryRun(t *testing.T) {
	app := newTestApp()
	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())

	out := captureStdout(t, func() {
		err := runEmailBulkMove(cmd, []string{"email1", "email2"}, app, "Archive", true, bulkInputOptions{BatchSize: defaultBulkBatchSize})
		if err != nil {
			t.Fatalf("runEmailBulkMove() dry-run unexpected error: %v", err)
		}
	})

	if !strings.Contains(out, "Would move 2 emails to Archive") {
		t.Fatalf("expected dry-run output to mention mailbox and count, got: %q", out)
	}
}

type fakeBulkMoveClient struct {
	mailboxes     []jmap.Mailbox
	moveResults   []*jmap.BulkResult
	moveErr       error
	moveCalls     [][]string
	targets       []string
	getMailboxErr error
}

func (f *fakeBulkMoveClient) GetMailboxes(_ context.Context) ([]jmap.Mailbox, error) {
	if f.getMailboxErr != nil {
		return nil, f.getMailboxErr
	}
	return f.mailboxes, nil
}

func (f *fakeBulkMoveClient) MoveEmails(_ context.Context, ids []string, targetMailboxID string) (*jmap.BulkResult, error) {
	if f.moveErr != nil {
		return nil, f.moveErr
	}
	copied := append([]string(nil), ids...)
	f.moveCalls = append(f.moveCalls, copied)
	f.targets = append(f.targets, targetMailboxID)

	if len(f.moveResults) == 0 {
		return &jmap.BulkResult{Succeeded: []string{}, Failed: map[string]string{}}, nil
	}
	result := f.moveResults[0]
	f.moveResults = f.moveResults[1:]
	return result, nil
}

func TestRunEmailBulkMoveWithClient_SuccessInBatches(t *testing.T) {
	app := newTestApp()
	app.Flags.Yes = true
	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())

	client := &fakeBulkMoveClient{
		mailboxes: []jmap.Mailbox{{ID: "archive-1", Name: "Archive", Role: "archive"}},
		moveResults: []*jmap.BulkResult{
			{Succeeded: []string{"id1", "id2"}, Failed: map[string]string{}},
			{Succeeded: []string{"id3"}, Failed: map[string]string{}},
		},
	}

	out := captureStdout(t, func() {
		err := runEmailBulkMoveWithClient(cmd, app, client, []string{"id1", "id2", "id3"}, "Archive", 2)
		if err != nil {
			t.Fatalf("runEmailBulkMoveWithClient error: %v", err)
		}
	})

	if len(client.moveCalls) != 2 {
		t.Fatalf("expected 2 batched MoveEmails calls, got %d", len(client.moveCalls))
	}
	if client.targets[0] != "archive-1" || client.targets[1] != "archive-1" {
		t.Fatalf("expected resolved mailbox ID archive-1, got %v", client.targets)
	}
	if !strings.Contains(out, "Processed 3 emails in 2 batches") {
		t.Fatalf("expected batch progress line, got: %q", out)
	}
	if !strings.Contains(out, "Moved 3 emails to Archive") {
		t.Fatalf("expected final moved summary, got: %q", out)
	}
}

func TestRunEmailBulkMoveWithClient_PartialFailure(t *testing.T) {
	app := newTestApp()
	app.Flags.Yes = true
	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())

	client := &fakeBulkMoveClient{
		mailboxes: []jmap.Mailbox{{ID: "archive-1", Name: "Archive", Role: "archive"}},
		moveResults: []*jmap.BulkResult{
			{Succeeded: []string{"id1"}, Failed: map[string]string{"id2": "notFound"}},
			{Succeeded: []string{"id3"}, Failed: map[string]string{}},
		},
	}

	out := captureStdout(t, func() {
		err := runEmailBulkMoveWithClient(cmd, app, client, []string{"id1", "id2", "id3"}, "Archive", 2)
		if err != nil {
			t.Fatalf("runEmailBulkMoveWithClient error: %v", err)
		}
	})

	if !strings.Contains(out, "Moved 2 emails to Archive, 1 failed:") {
		t.Fatalf("expected partial failure summary, got: %q", out)
	}
	if !strings.Contains(out, "id2: notFound") {
		t.Fatalf("expected failed ID details, got: %q", out)
	}
}

func TestRunEmailBulkMoveWithClient_Cancelled(t *testing.T) {
	app := newTestApp()
	app.Flags.Yes = false
	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())

	client := &fakeBulkMoveClient{
		mailboxes: []jmap.Mailbox{{ID: "archive-1", Name: "Archive", Role: "archive"}},
	}

	stdin := os.Stdin
	defer func() { os.Stdin = stdin }()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	os.Stdin = r
	_, _ = w.WriteString("n\n")
	_ = w.Close()

	stderr := captureStderr(t, func() {
		err := runEmailBulkMoveWithClient(cmd, app, client, []string{"id1"}, "Archive", 50)
		if err != nil {
			t.Fatalf("runEmailBulkMoveWithClient unexpected error: %v", err)
		}
	})

	if !strings.Contains(stderr, "Cancelled") {
		t.Fatalf("expected cancel message, got: %q", stderr)
	}
	if len(client.moveCalls) != 0 {
		t.Fatalf("expected no MoveEmails calls on cancel, got %d", len(client.moveCalls))
	}
}

func TestEmailBulkMoveCmd_RequiresArgs(t *testing.T) {
	app := newTestApp()
	cmd := newEmailBulkMoveCmd(app)

	// Set args to empty (no email IDs provided)
	cmd.SetArgs([]string{})

	// Execute should fail because bulk-move requires at least 1 email ID
	err := cmd.Execute()
	if err == nil {
		t.Error("expected error when no email IDs provided, got nil")
	}

	// Verify the error is related to args validation
	expectedErrPattern := "requires at least 1 arg"
	if err != nil && !strings.Contains(err.Error(), expectedErrPattern) {
		t.Errorf("expected error containing %q, got: %v", expectedErrPattern, err)
	}
}

func TestEmailBulkMoveCmd_AcceptsMultipleArgs(t *testing.T) {
	app := newTestApp()
	cmd := newEmailBulkMoveCmd(app)

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

func TestEmailBulkMoveCmd_AcceptsInputFlagsWithoutPositionalArgs(t *testing.T) {
	app := newTestApp()

	cmdStdin := newEmailBulkMoveCmd(app)
	if err := cmdStdin.Flags().Set("stdin", "true"); err != nil {
		t.Fatalf("set --stdin: %v", err)
	}
	if err := cmdStdin.Args(cmdStdin, []string{}); err != nil {
		t.Fatalf("expected --stdin to satisfy args requirement, got: %v", err)
	}

	cmdFile := newEmailBulkMoveCmd(app)
	if err := cmdFile.Flags().Set("ids-file", "/tmp/fm-ids.txt"); err != nil {
		t.Fatalf("set --ids-file: %v", err)
	}
	if err := cmdFile.Args(cmdFile, []string{}); err != nil {
		t.Fatalf("expected --ids-file to satisfy args requirement, got: %v", err)
	}
}

func TestEmailBulkMoveCmd_HasRequiredFlags(t *testing.T) {
	app := newTestApp()
	root := NewRootCmd(app)
	var emailCmd *cobra.Command
	for _, c := range root.Commands() {
		if c.Name() == "email" {
			emailCmd = c
			break
		}
	}
	if emailCmd == nil {
		t.Fatal("expected email command to exist on root")
	}

	var cmd *cobra.Command
	for _, c := range emailCmd.Commands() {
		if c.Name() == "bulk-move" {
			cmd = c
			break
		}
	}
	if cmd == nil {
		t.Fatal("expected bulk-move command to exist under email")
	}

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

	// Verify bulk input flags exist
	if cmd.Flags().Lookup("stdin") == nil {
		t.Error("expected --stdin flag to exist")
	}
	if cmd.Flags().Lookup("ids-file") == nil {
		t.Error("expected --ids-file flag to exist")
	}
	if cmd.Flags().Lookup("batch-size") == nil {
		t.Error("expected --batch-size flag to exist")
	}

	// Verify --yes flag exists (inherited from root persistent flags)
	yesFlag := cmd.InheritedFlags().Lookup("yes")
	if yesFlag == nil {
		t.Error("expected --yes flag to exist")
	}

	// Verify -y shorthand exists
	yShortFlag := cmd.InheritedFlags().ShorthandLookup("y")
	if yShortFlag == nil {
		t.Error("expected -y shorthand flag to exist")
	}
}

func TestEmailBulkMoveCmd_CommandMetadata(t *testing.T) {
	app := newTestApp()
	cmd := newEmailBulkMoveCmd(app)

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
	app := newTestApp()
	emailCmd := newEmailCmd(app)

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

func TestEmailBulkArchiveCmd_RequiresArgs(t *testing.T) {
	app := newTestApp()
	cmd := newEmailBulkArchiveCmd(app)

	// Set args to empty (no email IDs provided)
	cmd.SetArgs([]string{})

	// Execute should fail because bulk-archive requires at least 1 email ID
	err := cmd.Execute()
	if err == nil {
		t.Error("expected error when no email IDs provided, got nil")
	}

	// Verify the error is related to args validation
	expectedErrPattern := "requires at least 1 arg"
	if err != nil && !strings.Contains(err.Error(), expectedErrPattern) {
		t.Errorf("expected error containing %q, got: %v", expectedErrPattern, err)
	}
}

func TestEmailBulkArchiveCmd_AcceptsMultipleArgs(t *testing.T) {
	app := newTestApp()
	cmd := newEmailBulkArchiveCmd(app)

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

func TestEmailBulkArchiveCmd_AcceptsInputFlagsWithoutPositionalArgs(t *testing.T) {
	app := newTestApp()

	cmdStdin := newEmailBulkArchiveCmd(app)
	if err := cmdStdin.Flags().Set("stdin", "true"); err != nil {
		t.Fatalf("set --stdin: %v", err)
	}
	if err := cmdStdin.Args(cmdStdin, []string{}); err != nil {
		t.Fatalf("expected --stdin to satisfy args requirement, got: %v", err)
	}

	cmdFile := newEmailBulkArchiveCmd(app)
	if err := cmdFile.Flags().Set("ids-file", "/tmp/fm-ids.txt"); err != nil {
		t.Fatalf("set --ids-file: %v", err)
	}
	if err := cmdFile.Args(cmdFile, []string{}); err != nil {
		t.Fatalf("expected --ids-file to satisfy args requirement, got: %v", err)
	}
}

func TestEmailBulkArchiveCmd_HasRequiredFlags(t *testing.T) {
	app := newTestApp()
	root := NewRootCmd(app)
	var emailCmd *cobra.Command
	for _, c := range root.Commands() {
		if c.Name() == "email" {
			emailCmd = c
			break
		}
	}
	if emailCmd == nil {
		t.Fatal("expected email command to exist on root")
	}

	var cmd *cobra.Command
	for _, c := range emailCmd.Commands() {
		if c.Name() == "bulk-archive" {
			cmd = c
			break
		}
	}
	if cmd == nil {
		t.Fatal("expected bulk-archive command to exist under email")
	}

	// Verify --dry-run flag exists
	dryRunFlag := cmd.Flags().Lookup("dry-run")
	if dryRunFlag == nil {
		t.Error("expected --dry-run flag to exist")
	}

	// Verify bulk input flags exist
	if cmd.Flags().Lookup("stdin") == nil {
		t.Error("expected --stdin flag to exist")
	}
	if cmd.Flags().Lookup("ids-file") == nil {
		t.Error("expected --ids-file flag to exist")
	}
	if cmd.Flags().Lookup("batch-size") == nil {
		t.Error("expected --batch-size flag to exist")
	}

	// Verify --yes flag exists (inherited from root persistent flags)
	yesFlag := cmd.InheritedFlags().Lookup("yes")
	if yesFlag == nil {
		t.Error("expected --yes flag to exist")
	}
}

func TestEmailBulkArchiveCmd_CommandMetadata(t *testing.T) {
	app := newTestApp()
	cmd := newEmailBulkArchiveCmd(app)

	if cmd.Use != "bulk-archive <emailId>..." {
		t.Errorf("expected Use to be 'bulk-archive <emailId>...', got: %q", cmd.Use)
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

func TestEmailCmd_HasBulkArchiveSubcommand(t *testing.T) {
	app := newTestApp()
	emailCmd := newEmailCmd(app)

	// Find bulk-archive subcommand
	var found bool
	for _, cmd := range emailCmd.Commands() {
		if cmd.Name() == "bulk-archive" {
			found = true
			break
		}
	}

	if !found {
		t.Error("expected 'bulk-archive' to be registered as a subcommand of 'email'")
	}
}

func TestEmailBulkMarkReadCmd_RequiresArgs(t *testing.T) {
	app := newTestApp()
	cmd := newEmailBulkMarkReadCmd(app)

	// Set args to empty (no email IDs provided)
	cmd.SetArgs([]string{})

	// Execute should fail because bulk-mark-read requires at least 1 email ID
	err := cmd.Execute()
	if err == nil {
		t.Error("expected error when no email IDs provided, got nil")
	}

	// Verify the error is related to args validation
	expectedErrPattern := "requires at least 1 arg"
	if err != nil && !strings.Contains(err.Error(), expectedErrPattern) {
		t.Errorf("expected error containing %q, got: %v", expectedErrPattern, err)
	}
}

func TestEmailBulkMarkReadCmd_AcceptsMultipleArgs(t *testing.T) {
	app := newTestApp()
	cmd := newEmailBulkMarkReadCmd(app)

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

func TestEmailBulkMarkReadCmd_AcceptsInputFlagsWithoutPositionalArgs(t *testing.T) {
	app := newTestApp()

	cmdStdin := newEmailBulkMarkReadCmd(app)
	if err := cmdStdin.Flags().Set("stdin", "true"); err != nil {
		t.Fatalf("set --stdin: %v", err)
	}
	if err := cmdStdin.Args(cmdStdin, []string{}); err != nil {
		t.Fatalf("expected --stdin to satisfy args requirement, got: %v", err)
	}

	cmdFile := newEmailBulkMarkReadCmd(app)
	if err := cmdFile.Flags().Set("ids-file", "/tmp/fm-ids.txt"); err != nil {
		t.Fatalf("set --ids-file: %v", err)
	}
	if err := cmdFile.Args(cmdFile, []string{}); err != nil {
		t.Fatalf("expected --ids-file to satisfy args requirement, got: %v", err)
	}
}

func TestEmailBulkMarkReadCmd_HasRequiredFlags(t *testing.T) {
	app := newTestApp()
	root := NewRootCmd(app)
	var emailCmd *cobra.Command
	for _, c := range root.Commands() {
		if c.Name() == "email" {
			emailCmd = c
			break
		}
	}
	if emailCmd == nil {
		t.Fatal("expected email command to exist on root")
	}

	var cmd *cobra.Command
	for _, c := range emailCmd.Commands() {
		if c.Name() == "bulk-mark-read" {
			cmd = c
			break
		}
	}
	if cmd == nil {
		t.Fatal("expected bulk-mark-read command to exist under email")
	}

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

	// Verify bulk input flags exist
	if cmd.Flags().Lookup("stdin") == nil {
		t.Error("expected --stdin flag to exist")
	}
	if cmd.Flags().Lookup("ids-file") == nil {
		t.Error("expected --ids-file flag to exist")
	}
	if cmd.Flags().Lookup("batch-size") == nil {
		t.Error("expected --batch-size flag to exist")
	}

	// Verify inherited --yes exists
	if cmd.InheritedFlags().Lookup("yes") == nil {
		t.Error("expected --yes flag to exist")
	}
}

func TestEmailBulkMarkReadCmd_CommandMetadata(t *testing.T) {
	app := newTestApp()
	cmd := newEmailBulkMarkReadCmd(app)

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
	app := newTestApp()
	emailCmd := newEmailCmd(app)

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
