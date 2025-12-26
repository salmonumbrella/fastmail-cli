package cmd

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"text/tabwriter"
	"time"

	cerrors "github.com/salmonumbrella/fastmail-cli/internal/errors"
	"github.com/salmonumbrella/fastmail-cli/internal/jmap"
	"github.com/salmonumbrella/fastmail-cli/internal/outfmt"
	"github.com/spf13/cobra"
)

// emailRegex implements RFC 5322 email validation.
// This pattern validates the general structure of email addresses while being
// permissive enough for real-world usage but strict enough to reject obvious attacks.
var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9.!#$%&'*+/=?^_` + "`" + `{|}~-]+@[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(?:\.[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$`)

func newEmailCmd(flags *rootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "email",
		Short: "Email operations",
	}

	cmd.AddCommand(newEmailListCmd(flags))
	cmd.AddCommand(newEmailSearchCmd(flags))
	cmd.AddCommand(newEmailGetCmd(flags))
	cmd.AddCommand(newEmailSendCmd(flags))
	cmd.AddCommand(newEmailDeleteCmd(flags))
	cmd.AddCommand(newEmailBulkDeleteCmd(flags))
	cmd.AddCommand(newEmailMoveCmd(flags))
	cmd.AddCommand(newEmailBulkMoveCmd(flags))
	cmd.AddCommand(newEmailMarkReadCmd(flags))
	cmd.AddCommand(newEmailBulkMarkReadCmd(flags))
	cmd.AddCommand(newEmailThreadCmd(flags))
	cmd.AddCommand(newEmailAttachmentsCmd(flags))
	cmd.AddCommand(newEmailDownloadCmd(flags))
	cmd.AddCommand(newEmailMailboxesCmd(flags))
	cmd.AddCommand(newMailboxCreateCmd(flags))
	cmd.AddCommand(newMailboxDeleteCmd(flags))
	cmd.AddCommand(newMailboxRenameCmd(flags))
	cmd.AddCommand(newEmailImportCmd(flags))
	cmd.AddCommand(newEmailIdentitiesCmd(flags))

	return cmd
}

func newEmailListCmd(flags *rootFlags) *cobra.Command {
	var limit int
	var mailboxID string

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List emails",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := getClient(flags)
			if err != nil {
				return err
			}

			// Resolve mailbox ID or name
			if mailboxID != "" {
				var resolvedID string
				resolvedID, err = client.ResolveMailboxID(cmd.Context(), mailboxID)
				if err != nil {
					return fmt.Errorf("invalid mailbox: %w", err)
				}
				mailboxID = resolvedID
			}

			emails, err := client.GetEmails(cmd.Context(), mailboxID, limit)
			if err != nil {
				listErr := cerrors.WithContext(err, "listing emails")
				if strings.Contains(err.Error(), "401") || strings.Contains(err.Error(), "unauthorized") {
					return cerrors.WithSuggestion(listErr, cerrors.SuggestionReauth)
				}
				return listErr
			}

			if isJSON(cmd.Context()) {
				return printJSON(cmd, emails)
			}

			if len(emails) == 0 {
				outfmt.Errorf("No emails found")
				return nil
			}

			tw := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
			fmt.Fprintln(tw, "ID\tSUBJECT\tFROM\tDATE\tUNREAD")
			for _, email := range emails {
				from := formatEmailAddressList(email.From)
				date := formatEmailDate(email.ReceivedAt)
				unread := ""
				if email.Keywords != nil && !email.Keywords["$seen"] {
					unread = "*"
				}
				fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\n",
					email.ID,
					sanitizeTab(truncateString(email.Subject, 50)),
					sanitizeTab(truncateString(from, 30)),
					date,
					unread,
				)
			}
			tw.Flush()

			return nil
		},
	}

	cmd.Flags().IntVar(&limit, "limit", 25, "Maximum number of emails to list")
	cmd.Flags().StringVar(&mailboxID, "mailbox", "", "Mailbox ID or name to filter emails")

	return cmd
}

func newEmailSearchCmd(flags *rootFlags) *cobra.Command {
	var limit int
	var snippets bool

	cmd := &cobra.Command{
		Use:   "search <query>",
		Short: "Search emails",
		Long: `Search emails using JMAP query syntax.

Examples:
  fastmail email search "from:alice@example.com"
  fastmail email search --snippets "invoice"
  fastmail email search "subject:meeting after:2025-01-01"`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := getClient(flags)
			if err != nil {
				return err
			}

			var emails []jmap.Email
			var searchSnippets []jmap.SearchSnippet

			if snippets {
				emails, searchSnippets, err = client.SearchEmailsWithSnippets(cmd.Context(), args[0], limit)
			} else {
				emails, err = client.SearchEmails(cmd.Context(), args[0], limit)
			}

			if err != nil {
				searchErr := cerrors.WithContext(err, "searching emails")
				if strings.Contains(err.Error(), "401") || strings.Contains(err.Error(), "unauthorized") {
					return cerrors.WithSuggestion(searchErr, cerrors.SuggestionReauth)
				}
				return searchErr
			}

			if isJSON(cmd.Context()) {
				result := map[string]any{"emails": emails}
				if snippets && len(searchSnippets) > 0 {
					result["snippets"] = searchSnippets
				}
				return printJSON(cmd, result)
			}

			if len(emails) == 0 {
				outfmt.Errorf("No emails found matching '%s'", args[0])
				return nil
			}

			// Build snippet map for quick lookup
			snippetMap := make(map[string]jmap.SearchSnippet)
			for _, s := range searchSnippets {
				snippetMap[s.EmailID] = s
			}

			tw := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
			fmt.Fprintln(tw, "ID\tSUBJECT\tFROM\tDATE\tUNREAD")
			for _, email := range emails {
				from := formatEmailAddressList(email.From)
				date := formatEmailDate(email.ReceivedAt)
				unread := ""
				if email.Keywords != nil && !email.Keywords["$seen"] {
					unread = "*"
				}

				subject := email.Subject
				if snippets {
					if s, ok := snippetMap[email.ID]; ok && s.Subject != "" {
						subject = s.Subject // Use highlighted subject
					}
				}

				fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\n",
					email.ID,
					sanitizeTab(truncateString(subject, 50)),
					sanitizeTab(truncateString(from, 30)),
					date,
					unread,
				)

				// Show snippet preview if available
				if snippets {
					if s, ok := snippetMap[email.ID]; ok && s.Preview != "" {
						fmt.Fprintf(tw, "\t%s\t\t\t\n", sanitizeTab(truncateString(s.Preview, 80)))
					}
				}
			}
			tw.Flush()

			return nil
		},
	}

	cmd.Flags().IntVar(&limit, "limit", 25, "Maximum number of results")
	cmd.Flags().BoolVar(&snippets, "snippets", false, "Show highlighted search snippets")

	return cmd
}

