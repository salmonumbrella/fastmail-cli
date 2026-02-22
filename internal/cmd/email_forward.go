package cmd

import (
	"fmt"
	"strings"

	cerrors "github.com/salmonumbrella/fastmail-cli/internal/errors"
	"github.com/salmonumbrella/fastmail-cli/internal/jmap"
	"github.com/salmonumbrella/fastmail-cli/internal/validation"
	"github.com/spf13/cobra"
)

func newEmailForwardCmd(app *App) *cobra.Command {
	var to []string
	var fromIdentity string
	var body string

	cmd := &cobra.Command{
		Use:     "forward <emailId>",
		Aliases: []string{"fwd"},
		Short:   "Forward an email",
		Long: `Forward an email to one or more recipients.

By default, if the original email was received on a masked email address, the
forwarded email will be sent from that same masked email to maintain privacy.
Use --from to override this behavior.

Attachments from the original email are automatically included.

Examples:
  fastmail email forward Mf1234abc --to recipient@example.com
  fastmail email forward Mf1234abc --to user1@example.com --to user2@example.com
  fastmail email forward Mf1234abc --to recipient@example.com --body "FYI, see below"
  fastmail email forward Mf1234abc --to recipient@example.com --from my.identity@fastmail.com`,
		Args: cobra.ExactArgs(1),
		RunE: runE(app, func(cmd *cobra.Command, args []string, app *App) error {
			emailID := args[0]

			if len(to) == 0 {
				return fmt.Errorf("--to is required")
			}

			// Validate email addresses
			for _, addr := range to {
				if !validation.IsValidEmail(addr) {
					return fmt.Errorf("invalid email address: %s", addr)
				}
			}

			client, err := app.JMAPClient()
			if err != nil {
				return err
			}

			// Fetch the original email
			original, err := client.GetEmailByID(cmd.Context(), emailID)
			if err != nil {
				return cerrors.WithContext(err, "fetching email")
			}

			// Build forward options
			opts := jmap.ForwardEmailOpts{
				To:   to,
				From: fromIdentity,
				Body: body,
			}

			resolvedFrom, fromSource, err := client.ResolveForwardFrom(cmd.Context(), original, opts)
			if err != nil {
				return cerrors.WithContext(err, "resolving forward from")
			}

			// Use resolved address to avoid duplicate resolution
			opts.From = resolvedFrom

			// Forward the email
			submissionID, err := client.ForwardEmail(cmd.Context(), original, opts)
			if err != nil {
				return cerrors.WithContext(err, "forwarding email")
			}

			result := map[string]any{
				"submissionId":    submissionID,
				"status":          "sent",
				"originalEmailId": emailID,
				"forwardedTo":     to,
				"from":            resolvedFrom,
				"fromSource":      fromSource,
			}

			if app.IsJSON(cmd.Context()) {
				return app.PrintJSON(cmd, result)
			}

			fmt.Printf("Email forwarded successfully (submission ID: %s)\n", submissionID)
			fmt.Printf("  From: %s (%s)\n", resolvedFrom, fromSource)
			fmt.Printf("  To: %s\n", strings.Join(to, ", "))
			if len(original.Attachments) > 0 {
				fmt.Printf("  Attachments: %d included\n", len(original.Attachments))
			}

			return nil
		}),
	}

	cmd.Flags().StringSliceVar(&to, "to", nil, "Recipient email addresses (required)")
	cmd.Flags().StringVar(&fromIdentity, "from", "", "Send from this identity or masked email (default: auto-detect from original)")
	cmd.Flags().StringVar(&body, "body", "", "Optional message to prepend to the forwarded email")

	return cmd
}
