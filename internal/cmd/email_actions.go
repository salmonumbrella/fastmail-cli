package cmd

import (
	"context"
	"fmt"
	"strings"

	cerrors "github.com/salmonumbrella/fastmail-cli/internal/errors"
	"github.com/salmonumbrella/fastmail-cli/internal/jmap"
	"github.com/spf13/cobra"
)

type mailboxLookupClient interface {
	GetMailboxes(ctx context.Context) ([]jmap.Mailbox, error)
}

type bulkMoveClient interface {
	mailboxLookupClient
	MoveEmails(ctx context.Context, ids []string, targetMailboxID string) (*jmap.BulkResult, error)
}

func newEmailDeleteCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "delete <emailId>",
		Aliases: []string{"rm", "trash"},
		Short:   "Delete email (move to trash)",
		Args:    cobra.ExactArgs(1),
		RunE: runE(app, func(cmd *cobra.Command, args []string, app *App) error {
			client, err := app.JMAPClient()
			if err != nil {
				return err
			}

			err = client.DeleteEmail(cmd.Context(), args[0])
			if err != nil {
				return cerrors.WithContext(err, "deleting email")
			}

			if app.IsJSON(cmd.Context()) {
				return app.PrintJSON(cmd, map[string]any{
					"status":  "deleted",
					"deleted": args[0],
				})
			}

			fmt.Printf("Email %s moved to trash\n", args[0])
			return nil
		}),
	}

	return cmd
}

func newEmailBulkDeleteCmd(app *App) *cobra.Command {
	var dryRun bool
	var input bulkInputOptions

	cmd := &cobra.Command{
		Use:     "bulk-delete <emailId>...",
		Aliases: []string{"bulk-rm", "rm-many"},
		Short:   "Delete multiple emails (move to trash)",
		Example: `  fastmail email bulk-delete ID1 ID2
  fastmail email bulk-delete --ids-file /tmp/fm-ids.txt --yes
  fastmail email bulk-delete --stdin --yes < /tmp/fm-ids.txt`,
		Args: validateBulkInputArgs,
		RunE: runE(app, func(cmd *cobra.Command, args []string, app *App) error {
			ids, err := collectBulkIDs(args, input)
			if err != nil {
				return err
			}

			// Handle dry-run mode
			if dryRun {
				return printDryRunList(app, cmd, fmt.Sprintf("Would delete %d emails:", len(ids)), "wouldDelete", ids, map[string]any{
					"batchSize": input.BatchSize,
				})
			}

			client, err := app.JMAPClient()
			if err != nil {
				return err
			}

			// Prompt for confirmation unless --yes flag is set (global) or JSON output mode.
			confirmed, err := app.Confirm(cmd, false, fmt.Sprintf("Delete %d emails? [y/N] ", len(ids)), "y", "yes")
			if err != nil {
				return err
			}
			if !confirmed {
				printCancelled()
				return nil
			}

			// Delete emails using bulk API in client-side batches.
			results, batches, err := runBulkInBatches(ids, input.BatchSize, "deleting emails", func(batch []string) (*jmap.BulkResult, error) {
				return client.DeleteEmails(cmd.Context(), batch)
			})
			if err != nil {
				return cerrors.WithContext(err, "deleting emails")
			}

			// Handle JSON output
			if app.IsJSON(cmd.Context()) {
				output := map[string]any{
					"status":    "deleted",
					"succeeded": results.Succeeded,
					"batchSize": input.BatchSize,
					"batches":   batches,
				}
				if len(results.Failed) > 0 {
					output["failed"] = results.Failed
				}
				return app.PrintJSON(cmd, output)
			}

			if batches > 1 {
				fmt.Printf("Processed %d emails in %d batches (batch size %d)\n", len(ids), batches, input.BatchSize)
			}

			// Handle text output
			succeededCount := len(results.Succeeded)
			failedCount := len(results.Failed)
			printBulkResults("Deleted", "emails", succeededCount, failedCount, results.Failed)

			return nil
		}),
	}

	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show what would be deleted without making changes")
	addBulkInputFlags(cmd, &input)

	return cmd
}

func newEmailMoveCmd(app *App) *cobra.Command {
	var targetMailbox string

	cmd := &cobra.Command{
		Use:     "move <emailId>",
		Aliases: []string{"mv"},
		Short:   "Move email to mailbox",
		Args:    cobra.ExactArgs(1),
		RunE: runE(app, func(cmd *cobra.Command, args []string, app *App) error {
			client, err := app.JMAPClient()
			if err != nil {
				return err
			}

			// Resolve target mailbox ID + display name in one mailbox fetch.
			resolvedID, mailboxName, err := resolveMailboxTarget(cmd.Context(), client, targetMailbox)
			if err != nil {
				return err
			}

			err = client.MoveEmail(cmd.Context(), args[0], resolvedID)
			if err != nil {
				return cerrors.WithContext(err, "moving email")
			}

			if app.IsJSON(cmd.Context()) {
				return app.PrintJSON(cmd, map[string]any{
					"status":    "moved",
					"moved":     args[0],
					"mailbox":   mailboxName,
					"mailboxId": resolvedID,
				})
			}

			fmt.Printf("Email %s moved to mailbox %s\n", args[0], mailboxName)
			return nil
		}),
	}

	cmd.Flags().StringVar(&targetMailbox, "to", "", "Target mailbox ID or name")

	return cmd
}