func newEmailGetCmd(flags *rootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get <emailId>",
		Short: "Get email by ID",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := getClient(flags)
			if err != nil {
				return err
			}

			email, err := client.GetEmailByID(cmd.Context(), args[0])
			if err != nil {
				getErr := cerrors.WithContext(err, "fetching email")
				if strings.Contains(err.Error(), "401") || strings.Contains(err.Error(), "unauthorized") {
					return cerrors.WithSuggestion(getErr, cerrors.SuggestionReauth)
				}
				return getErr
			}

			if isJSON(cmd.Context()) {
				return printJSON(cmd, email)
			}

			// Text output
			fmt.Printf("ID:        %s\n", email.ID)
			fmt.Printf("Subject:   %s\n", email.Subject)
			fmt.Printf("From:      %s\n", formatEmailAddressList(email.From))
			fmt.Printf("To:        %s\n", formatEmailAddressList(email.To))
			if len(email.CC) > 0 {
				fmt.Printf("CC:        %s\n", formatEmailAddressList(email.CC))
			}
			fmt.Printf("Date:      %s\n", email.ReceivedAt)
			fmt.Printf("Thread ID: %s\n", email.ThreadID)
			fmt.Printf("Attachments: %d\n", len(email.Attachments))
			fmt.Println()

			// Print body
			if len(email.TextBody) > 0 && len(email.BodyValues) > 0 {
				for _, part := range email.TextBody {
					if body, ok := email.BodyValues[part.PartID]; ok {
						fmt.Println(body.Value)
					}
				}
			} else if email.Preview != "" {
				fmt.Println(email.Preview)
			}

			return nil
		},
	}

	return cmd
}

