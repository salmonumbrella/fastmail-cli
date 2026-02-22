package cmd

import (
	"fmt"

	"github.com/salmonumbrella/fastmail-cli/internal/config"
	"github.com/salmonumbrella/fastmail-cli/internal/jmap"
	"github.com/salmonumbrella/fastmail-cli/internal/validation"
	"github.com/spf13/cobra"
)

func newDraftCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "draft",
		Aliases: []string{"drafts"},
		Short:   "Draft email management",
	}

	cmd.AddCommand(newDraftListCmd(app))
	cmd.AddCommand(newDraftGetCmd(app))
	cmd.AddCommand(newDraftNewCmd(app))
	cmd.AddCommand(newDraftSendCmd(app))
	cmd.AddCommand(newDraftDeleteCmd(app))

	return cmd
}

func newDraftListCmd(app *App) *cobra.Command {
	var limit int
	var light bool

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List draft emails",
		RunE: runE(app, func(cmd *cobra.Command, args []string, app *App) error {
			client, err := app.JMAPClient()
			if err != nil {
				return err
			}

			drafts, err := client.GetDrafts(cmd.Context(), limit)
			if err != nil {
				return fmt.Errorf("failed to get drafts: %w", err)
			}

			// Fetch thread message counts (non-fatal)
			threadIDs := make([]string, 0, len(drafts))
			for _, email := range drafts {
				threadIDs = append(threadIDs, email.ThreadID)
			}
			threadCounts, err := client.GetThreadMessageCounts(cmd.Context(), threadIDs)
			if err != nil {
				threadCounts = map[string]int{}
			}

			if app.IsJSON(cmd.Context()) {
				if light {
					return app.PrintJSON(cmd, emailsToLightWithCounts(drafts, threadCounts))
				}
				return app.PrintJSON(cmd, emailsToOutputWithCounts(drafts, threadCounts))
			}

			if len(drafts) == 0 {
				printNoResults("No drafts found")
				return nil
			}

			printEmailList(drafts, threadCounts)
			return nil
		}),
	}

	cmd.Flags().IntVar(&limit, "limit", 20, "Maximum number of drafts to return")
	addLightFlag(cmd, &light)

	return cmd
}

func newDraftGetCmd(app *App) *cobra.Command {
	var light bool

	cmd := &cobra.Command{
		Use:   "get <draft-id>",
		Short: "Get a draft by ID",
		Args:  cobra.ExactArgs(1),
		RunE: runE(app, func(cmd *cobra.Command, args []string, app *App) error {
			client, err := app.JMAPClient()
			if err != nil {
				return err
			}

			draft, err := client.GetEmailByID(cmd.Context(), args[0])
			if err != nil {
				return fmt.Errorf("failed to get draft: %w", err)
			}

			// Verify it's actually a draft
			if draft.Keywords != nil && !draft.Keywords["$draft"] {
				return fmt.Errorf("email %s is not a draft", args[0])
			}

			if app.IsJSON(cmd.Context()) {
				if light {
					return app.PrintJSON(cmd, emailToLight(*draft))
				}
				return app.PrintJSON(cmd, emailToOutput(*draft))
			}

			printEmailDetails(draft)
			return nil
		}),
	}

	addLightFlag(cmd, &light)

	return cmd
}