func newEmailBulkMoveCmd(app *App) *cobra.Command {
	var targetMailbox string
	var dryRun bool
	var input bulkInputOptions

	cmd := &cobra.Command{
		Use:     "bulk-move <emailId>...",
		Aliases: []string{"bulk-mv"},
		Short:   "Move multiple emails to a mailbox",
		Example: `  fastmail email bulk-move --to Archive ID1 ID2
  fastmail email bulk-move --ids-file /tmp/fm-ids.txt --to Archive --yes
  fastmail email bulk-move --stdin --to Archive --yes < /tmp/fm-ids.txt`,
		Args: validateBulkInputArgs,
		RunE: runE(app, func(cmd *cobra.Command, args []string, app *App) error {
			return runEmailBulkMove(cmd, args, app, targetMailbox, dryRun, input)
		}),
	}

	cmd.Flags().StringVar(&targetMailbox, "to", "", "Target mailbox ID or name")
	cmd.Flags().StringVar(&targetMailbox, "mailbox", "", "Target mailbox ID or name (alias for --to)")
	_ = cmd.Flags().MarkHidden("mailbox") // Hidden alias for agent compatibility
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show what would be moved without making changes")
	addBulkInputFlags(cmd, &input)

	return cmd
}

func newEmailBulkArchiveCmd(app *App) *cobra.Command {
	var dryRun bool
	var input bulkInputOptions

	cmd := &cobra.Command{
		Use:     "bulk-archive <emailId>...",
		Aliases: []string{"archive-many", "bulk-arch"},
		Short:   "Archive multiple emails",
		Example: `  fastmail email bulk-archive ID1 ID2
  fastmail email bulk-archive --ids-file /tmp/fm-ids.txt --yes
  fastmail email bulk-archive --stdin --yes < /tmp/fm-ids.txt`,
		Args: validateBulkInputArgs,
		RunE: runE(app, func(cmd *cobra.Command, args []string, app *App) error {
			return runEmailBulkMove(cmd, args, app, "Archive", dryRun, input)
		}),
	}

	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show what would be moved without making changes")
	addBulkInputFlags(cmd, &input)

	return cmd
}

func runEmailBulkMove(cmd *cobra.Command, args []string, app *App, targetMailbox string, dryRun bool, input bulkInputOptions) error {
	// Validate required flags before accessing keyring
	if targetMailbox == "" {
		return fmt.Errorf("--to is required")
	}

	ids, err := collectBulkIDs(args, input)
	if err != nil {
		return err
	}

	// Handle dry-run mode without requiring keyring / network.
	if dryRun {
		return printDryRunList(app, cmd, fmt.Sprintf("Would move %d emails to %s:", len(ids), targetMailbox), "wouldMove", ids, map[string]any{
			"mailbox":   targetMailbox,
			"batchSize": input.BatchSize,
		})
	}

	client, err := app.JMAPClient()
	if err != nil {
		return err
	}

	return runEmailBulkMoveWithClient(cmd, app, client, ids, targetMailbox, input.BatchSize)
}

func runEmailBulkMoveWithClient(cmd *cobra.Command, app *App, client bulkMoveClient, ids []string, targetMailbox string, batchSize int) error {
	// Resolve target mailbox ID + display name in one mailbox fetch.
	resolvedID, mailboxName, err := resolveMailboxTarget(cmd.Context(), client, targetMailbox)
	if err != nil {
		return err
	}

	// Prompt for confirmation unless --yes flag is set (global) or JSON output mode.
	confirmed, err := app.Confirm(cmd, false, fmt.Sprintf("Move %d emails to %s? [y/N] ", len(ids), mailboxName), "y", "yes")
	if err != nil {
		return err
	}
	if !confirmed {
		printCancelled()
		return nil
	}

	// Move emails using bulk API in client-side batches.
	results, batches, err := runBulkInBatches(ids, batchSize, "moving emails", func(batch []string) (*jmap.BulkResult, error) {
		return client.MoveEmails(cmd.Context(), batch, resolvedID)
	})
	if err != nil {
		return cerrors.WithContext(err, "moving emails")
	}

	// Handle JSON output
	if app.IsJSON(cmd.Context()) {
		output := map[string]any{
			"status":    "moved",
			"mailbox":   mailboxName,
			"mailboxId": resolvedID,
			"succeeded": results.Succeeded,
			"batchSize": batchSize,
			"batches":   batches,
		}
		if len(results.Failed) > 0 {
			output["failed"] = results.Failed
		}
		return app.PrintJSON(cmd, output)
	}

	if batches > 1 {
		fmt.Printf("Processed %d emails in %d batches (batch size %d)\n", len(ids), batches, batchSize)
	}

	// Handle text output
	succeededCount := len(results.Succeeded)
	failedCount := len(results.Failed)
	printBulkResults("Moved", fmt.Sprintf("emails to %s", mailboxName), succeededCount, failedCount, results.Failed)

	return nil
}