func newEmailSendCmd(flags *rootFlags) *cobra.Command {
	var to, cc, bcc []string
	var subject, body, htmlBody string
	var draft bool
	var replyTo string
	var attachments []string
	var fromIdentity string

	cmd := &cobra.Command{
		Use:   "send",
		Short: "Send an email",
		Long: `Send an email with optional attachments.

Examples:
  fastmail email send --to user@example.com --subject "Hello" --body "Hi there"
  fastmail email send --to user@example.com --subject "Report" --body "See attached" --attach report.pdf
  fastmail email send --to user@example.com --subject "Q4 Results" --attach /docs/q4.pdf:Q4-Report.pdf`,
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := getClient(flags)
			if err != nil {
				return err
			}

			// For drafts with --reply-to, --to and --subject are optional (auto-filled)
			if !draft && replyTo == "" && len(to) == 0 {
				return fmt.Errorf("--to is required (or use --draft to save without sending)")
			}
			if replyTo == "" && subject == "" {
				return fmt.Errorf("--subject is required")
			}
			if body == "" && htmlBody == "" {
				return fmt.Errorf("--body or --html is required")
			}

			// Validate email addresses (only those provided)
			allAddrs := append(append(to, cc...), bcc...)
			for _, addr := range allAddrs {
				if !isValidEmail(addr) {
					return fmt.Errorf("invalid email address: %s", addr)
				}
			}

			// Process attachments
			var attachmentOpts []jmap.AttachmentOpts
			for _, att := range attachments {
				var attPath, attName string
				attPath, attName, err = parseAttachmentFlag(att)
				if err != nil {
					return fmt.Errorf("invalid attachment: %w", err)
				}

				// Verify file exists and get size
				var fileInfo os.FileInfo
				fileInfo, err = os.Stat(attPath)
				if err != nil {
					return fmt.Errorf("cannot access attachment '%s': %w", attPath, err)
				}
				if fileInfo.IsDir() {
					return fmt.Errorf("cannot attach directory: %s", attPath)
				}

				// Check file size before upload
				if fileInfo.Size() > jmap.MaxUploadSize {
					return fmt.Errorf("attachment '%s' too large (%s, max 50 MB)", attPath, formatSize(fileInfo.Size()))
				}

				// Open and upload the file
				var file *os.File
				file, err = os.Open(attPath)
				if err != nil {
					return fmt.Errorf("failed to open attachment '%s': %w", attPath, err)
				}

				mimeType := getMimeType(attPath)
				var uploadResult *jmap.UploadBlobResult
				uploadResult, err = client.UploadBlob(cmd.Context(), file, mimeType)
				_ = file.Close()
				if err != nil {
					return fmt.Errorf("failed to upload attachment '%s': %w", attPath, err)
				}

				attachmentOpts = append(attachmentOpts, jmap.AttachmentOpts{
					BlobID: uploadResult.BlobID,
					Name:   attName,
					Type:   mimeType,
				})
			}

			opts := jmap.SendEmailOpts{
				To:          to,
				CC:          cc,
				BCC:         bcc,
				Subject:     subject,
				TextBody:    body,
				HTMLBody:    htmlBody,
				From:        fromIdentity,
				Attachments: attachmentOpts,
			}

			if draft {
				var draftID string

				if replyTo != "" {
					// Create a threaded reply draft
					draftID, err = client.CreateReplyDraft(cmd.Context(), replyTo, opts)
				} else {
					// Create a standalone draft
					draftID, err = client.SaveDraft(cmd.Context(), opts)
				}

				if err != nil {
					return fmt.Errorf("failed to save draft: %w", err)
				}

				if isJSON(cmd.Context()) {
					return printJSON(cmd, map[string]any{
						"draftId": draftID,
						"status":  "draft",
						"replyTo": replyTo,
					})
				}

				fmt.Printf("Draft saved (ID: %s)\n", draftID)
				return nil
			}

			// Send the email
			submissionID, err := client.SendEmail(cmd.Context(), opts)
			if err != nil {
				sendErr := cerrors.WithContext(err, "sending email")
				if strings.Contains(err.Error(), "401") || strings.Contains(err.Error(), "unauthorized") {
					return cerrors.WithSuggestion(sendErr, cerrors.SuggestionReauth)
				}
				return sendErr
			}

			if isJSON(cmd.Context()) {
				return printJSON(cmd, map[string]any{
					"submissionId": submissionID,
					"status":       "sent",
				})
			}

			fmt.Printf("Email sent successfully (submission ID: %s)\n", submissionID)
			return nil
		},
	}

	cmd.Flags().StringSliceVar(&to, "to", nil, "Recipient email addresses")
	cmd.Flags().StringSliceVar(&cc, "cc", nil, "CC email addresses")
	cmd.Flags().StringSliceVar(&bcc, "bcc", nil, "BCC email addresses")
	cmd.Flags().StringVar(&subject, "subject", "", "Email subject")
	cmd.Flags().StringVar(&body, "body", "", "Email body (plain text)")
	cmd.Flags().StringVar(&htmlBody, "html", "", "Email body (HTML)")
	cmd.Flags().StringVar(&fromIdentity, "from", "", "Send from this identity email (see: fastmail email identities)")
	cmd.Flags().BoolVar(&draft, "draft", false, "Save as draft instead of sending")
	cmd.Flags().StringVar(&replyTo, "reply-to", "", "Email ID to reply to (threads the draft)")
	cmd.Flags().StringSliceVar(&attachments, "attach", nil, "Attach files (path or path:name)")

	return cmd
}

func newEmailDeleteCmd(flags *rootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete <emailId>",
		Short: "Delete email (move to trash)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := getClient(flags)
			if err != nil {
				return err
			}

			err = client.DeleteEmail(cmd.Context(), args[0])
			if err != nil {
				return cerrors.WithContext(err, "deleting email")
			}

			if isJSON(cmd.Context()) {
				return printJSON(cmd, map[string]any{
					"deleted": args[0],
				})
			}

			fmt.Printf("Email %s moved to trash\n", args[0])
			return nil
		},
	}

	return cmd
}

func newEmailBulkDeleteCmd(flags *rootFlags) *cobra.Command {
	var dryRun bool
	var yesFlag bool

	cmd := &cobra.Command{
		Use:   "bulk-delete <emailId>...",
		Short: "Delete multiple emails (move to trash)",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := getClient(flags)
			if err != nil {
				return err
			}

			// Handle dry-run mode
			if dryRun {
				if isJSON(cmd.Context()) {
					return printJSON(cmd, map[string]any{
						"dryRun":      true,
						"wouldDelete": args,
					})
				}

				fmt.Printf("Would delete %d emails:\n", len(args))
				for _, id := range args {
					fmt.Printf("  - %s\n", id)
				}
				return nil
			}

			// Prompt for confirmation unless --yes flag is set or JSON output mode
			if !isJSON(cmd.Context()) && !yesFlag {
				fmt.Fprintf(os.Stderr, "Delete %d emails? [y/N] ", len(args))
				scanner := bufio.NewScanner(os.Stdin)
				if !scanner.Scan() {
					return fmt.Errorf("cancelled")
				}
				response := strings.ToLower(strings.TrimSpace(scanner.Text()))
				if response != "y" && response != "yes" {
					return fmt.Errorf("cancelled")
				}
			}

			// Delete emails using bulk API
			results, err := client.DeleteEmails(cmd.Context(), args)
			if err != nil {
				deleteErr := cerrors.WithContext(err, "deleting emails")
				if strings.Contains(err.Error(), "401") || strings.Contains(err.Error(), "unauthorized") {
					return cerrors.WithSuggestion(deleteErr, cerrors.SuggestionReauth)
				}
				return deleteErr
			}

			// Handle JSON output
			if isJSON(cmd.Context()) {
				output := map[string]any{
					"succeeded": results.Succeeded,
				}
				if len(results.Failed) > 0 {
					output["failed"] = results.Failed
				}
				return printJSON(cmd, output)
			}

			// Handle text output
			succeededCount := len(results.Succeeded)
			failedCount := len(results.Failed)

			if failedCount == 0 {
				// All succeeded
				fmt.Printf("Deleted %d emails\n", succeededCount)
			} else {
				// Partial failure
				fmt.Printf("Deleted %d emails, %d failed:\n", succeededCount, failedCount)
				for id, errMsg := range results.Failed {
					fmt.Printf("  %s: %s\n", id, errMsg)
				}
			}

			return nil
		},
	}

	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show what would be deleted without making changes")
	cmd.Flags().BoolVarP(&yesFlag, "yes", "y", false, "Skip confirmation prompt")

	return cmd
}