func newDraftNewCmd(app *App) *cobra.Command {
	var to, cc, bcc []string
	var subject, body, htmlBody string
	var fromIdentity string
	var replyTo string

	cmd := &cobra.Command{
		Use:   "new",
		Short: "Create a new draft",
		Long: `Create a new draft email.

Examples:
  fastmail draft new --to user@example.com --subject "Hello" --body "Hi there"
  fastmail draft new --to user@example.com --subject "Hello" --html "<h1>Hi</h1>"
  fastmail draft new --reply-to <email-id> --body "Thanks!"  # Creates a threaded reply draft`,
		RunE: runE(app, func(cmd *cobra.Command, args []string, app *App) error {
			client, err := app.JMAPClient()
			if err != nil {
				return err
			}

			if replyTo == "" && subject == "" {
				return fmt.Errorf("--subject is required (or use --reply-to)")
			}
			if body == "" && htmlBody == "" {
				return fmt.Errorf("--body or --html is required")
			}

			// Validate email addresses (only those provided)
			allAddrs := append(append(to, cc...), bcc...)
			for _, addr := range allAddrs {
				if !validation.IsValidEmail(addr) {
					return fmt.Errorf("invalid email address: %s", addr)
				}
			}

			// Apply default identity if --from not specified
			effectiveFrom := fromIdentity
			if effectiveFrom == "" {
				accountEmail, accountErr := app.RequireAccount()
				if accountErr == nil {
					if defaultIdentity, _ := config.GetDefaultIdentity(accountEmail); defaultIdentity != "" {
						effectiveFrom = defaultIdentity
					}
				}
			}

			opts := jmap.SendEmailOpts{
				To:       to,
				CC:       cc,
				BCC:      bcc,
				Subject:  subject,
				TextBody: body,
				HTMLBody: htmlBody,
				From:     effectiveFrom,
			}

			var draftID string
			if replyTo != "" {
				draftID, err = client.CreateReplyDraft(cmd.Context(), replyTo, opts)
			} else {
				draftID, err = client.SaveDraft(cmd.Context(), opts)
			}
			if err != nil {
				return fmt.Errorf("failed to create draft: %w", err)
			}

			if app.IsJSON(cmd.Context()) {
				return app.PrintJSON(cmd, map[string]any{
					"draftId": draftID,
					"status":  "created",
				})
			}

			fmt.Printf("Draft created (ID: %s)\n", draftID)
			return nil
		}),
	}

	cmd.Flags().StringSliceVar(&to, "to", nil, "Recipient email addresses")
	cmd.Flags().StringSliceVar(&cc, "cc", nil, "CC email addresses")
	cmd.Flags().StringSliceVar(&bcc, "bcc", nil, "BCC email addresses")
	cmd.Flags().StringVar(&subject, "subject", "", "Email subject")
	cmd.Flags().StringVar(&body, "body", "", "Email body (plain text)")
	cmd.Flags().StringVar(&htmlBody, "html", "", "Email body (HTML)")
	cmd.Flags().StringVar(&fromIdentity, "from", "", "Send from this identity or masked email address")
	cmd.Flags().StringVar(&replyTo, "reply-to", "", "Email ID to reply to (threads the draft)")

	return cmd
}

func newDraftSendCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "send <draft-id>",
		Short: "Send a draft email",
		Aliases: []string{
			"submit",
		},
		Args: cobra.ExactArgs(1),
		RunE: runE(app, func(cmd *cobra.Command, args []string, app *App) error {
			client, err := app.JMAPClient()
			if err != nil {
				return err
			}

			draftID := args[0]

			// Get draft details for confirmation
			draft, err := client.GetEmailByID(cmd.Context(), draftID)
			if err != nil {
				return fmt.Errorf("failed to get draft: %w", err)
			}

			// Confirm before sending. Only print details in interactive mode.
			if !app.IsJSON(cmd.Context()) && (app.Flags == nil || !app.Flags.Yes) {
				toAddrs := make([]string, len(draft.To))
				for i, addr := range draft.To {
					toAddrs[i] = addr.Email
				}
				fmt.Printf("Subject: %s\n", draft.Subject)
				fmt.Printf("To: %v\n", toAddrs)
			}
			confirmed, confirmErr := app.Confirm(cmd, false, "Send this draft? [y/N] ", "y", "yes")
			if confirmErr != nil {
				return confirmErr
			}
			if !confirmed {
				fmt.Println("Cancelled")
				return nil
			}

			submissionID, err := client.SendDraft(cmd.Context(), draftID)
			if err != nil {
				return fmt.Errorf("failed to send draft: %w", err)
			}

			if app.IsJSON(cmd.Context()) {
				return app.PrintJSON(cmd, map[string]any{
					"submissionId": submissionID,
					"status":       "sent",
				})
			}

			fmt.Printf("Draft sent successfully\n")
			if submissionID != "" {
				fmt.Printf("Submission ID: %s\n", submissionID)
			}
			return nil
		}),
	}

	return cmd
}

func newDraftDeleteCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete <draft-id>",
		Short: "Delete a draft email",
		Aliases: []string{
			"rm",
			"remove",
		},
		Args: cobra.ExactArgs(1),
		RunE: runE(app, func(cmd *cobra.Command, args []string, app *App) error {
			client, err := app.JMAPClient()
			if err != nil {
				return err
			}

			draftID := args[0]

			confirmed, err := app.Confirm(cmd, false, "Delete this draft? [y/N] ", "y", "yes")
			if err != nil {
				return err
			}
			if !confirmed {
				fmt.Println("Cancelled")
				return nil
			}

			// DeleteEmail moves to trash
			if err := client.DeleteEmail(cmd.Context(), draftID); err != nil {
				return fmt.Errorf("failed to delete draft: %w", err)
			}

			if app.IsJSON(cmd.Context()) {
				return app.PrintJSON(cmd, map[string]any{
					"draftId": draftID,
					"status":  "deleted",
				})
			}

			fmt.Println("Draft deleted")
			return nil
		}),
	}

	return cmd
}