func resolveMailboxTarget(ctx context.Context, client mailboxLookupClient, targetMailbox string) (string, string, error) {
	targetMailbox = strings.TrimSpace(targetMailbox)
	if targetMailbox == "" {
		return "", "", fmt.Errorf("--to is required")
	}

	mailboxes, err := client.GetMailboxes(ctx)
	if err != nil {
		return "", "", fmt.Errorf("failed to get mailboxes: %w", err)
	}

	targetLower := strings.ToLower(targetMailbox)
	for _, mb := range mailboxes {
		if strings.ToLower(mb.Name) == targetLower || strings.ToLower(mb.Role) == targetLower {
			if mb.Name == "" {
				return mb.ID, mb.ID, nil
			}
			return mb.ID, mb.Name, nil
		}
	}

	for _, mb := range mailboxes {
		if mb.ID == targetMailbox {
			if mb.Name == "" {
				return mb.ID, mb.ID, nil
			}
			return mb.ID, mb.Name, nil
		}
	}

	return "", "", fmt.Errorf("invalid target mailbox: %w: %s", jmap.ErrMailboxNotFound, targetMailbox)
}

func newEmailMarkReadCmd(app *App) *cobra.Command {
	var unread bool

	cmd := &cobra.Command{
		Use:     "mark-read <emailId>",
		Aliases: []string{"read", "seen", "mark-seen"},
		Short:   "Mark email as read/unread",
		Args:    cobra.ExactArgs(1),
		RunE: runE(app, func(cmd *cobra.Command, args []string, app *App) error {
			client, err := app.JMAPClient()
			if err != nil {
				return err
			}

			err = client.MarkEmailRead(cmd.Context(), args[0], !unread)
			if err != nil {
				return fmt.Errorf("failed to update email: %w", err)
			}

			status := "read"
			if unread {
				status = "unread"
			}

			if app.IsJSON(cmd.Context()) {
				return app.PrintJSON(cmd, map[string]any{
					"emailId": args[0],
					"status":  status,
				})
			}

			fmt.Printf("Email %s marked as %s\n", args[0], status)
			return nil
		}),
	}

	cmd.Flags().BoolVar(&unread, "unread", false, "Mark as unread instead of read")

	return cmd
}

func newEmailBulkMarkReadCmd(app *App) *cobra.Command {
	var unread bool
	var dryRun bool
	var input bulkInputOptions

	cmd := &cobra.Command{
		Use:     "bulk-mark-read <emailId>...",
		Aliases: []string{"bulk-read", "bulk-seen"},
		Short:   "Mark multiple emails as read/unread",
		Example: `  fastmail email bulk-mark-read ID1 ID2
  fastmail email bulk-mark-read --ids-file /tmp/fm-ids.txt --yes
  fastmail email bulk-mark-read --stdin --unread --yes < /tmp/fm-ids.txt`,
		Args: validateBulkInputArgs,
		RunE: runE(app, func(cmd *cobra.Command, args []string, app *App) error {
			ids, err := collectBulkIDs(args, input)
			if err != nil {
				return err
			}

			status := "read"
			if unread {
				status = "unread"
			}

			// Handle dry-run mode
			if dryRun {
				return printDryRunList(app, cmd, fmt.Sprintf("Would mark %d emails as %s:", len(ids), status), "wouldMark", ids, map[string]any{
					"status":    status,
					"batchSize": input.BatchSize,
				})
			}

			client, err := app.JMAPClient()
			if err != nil {
				return err
			}

			// Mark emails using bulk API in client-side batches.
			results, batches, err := runBulkInBatches(ids, input.BatchSize, "marking emails", func(batch []string) (*jmap.BulkResult, error) {
				return client.MarkEmailsRead(cmd.Context(), batch, !unread)
			})
			if err != nil {
				return cerrors.WithContext(err, "marking emails")
			}

			// Handle JSON output
			if app.IsJSON(cmd.Context()) {
				output := map[string]any{
					"status":    status,
					"succeeded": results.Succeeded,
					"batchSize": input.BatchSize,
					"batches":   batches,
				}
				if len(results.Failed) > 0 {
					output["failed"] = results.Failed
				}
				return app.PrintJSON(cmd, output)
			}

			if batches > 1 {
				fmt.Printf("Processed %d emails in %d batches (batch size %d)\n", len(ids), batches, input.BatchSize)
			}

			// Handle text output
			succeededCount := len(results.Succeeded)
			failedCount := len(results.Failed)
			printBulkResults("Marked", fmt.Sprintf("emails as %s", status), succeededCount, failedCount, results.Failed)

			return nil
		}),
	}

	cmd.Flags().BoolVar(&unread, "unread", false, "Mark as unread instead of read")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show what would be changed without making changes")
	addBulkInputFlags(cmd, &input)

	return cmd
}