func newEmailMoveCmd(flags *rootFlags) *cobra.Command {
	var targetMailbox string

	cmd := &cobra.Command{
		Use:   "move <emailId>",
		Short: "Move email to mailbox",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := getClient(flags)
			if err != nil {
				return err
			}

			if targetMailbox == "" {
				return fmt.Errorf("--to is required")
			}

			// Resolve target mailbox ID or name
			resolvedID, err := client.ResolveMailboxID(cmd.Context(), targetMailbox)
			if err != nil {
				return fmt.Errorf("invalid target mailbox: %w", err)
			}
			targetMailbox = resolvedID

			err = client.MoveEmail(cmd.Context(), args[0], targetMailbox)
			if err != nil {
				return cerrors.WithContext(err, "moving email")
			}

			if isJSON(cmd.Context()) {
				return printJSON(cmd, map[string]any{
					"moved":   args[0],
					"mailbox": targetMailbox,
				})
			}

			fmt.Printf("Email %s moved to mailbox %s\n", args[0], targetMailbox)
			return nil
		},
	}

	cmd.Flags().StringVar(&targetMailbox, "to", "", "Target mailbox ID or name")

	return cmd
}

func newEmailBulkMoveCmd(flags *rootFlags) *cobra.Command {
	var targetMailbox string
	var dryRun bool
	var yesFlag bool

	cmd := &cobra.Command{
		Use:   "bulk-move <emailId>...",
		Short: "Move multiple emails to a mailbox",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Validate required flags before accessing keyring
			if targetMailbox == "" {
				return fmt.Errorf("--to is required")
			}

			client, err := getClient(flags)
			if err != nil {
				return err
			}

			// Resolve target mailbox ID or name
			resolvedID, err := client.ResolveMailboxID(cmd.Context(), targetMailbox)
			if err != nil {
				return fmt.Errorf("invalid target mailbox: %w", err)
			}

			// Get mailbox name for output
			mailboxes, err := client.GetMailboxes(cmd.Context())
			if err != nil {
				return fmt.Errorf("failed to get mailboxes: %w", err)
			}

			var mailboxName string
			for _, mb := range mailboxes {
				if mb.ID == resolvedID {
					mailboxName = mb.Name
					break
				}
			}
			if mailboxName == "" {
				mailboxName = resolvedID
			}

			// Handle dry-run mode
			if dryRun {
				if isJSON(cmd.Context()) {
					return printJSON(cmd, map[string]any{
						"dryRun":    true,
						"mailbox":   mailboxName,
						"wouldMove": args,
					})
				}

				fmt.Printf("Would move %d emails to %s:\n", len(args), mailboxName)
				for _, id := range args {
					fmt.Printf("  - %s\n", id)
				}
				return nil
			}

			// Prompt for confirmation unless --yes flag is set or JSON output mode
			if !isJSON(cmd.Context()) && !yesFlag {
				fmt.Fprintf(os.Stderr, "Move %d emails to %s? [y/N] ", len(args), mailboxName)
				scanner := bufio.NewScanner(os.Stdin)
				if !scanner.Scan() {
					return fmt.Errorf("cancelled")
				}
				response := strings.ToLower(strings.TrimSpace(scanner.Text()))
				if response != "y" && response != "yes" {
					return fmt.Errorf("cancelled")
				}
			}

			// Move emails using bulk API
			results, err := client.MoveEmails(cmd.Context(), args, resolvedID)
			if err != nil {
				moveErr := cerrors.WithContext(err, "moving emails")
				if strings.Contains(err.Error(), "401") || strings.Contains(err.Error(), "unauthorized") {
					return cerrors.WithSuggestion(moveErr, cerrors.SuggestionReauth)
				}
				return moveErr
			}

			// Handle JSON output
			if isJSON(cmd.Context()) {
				output := map[string]any{
					"mailbox":   mailboxName,
					"succeeded": results.Succeeded,
				}
				if len(results.Failed) > 0 {
					output["failed"] = results.Failed
				}
				return printJSON(cmd, output)
			}

			// Handle text output
			succeededCount := len(results.Succeeded)
			failedCount := len(results.Failed)

			if failedCount == 0 {
				// All succeeded
				fmt.Printf("Moved %d emails to %s\n", succeededCount, mailboxName)
			} else {
				// Partial failure
				fmt.Printf("Moved %d emails to %s, %d failed:\n", succeededCount, mailboxName, failedCount)
				for id, errMsg := range results.Failed {
					fmt.Printf("  %s: %s\n", id, errMsg)
				}
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&targetMailbox, "to", "", "Target mailbox ID or name")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show what would be moved without making changes")
	cmd.Flags().BoolVarP(&yesFlag, "yes", "y", false, "Skip confirmation prompt")

	return cmd
}

func newEmailMarkReadCmd(flags *rootFlags) *cobra.Command {
	var unread bool

	cmd := &cobra.Command{
		Use:   "mark-read <emailId>",
		Short: "Mark email as read/unread",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := getClient(flags)
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

			if isJSON(cmd.Context()) {
				return printJSON(cmd, map[string]any{
					"emailId": args[0],
					"status":  status,
				})
			}

			fmt.Printf("Email %s marked as %s\n", args[0], status)
			return nil
		},
	}

	cmd.Flags().BoolVar(&unread, "unread", false, "Mark as unread instead of read")

	return cmd
}

