package cmd

import (
	"fmt"

	"github.com/salmonumbrella/fastmail-cli/internal/format"
	"github.com/salmonumbrella/fastmail-cli/internal/outfmt"
	"github.com/spf13/cobra"
)

func newEmailThreadCmd(app *App) *cobra.Command {
	var light bool

	cmd := &cobra.Command{
		Use:     "thread <threadId>",
		Aliases: []string{"t"},
		Short:   "Get all emails in a thread",
		Args:    cobra.ExactArgs(1),
		RunE: runE(app, func(cmd *cobra.Command, args []string, app *App) error {
			client, err := app.JMAPClient()
			if err != nil {
				return err
			}

			emails, err := client.GetThread(cmd.Context(), args[0])
			if err != nil {
				return fmt.Errorf("failed to get thread: %w", err)
			}

			if app.IsJSON(cmd.Context()) {
				var emailData any
				if light {
					emailData = emailsToLight(emails)
				} else {
					emailData = emailsToOutput(emails)
				}
				return app.PrintJSON(cmd, map[string]any{
					"threadId": args[0],
					"emails":   emailData,
				})
			}

			if len(emails) == 0 {
				printNoResults("No emails found in thread")
				return nil
			}

			fmt.Printf("Thread: %s (%d messages)\n\n", args[0], len(emails))

			tw := outfmt.NewTabWriter()
			fmt.Fprintln(tw, "ID\tSUBJECT\tFROM\tDATE")
			for _, email := range emails {
				from := format.FormatEmailAddressList(email.From)
				date := format.FormatEmailDate(email.ReceivedAt)
				fmt.Fprintf(tw, "%s\t%s\t%s\t%s\n",
					email.ID,
					outfmt.SanitizeTab(format.Truncate(email.Subject, 40)),
					outfmt.SanitizeTab(format.Truncate(from, 25)),
					date,
				)
			}
			tw.Flush()

			return nil
		}),
	}

	addLightFlag(cmd, &light)

	return cmd
}
