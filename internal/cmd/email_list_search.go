package cmd

import (
	"fmt"
	"time"

	cerrors "github.com/salmonumbrella/fastmail-cli/internal/errors"
	"github.com/salmonumbrella/fastmail-cli/internal/format"
	"github.com/salmonumbrella/fastmail-cli/internal/jmap"
	"github.com/salmonumbrella/fastmail-cli/internal/outfmt"
	"github.com/spf13/cobra"
)

func newEmailListCmd(app *App) *cobra.Command {
	var limit int
	var mailboxID string
	var light bool

	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List emails",
		RunE: runE(app, func(cmd *cobra.Command, args []string, app *App) error {
			client, err := app.JMAPClient()
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
				return cerrors.WithContext(err, "listing emails")
			}

			// Fetch thread message counts
			threadIDs := make([]string, 0, len(emails))
			for _, email := range emails {
				threadIDs = append(threadIDs, email.ThreadID)
			}
			threadCounts, err := client.GetThreadMessageCounts(cmd.Context(), threadIDs)
			if err != nil {
				// Non-fatal: continue without thread counts
				threadCounts = map[string]int{}
			}

			if app.IsJSON(cmd.Context()) {
				if light {
					return app.PrintJSON(cmd, emailsToLightWithCounts(emails, threadCounts))
				}
				return app.PrintJSON(cmd, emailsToOutputWithCounts(emails, threadCounts))
			}

			if len(emails) == 0 {
				printNoResults("No emails found")
				return nil
			}

			tw := outfmt.NewTabWriter()
			fmt.Fprintln(tw, "ID\tSUBJECT\tFROM\tDATE\tUNREAD\tTHREAD")
			for _, email := range emails {
				from := format.FormatEmailAddressList(email.From)
				date := format.FormatEmailDate(email.ReceivedAt)
				unread := ""
				if email.Keywords != nil && !email.Keywords["$seen"] {
					unread = "*"
				}
				thread := formatThreadCount(threadCounts[email.ThreadID])
				fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\t%s\n",
					email.ID,
					outfmt.SanitizeTab(format.Truncate(email.Subject, 50)),
					outfmt.SanitizeTab(format.Truncate(from, 30)),
					date,
					unread,
					thread,
				)
			}
			tw.Flush()

			return nil
		}),
	}

	cmd.Flags().IntVar(&limit, "limit", 25, "Maximum number of emails to list")
	cmd.Flags().StringVar(&mailboxID, "mailbox", "", "Mailbox ID or name to filter emails")
	addLightFlag(cmd, &light)

	return cmd
}

func newEmailSearchCmd(app *App) *cobra.Command {
	var limit int
	var snippets bool
	var light bool

	cmd := &cobra.Command{
		Use:     "search <query>",
		Aliases: []string{"find", "s"},
		Short:   "Search emails",
		Long: `Search emails using JMAP query syntax.

Examples:
  fastmail email search "from:alice@example.com"
  fastmail email search --snippets "invoice"
  fastmail email search "subject:meeting after:2025-01-01"
  fastmail email search "subject:meeting after:yesterday"
  fastmail email search "subject:meeting after:'2h ago'"`,
		Args: cobra.ExactArgs(1),
		RunE: runE(app, func(cmd *cobra.Command, args []string, app *App) error {
			client, err := app.JMAPClient()
			if err != nil {
				return err
			}

			var emails []jmap.Email
			var searchSnippets []jmap.SearchSnippet

			// Parse the query into JMAP filter components
			filter, err := parseEmailSearchFilter(args[0], time.Now())
			if err != nil {
				return err
			}

			if snippets {
				emails, searchSnippets, err = client.SearchEmailsWithSnippets(cmd.Context(), filter, limit)
			} else {
				emails, err = client.SearchEmails(cmd.Context(), filter, limit)
			}

			if err != nil {
				return cerrors.WithContext(err, "searching emails")
			}

			// Fetch thread message counts
			threadIDs := make([]string, 0, len(emails))
			for _, email := range emails {
				threadIDs = append(threadIDs, email.ThreadID)
			}
			threadCounts, err := client.GetThreadMessageCounts(cmd.Context(), threadIDs)
			if err != nil {
				// Non-fatal: continue without thread counts
				threadCounts = map[string]int{}
			}

			if app.IsJSON(cmd.Context()) {
				var emailData any
				if light {
					emailData = emailsToLightWithCounts(emails, threadCounts)
				} else {
					emailData = emailsToOutputWithCounts(emails, threadCounts)
				}
				result := map[string]any{"emails": emailData}
				if snippets && len(searchSnippets) > 0 {
					result["snippets"] = searchSnippets
				}
				return app.PrintJSON(cmd, result)
			}

			if len(emails) == 0 {
				printNoResults("No emails found matching '%s'", args[0])
				return nil
			}

			// Build snippet map for quick lookup
			snippetMap := make(map[string]jmap.SearchSnippet)
			for _, s := range searchSnippets {
				snippetMap[s.EmailID] = s
			}

			tw := outfmt.NewTabWriter()
			fmt.Fprintln(tw, "ID\tSUBJECT\tFROM\tDATE\tUNREAD\tTHREAD")
			for _, email := range emails {
				from := format.FormatEmailAddressList(email.From)
				date := format.FormatEmailDate(email.ReceivedAt)
				unread := ""
				if email.Keywords != nil && !email.Keywords["$seen"] {
					unread = "*"
				}
				thread := formatThreadCount(threadCounts[email.ThreadID])

				subject := email.Subject
				if snippets {
					if s, ok := snippetMap[email.ID]; ok && s.Subject != "" {
						subject = s.Subject // Use highlighted subject
					}
				}

				fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\t%s\n",
					email.ID,
					outfmt.SanitizeTab(format.Truncate(subject, 50)),
					outfmt.SanitizeTab(format.Truncate(from, 30)),
					date,
					unread,
					thread,
				)

				// Show snippet preview if available
				if snippets {
					if s, ok := snippetMap[email.ID]; ok && s.Preview != "" {
						fmt.Fprintf(tw, "\t%s\t\t\t\t\n", outfmt.SanitizeTab(format.Truncate(s.Preview, 80)))
					}
				}
			}
			tw.Flush()

			return nil
		}),
	}

	cmd.Flags().IntVar(&limit, "limit", 25, "Maximum number of results")
	cmd.Flags().BoolVar(&snippets, "snippets", false, "Show highlighted search snippets")
	addLightFlag(cmd, &light)

	return cmd
}

// formatThreadCount formats a thread message count for display.
// Returns "-" for single-message threads, "[N msgs]" for multi-message threads.
func formatThreadCount(count int) string {
	if count <= 1 {
		return "-"
	}
	return fmt.Sprintf("[%d msgs]", count)
}