func newEmailBulkMarkReadCmd(flags *rootFlags) *cobra.Command {
	var unread bool
	var dryRun bool

	cmd := &cobra.Command{
		Use:   "bulk-mark-read <emailId>...",
		Short: "Mark multiple emails as read/unread",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := getClient(flags)
			if err != nil {
				return err
			}

			status := "read"
			if unread {
				status = "unread"
			}

			// Handle dry-run mode
			if dryRun {
				if isJSON(cmd.Context()) {
					return printJSON(cmd, map[string]any{
						"dryRun":    true,
						"status":    status,
						"wouldMark": args,
					})
				}

				fmt.Printf("Would mark %d emails as %s:\n", len(args), status)
				for _, id := range args {
					fmt.Printf("  - %s\n", id)
				}
				return nil
			}

			// Mark emails using bulk API
			results, err := client.MarkEmailsRead(cmd.Context(), args, !unread)
			if err != nil {
				markErr := cerrors.WithContext(err, "marking emails")
				if strings.Contains(err.Error(), "401") || strings.Contains(err.Error(), "unauthorized") {
					return cerrors.WithSuggestion(markErr, cerrors.SuggestionReauth)
				}
				return markErr
			}

			// Handle JSON output
			if isJSON(cmd.Context()) {
				output := map[string]any{
					"status":    status,
					"succeeded": results.Succeeded,
				}
				if len(results.Failed) > 0 {
					output["failed"] = results.Failed
				}
				return printJSON(cmd, output)
			}

			// Handle text output
			succeededCount := len(results.Succeeded)
			failedCount := len(results.Failed)

			if failedCount == 0 {
				// All succeeded
				fmt.Printf("Marked %d emails as %s\n", succeededCount, status)
			} else {
				// Partial failure
				fmt.Printf("Marked %d emails as %s, %d failed:\n", succeededCount, status, failedCount)
				for id, errMsg := range results.Failed {
					fmt.Printf("  %s: %s\n", id, errMsg)
				}
			}

			return nil
		},
	}

	cmd.Flags().BoolVar(&unread, "unread", false, "Mark as unread instead of read")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show what would be changed without making changes")

	return cmd
}

func newEmailThreadCmd(flags *rootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "thread <threadId>",
		Short: "Get all emails in a thread",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := getClient(flags)
			if err != nil {
				return err
			}

			emails, err := client.GetThread(cmd.Context(), args[0])
			if err != nil {
				return fmt.Errorf("failed to get thread: %w", err)
			}

			if isJSON(cmd.Context()) {
				return printJSON(cmd, map[string]any{
					"threadId": args[0],
					"emails":   emails,
				})
			}

			if len(emails) == 0 {
				outfmt.Errorf("No emails found in thread")
				return nil
			}

			fmt.Printf("Thread: %s (%d messages)\n\n", args[0], len(emails))

			tw := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
			fmt.Fprintln(tw, "ID\tSUBJECT\tFROM\tDATE")
			for _, email := range emails {
				from := formatEmailAddressList(email.From)
				date := formatEmailDate(email.ReceivedAt)
				fmt.Fprintf(tw, "%s\t%s\t%s\t%s\n",
					email.ID,
					sanitizeTab(truncateString(email.Subject, 40)),
					sanitizeTab(truncateString(from, 25)),
					date,
				)
			}
			tw.Flush()

			return nil
		},
	}

	return cmd
}

func newEmailAttachmentsCmd(flags *rootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "attachments <emailId>",
		Short: "List attachments for an email",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := getClient(flags)
			if err != nil {
				return err
			}

			attachments, err := client.GetEmailAttachments(cmd.Context(), args[0])
			if err != nil {
				return fmt.Errorf("failed to get attachments: %w", err)
			}

			if isJSON(cmd.Context()) {
				return printJSON(cmd, map[string]any{
					"emailId":     args[0],
					"attachments": attachments,
				})
			}

			if len(attachments) == 0 {
				fmt.Println("No attachments")
				return nil
			}

			tw := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
			fmt.Fprintln(tw, "BLOB ID\tNAME\tTYPE\tSIZE")
			for _, att := range attachments {
				fmt.Fprintf(tw, "%s\t%s\t%s\t%s\n",
					att.BlobID,
					sanitizeTab(att.Name),
					att.Type,
					formatSize(att.Size),
				)
			}
			tw.Flush()

			return nil
		},
	}

	return cmd
}

func newEmailMailboxesCmd(flags *rootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "mailboxes",
		Short: "List mailboxes (folders)",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := getClient(flags)
			if err != nil {
				return err
			}

			mailboxes, err := client.GetMailboxes(cmd.Context())
			if err != nil {
				return fmt.Errorf("failed to get mailboxes: %w", err)
			}

			if isJSON(cmd.Context()) {
				return printJSON(cmd, mailboxes)
			}

			tw := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
			fmt.Fprintln(tw, "ID\tNAME\tROLE\tUNREAD\tTOTAL")
			for _, mb := range mailboxes {
				fmt.Fprintf(tw, "%s\t%s\t%s\t%d\t%d\n",
					mb.ID,
					sanitizeTab(mb.Name),
					mb.Role,
					mb.UnreadEmails,
					mb.TotalEmails,
				)
			}
			tw.Flush()

			return nil
		},
	}

	return cmd
}

func newEmailImportCmd(flags *rootFlags) *cobra.Command {
	var mailbox string
	var markRead bool

	cmd := &cobra.Command{
		Use:   "import <file.eml>",
		Short: "Import an email from a .eml file",
		Long: `Import a raw RFC 5322 email message (.eml file) into your mailbox.

The email will be imported with its original headers and content.
By default, emails are imported to the Inbox and marked as unread.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := getClient(flags)
			if err != nil {
				return err
			}

			emlPath := args[0]

			// Verify file exists
			fileInfo, err := os.Stat(emlPath)
			if err != nil {
				return fmt.Errorf("cannot access file '%s': %w", emlPath, err)
			}
			if fileInfo.IsDir() {
				return fmt.Errorf("cannot import directory: %s", emlPath)
			}

			// Determine target mailbox
			targetMailboxID := mailbox
			if targetMailboxID == "" {
				// Default to inbox
				var inbox *jmap.Mailbox
				inbox, err = client.GetMailboxByName(cmd.Context(), "inbox")
				if err != nil {
					return fmt.Errorf("failed to find inbox: %w", err)
				}
				targetMailboxID = inbox.ID
			} else {
				// Resolve mailbox name/ID
				var resolvedID string
				resolvedID, err = client.ResolveMailboxID(cmd.Context(), targetMailboxID)
				if err != nil {
					return fmt.Errorf("invalid mailbox: %w", err)
				}
				targetMailboxID = resolvedID
			}

			// Open and upload the .eml file
			file, err := os.Open(emlPath)
			if err != nil {
				return fmt.Errorf("failed to open file '%s': %w", emlPath, err)
			}
			defer file.Close()

			uploadResult, err := client.UploadBlob(cmd.Context(), file, "message/rfc822")
			if err != nil {
				return fmt.Errorf("failed to upload email: %w", err)
			}

			// Build import options
			opts := jmap.ImportEmailOpts{
				BlobID:     uploadResult.BlobID,
				MailboxIDs: map[string]bool{targetMailboxID: true},
			}

			if markRead {
				opts.Keywords = map[string]bool{"$seen": true}
			}

			emailID, err := client.ImportEmail(cmd.Context(), opts)
			if err != nil {
				importErr := cerrors.WithContext(err, "importing email")
				if strings.Contains(err.Error(), "401") || strings.Contains(err.Error(), "unauthorized") {
					return cerrors.WithSuggestion(importErr, cerrors.SuggestionReauth)
				}
				return importErr
			}

			if isJSON(cmd.Context()) {
				return printJSON(cmd, map[string]any{
					"emailId":   emailID,
					"blobId":    uploadResult.BlobID,
					"mailboxId": targetMailboxID,
					"file":      emlPath,
				})
			}

			fmt.Printf("Imported email (ID: %s) from %s\n", emailID, emlPath)
			return nil
		},
	}

	cmd.Flags().StringVar(&mailbox, "mailbox", "", "Target mailbox ID or name (default: Inbox)")
	cmd.Flags().BoolVar(&markRead, "read", false, "Mark imported email as read")

	return cmd
}

// Helper functions

func formatEmailAddressList(addrs []jmap.EmailAddress) string {
	if len(addrs) == 0 {
		return ""
	}
	parts := make([]string, len(addrs))
	for i, addr := range addrs {
		if addr.Name != "" {
			parts[i] = fmt.Sprintf("%s <%s>", addr.Name, addr.Email)
		} else {
			parts[i] = addr.Email
		}
	}
	return strings.Join(parts, ", ")
}

func formatEmailDate(dateStr string) string {
	t, err := time.Parse(time.RFC3339, dateStr)
	if err != nil {
		return dateStr
	}
	return t.Format("2006-01-02 15:04")
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

func formatSize(size int64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
	)
	switch {
	case size >= GB:
		return fmt.Sprintf("%.1f GB", float64(size)/GB)
	case size >= MB:
		return fmt.Sprintf("%.1f MB", float64(size)/MB)
	case size >= KB:
		return fmt.Sprintf("%.1f KB", float64(size)/KB)
	default:
		return fmt.Sprintf("%d B", size)
	}
}

func newEmailDownloadCmd(flags *rootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "download <emailId> <blobId> [output-file]",
		Short: "Download an email attachment",
		Long: `Download an email attachment by blob ID.

If output-file is not specified, the attachment will be saved with its original name
in the current directory. You can get the blob ID from the 'attachments' command.`,
		Args: cobra.RangeArgs(2, 3),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := getClient(flags)
			if err != nil {
				return err
			}

			emailID := args[0]
			blobID := args[1]
			var outputFile string

			// If output file not specified, get the attachment name from the email
			if len(args) < 3 {
				var attachments []jmap.Attachment
				attachments, err = client.GetEmailAttachments(cmd.Context(), emailID)
				if err != nil {
					return fmt.Errorf("failed to get attachments: %w", err)
				}

				// Find the attachment with matching blob ID
				var found bool
				for _, att := range attachments {
					if att.BlobID == blobID {
						outputFile = att.Name
						found = true
						break
					}
				}

				if !found {
					return fmt.Errorf("blob ID '%s' not found in email '%s'", blobID, emailID)
				}

				if outputFile == "" {
					outputFile = "attachment"
				}
			} else {
				outputFile = args[2]
			}

			// SECURITY: Sanitize filename to prevent path traversal attacks
			outputFile = sanitizeFilename(outputFile)

			// Check if file already exists
			if _, statErr := os.Stat(outputFile); statErr == nil {
				return fmt.Errorf("file '%s' already exists. Specify a different output file", outputFile)
			}

			// Download the blob
			reader, err := client.DownloadBlob(cmd.Context(), blobID)
			if err != nil {
				downloadErr := cerrors.WithContext(err, "downloading attachment")
				if strings.Contains(err.Error(), "401") || strings.Contains(err.Error(), "unauthorized") {
					return cerrors.WithSuggestion(downloadErr, cerrors.SuggestionReauth)
				}
				return downloadErr
			}
			defer reader.Close()

			// Create output file
			outFile, err := os.Create(outputFile)
			if err != nil {
				return fmt.Errorf("failed to create output file: %w", err)
			}
			defer outFile.Close()

			// Copy content
			written, err := io.Copy(outFile, reader)
			if err != nil {
				return fmt.Errorf("failed to write attachment: %w", err)
			}

			if isJSON(cmd.Context()) {
				return printJSON(cmd, map[string]any{
					"emailId":    emailID,
					"blobId":     blobID,
					"outputFile": outputFile,
					"size":       written,
				})
			}

			fmt.Printf("Downloaded attachment to %s (%s)\n", outputFile, formatSize(written))
			return nil
		},
	}

	return cmd
}

// isValidEmail validates email addresses using RFC 5322 compliant regex.
// SECURITY: Rejects malformed addresses, control characters, and potential injection attempts.
func isValidEmail(email string) bool {
	// Length limits: RFC 5321 specifies max 254 characters for email address
	if len(email) < 3 || len(email) > 254 {
		return false
	}

	// SECURITY: Reject null bytes and control characters (potential injection)
	// Covers ASCII control chars (0x00-0x1F, 0x7F) and Unicode C1 controls (0x80-0x9F)
	for _, r := range email {
		if r < 32 || r == 127 || (r >= 0x80 && r <= 0x9F) {
			return false
		}
	}

	// SECURITY: Reject angle brackets (potential header injection)
	if strings.ContainsAny(email, "<>") {
		return false
	}

	// Validate against RFC 5322 pattern
	return emailRegex.MatchString(email)
}

// parseAttachmentFlag parses an attachment flag value.
// Format: /path/to/file[:displayname]
// Returns the file path and display name (defaults to basename if not specified).
func parseAttachmentFlag(value string) (path, name string, err error) {
	if value == "" {
		return "", "", fmt.Errorf("attachment path cannot be empty")
	}

	// Check for custom name separator (last colon that's not part of Windows drive letter)
	// Handle Windows paths like C:\path\file.pdf
	lastColon := strings.LastIndex(value, ":")

	// On Windows, skip the drive letter colon (e.g., C:)
	isWindowsDrive := lastColon == 1 && len(value) > 2 && (value[2] == '\\' || value[2] == '/')

	if lastColon > 1 && !isWindowsDrive {
		// Found a colon for custom name
		path = value[:lastColon]
		name = value[lastColon+1:]
		if name == "" {
			name = filepath.Base(path)
		}
		return path, name, nil
	}

	// No custom name specified (or Windows drive letter)
	path = value
	name = filepath.Base(path)
	return path, name, nil
}

// getMimeType returns the MIME type for a file based on extension.
func getMimeType(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))
	mimeTypes := map[string]string{
		".pdf":  "application/pdf",
		".doc":  "application/msword",
		".docx": "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
		".xls":  "application/vnd.ms-excel",
		".xlsx": "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
		".ppt":  "application/vnd.ms-powerpoint",
		".pptx": "application/vnd.openxmlformats-officedocument.presentationml.presentation",
		".png":  "image/png",
		".jpg":  "image/jpeg",
		".jpeg": "image/jpeg",
		".gif":  "image/gif",
		".svg":  "image/svg+xml",
		".txt":  "text/plain",
		".html": "text/html",
		".css":  "text/css",
		".js":   "application/javascript",
		".json": "application/json",
		".xml":  "application/xml",
		".zip":  "application/zip",
		".tar":  "application/x-tar",
		".gz":   "application/gzip",
		".mp3":  "audio/mpeg",
		".mp4":  "video/mp4",
		".wav":  "audio/wav",
	}
	if mime, ok := mimeTypes[ext]; ok {
		return mime
	}
	return "application/octet-stream"
}

// sanitizeFilename removes path components and dangerous characters to prevent
// path traversal attacks. Returns only the base filename.
// SECURITY: Handles null bytes, control characters, reserved names (Windows),
// and enforces length limits.
func sanitizeFilename(name string) string {
	// SECURITY: Remove null bytes first (can bypass filesystem checks)
	name = strings.ReplaceAll(name, "\x00", "")

	// SECURITY: Remove control characters (0x00-0x1F and 0x7F)
	var clean strings.Builder
	for _, r := range name {
		if r >= 32 && r != 127 {
			clean.WriteRune(r)
		}
	}
	name = clean.String()

	// Remove any path components (prevents ../../etc/passwd attacks)
	name = filepath.Base(name)

	// Trim whitespace (prevents " .bashrc" becoming valid after dot trim)
	name = strings.TrimSpace(name)

	// Remove leading dots (prevents hidden files)
	name = strings.TrimLeft(name, ".")

	// SECURITY: Check for Windows reserved names (CON, PRN, AUX, NUL, COM1-9, LPT1-9)
	// These can cause issues even on non-Windows systems when files are transferred
	reservedNames := []string{
		"CON", "PRN", "AUX", "NUL",
		"COM1", "COM2", "COM3", "COM4", "COM5", "COM6", "COM7", "COM8", "COM9",
		"LPT1", "LPT2", "LPT3", "LPT4", "LPT5", "LPT6", "LPT7", "LPT8", "LPT9",
	}
	nameUpper := strings.ToUpper(name)
	// Check both exact match and "RESERVED.ext" pattern
	for _, reserved := range reservedNames {
		if nameUpper == reserved || strings.HasPrefix(nameUpper, reserved+".") {
			name = "_" + name
			break
		}
	}

	// SECURITY: Limit filename length (most filesystems max 255 bytes)
	if len(name) > 255 {
		// Preserve extension if possible
		ext := filepath.Ext(name)
		if len(ext) < 20 && len(ext) > 0 {
			name = name[:255-len(ext)] + ext
		} else {
			name = name[:255]
		}
	}

	// Handle empty or dangerous names
	if name == "" || name == "." || name == ".." {
		return "attachment"
	}

	return name
}

func newMailboxCreateCmd(flags *rootFlags) *cobra.Command {
	var parentID string

	cmd := &cobra.Command{
		Use:   "mailbox-create <name>",
		Short: "Create a new mailbox (folder)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := getClient(flags)
			if err != nil {
				return err
			}

			opts := jmap.CreateMailboxOpts{
				Name:     args[0],
				ParentID: parentID,
			}

			mailbox, err := client.CreateMailbox(cmd.Context(), opts)
			if err != nil {
				return fmt.Errorf("failed to create mailbox: %w", err)
			}

			if isJSON(cmd.Context()) {
				return printJSON(cmd, mailbox)
			}

			fmt.Printf("Created mailbox '%s' (ID: %s)\n", mailbox.Name, mailbox.ID)
			return nil
		},
	}

	cmd.Flags().StringVar(&parentID, "parent", "", "Parent mailbox ID (for nested folders)")

	return cmd
}

func newMailboxDeleteCmd(flags *rootFlags) *cobra.Command {
	var yesFlag bool

	cmd := &cobra.Command{
		Use:   "mailbox-delete <mailbox-id-or-name>",
		Short: "Delete a mailbox (folder)",
		Long:  "Delete a mailbox. Emails in the mailbox will be moved to trash.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := getClient(flags)
			if err != nil {
				return err
			}

			// First resolve the mailbox ID
			mailboxID, err := client.ResolveMailboxID(cmd.Context(), args[0])
			if err != nil {
				return fmt.Errorf("invalid mailbox: %w", err)
			}

			// Then get mailboxes to find the name for the prompt
			mailboxes, err := client.GetMailboxes(cmd.Context())
			if err != nil {
				return fmt.Errorf("failed to get mailboxes: %w", err)
			}

			// Find the mailbox name for the confirmation prompt
			var mailboxName string
			for _, mb := range mailboxes {
				if mb.ID == mailboxID {
					mailboxName = mb.Name
					break
				}
			}

			// Prompt for confirmation unless --yes flag is set or JSON output mode
			if !isJSON(cmd.Context()) && !yesFlag {
				fmt.Fprintf(os.Stderr, "Delete mailbox '%s' (ID: %s)? [y/N] ", mailboxName, mailboxID)
				scanner := bufio.NewScanner(os.Stdin)
				if !scanner.Scan() {
					return fmt.Errorf("cancelled")
				}
				response := strings.ToLower(strings.TrimSpace(scanner.Text()))
				if response != "y" && response != "yes" {
					return fmt.Errorf("cancelled")
				}
			}

			err = client.DeleteMailbox(cmd.Context(), mailboxID)
			if err != nil {
				return fmt.Errorf("failed to delete mailbox: %w", err)
			}

			if isJSON(cmd.Context()) {
				return printJSON(cmd, map[string]any{
					"deleted": mailboxID,
				})
			}

			fmt.Printf("Deleted mailbox %s\n", mailboxID)
			return nil
		},
	}

	cmd.Flags().BoolVarP(&yesFlag, "yes", "y", false, "Skip confirmation prompt")

	return cmd
}

func newMailboxRenameCmd(flags *rootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "mailbox-rename <mailbox-id-or-name> <new-name>",
		Short: "Rename a mailbox (folder)",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := getClient(flags)
			if err != nil {
				return err
			}

			// Resolve mailbox ID or name
			mailboxID, err := client.ResolveMailboxID(cmd.Context(), args[0])
			if err != nil {
				return fmt.Errorf("invalid mailbox: %w", err)
			}

			err = client.RenameMailbox(cmd.Context(), mailboxID, args[1])
			if err != nil {
				return fmt.Errorf("failed to rename mailbox: %w", err)
			}

			if isJSON(cmd.Context()) {
				return printJSON(cmd, map[string]any{
					"mailboxId": mailboxID,
					"newName":   args[1],
				})
			}

			fmt.Printf("Renamed mailbox %s to '%s'\n", mailboxID, args[1])
			return nil
		},
	}

	return cmd
}

func newEmailIdentitiesCmd(flags *rootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "identities",
		Short: "List sending identities (aliases)",
		Long:  "List all email identities/aliases you can send from.",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := getClient(flags)
			if err != nil {
				return err
			}

			identities, err := client.GetIdentities(cmd.Context())
			if err != nil {
				idErr := cerrors.WithContext(err, "fetching identities")
				if strings.Contains(err.Error(), "401") || strings.Contains(err.Error(), "unauthorized") {
					return cerrors.WithSuggestion(idErr, cerrors.SuggestionReauth)
				}
				return idErr
			}

			if isJSON(cmd.Context()) {
				return printJSON(cmd, identities)
			}

			if len(identities) == 0 {
				outfmt.Errorf("No identities found")
				return nil
			}

			tw := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
			fmt.Fprintln(tw, "ID\tEMAIL\tNAME\tDEFAULT")
			for _, id := range identities {
				// MayDelete=false indicates the primary account identity
				isDefault := ""
				if !id.MayDelete {
					isDefault = "*"
				}
				fmt.Fprintf(tw, "%s\t%s\t%s\t%s\n",
					id.ID,
					id.Email,
					sanitizeTab(id.Name),
					isDefault,
				)
			}
			tw.Flush()

			return nil
		},
	}

	return cmd
}
